package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/instructor-ai/instructor-go/pkg/instructor"
	"github.com/joho/godotenv"
	"github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/require"
	"github.com/wricardo/code-surgeon/ai"
)

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
var DEBUG TMode = "debug"
var HELP TMode = "help"

// Keywords for detecting different modes
var modeKeywords = map[string]TMode{
	"exit":     EXIT,
	"quit":     EXIT,
	"bye":      EXIT,
	"code":     CODE,
	"add_test": ADD_TEST,
	"qa":       QUESTION_ANSWER,
	"question": QUESTION_ANSWER,
	"cypher":   CYPHER,
	"neo4j":    CYPHER,
	"debug":    DEBUG,
	"help":     HELP,
}

type Mode interface {
	Start() (string, Command, error)
	HandleResponse(input string) (string, Command, error)
	Stop() error
}

// IChat is an interface for chat operations
type IChat interface {
	HandleUserMessage(userMessage string) (string, Command, error)
	GetHistory() []Message
	PrintHistory()
}

// ModeHandler is an interface for different types of modes
type ModeHandler interface {
	Start() (string, error)
	HandleResponse(userMessage string) (string, Command, error)
}

// Message represents a single chat message
type Message struct {
	Sender  string
	Content string
}

// Chat handles the chat functionality
type Chat struct {
	mutex               sync.Mutex
	history             []Message
	aiClient            AIClient
	conversationSummary string
	instructor          *instructor.InstructorOpenAI

	modeManager *ModeManager
}

// NewChat creates a new Chat instance
func NewChat(aiClient AIClient) *Chat {
	instructorClient := ai.GetInstructor()
	return &Chat{
		aiClient:    aiClient,
		mutex:       sync.Mutex{},
		history:     []Message{},
		modeManager: &ModeManager{},
		instructor:  instructorClient,
	}
}

func (c *Chat) GetModeText() string {
	if c.modeManager.currentMode != nil {
		return fmt.Sprintf("%T", c.modeManager.currentMode)
	}
	return ""
}

// addMessage adds a message to the chat history
func (c *Chat) addMessage(sender, content string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	msg := Message{Sender: sender, Content: content}
	c.history = append(c.history, msg)
}

func (c *Chat) generateConversationSummary() {
	latestMessages := ""
	for _, msg := range c.GetHistory() {
		latestMessages += fmt.Sprintf("%s: %s\n", msg.Sender, msg.Content)
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

// GetAIResponse calls the OpenAI API to get a response based on user input
func (c *Chat) GetAIResponse(userMessage string) string {

	prompt := fmt.Sprintf(`
	Conversation Summary:
	%s
	Given the user message: 
	%s
	Generate a response.`, c.GetConversationSummary(), userMessage)
	response, err := c.aiClient.GetResponse(prompt)
	log.Printf("GetAIResponse Prompt: %s", prompt)
	if err != nil {
		return "Sorry, I couldn't process that."
	}
	return response
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
		history += fmt.Sprintf("%s: %s\n", msg.Sender, msg.Content)
	}
	history += fmt.Sprintf("====================\n")
	return history
}

// DetectMode detects the Mode based on user input, if any
func (c *Chat) DetectMode(userMessage string) (TMode, bool) {
	userInputLower := strings.ToLower(strings.TrimSpace(userMessage))
	if mode, exists := modeKeywords[userInputLower]; exists {
		return mode, true
	}
	return "", false
}

func (c *Chat) HandleUserMessage(userMessage string) (string, Command, error) {
	c.generateConversationSummary()

	// If in a mode, delegate input handling to the mode manager
	if c.modeManager.currentMode != nil {
		if userMessage == "/exit" || userMessage == "/quit" || userMessage == "/bye" || userMessage == "exit" || userMessage == "quit" || userMessage == "bye" {
			c.modeManager.StopMode()
			return "Exited mode.", MODE_QUIT, nil
		}
		response, command, err := c.modeManager.HandleInput(userMessage)
		if command != SILENT {
			c.addMessage("You", userMessage)
			c.addMessage("AI", response)
		}
		return response, command, err
	}

	// Detect and start new modes using modeManager
	if modeName, detected := c.DetectMode(userMessage); detected {
		mode, err := c.CreateMode(modeName)
		if err != nil {
			return "", NOOP, err
		}
		response, command, err := c.modeManager.StartMode(mode)
		if err != nil {
			return "", NOOP, err
		}
		if command != SILENT {
			c.addMessage("You", userMessage)
			c.addMessage("AI", response)
		}
		return response, command, nil
	}
	c.addMessage("You", userMessage)

	// Regular AI response
	aiResponse := c.GetAIResponse(userMessage)
	c.addMessage("AI", aiResponse)
	return aiResponse, NOOP, nil
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
func (c *Chat) GetHistory() []Message {
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
			log.Fatalf("ListenAndServe(): %s", err)
		}
	}()

	// Wait for shutdown signal
	<-s.shutdownChan
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server Shutdown Failed:%+v", err)
	}
	log.Println("Server exited properly")
}

func (s *HttpChat) ShutdownServer() {
	defer s.chat.PrintHistory()
	fmt.Println("Shutting down server...")
	if err := syscall.Kill(syscall.Getpid(), syscall.SIGINT); err != nil {
		fmt.Println("Error shutting down server:", err)
	}
}

func (s *HttpChat) handlePostMessage(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	response, command, err := s.chat.HandleUserMessage(request.Message)
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
		Response string  `json:"response"`
		Command  *string `json:"command,omitempty"`
	}{
		Response: response,
	}
	if command != NOOP {
		tmp := string(command)
		res.Command = &tmp
	}
	json.NewEncoder(w).Encode(res)
}

func (s *HttpChat) handleGetHistory(w http.ResponseWriter, r *http.Request) {
	history := s.chat.GetHistory()
	json.NewEncoder(w).Encode(history)
}

func initLogger() *os.File {
	logFile, err := os.OpenFile("chatapp.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	log.SetOutput(logFile)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	return logFile
}

func main() {
	// Initialize logger
	logFile := initLogger()
	defer logFile.Close()

	// Setup signal handling for graceful exit
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	shutdownChan := make(chan struct{})

	// Instantiate chat service
	chat := NewChat(NewGptAiClient())

	go func() {
		// Decide to run CLI or HTTP server based on flag or environment
		if len(os.Args) > 1 && os.Args[1] == "http" {
			// Start HTTP Server
			httpServer := NewHTTPServer(chat, shutdownChan)
			httpServer.Start()
		} else if len(os.Args) > 1 && os.Args[1] == "bubble_tea" {
			mainBubbleTea(chat, shutdownChan)
		} else {
			// Start CLI
			runCLI(chat, shutdownChan)
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
	var myEnv map[string]string
	myEnv, err := godotenv.Read()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

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
func (mm *ModeManager) StartMode(mode Mode) (string, Command, error) {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	if mm.currentMode != nil {
		if err := mm.currentMode.Stop(); err != nil {
			return "", NOOP, err
		}
	}
	mm.currentMode = mode
	res, command, err := mode.Start()
	if command == MODE_QUIT {
		if err := mm.currentMode.Stop(); err != nil {
			return "", NOOP, err
		}
		mm.currentMode = nil
	}
	return res, command, err

}

// HandleInput handles the user input based on the current mode
func (mm *ModeManager) HandleInput(input string) (string, Command, error) {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	if mm.currentMode != nil {
		response, command, err := mm.currentMode.HandleResponse(input)
		if command == MODE_QUIT {
			if err := mm.currentMode.Stop(); err != nil {
				return "", NOOP, err
			}
			mm.currentMode = nil
		}
		return response, command, err
	}
	return "", NOOP, fmt.Errorf("no mode is currently active")
}

// StopMode stops the current mode
func (mm *ModeManager) StopMode() error {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	if mm.currentMode != nil {
		if err := mm.currentMode.Stop(); err != nil {
			return err
		}
		mm.currentMode = nil
	}
	return nil
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
	chat := NewChat(mockAIClient)

	response, command, err := chat.HandleUserMessage("hello")
	require.NoError(t, err)
	require.Equal(t, "Hi there!", response)
	require.Equal(t, NOOP, command)
}

func TestChat_DetectMode(t *testing.T) {
	chat := NewChat(&MyMockAiClient{})

	mode, detected := chat.DetectMode("exit")
	require.True(t, detected)
	require.Equal(t, EXIT, mode)

	mode, detected = chat.DetectMode("code")
	require.True(t, detected)
	require.Equal(t, CODE, mode)

	mode, detected = chat.DetectMode("add_test")
	require.True(t, detected)
	require.Equal(t, ADD_TEST, mode)

	mode, detected = chat.DetectMode("invalid")
	require.False(t, detected)
	require.Equal(t, TMode(""), mode)
}
