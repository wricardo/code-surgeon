package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/sashabaranov/go-openai"
	codesurgeon "github.com/wricardo/code-surgeon"
)

func main() {
	request := getUserRequest()

	fmt.Println("AI Software Engineer response:")
	fileContent, err := readFileContents("dynamic.go")
	if err != nil {
		log.Fatalf("Failed to read file: %v", err)
	}
	implementationsMap := analyzeRequest(fileContent, request)
	fmt.Println(implementationsMap)

	codesurgeon.InsertCodeFragments(implementationsMap)
}

// Simulate the AI analysis and response
func analyzeRequest(fileContents, request string) map[string][]codesurgeon.CodeFragment {
	client := openai.NewClient(os.Getenv("OPENAI_API_KEY"))
	resp, err := client.CreateChatCompletion(context.Background(), openai.ChatCompletionRequest{
		Model: openai.GPT3Dot5Turbo,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: fileContents,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: request,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: "write only the golang function without any package or import. Just the code block with the golang function.",
			},
		},
	})

	if err != nil {
		log.Fatalf("Failed to get response from OpenAI: %v", err)
	}

	fn := func(str string) codesurgeon.CodeFragment {
		str = extractCodeBlock(str, "go")
		return codesurgeon.CodeFragment{Content: str}
	}

	// Process the AI response to map it into the fragments
	// For this example, we'll just use the response directly
	return map[string][]codesurgeon.CodeFragment{
		"dynamic.go": {
			fn(resp.Choices[0].Message.Content),
		},
	}
}

// Get user input for request
func getUserRequest() string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Please enter your modification request: ")
	request, _ := reader.ReadString('\n')
	return strings.TrimSpace(request)
}

func readFileContents(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var builder strings.Builder
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		builder.WriteString(scanner.Text() + "\n")
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return builder.String(), nil
}

// extractCodeBlock extracts the content of a code block with the specified language identifier (like "xyz").
func extractCodeBlock(input, language string) string {
	// Regular expression pattern to match the code block with the specified language
	pattern := fmt.Sprintf("(?s)```%s\\s*(.*?)\\s*```", regexp.QuoteMeta(language))
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(input)
	if len(matches) > 1 {
		// Return the captured content inside the code block
		return matches[1]
	}
	return ""
}
