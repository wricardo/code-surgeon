package main

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

// Message represents a single chat message
type Message struct {
	Sender  string
	Content string
}

// Chat handles the chat functionality
type Chat struct {
	history              []Message
	isInCodeSession      bool
	codeSessionData      map[string]string
	codeQuestions        []string
	currentQuestionIndex int
}

// NewChat creates a new Chat instance
func NewChat() *Chat {
	return &Chat{
		history:         []Message{},
		isInCodeSession: false,
		codeSessionData: make(map[string]string),
		codeQuestions: []string{
			"What programming language would you like to use?",
			"What is the primary purpose of the code? (e.g., web server, data processing)",
			"Do you have any specific libraries or frameworks in mind?",
			"Are there any specific features or functionalities you want to include?",
		},
		currentQuestionIndex: 0,
	}
}

// AddMessage adds a message to the chat history
func (c *Chat) AddMessage(sender, content string) {
	msg := Message{Sender: sender, Content: content}
	c.history = append(c.history, msg)
}

// GetAIResponse generates an AI response based on chat history and user input
func (c *Chat) GetAIResponse(userInput string) string {
	// This function should call the AI model to get a response based on the chat history and user input.
	// For now, we will just hardcode a response with a simulated delay.
	time.Sleep(1 * time.Second) // Simulate processing delay
	return "This is a simulated AI response."
}

// PrintHistory prints the entire chat history
func (c *Chat) PrintHistory() {
	for _, msg := range c.history {
		fmt.Printf("%s: %s\n", msg.Sender, msg.Content)
	}
}

// StartCodeSession initiates the code session
func (c *Chat) StartCodeSession() {
	c.isInCodeSession = true
	c.codeSessionData = make(map[string]string)
	c.currentQuestionIndex = 0
	fmt.Println("Entered code session. Let's gather the requirements.")
	c.AskNextQuestion()
}

// AskNextQuestion asks the next question in the code session
func (c *Chat) AskNextQuestion() {
	if c.currentQuestionIndex < len(c.codeQuestions) {
		question := c.codeQuestions[c.currentQuestionIndex]
		fmt.Println("AI: " + question)
	} else {
		c.GenerateCode()
	}
}

// HandleCodeResponse processes the user's response during a code session
func (c *Chat) HandleCodeResponse(userInput string) {
	// Store the response
	currentQuestion := c.codeQuestions[c.currentQuestionIndex]
	c.codeSessionData[currentQuestion] = userInput
	c.AddMessage("You", userInput)

	// Move to the next question
	c.currentQuestionIndex++
	if c.currentQuestionIndex < len(c.codeQuestions) {
		c.AskNextQuestion()
	} else {
		c.GenerateCode()
	}
}

// GenerateCode creates a code snippet based on the collected data
func (c *Chat) GenerateCode() {
	fmt.Println("AI: Thank you for the information. Generating code...")
	// Simulate processing delay
	time.Sleep(2 * time.Second)

	// Simple code generation logic based on user input
	language := c.codeSessionData["What programming language would you like to use?"]
	purpose := c.codeSessionData["What is the primary purpose of the code? (e.g., web server, data processing)"]
	libraries := c.codeSessionData["Do you have any specific libraries or frameworks in mind?"]
	features := c.codeSessionData["Are there any specific features or functionalities you want to include?"]

	codeSnippet := fmt.Sprintf("// Generated Code\n// Language: %s\n// Purpose: %s\n// Libraries/Frameworks: %s\n// Features: %s\n\nfunc main() {\n\t// TODO: Implement the %s\n}", language, purpose, libraries, features, purpose)

	fmt.Println("AI: Here is your generated code:")
	fmt.Println(codeSnippet)

	// Add the code snippet to chat history
	c.AddMessage("AI", codeSnippet)

	// End the code session
	c.isInCodeSession = false
	fmt.Println("Exited code session.")
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

	reader := bufio.NewReader(os.Stdin)
	chat := NewChat()

	fmt.Println("ChatCLI - Type your message and press Enter. Type /help for commands.")

	for {
		if chat.isInCodeSession {
			// During code session, prompt for input without "You: "
			userInput, _ := reader.ReadString('\n')
			userInput = strings.TrimSpace(userInput)

			if userInput == "/exit" {
				fmt.Println("Bye")
				break
			}

			chat.HandleCodeResponse(userInput)
		} else {
			// Regular chat mode
			fmt.Print("You: ")
			userInput, _ := reader.ReadString('\n')
			userInput = strings.TrimSpace(userInput)

			// Handle commands
			if userInput == "/exit" {
				fmt.Println("Bye")
				break
			} else if userInput == "/help" {
				fmt.Println("Available commands: /help, /exit, /code")
				continue
			} else if userInput == "/code" {
				chat.StartCodeSession()
				continue
			}

			// Add user message to history
			chat.AddMessage("You", userInput)

			// Get AI response
			aiResponse := chat.GetAIResponse(userInput)
			fmt.Println("AI: " + aiResponse)

			// Add AI response to history
			chat.AddMessage("AI", aiResponse)
		}
	}
}
