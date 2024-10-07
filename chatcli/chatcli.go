package chatcli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"

	"io/ioutil"

	"connectrpc.com/connect"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/charmbracelet/huh"
	"github.com/instructor-ai/instructor-go/pkg/instructor"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/sashabaranov/go-openai"
	"github.com/wricardo/code-surgeon/api"
	"github.com/wricardo/code-surgeon/api/apiconnect"
	"github.com/wricardo/code-surgeon/log2"
)

var GLOBAL_CHAT *ChatImpl

// INTENT_DATA is the data used to train the AI model to detect intents
var INTENT_DATA string = `
g: "exit" i: "exit"
g: "write some code" i: "code"
g: "i want to write some code" i: "code"
g: "I want to query neo4j" i: "cypher"
g: "what can you do" i: "help"
g: "ask a question" i: "question_answer"
g: "add a question and answer to knowledge base" i: "question_answer"
g: "I want to query postgres" i: "postgres"
g: "I want to fetch data from postgres" i: "postgres"
g: "I want run some bash script" i: "bash"
g: "I want to know information about a local golang package, directory or file." i: "codeparser"
g: "I want to know the signature/info of function XYZ from folder X" i: "codeparser"
g: "I want to know the signature/info of function XYZ from file X" i: "codeparser"
g: "I want to parse the code in ./xyz directory" i: "codeparser"
g: "I want to parse the code in ./xyz.go file" i: "codeparser"
g: "I want to see the summary of the conversation" i: "debug"
g: "I want to see the history of the conversation" i: "debug"
g: "I want to save information to the knowledge base(KB)" i: "teacher"
g: "I want to make an http request" i: "resty"
`

const (
	SenderAI  = "AI"
	SenderYou = "You"
)

var NOOP *api.Command = &api.Command{Name: "noop"}
var QUIT *api.Command = &api.Command{Name: "quit"}
var MODE_QUIT *api.Command = &api.Command{Name: "mode_quit"}
var MODE_START *api.Command = &api.Command{Name: "mode_start"}
var SILENT *api.Command = &api.Command{Name: "silent"} // this will not save to history

type TMode string

var EXIT TMode = "exit"

// Keywords for detecting different modes
// modes are added here by init() functions in the mode files using the function RegisterMode
var modeKeywords = map[string]TMode{
	"/exit":  EXIT,
	"/quit":  EXIT,
	"/bye":   EXIT,
	"/debug": DEBUG,
	"/help":  HELP,
}

type ChatState struct {
	History             []*api.Message `json:"history"`
	ConversationSummary string         `json:"conversation_summary"`
}

// SaveState saves the chat state to a file
func (c *ChatImpl) SaveState(filename string) error {
	return nil
	if c == nil {
		panic("chat is nil, on chat.SaveState")
	}
	log.Debug().Str("filename", filename).Msg("SaveState")
	c.mutex.Lock()
	defer c.mutex.Unlock()

	state := ChatState{
		History:             c.history,
		ConversationSummary: c.conversationSummary,
	}

	data, err := json.Marshal(state)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filename, data, 0644)
}

// LoadState loads the chat state from a file
func (c *ChatImpl) LoadState(filename string) error {
	log.Debug().Str("filename", filename).Msg("LOAAADD")
	c.mutex.Lock()
	defer c.mutex.Unlock()

	data, err := ioutil.ReadFile(filename)
	log.Debug().Str("filename", filename).Msg("LoadState")
	if err != nil {
		return err
	}

	var state ChatState
	if err := json.Unmarshal(data, &state); err != nil {
		return err
	}

	c.history = state.History
	c.conversationSummary = state.ConversationSummary
	logger := log.Debug().Str("conversationSummary", c.conversationSummary).Int("historyLength", len(c.history))
	if log.Logger.GetLevel() == zerolog.TraceLevel {
		logger.Str("history", fmt.Sprintf("%v", c.history))
	}
	logger.Msg("LoadState")
	return nil
}

func ModeFromString(mode string) TMode {
	mode = "/" + mode
	if m, ok := modeKeywords[mode]; ok {
		return m
	}
	return ""
}

var modeRegistry = make(map[TMode]func(*ChatImpl) IMode)

// RegisterMode registers a mode constructor in the registry
func RegisterMode[T IMode](name TMode, constructor func(*ChatImpl) T) {
	// Explicitly convert the constructor to func(*Chat) Mode
	modeRegistry[name] = func(chat *ChatImpl) IMode {
		return constructor(chat) // Cast to Mode
	}
	if _, ok := modeKeywords[string(name)]; !ok {
		modeKeywords["/"+string(name)] = name
	}
}

// Mode is a chatbot specialized for a particular task, like coding or answering questions or playing a game o top of your data
type IMode interface {
	Start() (*api.Message, *api.Command, error)
	BestShot(msg *api.Message) (*api.Message, *api.Command, error)
	HandleIntent(msg *api.Message, intent Intent) (*api.Message, *api.Command, error)
	HandleResponse(input *api.Message) (*api.Message, *api.Command, error)
	Stop() error
}

// Message represents a message either from User to Ai or Ai to User

func TextMessage(text string) *api.Message {
	return &api.Message{
		Text: text,
	}
}

// IChat defines the interface for chat operations.
type IChat interface {
	HandleUserMessage(msg *api.Message) (responseMsg *api.Message, responseCmd *api.Command, err error)
	GetHistory() []*api.Message
	PrintHistory()
	SaveState(filename string) error
	LoadState(filename string) error

	// TODO: Remove this from the interface
	GetModeText() string
}

// ModeHandler is an interface for different types of modes
type ModeHandler interface {
	Start() (*api.Message, error)
	HandleResponse(msg *api.Message) (*api.Message, *api.Command, error)
}

// MessagePayload represents a single chat message
type MessagePayload struct {
	Sender  string
	Message *api.Message
}

// ChatImpl handles the chat functionality
type ChatImpl struct {
	id         string
	test       bool // test mode, tests will set this to true
	driver     *neo4j.DriverWithContext
	instructor *instructor.InstructorOpenAI

	mutex               sync.Mutex
	history             []*api.Message
	conversationSummary string

	modeManager *ModeManager
}

func (c *ChatImpl) checkIfExit(msg *api.Message) (bool, *api.Message) {
	userMessage := strings.TrimSpace(msg.Text)
	if userMessage == "/exit" || userMessage == "/quit" || userMessage == "/bye" || userMessage == "/stop" {
		return true, TextMessage("Exited.")
	}
	return false, TextMessage("")
}

// HandleUserMessage handles the user message. This is the main loop of the chat where we detect modes, intents, and handle user input and AI responses.
func (c *ChatImpl) HandleUserMessage(msg *api.Message) (responseMsg *api.Message, responseCmd *api.Command, err error) {
	defer func() {
		log.Debug().
			Any("responseMsg", &responseMsg).
			Any("responseCmd", responseCmd).
			Any("err", err).
			Msg("Chat.HandleUserMessage completed.")
	}()
	log.Debug().Any("msg", msg).Msg("Chat.HandleUserMessage started.")

	if c == nil {
		return nil, NOOP, fmt.Errorf("chat is nil on HandleUserMessage")
	}
	c.generateConversationSummary()

	if c.modeManager == nil {
		return nil, NOOP, fmt.Errorf("modeManager is nil on HandleUserMessage")
	}

	// If in a mode, delegate input handling to the mode manager
	if c.modeManager.currentMode != nil {
		// if user wants to exit mode
		if exit, response := c.checkIfExit(msg); exit {
			c.modeManager.StopMode()
			return response, MODE_QUIT, nil
		}

		response, command, err := c.modeManager.HandleInput(msg)
		if command != SILENT {
			//  the add message being after handle input causes add history from inside the mode be ahead of the user message
			c.addMessage("You", msg)
			c.addMessage("AI", response)
		}
		return response, command, err
	}

	// if user wants to exit mode
	if exit, response := c.checkIfExit(msg); exit {
		return response, MODE_QUIT, nil
	}

	// Detect and start new modes using modeManager
	if modeName, detected := c.DetectMode(msg); detected {
		mode, err := c.modeManager.CreateMode(c, modeName)
		if err != nil {
			return &api.Message{}, NOOP, err
		}
		response, command, err := c.modeManager.BestShot(mode, msg)
		if err != nil {
			return &api.Message{}, NOOP, err
		}
		if command != SILENT {
			c.addMessage("You", msg)
			c.addMessage("AI", response)
		}
		log.Debug().Str("mode", fmt.Sprintf("%T", mode)).Msg("Started new mode from detect")
		return response, command, nil
	}

	// detect intent
	intent2, detected := c.DetectIntent(msg)
	if detected {
		// handle intent
		if intent2.TMode != "" {
			response, command, err := c.modeManager.HandleIntent(intent2, msg)
			if err != nil {
				return &api.Message{}, NOOP, err
			}
			if command != SILENT {
				c.addMessage("You", msg)
				c.addMessage("AI", response)
			}
			return response, command, nil
		}
	}

	// Regular top level AI response
	c.AddMessageYou(msg)
	aiResponse := c.getAIResponse(msg)
	c.AddMessageAI(aiResponse)
	return aiResponse, NOOP, nil
}

// NewChat creates a new Chat instance
func NewChat(driver *neo4j.DriverWithContext, instructorClient *instructor.InstructorOpenAI) *ChatImpl {
	return &ChatImpl{
		driver:      driver,
		mutex:       sync.Mutex{},
		history:     []*api.Message{},
		modeManager: &ModeManager{},
		instructor:  instructorClient,
	}
}

func (c *ChatImpl) GetModeText() string {
	if c.modeManager.currentMode != nil {
		return fmt.Sprintf("%T", c.modeManager.currentMode)
	}
	return ""
}

// internal function to chat with AI
func (c *ChatImpl) Chat(aiOut interface{}, msgs []openai.ChatCompletionMessage) error {
	ctx := context.Background()
	history := c.GetHistory()
	summary := c.GetConversationSummary()
	gptMessages := make([]openai.ChatCompletionMessage, 0, len(history)+2)

	defer func() {
		logger := log.Debug().Any("aiOut", aiOut)
		logger.RawJSON("gpt_messages", func() []byte {
			b, _ := json.Marshal(gptMessages)
			return b
		}())
		if len(msgs) > 0 {
			logger = logger.Any("last_msg", msgs[len(msgs)-1])
		}

		logger.Msg("Chat completed.")
	}()

	// add history, last 10 messages
	from := len(history) - 10
	if from < 0 {
		from = 0
	}
	history = history[from:]
	for _, msg := range history {
		role := openai.ChatMessageRoleUser
		if msg.Sender != SenderYou {
			role = openai.ChatMessageRoleAssistant
		}
		gptMessages = append(gptMessages, openai.ChatCompletionMessage{
			Role:    role,
			Content: msg.ChatString(),
		})
	}
	gptMessages = append(gptMessages, openai.ChatCompletionMessage{
		Role:    "user",
		Content: "information for context: " + summary,
	})

	gptMessages = append(gptMessages, msgs...)

	_, err := c.instructor.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:     openai.GPT4o,
		Messages:  gptMessages,
		MaxTokens: 1000,
	}, aiOut)

	if err != nil {
		return fmt.Errorf("Failed to generate cypher query: %v", err)
	}

	return nil
}

func (c *ChatImpl) AddMessageYou(msg *api.Message) {
	c.addMessage(SenderYou, msg)
}

func (c *ChatImpl) AddMessageAI(msg *api.Message) {
	c.addMessage(SenderAI, msg)
}

// addMessage adds a message to the chat history
func (c *ChatImpl) addMessage(sender string, msg *api.Message) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	msg.Sender = sender
	log2.Debugf("Adding message to chat history: %s %s", sender, msg.String())

	c.history = append(c.history, msg)
}

func (c *ChatImpl) generateConversationSummary() {
	if DISABLE_CONVERSATION_SUMMARY {
		return
	}
	latestMessages := ""
	for _, mp := range c.GetLastNMessages(4) {
		latestMessages += fmt.Sprintf("%s: %s\n", mp.Sender, mp.ChatString())
	}

	type AiOutput struct {
		Summary string `json:"summary" jsonschema:"title=summary,description=the summary of the conversation."`
	}
	ctx := context.Background()

	var aiOut AiOutput
	prompt := fmt.Sprintf(`
	Conversation Summary:
	%s
	Converation Context:
	%s
	Latest Messages:
	%s
	Given a conversation summary, conversation context, and the latest messages since last summary, generate a summary of the conversation.
	`, c.GetConversationSummary(), "", latestMessages)
	_, err := c.instructor.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: openai.GPT4o,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		MaxTokens: 1000,
	}, &aiOut)
	log.Printf("Summary Prompt: %s", prompt)

	if err != nil {
		log.Printf("Failed to generate conversation summary: %v", err)
		return
	}

	c.mutex.Lock()
	c.conversationSummary = aiOut.Summary
	defer c.mutex.Unlock()
}

// getAIResponse calls the OpenAI API to get a response based on user input at the top level of the chat, not in a mode
func (c *ChatImpl) getAIResponse(msg *api.Message) *api.Message {
	userMessage := strings.TrimSpace(msg.Text)
	prompt := fmt.Sprintf(`
Given the user message: 
{{{
%s
}}}
Be a helpful assistant and respond to the users resquest if it's a request. If the user is asking something that's not public knowledge that an llm has access to or the chat history, the use should use the /question_answer mode.  You must output your response in a JSON object with a "response" key.`,
		userMessage)

	type AiOutput struct {
		Response string `json:"response" jsonschema:"title=response,description=your response to user message."`
	}
	var aiOut AiOutput

	if c.instructor == nil {
		return TextMessage("Sorry, I couldn't process that.")
	}

	err := c.Chat(&aiOut, []openai.ChatCompletionMessage{
		{
			Role:    "user",
			Content: prompt,
		},
	})

	if err != nil {
		log.Printf("Failed to generate AI response: %v", err)
		return TextMessage("Sorry, I couldn't process that.")
	}

	return TextMessage(aiOut.Response)
}

// PrintHistory prints the entire chat history
func (c *ChatImpl) PrintHistory() {
	fmt.Println(c.SprintHistory())
}

func (c *ChatImpl) SprintHistory() string {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	// use string builder
	history := ""
	history += "=== Chat History ===\n"
	for _, msg := range c.history {
		history += fmt.Sprintf("%s: %s\n", msg.Sender, msg.String())
	}
	history += fmt.Sprintf("====================\n")
	return history
}

// DetectMode detects the Mode based on user input, if any
func (c *ChatImpl) DetectMode(msg *api.Message) (TMode, bool) {
	userMessage := strings.TrimSpace(msg.Text)
	userInputLower := strings.ToLower(strings.TrimSpace(userMessage))
	if strings.HasPrefix(userInputLower, "/") {
		parts := strings.Split(userInputLower, " ")
		if len(parts) > 0 {
			cmd := parts[0]
			if mode, exists := modeKeywords[cmd]; exists {
				return mode, true
			}
		}
	}
	if mode, exists := modeKeywords[userInputLower]; exists {
		return mode, true
	}
	return "", false
}

type Intent struct {
	Mode                   IMode
	TMode                  TMode
	ParsedIntentAttributes map[string]string
}

// DetectIntent detects the intent based on user input, if any
func (c *ChatImpl) DetectIntent(msg *api.Message) (detectedIntent Intent, ok bool) {
	log.Debug().Any("msg", msg).Msg("DetectIntent started.")
	defer func() {
		log.Debug().Any("detectedMode", detectedIntent).Bool("ok", ok).Msg("DetectIntent completed.")
	}()

	userMessage := strings.TrimSpace(msg.Text)
	// Get intent from instructor
	type AiOutput struct {
		Intent     string            `json:"intent" jsonschema:"title=intent,description=the intent of the user message."`
		Attributes map[string]string `json:"attributes" jsonschema:"title=attributes,description=the attributes of the intent."`
	}
	var aiOut AiOutput
	err := c.Chat(&aiOut, []openai.ChatCompletionMessage{
		{
			Role: "user",
			Content: `Given the user message and intent examples, identify the intent of the user message and attributes of the intent.
				Examples:
` + INTENT_DATA + `
				given the user message: "I want ask what collor is Thido on the last episode of Pula?". Identify the intent and attributes of the user message.
				`,
		},
		{
			Role:    "assistant",
			Content: `{"intent": "code", "attributes": {"question": "what is the color of Thido on the last episode of Pula?"}}`,
		},
		{
			Role:    "user",
			Content: `Given the user message: ` + userMessage + `. Identify the intent and attributes of the user message.`,
		},
	})
	if err != nil {
		log.Warn().Err(err).Msg("Failed to detect intent")
		return Intent{}, false
	}

	intent := aiOut.Intent
	m := ModeFromString(intent)
	mode2, err := c.modeManager.CreateMode(c, m)
	if err != nil {
		return Intent{}, false
	}

	return Intent{
		Mode:                   mode2,
		TMode:                  m,
		ParsedIntentAttributes: aiOut.Attributes,
	}, (m != "")
}

// func (c *Chat) CreateMode(modeName TMode) (Mode, error) {
// 	if constructor, exists := modeRegistry[modeName]; exists {
// 		return constructor(c), nil
// 	}
// 	return nil, fmt.Errorf("unknown mode: %s", modeName)
// }

// GetHistory returns the chat history
func (c *ChatImpl) GetHistory() []*api.Message {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.history
}

func (c *ChatImpl) GetLastNMessages(n int) []*api.Message {
	history := c.GetHistory()
	from := len(history) - 10
	if from < 0 {
		from = 0
	}
	return history[from:]
}

func (c *ChatImpl) GetConversationSummary() string {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.conversationSummary
}

// RequestResponseChat is a chat that uses requests and response pattern for communication with the user.
type RequestResponseChat struct {
	shutdownChan chan struct{}
	chat         IChat
}

// ModeManager manages the different modes in the chat
type ModeManager struct {
	currentMode IMode
	mutex       sync.Mutex
}

func (mm *ModeManager) BestShot(mode IMode, msg *api.Message) (*api.Message, *api.Command, error) {
	return mode.BestShot(msg)
}

func (mm *ModeManager) CreateMode(c *ChatImpl, modeName TMode) (IMode, error) {
	if constructor, exists := modeRegistry[modeName]; exists {
		return constructor(c), nil
	}
	return nil, fmt.Errorf("unknown mode: %s", modeName)
}

// StartMode starts a new mode
func (mm *ModeManager) StartMode(mode IMode) (*api.Message, *api.Command, error) {
	log2.Debugf("ModeManager.StartMode: %T", mode)
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	if mm.currentMode != nil {
		log.Warn().Any("currentMode", mm.currentMode).Any("mode", mode).Msg("start new mode while currentMode != nil")
		if err := mm.currentMode.Stop(); err != nil {
			return &api.Message{}, NOOP, err
		}
	}
	mm.currentMode = mode
	res, command, err := mode.Start()
	if command == MODE_QUIT {
		if err := mm.currentMode.Stop(); err != nil {
			return &api.Message{}, NOOP, err
		}
		mm.currentMode = nil
	}
	return res, command, err

}

// HandleInput handles the user input based on the current mode
func (mm *ModeManager) HandleInput(msg *api.Message) (responseMsg *api.Message, responseCmd *api.Command, err error) {
	defer func() {
		log.Debug().
			Any("responseMsg", responseMsg).
			Any("responseCmd", responseCmd).
			Any("err", err).
			Msg("ModeManager.HandleInput completed.")
	}()
	log.Debug().Any("msg", msg).Msg("ModeManager.HandleInput started.")
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	if mm.currentMode != nil {
		response, command, err := mm.currentMode.HandleResponse(msg)
		if command == MODE_QUIT {
			if err := mm.currentMode.Stop(); err != nil {
				return &api.Message{}, NOOP, err
			}
			mm.currentMode = nil
		}
		return response, command, err
	}
	return &api.Message{}, NOOP, fmt.Errorf("no mode is currently active")
}

// StopMode stops the current mode
func (mm *ModeManager) StopMode() error {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	if mm.currentMode != nil {
		log2.Debugf("ModeManager.StopMode: %s", mm.currentMode)
		if err := mm.currentMode.Stop(); err != nil {
			return err
		}
		mm.currentMode = nil
	}
	return nil
}

func (mm *ModeManager) HandleIntent(intent Intent, msg *api.Message) (responseMsg *api.Message, responseCmd *api.Command, err error) {
	log.Debug().Any("mode", intent.TMode).Any("msg", msg).Msg("ModeManager.HandleIntent started.")
	defer func() {
		log.Debug().
			Any("responseMsg", responseMsg).
			Any("responseCmd", responseCmd).
			Any("err", err).
			Msg("ModeManager.HandleIntent completed.")
	}()

	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	if mm.currentMode != nil {
		if err := mm.currentMode.Stop(); err != nil {
			return &api.Message{}, NOOP, err
		}
	}
	mm.currentMode = intent.Mode
	res, command, err := mm.currentMode.HandleIntent(msg, intent)

	if command == MODE_QUIT {
		if err := mm.currentMode.Stop(); err != nil {
			return &api.Message{}, NOOP, err
		}
		mm.currentMode = nil
	}
	return res, command, err
}

func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	str, ok := v.(string)
	if !ok {
		return ""
	}
	return str
}

func toFloat32Slice(v interface{}) []float32 {
	if v == nil {
		return nil
	}
	floats, ok := v.([]float32)
	if !ok {
		return nil
	}
	return floats
}

func toFloat64(v interface{}) float64 {
	if v == nil {
		return 0
	}
	f, ok := v.(float64)
	if !ok {
		return 0
	}
	return f
}

type CliChat struct {
	chat   IChat
	client apiconnect.GptServiceClient
	mux    sync.Mutex
}

func NewCliChat(url string, chat *ChatImpl) *CliChat {
	client := apiconnect.NewGptServiceClient(http.DefaultClient, url) // replace with actual server URL
	return &CliChat{
		chat:   chat, // legacy, remve
		client: client,
	}
}

func (cli *CliChat) Start(shutdownChan chan struct{}) {
	// Initialize ConnectRPC client for the GptService
	ctx := context.Background()

	// Create a new chat session
	// newChatResp, err := client.NewChat(ctx, connect.NewRequest(&codesurgeon.NewChatRequest{}))
	// if err != nil {
	// 	fmt.Println("Error creating new chat:", err)
	// 	return
	// }
	// chatID := newChatResp.Msg.Chat.Id

	reader := bufio.NewReader(os.Stdin)
	fmt.Println("ChatCLI - Type your message and press Enter. Type /help for commands.")

	currentModeName := ""
	for {
		select {
		case <-shutdownChan:
			return
		default:
		}

		// Display prompt and read user input
		if currentModeName == "" {
			fmt.Print("🧓: ")
		} else {
			fmt.Printf("🧓(%s): ", currentModeName)
		}
		userMessage, _ := reader.ReadString('\n')
		userMessage = strings.TrimSpace(userMessage)
		if userMessage == "" {
			continue
		}

		// Prepare and send the message to the server
		message := &api.Message{Text: userMessage}
		sendMsgReq := &api.SendMessageRequest{Message: message}

		response, err := cli.client.SendMessage(ctx, connect.NewRequest(sendMsgReq))
		if err != nil {
			fmt.Println("Error:", err)
			return
		}

		if response.Msg.Mode != nil {
			currentModeName = response.Msg.Mode.Name
		}
		// Handle the response from the server
		if response.Msg.Message.Text != "" {
			fmt.Printf("🤖(%s): %s\n", currentModeName, response.Msg.Message.Text)
		}

		// Handle form response if present
		if response.Msg.Message.Form != nil {
			formResponse, cmd, err := cli.handleForm(response.Msg.Message)
			if err != nil {
				fmt.Println("Error handling form:", err)
				return
			}
			if cmd == QUIT {
				log.Debug().Msg("Exiting chat")
				return
			}

			// Send form response back to the server
			sendFormReq := &api.SendMessageRequest{Message: formResponse}

			response, err = cli.client.SendMessage(ctx, connect.NewRequest(sendFormReq))
			if err != nil {
				fmt.Println("Error sending form response:", err)
				return
			}

			if response.Msg.Message.Text != "" {
				fmt.Println("🤖:", response.Msg.Message.Text)
			} else if response.Msg.Message.Form != nil {
				fmt.Println("handling form")
				newresponse, _, err := cli.handleForm(response.Msg.Message)
				if err != nil {
					fmt.Println("Error:", err)
					return
				}
				log2.Debugf("CLI handleForm response: \n%v", newresponse)
				sendFormReq.Message = newresponse
				// send the response back to the chat to process the form submission by the user
				continue
			}
			// formResponse, cmd, err := cli.handleForm(response.Msg.Message)
			// if err != nil {
			// 	fmt.Println("Error handling form:", err)
			// 	return
			// }
			// if cmd == QUIT {
			// 	log.Debug().Msg("Exiting chat")
			// 	return
			// }

			// // Send form response back to the server
			// sendFormReq := &api.SendMessageRequest{Message: formResponse}
			// response, err = client.SendMessage(ctx, connect.NewRequest(sendFormReq))
			// if err != nil {
			// 	fmt.Println("Error sending form response:", err)
			// 	return
			// }

			// fmt.Println("🤖:", response.Msg.Message.Text)
		}

		if response.Msg.Command != nil {
			switch response.Msg.Command.Name {
			case string(QUIT.Name):
				fmt.Println("Exiting chat")
				return
			}
		}

		// Process other commands based on the server response
		// (e.g., QUIT, MODE_START, etc.) if needed
	}
}

func (cli *CliChat) StartOriginal(shutdownChan chan struct{}) {
	chat := cli.chat
	if PRINT_HISTORY_ON_EXIT {
		defer chat.PrintHistory()
	}
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("ChatCLI - Type your message and press Enter. Type /help for commands.")
	var incomingMessage *api.Message
	for {
		select {
		case <-shutdownChan:
			return
		default:
		}

		mode := chat.GetModeText()
		log2.Debugf("CLI waiting for user input")
		if mode != "" {
			fmt.Printf("%s:\n", mode)
		} else {
			fmt.Print("🧓:\n")
		}
		userMessage, _ := reader.ReadString('\n')
		userMessage = strings.TrimSpace(userMessage)
		if userMessage == "" {
			continue
		}
		incomingMessage = TextMessage(userMessage)

		// TODO - fix this, 5 is justt arbitrary
		var response *api.Message
		var command *api.Command
		var err error
		for i := 0; i < 5; i++ {
			response, command, err = chat.HandleUserMessage(incomingMessage)
			if err != nil {
				fmt.Println("Error:", err)
				return
			} else if response.Text != "" {
				fmt.Println("🤖:\n" + response.Text)
			} else if response.Form != nil {
				fmt.Println("handling form")
				newresponse, _, err := cli.handleForm(response)
				if err != nil {
					fmt.Println("Error:", err)
					return
				}
				log2.Debugf("CLI handleForm response: \n%v", newresponse)
				incomingMessage = newresponse
				// send the response back to the chat to process the form submission by the user
				continue
			}
			break
		}

		switch command {
		case QUIT:
			return
		case NOOP:
			continue
		case MODE_QUIT:
			continue
		case MODE_START:
			continue
		case SILENT:
			continue
		case nil:
			continue
		default:
			fmt.Printf("Command not recognized: %s\n", command)
		}
	}
}

func (cli *CliChat) handleForm(msg *api.Message) (responseMsg *api.Message, responseCmd *api.Command, err error) {
	log.Debug().Any("msg", msg).Msg("CliChat.handleForm started.")
	defer func() {
		log.Debug().
			Any("responseMsg", responseMsg).
			Any("responseCmd", responseCmd).
			Any("err", err).
			Msg("CliChat.handleForm completed.")
	}()

	// Create a slice of inputs to collect responses
	var fields []huh.Field

	// Iterate over the questions in the form and add them to the group
	for k, qa := range msg.Form.Questions {
		if qa.Question == "" {
			continue
		}

		// Create a new input for each question
		input := huh.NewInput().
			Title(qa.Question).
			Value(&msg.Form.Questions[k].Answer)

		// Add the input field to the list of fields
		fields = append(fields, input)

	}
	if len(fields) == 0 {
		return &api.Message{Text: "warn: no questions in form"}, NOOP, nil
	}

	// Create a form group with all questions
	form := huh.NewForm(
		huh.NewGroup(fields...),
	)

	// Run the form group to get all responses
	err = form.Run()
	if err != nil {
		return &api.Message{}, NOOP, err // Return error if form fails
	}

	log.Debug().Any("msg", msg).Msg("CliChat.handleForm afterrun.")
	return msg, NOOP, nil
}

type Form struct {
	Questions []FormQuestion
}

type FormQuestion struct {
	Question   string
	Answer     string
	Required   bool
	Validation func(string) bool
}

func NewForm() *Form {
	return &Form{
		Questions: []FormQuestion{},
	}
}

func (pf *Form) ClearAnswers() {
	for i := range pf.Questions {
		pf.Questions[i].Answer = ""
	}
}

func (pf *Form) AddQuestion(question string, required bool, validation func(string) bool, defaultValue string) StringPromise {
	q := FormQuestion{
		Question:   question,
		Required:   required,
		Validation: validation,
		Answer:     defaultValue,
	}
	pf.Questions = append(pf.Questions, q)
	return func() string {
		for _, q := range pf.Questions {
			if q.Question == question {
				return q.Answer
			}
		}
		return ""
	}
}

func (pf *Form) IsQuestionFilled(q FormQuestion) bool {
	if q.Required && q.Answer == "" {
		return false
	} else if q.Answer != "" {
		if q.Validation != nil && !q.Validation(q.Answer) {
			return false
		}
	}
	return true
}

func (pf *Form) IsFilled() bool {
	if pf == nil || len(pf.Questions) == 0 {
		return true
	}
	for _, q := range pf.Questions {
		if !pf.IsQuestionFilled(q) {
			return false
		}
	}
	return true
}

func (pf *Form) Answer(question, answer string) {
	log.Debug().Str("question", question).Str("answer", answer).Msg("Form.Answer")
	if pf == nil || len(pf.Questions) == 0 {
		log.Warn().Msg("Form is nil or empty here")
		return
	}
	for i, q := range pf.Questions {
		if q.Question == question {
			pf.Questions[i].Answer = answer
			log.Warn().Str("question", question).Str("answer", answer).Msg("Form.Answer answered for real")
			return
		}
	}
	log.Warn().Str("question", question).Str("answer", answer).Msg("Form.Answer question NOT FOUND")
}

func (pf *Form) MakeFormMessage() *api.FormMessage {
	if pf == nil || len(pf.Questions) == 0 {
		log.Warn().Msg("Form is nil or empty here222")
		return &api.FormMessage{
			Questions: []*api.QuestionAnswer{},
		}
	}
	form_ := &api.FormMessage{
		Questions: []*api.QuestionAnswer{},
	}
	for _, q := range pf.Questions {
		form_.Questions = append(form_.Questions, &api.QuestionAnswer{
			Question: q.Question,
			Answer:   q.Answer,
		})
	}
	return form_

}

type StringPromise func() string

func fnValidateNotEmpty[T comparable](s T) bool {
	var z T
	return s != z
}
