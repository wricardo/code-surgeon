package main

import (
	"bufio"
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
var modeKeywords = map[Mode][]string{
	// QUIT:     {"exit", "quit", "bye"},
	CODE:     {"code", "coding", "program", "develop"},
	ADD_TEST: {"test", "testing", "unit test", "write test"},
}

// ChatService is an interface for chat operations
type ChatService interface {
	StartMode(modeName Mode) (string, Command, error)
	HandleUserMessage(userInput string) (string, Command, error)
	GetHistory() []Message
}

// ModeHandler is an interface for different types of modes
type ModeHandler interface {
	Start() (string, error)
	HandleResponse(userInput string) (string, Command, error)
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
}

// NewChat creates a new Chat instance
func NewChat() *Chat {
	return &Chat{
		mutex:   sync.Mutex{},
		history: []Message{},
	}
}

// AddMessage adds a message to the chat history
func (c *Chat) AddMessage(sender, content string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	msg := Message{Sender: sender, Content: content}
	c.history = append(c.history, msg)
}

func (c *Chat) GetAIResponse(userInput string) string {
	response, err := callOpenAIApi(userInput)
	if err != nil {
		return "AI: Sorry, I couldn't process that."
	}
	return response
}

// PrintHistory prints the entire chat history
func (c *Chat) PrintHistory() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	for _, msg := range c.history {
		fmt.Printf("%s: %s\n", msg.Sender, msg.Content)
	}
}

func (c *Chat) DetectMode(userInput string) (Mode, bool) {
	userInputLower := strings.ToLower(userInput)
	for modeName, keywords := range modeKeywords {
		for _, keyword := range keywords {
			if strings.Contains(userInputLower, keyword) {
				return modeName, true
			}
		}
	}
	return "", false
}

// Function to call OpenAI API (pseudo-code)
func callOpenAIApi(prompt string) (string, error) {
	return "aaaa", nil
}

func main() {
	// Setup signal handling for graceful exit
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-signalChan
		fmt.Println("\nBye")
		os.Exit(0)
	}()

	// Instantiate chat service
	chat := NewChat()

	// Decide to run CLI or HTTP server based on flag or environment
	if len(os.Args) > 1 && os.Args[1] == "http" {
		// Start HTTP Server
		httpServer := NewHTTPServer(chat)
		httpServer.Start()
	} else {
		// Start CLI
		runCLI(chat)
	}
}

func runCLI(chat *Chat) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("ChatCLI - Type your message and press Enter. Type /help for commands.")
	for {
		fmt.Print("You: ")
		userInput, _ := reader.ReadString('\n')
		userInput = strings.TrimSpace(userInput)

		if userInput == "/exit" {
			fmt.Println("Bye")
			break
		}

		response, command, err := chat.HandleUserMessage(userInput)
		if err != nil {
			fmt.Println("Error:", err)
		} else if response != "" {
			fmt.Println(response)
		}
		log.Println("Command:", command)
		switch command {
		case QUIT:
			return
		case NOOP:
			continue
		case MODE_QUIT:
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
	chat              *Chat
	questionAnswerMap map[string]string
	questions         []string
	questionIndex     int
}

func NewCodeMode(chat *Chat) *CodeMode {
	return &CodeMode{
		chat:              chat,
		questionAnswerMap: make(map[string]string),
		questions: []string{
			"What programming language would you like to use?",
			"What is the primary purpose of the code? (e.g., web server, data processing)",
			"Do you have any specific libraries or frameworks in mind?",
			"Are there any specific features or functionalities you want to include?",
		},
		questionIndex: 0,
	}
}

func (cs *CodeMode) Start() (string, error) {
	message := "AI: Starting code mode. I will ask you some questions to generate code."
	question, _ := cs.AskNextQuestion()
	return message + "\n" + question, nil
}

func (cs *CodeMode) HandleResponse(userInput string) (string, Command, error) {
	trimmedInput := strings.TrimSpace(userInput)
	if trimmedInput == "" {
		question, _ := cs.AskNextQuestion()
		return "AI: Input cannot be empty. Please provide a valid response.\n" + question, NOOP, nil
	}

	cs.questionAnswerMap[cs.questions[cs.questionIndex]] = userInput
	cs.questionIndex++
	if cs.questionIndex < len(cs.questions) {
		question, _ := cs.AskNextQuestion()
		return question, NOOP, nil
	} else {
		response, _ := cs.GenerateCode()
		return response, MODE_QUIT, nil
	}
}

func (cs *CodeMode) AskNextQuestion() (string, error) {
	if cs.questionIndex >= len(cs.questions) {
		response, _ := cs.GenerateCode()
		cs.chat.currentMode = nil // End the mode
		return response, nil
	}
	question := "AI: " + cs.questions[cs.questionIndex]
	return question, nil
}

func (cs *CodeMode) GenerateCode() (string, error) {
	language := cs.questionAnswerMap["What programming language would you like to use?"]
	purpose := cs.questionAnswerMap["What is the primary purpose of the code? (e.g., web server, data processing)"]
	libraries := cs.questionAnswerMap["Do you have any specific libraries or frameworks in mind?"]
	features := cs.questionAnswerMap["Are there any specific features or functionalities you want to include?"]

	codeSnippet := fmt.Sprintf(
		"// Generated Code\n// Language: %s\n// Purpose: %s\n// Libraries/Frameworks: %s\n// Features: %s\n\nfunc main() {\n\t// TODO: Implement the %s\n}",
		language, purpose, libraries, features, purpose,
	)

	cs.chat.AddMessage("AI", codeSnippet)
	return "AI: Thank you for the information. Generating code...\nAI: Here is your generated code:\n" + codeSnippet + "\nExited code mode.", nil
}

type AddTestMode struct {
	chat              *Chat
	questionAnswerMap map[string]string
	questions         []string
	questionIndex     int
}

func NewAddTestMode(chat *Chat) *AddTestMode {
	return &AddTestMode{
		chat:              chat,
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

func (ats *AddTestMode) HandleResponse(userInput string) (string, Command, error) {
	ats.questionAnswerMap[ats.questions[ats.questionIndex]] = userInput
	ats.questionIndex++
	if ats.questionIndex < len(ats.questions) {
		question, _ := ats.AskNextQuestion()
		return question, NOOP, nil
	} else {
		response, _ := ats.GenerateTestCode()
		ats.chat.currentMode = nil // End the mode
		return response, MODE_QUIT, nil
	}
}

func (ats *AddTestMode) AskNextQuestion() (string, error) {
	if ats.questionIndex >= len(ats.questions) {
		response, _ := ats.GenerateTestCode()
		ats.chat.currentMode = nil // End the mode
		return response, nil
	}
	question := "AI: " + ats.questions[ats.questionIndex]
	return question, nil
}

func (ats *AddTestMode) GenerateTestCode() (string, error) {
	// Generate test code based on user inputs
	testCode := "<<GENERATED TEST CODE>>"
	ats.chat.AddMessage("AI", testCode)
	return "AI: Generating test code based on your inputs...\n" + testCode, nil
}

func (c *Chat) StartMode(modeName Mode) (string, Command, error) {
	var mode ModeHandler
	switch modeName {
	case CODE:
		mode = NewCodeMode(c)
	case ADD_TEST:
		mode = NewAddTestMode(c)
	case EXIT:
		c.currentMode = nil
		return "AI: Exiting current mode.", QUIT, nil
	default:
		return "", NOOP, fmt.Errorf("unknown mode: %s", modeName)
	}
	c.currentMode = mode
	response, err := mode.Start()
	return response, MODE_START, err
}

func (c *Chat) HandleUserMessage(userInput string) (string, Command, error) {
	if c.currentMode != nil {
		response, command, err := c.currentMode.HandleResponse(userInput)
		if command == MODE_QUIT {
			c.currentMode = nil
		}
		return response, command, err
	}

	if modeName, detected := c.DetectMode(userInput); detected {
		response, command, err := c.StartMode(modeName)
		if err != nil {
			return "", command, err
		}
		return response, command, nil
	}

	c.AddMessage("You", userInput)
	aiResponse := c.GetAIResponse(userInput)
	c.AddMessage("AI", aiResponse)
	return aiResponse, NOOP, nil
}
func (c *Chat) GetHistory() []Message {
	return c.history
}

type HTTPServer struct {
	chatService ChatService
	mux         sync.Mutex
}

func NewHTTPServer(chatService ChatService) *HTTPServer {
	return &HTTPServer{chatService: chatService}
}

func (s *HTTPServer) Start() {
	http.HandleFunc("/post-message", s.handlePostMessage)
	http.HandleFunc("/get-history", s.handleGetHistory)

	fmt.Println("Starting server on :8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func (s *HTTPServer) ShutdownServer() {
	fmt.Println("Shutting down server...")
	if err := syscall.Kill(syscall.Getpid(), syscall.SIGKILL); err != nil {
		fmt.Println("Error shutting down server:", err)
	}
}

func (s *HTTPServer) handlePostMessage(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	response, command, err := s.chatService.HandleUserMessage(request.Message)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	switch command {
	case QUIT:
		w.Write([]byte("Exited mode."))
		go func() { time.Sleep(time.Millisecond * 100); s.ShutdownServer() }()
	case NOOP:
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

func (s *HTTPServer) handleGetHistory(w http.ResponseWriter, r *http.Request) {
	history := s.chatService.GetHistory()
	json.NewEncoder(w).Encode(history)
}
