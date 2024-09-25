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

var sessionKeywords = map[string][]string{
	"exit":     {"exit", "quit", "bye"},
	"code":     {"code", "coding", "program", "develop"},
	"add_test": {"test", "testing", "unit test", "write test"},
}

// ChatService is an interface for chat operations
type ChatService interface {
	StartSession(sessionName string) error
	HandleUserMessage(userInput string) (string, error)
	GetHistory() []Message
}

// Session is an interface for different types of sessions
type Session interface {
	Start()
	HandleResponse(userInput string)
}

// Message represents a single chat message
type Message struct {
	Sender  string
	Content string
}

// Chat handles the chat functionality
type Chat struct {
	mutex          sync.Mutex
	history        []Message
	currentSession Session
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

func (c *Chat) DetectSession(userInput string) (string, bool) {
	userInputLower := strings.ToLower(userInput)
	for sessionName, keywords := range sessionKeywords {
		for _, keyword := range keywords {
			if strings.Contains(userInputLower, keyword) {
				return sessionName, true
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

		if chat.currentSession != nil {
			chat.currentSession.HandleResponse(userInput)
			continue
		}

		if strings.HasPrefix(userInput, "/") {
			handleCommand(userInput, chat)
			continue
		}

		if sessionName, detected := chat.DetectSession(userInput); detected {
			promptSessionStart(sessionName, chat, reader)
			continue
		}

		chat.AddMessage("You", userInput)
		aiResponse := chat.GetAIResponse(userInput)
		fmt.Println("AI: " + aiResponse)
		chat.AddMessage("AI", aiResponse)
	}
}

/*
// Main function
func main() {
	// Setup signal handling for graceful exit
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-signalChan
		fmt.Println("\nBye")
		os.Exit(0)
	}()

	reader := bufio.NewReader(os.Stdin)
	chat := NewChat()

	fmt.Println("ChatCLI - Type your message and press Enter. Type /help for commands.")
	for {
		fmt.Print("You: ")
		userInput, _ := reader.ReadString('\n')
		userInput = strings.TrimSpace(userInput)

		if userInput == "/exit" {
			fmt.Println("Bye")
			break
		}

		if chat.currentSession != nil {
			chat.currentSession.HandleResponse(userInput)
			continue
		}

		if strings.HasPrefix(userInput, "/") {
			handleCommand(userInput, chat)
			continue
		}

		if sessionName, detected := chat.DetectSession(userInput); detected {
			promptSessionStart(sessionName, chat, reader)
			continue
		}

		chat.AddMessage("You", userInput)
		aiResponse := chat.GetAIResponse(userInput)
		fmt.Println("AI: " + aiResponse)
		chat.AddMessage("AI", aiResponse)
	}
}
*/

func promptSessionStart(sessionName string, chat *Chat, reader *bufio.Reader) {
	fmt.Printf("AI: It sounds like you might want to start a '/%s' session. Would you like to do that? (yes/no)\n", sessionName)
	fmt.Print("You: ")
	confirmation, _ := reader.ReadString('\n')
	confirmation = strings.TrimSpace(strings.ToLower(confirmation))
	if confirmation == "yes" || confirmation == "y" {
		switch sessionName {
		case "code":
			chat.currentSession = NewCodeSession(chat)
			chat.currentSession.Start()
		case "add_test":
			chat.currentSession = NewAddTestSession(chat)
			chat.currentSession.Start()
		case "exit":
			fmt.Println("Bye")
			os.Exit(0)
		}
	} else {
		fmt.Println("AI: Alright, let's continue our chat.")
	}
}

func handleCommand(command string, chat *Chat) {
	switch command {
	case "/help":
		displayHelp()
	case "/code":
		chat.currentSession = NewCodeSession(chat)
		chat.currentSession.Start()
	case "/add_test":
		chat.currentSession = NewAddTestSession(chat)
		chat.currentSession.Start()
	default:
		fmt.Println("Unknown command. Type /help for available commands.")
	}
}
func displayHelp() {
	fmt.Println("Available commands: /help, /exit, /code, /add_test")
}

type CodeSession struct {
	chat          *Chat
	sessionData   map[string]string
	questions     []string
	questionIndex int
}

func NewCodeSession(chat *Chat) *CodeSession {
	return &CodeSession{
		chat:        chat,
		sessionData: make(map[string]string),
		questions: []string{
			"What programming language would you like to use?",
			"What is the primary purpose of the code? (e.g., web server, data processing)",
			"Do you have any specific libraries or frameworks in mind?",
			"Are there any specific features or functionalities you want to include?",
		},
		questionIndex: 0,
	}
}

func (cs *CodeSession) Start() {
	fmt.Println("AI: Starting code session. I will ask you some questions to generate code.")
	cs.AskNextQuestion()
}

func (cs *CodeSession) HandleResponse(userInput string) {
	trimmedInput := strings.TrimSpace(userInput)
	if trimmedInput == "" {
		fmt.Println("AI: Input cannot be empty. Please provide a valid response.")
		cs.AskNextQuestion()
		return
	}

	cs.sessionData[cs.questions[cs.questionIndex]] = userInput
	cs.questionIndex++
	if cs.questionIndex < len(cs.questions) {
		cs.AskNextQuestion()
	} else {
		cs.GenerateCode()
		cs.chat.currentSession = nil // End the session
	}
}

func (cs *CodeSession) AskNextQuestion() {
	if cs.questionIndex >= len(cs.questions) {
		cs.GenerateCode()
		cs.chat.currentSession = nil // End the session
		return
	}
	question := cs.questions[cs.questionIndex]
	fmt.Println("AI: " + question)
}

func (cs *CodeSession) GenerateCode() {
	fmt.Println("AI: Thank you for the information. Generating code...")
	// Simulate processing delay
	time.Sleep(2 * time.Second)

	// Simple code generation logic based on user input
	language := cs.sessionData["What programming language would you like to use?"]
	purpose := cs.sessionData["What is the primary purpose of the code? (e.g., web server, data processing)"]
	libraries := cs.sessionData["Do you have any specific libraries or frameworks in mind?"]
	features := cs.sessionData["Are there any specific features or functionalities you want to include?"]

	codeSnippet := fmt.Sprintf("// Generated Code\n// Language: %s\n// Purpose: %s\n// Libraries/Frameworks: %s\n// Features: %s\n\nfunc main() {\n\t// TODO: Implement the %s\n}", language, purpose, libraries, features, purpose)

	fmt.Println("AI: Here is your generated code:")
	fmt.Println(codeSnippet)

	// Add the code snippet to chat history
	cs.chat.AddMessage("AI", codeSnippet)
	fmt.Println("Exited code session.")
}

type AddTestSession struct {
	chat          *Chat
	sessionData   map[string]string
	questions     []string
	questionIndex int
}

func NewAddTestSession(chat *Chat) *AddTestSession {
	return &AddTestSession{
		chat:        chat,
		sessionData: make(map[string]string),
		questions: []string{
			"Which function would you like to test?",
			"In which file is this function located?",
			"Where should the test file be saved?",
			"Are there any specific edge cases you want to cover?",
		},
		questionIndex: 0,
	}
}

func (ats *AddTestSession) Start() {
	fmt.Println("AI: Starting add test session. I will ask you some questions to generate a test function.")
	ats.AskNextQuestion()
}

func (ats *AddTestSession) HandleResponse(userInput string) {
	ats.sessionData[ats.questions[ats.questionIndex]] = userInput
	ats.questionIndex++
	if ats.questionIndex < len(ats.questions) {
		ats.AskNextQuestion()
	} else {
		ats.GenerateTestCode()
		ats.chat.currentSession = nil // End the session
	}
}

func (ats *AddTestSession) AskNextQuestion() {
	if ats.questionIndex >= len(ats.questions) {
		ats.GenerateTestCode()
		ats.chat.currentSession = nil // End the session
		return
	}
	question := ats.questions[ats.questionIndex]
	fmt.Println("AI: " + question)
}

func (ats *AddTestSession) GenerateTestCode() {
	fmt.Println("AI: Generating test code based on your inputs...")
	fmt.Println("AI: Here is the generated test code XYZ.")
}

func (c *Chat) StartSession(sessionName string) error {
	switch sessionName {
	case "code":
		c.currentSession = NewCodeSession(c)
	case "add_test":
		c.currentSession = NewAddTestSession(c)
	default:
		return fmt.Errorf("unknown session: %s", sessionName)
	}
	c.currentSession.Start()
	return nil
}

func (c *Chat) HandleUserMessage(userInput string) (string, error) {
	if c.currentSession != nil {
		c.currentSession.HandleResponse(userInput)
		return "", nil
	}

	if sessionName, detected := c.DetectSession(userInput); detected {
		c.StartSession(sessionName)
		return fmt.Sprintf("AI: Started %s session.", sessionName), nil
	}

	c.AddMessage("You", userInput)
	aiResponse := c.GetAIResponse(userInput)
	c.AddMessage("AI", aiResponse)
	return aiResponse, nil
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
	http.HandleFunc("/start-session", s.handleStartSession)
	http.HandleFunc("/post-message", s.handlePostMessage)
	http.HandleFunc("/get-history", s.handleGetHistory)

	fmt.Println("Starting server on :8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func (s *HTTPServer) handleStartSession(w http.ResponseWriter, r *http.Request) {
	sessionName := r.URL.Query().Get("session")
	if sessionName == "" {
		http.Error(w, "Missing session name", http.StatusBadRequest)
		return
	}

	if err := s.chatService.StartSession(sessionName); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write([]byte(fmt.Sprintf("Started session: %s", sessionName)))
}

func (s *HTTPServer) handlePostMessage(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	response, err := s.chatService.HandleUserMessage(request.Message)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"response": response,
	})
}

func (s *HTTPServer) handleGetHistory(w http.ResponseWriter, r *http.Request) {
	history := s.chatService.GetHistory()
	json.NewEncoder(w).Encode(history)
}
