package chatcli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/uuid"
	"github.com/instructor-ai/instructor-go/pkg/instructor"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/rs/zerolog/log"
	"github.com/sashabaranov/go-openai"
	"github.com/wricardo/code-surgeon/api"
	"github.com/wricardo/code-surgeon/log2"
)

type IChat interface {
	HandleUserMessage(msg *api.Message) (responseMsg *api.Message, responseCmd *api.Command, err error)
	GetHistory() []*api.Message
	PrintHistory()
	SaveState() error
	LoadState() error

	// TODO: Remove this from the interface
	GetModeText() string
}

type ChatImpl struct {
	driver         *neo4j.DriverWithContext     `json:"-"`
	instructor     *instructor.InstructorOpenAI `json:"-"`
	rwmutex        sync.RWMutex
	modeManager    *ModeManager `json:"-"`
	clearHistory   func() error
	chatRepository ChatRepository

	Id                  string
	IsTest              bool // test mode, tests will set this to true
	History             []*api.Message
	ConversationSummary string
	ModesStates         map[string]map[string]string
}

// NewChat creates a new Chat instance
func NewChat(driver *neo4j.DriverWithContext, instructorClient *instructor.InstructorOpenAI, chatRepository ChatRepository) *ChatImpl {
	return (&ChatImpl{
		Id:             uuid.New().String(),
		History:        []*api.Message{},
		ModesStates:    map[string]map[string]string{},
		chatRepository: chatRepository,
	}).Setup(driver, instructorClient, chatRepository)
}

func (c *ChatImpl) checkIfExit(msg *api.Message) (bool, *api.Message) {
	userMessage := strings.TrimSpace(msg.Text)
	if userMessage == "/exit" || userMessage == "/quit" || userMessage == "/bye" || userMessage == "/stop" {
		return true, TextMessage("Exited.")
	}
	return false, TextMessage("")
}

func (c *ChatImpl) ClearHistory() error {
	c.rwmutex.Lock()
	defer c.rwmutex.Unlock()
	c.History = []*api.Message{}
	c.ConversationSummary = ""
	return nil
}

// HandleUserMessage handles the user message. This is the main loop of the chat where we detect modes, intents, and handle user input and AI responses.
func (c *ChatImpl) HandleUserMessage(msg *api.Message) (responseMsg *api.Message, responseCmd *api.Command, err error) {
	defer func() {
		log.Debug().
			Any("responseMsg", &responseMsg).
			Any("responseCmd", responseCmd).
			Any("mode", c.modeManager.currentMode).
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
		log.Debug().Str("mode", fmt.Sprintf("%v", c.modeManager.currentMode)).Msg("HandleUserMessage: in mode")
		// if user wants to exit mode
		if exit, response := c.checkIfExit(msg); exit {
			c.modeManager.StopMode()
			return response, MODE_QUIT, nil
		}

		response, command, err := c.modeManager.HandleInput(msg)
		if command != SILENT && command != SILENT_MODE_QUIT && command != SILENT_MODE_QUIT_CLEAR {
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
	if modeName, detected, interactiveMode := c.DetectMode(msg); detected {
		mode, err := c.modeManager.CreateMode(c, modeName)
		if err != nil {
			return &api.Message{}, NOOP, err
		}
		var response *api.Message
		var command *api.Command
		method := "BestShot"
		if interactiveMode {
			method = "Start"
			response, command, err = c.modeManager.StartMode(mode)
			if err != nil {
				return &api.Message{}, NOOP, err
			}
		} else {
			response, command, err = c.modeManager.BestShot(mode, msg)
			if err != nil {
				return &api.Message{}, NOOP, err
			}
		}

		if command != SILENT && command != SILENT_MODE_QUIT && command != SILENT_MODE_QUIT_CLEAR {
			c.addMessage("You", msg)
			c.addMessage("AI", response)
		}
		log.Debug().Str("mode", fmt.Sprintf("%T", mode)).Any("command", command).Str("method", method).Msg("Response from detected mode")
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
			if command != SILENT && command != SILENT_MODE_QUIT && command != SILENT_MODE_QUIT_CLEAR {
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

// Called after an unmarshal, or NewChat
func (c *ChatImpl) Setup(driver *neo4j.DriverWithContext, instructorClient *instructor.InstructorOpenAI, chatRepository ChatRepository) *ChatImpl {
	log.Debug().Msg("Chat.Setup started.")
	c.rwmutex = sync.RWMutex{}
	c.driver = driver
	c.instructor = instructorClient
	c.modeManager = &ModeManager{}
	c.chatRepository = chatRepository
	return c
}

func (c *ChatImpl) GetModeText() string {
	log.Debug().Any("currentMode", c.modeManager.currentMode).Msg("GetModeText")
	if c.modeManager != nil && c.modeManager.currentMode != nil {
		return c.modeManager.currentMode.Name()
	}
	return "main"
}

// internal function to chat with AI
func (c *ChatImpl) Chat(aiOut interface{}, msgs []openai.ChatCompletionMessage) error {
	ctx := context.Background()
	history := c.GetHistory()
	summary := c.GetConversationSummary()
	gptMessages := make([]openai.ChatCompletionMessage, 0, len(history)+2)

	defer func() {
		logger := log.Trace().Any("aiOut", aiOut)
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
		Content: "conversation summary for context: " + summary,
	})

	gptMessages = append(gptMessages, msgs...)

	if c.instructor == nil {
		return fmt.Errorf("instructor is nil on chatimpl")
	}
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
	c.rwmutex.Lock()
	defer c.rwmutex.Unlock()
	msg.Sender = sender
	log2.Debugf("Adding message to chat history: %s %s", sender, msg.String())

	c.History = append(c.History, msg)
}

func (c *ChatImpl) generateConversationSummary() {
	if DISABLE_CONVERSATION_SUMMARY {
		log.Debug().Msg("Conversation summary generation is disabled.")
		return
	}

	// Acquire read lock and copy data needed
	c.rwmutex.RLock()
	conversationSummary := c.ConversationSummary
	historyCopy := make([]*api.Message, len(c.History))
	copy(historyCopy, c.History)
	c.rwmutex.RUnlock()

	latestMessages := ""
	from := len(historyCopy) - 4
	if from < 0 {
		from = 0
	}
	lastMessages := historyCopy[from:]
	for _, mp := range lastMessages {
		latestMessages += fmt.Sprintf("%s: %s\n", mp.Sender, mp.ChatString())
	}

	type AiOutput struct {
		Summary string `json:"summary"`
	}
	ctx := context.Background()
	var aiOut AiOutput
	prompt := fmt.Sprintf(`
    Conversation Summary:
    %s
    Latest Messages:
    %s
    Given the conversation summary and the latest messages, generate a new summary. Output in json, use "summary" object key`,
		conversationSummary, latestMessages)

	_, err := c.instructor.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: openai.GPT4o,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		MaxTokens: 2000,
	}, &aiOut)
	if err != nil {
		log.Printf("Failed to generate conversation summary: %v", err)
		return
	}

	// Acquire write lock to update the summary
	c.rwmutex.Lock()
	c.ConversationSummary = aiOut.Summary
	c.rwmutex.Unlock()
	log2.Debugf("Generated conversation summary: %s", aiOut.Summary)
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
	// use string builder
	history := ""
	history += "=== Chat History ===\n"
	historySlice := c.GetHistory()
	for _, msg := range historySlice {
		history += fmt.Sprintf("%s: %s\n", msg.Sender, msg.String())
	}
	history += fmt.Sprintf("====================\n")
	return history
}

// DetectMode detects the Mode based on user input, if any.
// first boolean indicates if a mode was detected
// second boolean indicates if the mode was detected by a single word
func (c *ChatImpl) DetectMode(msg *api.Message) (TMode, bool, bool) {
	userMessage := strings.TrimSpace(msg.Text)
	userInputLower := strings.ToLower(strings.TrimSpace(userMessage))
	if strings.HasPrefix(userInputLower, "/") {
		parts := strings.Split(userInputLower, " ")
		if len(parts) > 1 {
			cmd := parts[0]
			if mode, exists := modeKeywords[cmd]; exists {
				return mode, true, false
			}
		}
		if len(parts) == 1 {
			if mode, exists := modeKeywords[userInputLower]; exists {
				return mode, true, true
			}
		}
	}
	if mode, exists := modeKeywords[userInputLower]; exists {
		return mode, true, true
	}
	return "", false, false
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
		panic(err)
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

//	func (c *Chat) CreateMode(modeName TMode) (Mode, error) {
//		if constructor, exists := modeRegistry[modeName]; exists {
//			return constructor(c), nil
//		}
//		return nil, fmt.Errorf("unknown mode: %s", modeName)
//	}
func (c *ChatImpl) GetModeState() []*api.ModeState {
	res := make([]*api.ModeState, 0, len(c.ModesStates))
	for modeName, state := range c.ModesStates {
		for k, v := range state {
			res = append(res, &api.ModeState{
				ModeName: modeName,
				Key:      k,
				Value:    v,
			})
		}
	}
	return res
}

// GetHistory returns the chat history
func (c *ChatImpl) GetHistory() []*api.Message {
	c.rwmutex.RLock()
	defer c.rwmutex.RUnlock()
	return c.History
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
	c.rwmutex.RLock()
	defer c.rwmutex.RUnlock()
	return c.ConversationSummary
}

func (c *ChatImpl) LoadModeState(mode IMode) (ModeStateMap, error) {
	if c.ModesStates == nil {
		c.ModesStates = map[string]map[string]string{}
		// return nil, errors.New("ModesStates is nil on load")
	}
	if c.ModesStates[mode.Name()] == nil {
		c.ModesStates[mode.Name()] = map[string]string{}
	}
	return c.ModesStates[mode.Name()], nil
}

func (c *ChatImpl) SaveModeState(mode IMode, state map[string]string) error {
	log.Debug().Any("mode", mode).Any("state", state).Msg("SaveModeState started.")
	fmt.Printf("state\n%s", spew.Sdump(state)) // TODO: wallace debug
	if c.ModesStates == nil {
		c.ModesStates = map[string]map[string]string{}
		// return errors.New("ModesStates is nil on save")
	}
	if c.ModesStates[mode.Name()] == nil {
		c.ModesStates[mode.Name()] = map[string]string{}
	}
	c.ModesStates[mode.Name()] = state
	err := c.chatRepository.SaveChat(c.Id, c)

	if err != nil {
		log.Error().Err(err).Msg("Failed to save mode state")
		return err
	}
	log.Info().Msg("Saved mode state on postgres")
	return err
}
