package main

import (
	"fmt"
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/charmbracelet/huh"
	"github.com/instructor-ai/instructor-go/pkg/instructor"
	"github.com/joho/godotenv"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/require"
	"github.com/wricardo/code-surgeon/ai"
	"github.com/wricardo/code-surgeon/log2"
	"github.com/wricardo/code-surgeon/neo4j2"
)

// INTENT_DATA is the data used to train the AI model to detect intents
var INTENT_DATA string = `
g: "exit" i: "exit"
g: "write some code" i: "code"
g: "i want to write some code" i: "code"
g: "what can you do" i: "help"
g: "ask a question" i: "question_answer"
g: "add a question and answer to knowledge base" i: "question_answer"
`

const (
	SenderAI  = "AI"
	SenderYou = "You"
)

type Command string

var NOOP Command = "noop"
var QUIT Command = "quit"
var MODE_QUIT Command = "mode_quit"
var MODE_START Command = "mode_start"
var SILENT Command = "silent" // this will not save to history

type TMode string

var EXIT TMode = "exit"
var CODE TMode = "code"
var ADD_TEST TMode = "add_test"
var QUESTION_ANSWER TMode = "question_answer"
var CYPHER TMode = "cypher"
var TEACHER TMode = "teacher"
var DEBUG TMode = "debug"
var HELP TMode = "help"

func ModeFromString(mode string) TMode {
	switch mode {
	case "exit":
		return EXIT
	case "code":
		return CODE
	case "add_test":
		return ADD_TEST
	case "question_answer":
		return QUESTION_ANSWER
	case "cypher":
		return CYPHER
	case "teacher":
		return TEACHER
	case "debug":
		return DEBUG
	case "help":
		return HELP
	default:
		log.Printf("WARN: Unknown mode in ModeFromString: %s", mode)
		return ""
	}
}

// Keywords for detecting different modes
var modeKeywords = map[string]TMode{
	"/exit":     EXIT,
	"/quit":     EXIT,
	"/bye":      EXIT,
	"/code":     CODE,
	"/add_test": ADD_TEST,
	"/qa":       QUESTION_ANSWER,
	"/question": QUESTION_ANSWER,
	"/cypher":   CYPHER,
	"/neo4j":    CYPHER,
	"/debug":    DEBUG,
	"/help":     HELP,
	"/teacher":  TEACHER,
}

// Mode is a chatbot specialized for a particular task, like coding or answering questions or playing a game o top of your data
type Mode interface {
	Start() (Message, Command, error)
	HandleIntent(msg Message) (Message, Command, error)
	HandleResponse(input Message) (Message, Command, error)
	Stop() error
}

// Form represents the form for the user to fill
type Form struct {
	Questions []QuestionAnswer
}

func NewForm(questions []QuestionAnswer) *Form {
	return &Form{
		Questions: questions,
	}
}

// Message represents a message either from User to Ai or Ai to User
type Message struct {
	Text string
	Form *Form
}

func (m Message) String() string {
	if m.Form != nil {
		ret := ""
		for _, qa := range m.Form.Questions {
			ret += fmt.Sprintf("Q:\n%s\n\nA:\n%s\n", qa.Question, qa.Answer)
		}
		return ret
	}
	return m.Text
}

func TextMessage(text string) Message {
	return Message{
		Text: text,
	}
}

// IChat is an interface for chat operations
type IChat interface {
	HandleUserMessage(msg Message) (Message, Command, error)
	GetHistory() []MessagePayload
	PrintHistory()

	// TODO: Remove this from the interface
	GetModeText() string
}

// ModeHandler is an interface for different types of modes
type ModeHandler interface {
	Start() (Message, error)
	HandleResponse(msg Message) (Message, Command, error)
}

// MessagePayload represents a single chat message
type MessagePayload struct {
	Sender  string
	Message Message
}

// Chat handles the chat functionality
type Chat struct {
	driver     *neo4j.DriverWithContext
	instructor *instructor.InstructorOpenAI

	mutex               sync.Mutex
	history             []MessagePayload
	conversationSummary string

	modeManager *ModeManager

	disableConversationSummary bool
}

func (c *Chat) checkIfExit(msg Message) (bool, Message) {
	userMessage := strings.TrimSpace(msg.Text)
	if userMessage == "/exit" || userMessage == "/quit" || userMessage == "/bye" || userMessage == "/stop" {
		return true, TextMessage("Exited mode.")
	}
	return false, TextMessage("")
}

// HandleUserMessage handles the user message. This is the main loop of the chat where we detect modes, intents, and handle user input and AI responses.
func (c *Chat) HandleUserMessage(msg Message) (responseMsg Message, responseCmd Command, err error) {
	defer func() {
		log.Debug().
			Any("responseMsg", responseMsg).
			Any("responseCmd", responseCmd).
			Any("err", err).
			Msg("Chat.HandleUserMessage completed.")
	}()
	log.Debug().Any("msg", msg).Msg("Chat.HandleUserMessage started.")
	c.generateConversationSummary()

	// If in a mode, delegate input handling to the mode manager
	if c.modeManager.currentMode != nil {
		// if user wants to exit mode
		if exit, response := c.checkIfExit(msg); exit {
			c.modeManager.StopMode()
			return response, MODE_QUIT, nil
		}

		response, command, err := c.modeManager.HandleInput(msg)
		if command != SILENT {
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
		mode, err := c.CreateMode(modeName)
		if err != nil {
			return Message{}, NOOP, err
		}
		response, command, err := c.modeManager.StartMode(mode)
		if err != nil {
			return Message{}, NOOP, err
		}
		if command != SILENT {
			c.addMessage("You", msg)
			c.addMessage("AI", response)
		}
		log.Debug().Str("mode", fmt.Sprintf("%T", mode)).Msg("Started new mode from detect")
		return response, command, nil
	}

	// detect intent
	mode, detected := c.DetectIntent(msg)
	if detected {
		// handle intent
		if mode != "" {
			mode, err := c.CreateMode(mode)
			if err != nil {
				return Message{}, NOOP, err
			}
			response, command, err := c.modeManager.HandleIntent(mode, msg)
			if err != nil {
				return Message{}, NOOP, err
			}
			if command != SILENT {
				c.addMessage("You", msg)
				c.addMessage("AI", response)
			}
			return response, command, nil
		}
	}

	// Regular top level AI response
	c.addMessage("You", msg)
	aiResponse := c.getAIResponse(msg)
	c.addMessage("AI", aiResponse)
	return aiResponse, NOOP, nil
}

// NewChat creates a new Chat instance
func NewChat(aiClient AIClient, driver *neo4j.DriverWithContext) *Chat {
	instructorClient := ai.GetInstructor()
	return &Chat{
		driver:                     driver,
		mutex:                      sync.Mutex{},
		history:                    []MessagePayload{},
		modeManager:                &ModeManager{},
		instructor:                 instructorClient,
		disableConversationSummary: false,
	}
}

func (c *Chat) GetModeText() string {
	if c.modeManager.currentMode != nil {
		return fmt.Sprintf("%T", c.modeManager.currentMode)
	}
	return ""
}

// internal function to chat with AI
func (c *Chat) Chat(aiOut interface{}, msgs []openai.ChatCompletionMessage) error {
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
		if msg.Sender == SenderAI {
			role = openai.ChatMessageRoleAssistant
		}
		gptMessages = append(gptMessages, openai.ChatCompletionMessage{
			Role:    role,
			Content: msg.Message.Text,
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

// addMessage adds a message to the chat history
func (c *Chat) addMessage(sender string, msg Message) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	log2.Debugf("Adding message to chat history: %s %s", sender, msg.String())

	mp := MessagePayload{Sender: sender, Message: msg}
	c.history = append(c.history, mp)
}

func (c *Chat) generateConversationSummary() {
	if c.disableConversationSummary {
		return
	}
	latestMessages := ""
	for _, mp := range c.GetHistory() {
		latestMessages += fmt.Sprintf("%s: %s\n", mp.Sender, mp.Message.String())
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
func (c *Chat) getAIResponse(msg Message) Message {
	userMessage := strings.TrimSpace(msg.Text)
	prompt := fmt.Sprintf(`
Given the user message: 
%s
Generate a response. You must output your response in a JSON object with a "response" key.`,
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
func (c *Chat) PrintHistory() {
	fmt.Println(c.SprintHistory())
}

func (c *Chat) SprintHistory() string {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	// use string builder
	history := ""
	history += "=== Chat History ===\n"
	for _, msg := range c.history {
		history += fmt.Sprintf("%s: %s\n", msg.Sender, msg.Message.Text)
	}
	history += fmt.Sprintf("====================\n")
	return history
}

// DetectMode detects the Mode based on user input, if any
func (c *Chat) DetectMode(msg Message) (TMode, bool) {
	userMessage := strings.TrimSpace(msg.Text)
	userInputLower := strings.ToLower(strings.TrimSpace(userMessage))
	if mode, exists := modeKeywords[userInputLower]; exists {
		return mode, true
	}
	return "", false
}

type Intent int

const (
	Unknown Intent = iota
	Exit
	WriteCode
)

// String returns the string representation of the Intent
func (i Intent) String() string {
	return [...]string{"Unknown", "Exit", "WriteCode"}[i]
}

// DetectIntent detects the intent based on user input, if any
func (c *Chat) DetectIntent(msg Message) (detectedMode TMode, ok bool) {
	log.Debug().Any("msg", msg).Msg("DetectIntent started.")
	defer func() {
		log.Debug().Any("detectedMode", detectedMode).Bool("ok", ok).Msg("DetectIntent completed.")
	}()

	userMessage := strings.TrimSpace(msg.Text)
	// Get intent from instructor
	type AiOutput struct {
		Intent string `json:"intent" jsonschema:"title=intent,description=the intent of the user message."`
	}
	var aiOut AiOutput
	_, err := c.instructor.CreateChatCompletion(context.Background(), openai.ChatCompletionRequest{
		Model: openai.GPT4o,
		Messages: []openai.ChatCompletionMessage{
			{
				Role: "user",
				Content: `Given the user message and intent examples, Generate the intent of the user message.
				Examples:
` + INTENT_DATA + `
				given the user message: "i want to code something". Generate the intent of the user message.
				`,
			},
			{
				Role:    "assistant",
				Content: `{"intent": "code" }`,
			},
			{
				Role:    "user",
				Content: `Given the user message: ` + userMessage + `. Generate the intent of the user message.`,
			},
		},
		MaxTokens: 1000,
	}, &aiOut)
	if err != nil {
		log.Printf("Failed to detect intent: %v", err)
		return "", false
	}

	intent := aiOut.Intent
	m := ModeFromString(intent)
	return m, (m != "")
}

func (c *Chat) HandleIntent(intent Intent) (TMode, error) {
	switch intent {
	case Exit:
		return EXIT, nil
	case WriteCode:
		return CODE, nil

	default:
		log.Printf("WARN: Unknown intent: %s", intent)
		return "", nil
	}
}

func (c *Chat) CreateMode(modeName TMode) (Mode, error) {
	switch modeName {
	case CODE:
		return NewCodeMode(c), nil
	case ADD_TEST:
		return NewAddTestMode(c), nil
	case QUESTION_ANSWER:
		return NewQuestionAnswerMode(c), nil
	case CYPHER:
		return NewCypherMode(c), nil
	case TEACHER:
		return NewTeacherMode(c), nil
	case DEBUG:
		return NewDebugMode(c), nil
	case HELP:
		return NewHelpMode(c), nil
	case EXIT:
		return nil, fmt.Errorf("exit forced error")
	default:
		return nil, fmt.Errorf("unknown mode: %s", modeName)
	}
}

// GetHistory returns the chat history
func (c *Chat) GetHistory() []MessagePayload {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.history
}

func (c *Chat) GetConversationSummary() string {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.conversationSummary
}

type HttpChat struct {
	shutdownChan chan struct{}
	chat         IChat
	mux          sync.Mutex
}

func NewHTTPServer(chat IChat, shutdownChan chan struct{}) *HttpChat {
	return &HttpChat{chat: chat, shutdownChan: shutdownChan}
}

func (s *HttpChat) Start() {
	srv := &http.Server{Addr: ":8080"}
	http.HandleFunc("/post-message", s.handlePostMessage)
	http.HandleFunc("/get-history", s.handleGetHistory)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Msgf("ListenAndServe(): %s", err)
		}
	}()

	// Wait for shutdown signal
	<-s.shutdownChan
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal().Msgf("Server Shutdown Failed:%+v", err)
	}
	log.Print("Server exited properly")
}

func (s *HttpChat) ShutdownServer() {
	defer s.chat.PrintHistory()
	fmt.Println("Shutting down server...")
	if err := syscall.Kill(syscall.Getpid(), syscall.SIGINT); err != nil {
		fmt.Println("Error shutting down server:", err)
	}
}

func (s *HttpChat) handlePostMessage(w http.ResponseWriter, r *http.Request) {
	// Updated request structure to accept a Message
	var request struct {
		Text string `json:"text"`
		Form *Form  `json:"form,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Create a Message from the request
	msg := Message{
		Text: request.Text,
		Form: request.Form,
	}

	// Handle the user message, which now includes form input if present
	response, command, err := s.chat.HandleUserMessage(msg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	switch command {
	case QUIT:
		w.Write([]byte("Exited mode."))
		go func() { time.Sleep(time.Millisecond * 100); s.ShutdownServer() }()
	case NOOP, MODE_START:
		// nothing
	case MODE_QUIT:
		// nothing
	case "":
		// nothing
	default:
		fmt.Printf("Command not recognized: %s\n", command)
	}

	res := struct {
		Command *string `json:"command,omitempty"`
		Text    string  `json:"Text,omitempty"`
		Form    *Form   `json:"form,omitempty"` // Add form to the response
	}{
		Text: response.Text,
	}

	// Check if the response includes a form and serialize it if present
	if response.Form != nil {
		res.Form = response.Form
	} else {
		if command != NOOP {
			tmp := string(command)
			res.Command = &tmp
		}
	}

	// Encode the response with the form included
	json.NewEncoder(w).Encode(res)
}

func (s *HttpChat) handleGetHistory(w http.ResponseWriter, r *http.Request) {
	history := s.chat.GetHistory()
	json.NewEncoder(w).Encode(history)
}

func main() {
	hello()
	// Initialize logger
	log2.Configure()

	// Setup signal handling for graceful exit
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	shutdownChan := make(chan struct{})

	var myEnv map[string]string
	myEnv, err := godotenv.Read()
	if err != nil {
		log.Fatal().Msg("Error loading .env file")
	}
	neo4jDbUri, _ := myEnv["NEO4J_DB_URI"]
	neo4jDbUser, _ := myEnv["NEO4J_DB_USER"]
	neo4jDbPassword, _ := myEnv["NEO4J_DB_PASSWORD"]
	ctx := context.Background()
	driver, closeFn, err := neo4j2.Connect(ctx, neo4jDbUri, neo4jDbUser, neo4jDbPassword)
	if err != nil {
		log.Fatal().Msgf("Failed to connect to Neo4j: %v", err)
	}
	defer closeFn()

	apiClient := NewGptAiClient()
	// Instantiate chat service
	chat := NewChat(apiClient, &driver)

	go func() {
		// Decide to run CLI or HTTP server based on flag or environment
		if len(os.Args) > 1 && os.Args[1] == "http" {
			// Start HTTP Server
			httpServer := NewHTTPServer(chat, shutdownChan)
			httpServer.Start()
		} else {
			// Start CLI
			cliChat := NewCliChat(chat)
			cliChat.Start(shutdownChan)
			os.Exit(0)
		}
	}()

	<-signalChan
	close(shutdownChan)
	fmt.Println("\nBye")
}

type GptAiClient struct {
	instructor *instructor.InstructorOpenAI
}

func NewGptAiClient() *GptAiClient {

	instructorClient := ai.GetInstructor()

	return &GptAiClient{
		instructor: instructorClient,
	}
}

func (c *GptAiClient) GetResponse(prompt string) (string, error) {
	ctx := context.Background()
	type AiOutput struct {
		Response string `json:"response" jsonschema:"title=response,description=your response to user message."`
	}
	var aiOut AiOutput
	if c.instructor == nil {
		return "", fmt.Errorf("instructor client is nil")
	}

	_, err := c.instructor.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: openai.GPT4o,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    "user",
				Content: prompt + ". You must output your response in a json object with a response key.",
			},
		},
		MaxTokens: 1000,
	}, &aiOut)

	if err != nil {
		return "", err
	}

	return aiOut.Response, nil
}

type AIClient interface {
	GetResponse(prompt string) (string, error)
}

type MyMockAiClient struct{}

func (c *MyMockAiClient) GetResponse(prompt string) (string, error) {
	return "This is a mock response \n with new line.", nil
}

// ModeManager manages the different modes in the chat
type ModeManager struct {
	currentMode Mode
	mutex       sync.Mutex
}

// StartMode starts a new mode
func (mm *ModeManager) StartMode(mode Mode) (Message, Command, error) {
	log2.Debugf("ModeManager.StartMode: %T", mode)
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	if mm.currentMode != nil {
		log.Warn().Any("currentMode", mm.currentMode).Any("mode", mode).Msg("start new mode while currentMode != nil")
		if err := mm.currentMode.Stop(); err != nil {
			return Message{}, NOOP, err
		}
	}
	mm.currentMode = mode
	res, command, err := mode.Start()
	if command == MODE_QUIT {
		if err := mm.currentMode.Stop(); err != nil {
			return Message{}, NOOP, err
		}
		mm.currentMode = nil
	}
	return res, command, err

}

// HandleInput handles the user input based on the current mode
func (mm *ModeManager) HandleInput(msg Message) (responseMsg Message, responseCmd Command, err error) {
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
				return Message{}, NOOP, err
			}
			mm.currentMode = nil
		}
		return response, command, err
	}
	return Message{}, NOOP, fmt.Errorf("no mode is currently active")
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

func (mm *ModeManager) HandleIntent(mode Mode, msg Message) (responseMsg Message, responseCmd Command, err error) {
	log.Debug().Any("mode", mode).Any("msg", msg).Msg("ModeManager.HandleIntent started.")
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
			return Message{}, NOOP, err
		}
	}
	mm.currentMode = mode
	res, command, err := mode.HandleIntent(msg)
	if command == MODE_QUIT {
		if err := mm.currentMode.Stop(); err != nil {
			return Message{}, NOOP, err
		}
		mm.currentMode = nil
	}
	return res, command, err
}

// MockAIClient is a mock implementation of the AIClient interface
type MockAIClient struct {
	Responses map[string]string
}

// GetResponse returns a mock response based on the prompt
func (client *MockAIClient) GetResponse(prompt string) (string, error) {
	if response, ok := client.Responses[prompt]; ok {
		return response, nil
	}
	return "Default mock response", nil
}

// TESTS
// TESTS
// TESTS

func TestEverything(t *testing.T) {
	t.Run("TestChat_HandleUserMessage", TestChat_HandleUserMessage)
	t.Run("TestChat_DetectMode", TestChat_DetectMode)
}

func TestChat_HandleUserMessage(t *testing.T) {
	mockAIClient := &MockAIClient{
		Responses: map[string]string{
			"hello": "Hi there!",
		},
	}
	chat := NewChat(mockAIClient, nil)

	response, command, err := chat.HandleUserMessage(TextMessage("hello"))
	require.NoError(t, err)
	require.Equal(t, "Hi there!", response)
	require.Equal(t, NOOP, command)
}

func TestChat_DetectMode(t *testing.T) {
	chat := NewChat(&MyMockAiClient{}, nil)

	mode, detected := chat.DetectMode(TextMessage("exit"))
	require.True(t, detected)
	require.Equal(t, EXIT, mode)

	mode, detected = chat.DetectMode(TextMessage("code"))
	require.True(t, detected)
	require.Equal(t, CODE, mode)

	mode, detected = chat.DetectMode(TextMessage("add_test"))
	require.True(t, detected)
	require.Equal(t, ADD_TEST, mode)

	mode, detected = chat.DetectMode(TextMessage("invalid"))
	require.False(t, detected)
	require.Equal(t, TMode(""), mode)
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
	chat IChat
	mux  sync.Mutex
}

func NewCliChat(chat *Chat) *CliChat {
	return &CliChat{
		chat: chat,
	}
}

func (cli *CliChat) Start(shutdownChan chan struct{}) {
	chat := cli.chat
	defer chat.PrintHistory()
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("ChatCLI - Type your message and press Enter. Type /help for commands.")
	var incomingMessage Message
	for {
		select {
		case <-shutdownChan:
			return
		default:
		}

		mode := chat.GetModeText()
		if mode != "" {
			fmt.Printf("%s:\n", mode)
		} else {
			fmt.Print("🧓:\n")
		}
		log2.Debugf("CLI waiting for user input")
		userMessage, _ := reader.ReadString('\n')
		userMessage = strings.TrimSpace(userMessage)
		incomingMessage = TextMessage(userMessage)

		// TODO - fix this, 5 is justt arbitrary
		var response Message
		var command Command
		var err error
		for i := 0; i < 5; i++ {
			response, command, err = chat.HandleUserMessage(incomingMessage)
			if err != nil {
				fmt.Println("Error:", err)
				return
			} else if response.Text != "" {
				fmt.Println("🤖:\n" + response.Text)
			} else if response.Form != nil {
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
		case "":
			continue
		default:
			fmt.Printf("Command not recognized: %s\n", command)
		}
	}
}

func (cli *CliChat) handleForm(msg Message) (responseMsg Message, responseCmd Command, err error) {
	log.Debug().Any("msg", msg).Msg("CliChat.handleForm started.")
	defer func() {
		log.Debug().
			Any("responseMsg", responseMsg).
			Any("responseCmd", responseCmd).
			Any("err", err).
			Msg("CliChat.handleForm completed.")
	}()

	// Iterate over the questions in the form and collect responses
	for k, qa := range msg.Form.Questions {
		var response string
		// Create a new input for each question
		input := huh.NewInput().
			Title(qa.Question).
			Value(&response)

		// // If there's a validation function in the question, add it
		// if qa.ValidateFunc != nil {
		// 	input.Validate(qa.ValidateFunc)
		// }

		// Run the input to get the response
		err := input.Run()
		if err != nil {
			return Message{}, NOOP, err // Return error if input fails
		}

		msg.Form.Questions[k].Answer = response
	}

	return Message{
		Form: msg.Form,
	}, NOOP, nil

}
