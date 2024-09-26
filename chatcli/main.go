package main

import (
	"bufio"
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
	"time"
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

type Mode string

var EXIT Mode = "exit"
var CODE Mode = "code"
var ADD_TEST Mode = "add_test"

// Keywords for detecting different modes
var modeKeywords = map[string]Mode{
	"exit":     EXIT,
	"quit":     EXIT,
	"bye":      EXIT,
	"code":     CODE,
	"add_test": ADD_TEST,
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
	mutex       sync.Mutex
	history     []Message
	currentMode ModeHandler
	aiClient    AIClient
}

// NewChat creates a new Chat instance
func NewChat() *Chat {
	return &Chat{
		aiClient: &MyMockAiClient{},
		mutex:    sync.Mutex{},
		history:  []Message{},
	}
}

// AddMessage adds a message to the chat history
func (c *Chat) AddMessage(sender, content string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	msg := Message{Sender: sender, Content: content}
	c.history = append(c.history, msg)
}

// GetAIResponse calls the OpenAI API to get a response based on user input
func (c *Chat) GetAIResponse(userMessage string) string {

	response, err := c.aiClient.GetResponse(userMessage)
	if err != nil {
		return "AI: Sorry, I couldn't process that."
	}
	return response
}

// PrintHistory prints the entire chat history
func (c *Chat) PrintHistory() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	fmt.Printf("=== Chat History ===\n")
	for _, msg := range c.history {
		fmt.Printf("%s: %s\n", msg.Sender, msg.Content)
	}
	fmt.Printf("====================\n")
}

// DetectMode detects the Mode based on user input, if any
func (c *Chat) DetectMode(userMessage string) (Mode, bool) {
	userInputLower := strings.ToLower(strings.TrimSpace(userMessage))
	if mode, exists := modeKeywords[userInputLower]; exists {
		return mode, true
	}
	return "", false
}

// StartMode puts the chat in a mode based on the Mode name
func (c *Chat) StartMode(modeName Mode) (string, Command, error) {
	var mode ModeHandler
	switch modeName {
	case CODE:
		mode = NewCodeMode(c)
	case ADD_TEST:
		mode = NewAddTestMode(c)
	case EXIT:
		c.currentMode = nil
		return "bye", QUIT, nil
	default:
		return "", NOOP, fmt.Errorf("unknown mode: %s", modeName)
	}
	c.currentMode = mode
	response, err := mode.Start()
	return response, MODE_START, err
}

// HandleUserMessage handles the user input and returns the AI response
func (c *Chat) HandleUserMessage(userMessage string) (string, Command, error) {
	c.AddMessage(SenderYou, userMessage)

	// if in a mode(code/test/etc), handle the response in the mode
	if c.currentMode != nil {
		response, command, err := c.currentMode.HandleResponse(userMessage)
		if command == MODE_QUIT {
			c.currentMode = nil
		}
		c.AddMessage(SenderAI, response)
		return response, command, err
	}

	if modeName, detected := c.DetectMode(userMessage); detected {
		response, command, err := c.StartMode(modeName)
		if err != nil {
			return "", command, err
		}
		c.AddMessage(SenderAI, response)
		return response, command, nil
	}

	aiResponse := c.GetAIResponse(userMessage)
	c.AddMessage(SenderAI, aiResponse)
	return aiResponse, NOOP, nil
}

// GetHistory returns the chat history
func (c *Chat) GetHistory() []Message {
	return c.history
}

func runCLI(chat *Chat, shutdownChan chan struct{}) {
	defer chat.PrintHistory()
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("ChatCLI - Type your message and press Enter. Type /help for commands.")
	for {
		select {
		case <-shutdownChan:
			return
		default:
		}

		fmt.Print("You: ")
		userMessage, _ := reader.ReadString('\n')
		userMessage = strings.TrimSpace(userMessage)

		if strings.HasPrefix(userMessage, "/") {
			switch userMessage {
			case "/exit":
				fmt.Println("Bye")
				break
			case "/help":
				displayHelp()
				continue
			case "/code":
				response, _, err := chat.StartMode(CODE)
				if err != nil {
					fmt.Println("Error:", err)
				} else {
					fmt.Println(response)
				}
				continue
			case "/add_test":
				response, _, err := chat.StartMode(ADD_TEST)
				if err != nil {
					fmt.Println("Error:", err)
				} else {
					fmt.Println(response)
				}
				continue
			default:
				fmt.Println("Unknown command:", userMessage)
				continue
			}
		}

		response, command, err := chat.HandleUserMessage(userMessage)
		if err != nil {
			fmt.Println("Error:", err)
		} else if response != "" {
			fmt.Println(response)
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
		case "":
			continue
		default:
			fmt.Printf("Command not recognized: %s\n", command)
		}
	}
}

func displayHelp() {
	fmt.Println("Available commands: /help, /exit, /code, /add_test")
}

type CodeMode struct {
	questionAnswerMap map[string]string
	questions         []string
	questionIndex     int
}

func NewCodeMode(chat *Chat) *CodeMode {
	return &CodeMode{
		// chat:              chat,
		questionAnswerMap: make(map[string]string),
		questions: []string{
			"What kind of api would you like to build?",
			"What's the file name for the main.go file?",
		},
		questionIndex: 0,
	}
}

func (cs *CodeMode) Start() (string, error) {
	question, _, _ := cs.AskNextQuestion()
	return "AI: Starting code mode. I will ask you some questions to generate code.\n" + question, nil
}

func (cs *CodeMode) HandleResponse(userMessage string) (string, Command, error) {
	trimmedInput := strings.TrimSpace(userMessage)
	if trimmedInput == "" {
		question, command, _ := cs.AskNextQuestion()
		return "AI: Input cannot be empty. Please provide a valid response.\n" + question, command, nil
	}

	cs.questionAnswerMap[cs.questions[cs.questionIndex]] = userMessage
	cs.questionIndex++
	if cs.questionIndex < len(cs.questions) {
		question, command, _ := cs.AskNextQuestion()
		return question, command, nil
	} else {
		response, _ := cs.GenerateCode()
		return response, MODE_QUIT, nil
	}
}

func (cs *CodeMode) AskNextQuestion() (string, Command, error) {
	if cs.questionIndex >= len(cs.questions) {
		response, _ := cs.GenerateCode()
		return response, MODE_QUIT, nil
	}
	question := "AI: " + cs.questions[cs.questionIndex]
	return question, NOOP, nil
}

func (cs *CodeMode) GenerateCode() (string, error) {
	codeSnippet := ""
	codeSnippet += fmt.Sprintf("// generated based on these questions:\n")
	for _, q := range cs.questions {
		codeSnippet += fmt.Sprintf("// %s: %s\n", q, cs.questionAnswerMap[q])
	}
	codeSnippet += "<<GENERATED CODE>>"

	return "AI: Thank you for the information. Generating code...\n\n" + codeSnippet, nil
}

type AddTestMode struct {
	questionAnswerMap map[string]string
	questions         []string
	questionIndex     int
}

func NewAddTestMode(chat *Chat) *AddTestMode {
	return &AddTestMode{
		questionAnswerMap: make(map[string]string),
		questions: []string{
			"Which function would you like to test?",
			"In which file is this function located?",
			"Where should the test file be saved?",
			"Are there any specific edge cases you want to cover?",
		},
		questionIndex: 0,
	}
}

func (ats *AddTestMode) Start() (string, error) {
	message := "AI: Starting add test mode. I will ask you some questions to generate a test function."
	question, _ := ats.AskNextQuestion()
	return message + "\n" + question, nil
}

func (ats *AddTestMode) HandleResponse(userMessage string) (string, Command, error) {
	ats.questionAnswerMap[ats.questions[ats.questionIndex]] = userMessage
	ats.questionIndex++
	if ats.questionIndex < len(ats.questions) {
		question, _ := ats.AskNextQuestion()
		return question, NOOP, nil
	} else {
		response, _ := ats.GenerateTestCode()
		return response, MODE_QUIT, nil
	}
}

func (ats *AddTestMode) AskNextQuestion() (string, error) {
	if ats.questionIndex >= len(ats.questions) {
		response, _ := ats.GenerateTestCode()
		return response, nil
	}
	question := "AI: " + ats.questions[ats.questionIndex]
	return question, nil
}

func (ats *AddTestMode) GenerateTestCode() (string, error) {
	// Generate test code based on user inputs
	testCode := "<<GENERATED TEST CODE>>"
	return "AI: Generating test code based on your inputs...\n" + testCode, nil
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

func main() {
	// Setup signal handling for graceful exit
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	shutdownChan := make(chan struct{})

	// Instantiate chat service
	chat := NewChat()

	go func() {
		// Decide to run CLI or HTTP server based on flag or environment
		if len(os.Args) > 1 && os.Args[1] == "http" {
			// Start HTTP Server
			httpServer := NewHTTPServer(chat, shutdownChan)
			httpServer.Start()
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

type AIClient interface {
	GetResponse(prompt string) (string, error)
}

type MyMockAiClient struct{}

func (c *MyMockAiClient) GetResponse(prompt string) (string, error) {
	return "AI: This is a mock response.", nil
}
