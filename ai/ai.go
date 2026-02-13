// Package ai provides unified AI integration with OpenAI and Anthropic APIs.
// It uses instructor-go for type-safe structured outputs and supports operations
// like question answering, cypher query generation, embeddings, and problem analysis.
package ai

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"text/template"

	"github.com/Jeffail/gabs"
	"github.com/instructor-ai/instructor-go/pkg/instructor"
	"github.com/joho/godotenv"
	"github.com/sashabaranov/go-openai"
	codesurgeon "github.com/wricardo/code-surgeon"
)

// ThinkThroughProblemRequest represents a request to think through a problem
type ThinkThroughProblemRequest struct {
	Goal             string
	Context          string
	ProblemStatement string
	Questions        []string
	QuestionsString  string
}

// ThinkThroughProblemResponse represents the response from thinking through a problem
type ThinkThroughProblemResponse struct {
	Answers          []*QuestionAnswer
	Observations     string
	SimilarQuestions []*QuestionAnswer
}

// QuestionAnswer represents a question and answer pair
type QuestionAnswer struct {
	Question string
	Answer   string
}

// ThinkThroughProblem uses AI to analyze a problem statement and answer related questions.
// It takes a request with context, problem statement, and questions, along with optional
// similar question-answer pairs for reference. Returns answers, observations, and the provided
// similar questions for context.
func ThinkThroughProblem(client *instructor.InstructorOpenAI, req *ThinkThroughProblemRequest, similarQuestionsAnswers []*QuestionAnswer) (res *ThinkThroughProblemResponse, err error) {
	type InternalQuestionAnswer struct {
		Question string `json:"question" jsonschema:"title=question,description=the question asked by the user."`
		Answer   string `json:"answer" jsonschema:"title=answer,description=the answer to the question asked by the user."`
	}
	type AiOutput struct {
		Answers      []*InternalQuestionAnswer `json:"answers,omitempty"`
		Observations string                    `protobuf:"bytes,2,opt,name=observations,proto3" json:"observations,omitempty"`
	}

	ctx := context.Background()
	var aiOut AiOutput

	similarQuestionsAnswersTxt := ""
	for _, qa := range similarQuestionsAnswers {
		similarQuestionsAnswersTxt += fmt.Sprintf(`
		<QA>
		Q:%s
		A:%s
		</QA>
		`, qa.Question, qa.Answer)
	}

	_, err = client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: openai.GPT4o,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    "system",
				Content: "You are a helpful assistant which can help the user think through a problem based on information from contextt and previous seen questions. You must answer in json format with the keys being 'answers' and 'observations'.",
			},
			{
				Role:    "user",
				Content: fmt.Sprintf(`<previously_seen_questions>%s</previously_seen_questions> <context>%s</context> <problemStatement>%s</problemStatement> <questions>%s</questions>`, similarQuestionsAnswersTxt, req.Context, req.ProblemStatement, strings.Join(req.Questions, ",")),
			},
		},
		MaxTokens: 1000,
	}, &aiOut)

	if err != nil {
		return res, fmt.Errorf("Failed to think through problem: %v", err)
	}

	if len(aiOut.Answers) == 0 {
		return res, nil
	}

	finalRes := ThinkThroughProblemResponse{}
	for _, qa := range aiOut.Answers {
		finalRes.Answers = append(finalRes.Answers, &QuestionAnswer{Question: qa.Question, Answer: qa.Answer})
	}

	finalRes.Observations = aiOut.Observations
	finalRes.SimilarQuestions = similarQuestionsAnswers

	return &finalRes, nil
}

// GetGPTInstructions generates AI assistant instructions from an OpenAPI specification.
// It extracts available actions from the OpenAPI definition and returns a formatted
// instruction string for configuring an AI assistant's behavior.
func GetGPTInstructions(openapi string) (string, error) {
	actions, err := getActionsFromOpenApiDev(openapi)
	if err != nil {
		return "", err
	}
	m := map[string]interface{}{
		"Actions": actions,
	}
	return codesurgeon.RenderTemplate(`
	{{.Actions}}

You are a helpful assistant with instructions on software engineering.
Let's have a conversation and whenever I tell you to MSG:<something>, you call the action SendMessage with <something> in the message as a text.
`, m)
}

// GetGPTIntroduction generates an introduction message for an AI assistant based on an OpenAPI specification.
// It creates a friendly greeting that includes available actions extracted from the OpenAPI definition.
func GetGPTIntroduction(openapiDef string) (string, error) {
	actions, err := getActionsFromOpenApiDev(openapiDef)
	if err != nil {
		return "", err
	}
	// if symbolsByFileCache == "" {
	// 	CacheSymbols()
	// }

	return codesurgeon.RenderTemplate(`
	Hi, I'm Patna and I'm a helpful and experienced golang developer. I can help you with your project.
			{{.Actions}}

			`, map[string]any{
		"Actions": actions,
	})
}

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

func Render(tempstring string, data interface{}) string {
	tmpl, err := template.New("").Parse(tempstring)
	if err != nil {
		log.Fatalf("Failed to parse template: %v", err)
	}

	var builder strings.Builder
	err = tmpl.Execute(&builder, data)
	if err != nil {
		log.Fatalf("Failed to execute template: %v", err)
	}

	return builder.String()
}

func getActionsFromOpenApiDev(openapi string) (string, error) {
	actions := []string{}
	parsed, err := gabs.ParseJSON([]byte(openapi))
	if err != nil {
		return "", err
	}

	paths, err := parsed.Path("paths").ChildrenMap()
	if err != nil {
		return "", err
	}

	for _, pathData := range paths {
		methods, err := pathData.ChildrenMap()
		if err != nil {
			return "", err
		}
		// get operationId and append to actions
		for _, methodData := range methods {
			operationID := methodData.Path("operationId").Data().(string)
			actions = append(actions, operationID)
		}

	}

	return codesurgeon.RenderTemplate(`You may use these Actions:
			{{range .Actions}}
			- {{.}}
			{{end}}
			`, map[string]interface{}{
		"Actions": actions,
	})
}

// EmbedText embeds a user's question into a vector representation using a predefined embedding model.
func EmbedText(client *openai.Client, text string) ([]float32, error) {
	resp, err := client.CreateEmbeddings(context.Background(), openai.
		EmbeddingRequest{Input: []string{text}, Model: openai.AdaEmbeddingV2})
	if err != nil {
		return nil, err
	}
	if len(resp.Data) == 0 {
		return nil, errors.New("no embedding found in response")
	}
	return resp.Data[0].Embedding, nil
}

// GenerateCypher uses AI to generate a Cypher query based on a natural language request.
// It takes a question or description and returns a Neo4j Cypher query string that
// can be executed against a Neo4j database.
func GenerateCypher(client *instructor.InstructorOpenAI, ask string) (string, error) {
	type AiOutput struct {
		Cypher string `json:"cypher" jsonschema:"title=cypher,description=the cypher query to be executed on the neo4j database."`
	}

	ctx := context.Background()
	var aiOut AiOutput

	_, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: openai.GPT4o,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    "system",
				Content: "You are a helpful assistant which can generate cypher queries based on the user's request. You must answer in json format with the key being 'cypher'.",
			},
			{
				Role:    "user",
				Content: ask,
			},
		},
		MaxTokens: 1000,
	}, &aiOut)

	if err != nil {
		return "", fmt.Errorf("Failed to generate cypher query: %v", err)
	}

	if aiOut.Cypher == "" {
		return "", nil
	}

	return aiOut.Cypher, nil
}

// GenerateFinalAnswer synthesizes an answer to a question based on previous question-answer pairs.
// It uses recent Q&A context to provide factually grounded responses. The questionsAnswers slice
// is reversed internally to prioritize the most recent information.
func GenerateFinalAnswer(client *instructor.InstructorOpenAI, question string, questionsAnswers []QuestionAnswer) (string, error) {
	// Create a struct to capture the AI's response
	type AiOutput struct {
		FinalAnswer string `json:"final_answer" jsonschema:"title=final_answer,description=the final answer to the user's question."`
	}

	// Context for the OpenAI API call
	ctx := context.Background()

	// reverse the order of the questionsAnswers
	for i, j := 0, len(questionsAnswers)-1; i < j; i, j = i+1, j-1 {
		questionsAnswers[i], questionsAnswers[j] = questionsAnswers[j], questionsAnswers[i]
	}

	// Construct the messages for the chat completion
	messages := []openai.ChatCompletionMessage{
		{
			Role:    "system",
			Content: "You are a helpful assistant which can answer questions based on previous knowledge and more importantly recent questions and answers given. Answer to the best of your ability but stay close to the facts seen on recent questions and answers. You must answer in json format with the key being 'final_answer'.",
		},
		{
			Role: "user",
			Content: Render(`
			Recent questions and answers:
			{{range .questionsAnswers}}
			<section>
			Q:{{.Question}}
			A:{{.Answer}}
			</section>
			{{end}}
			Now, answer the following question:
			{{.question}}
			`, map[string]interface{}{"questionsAnswers": questionsAnswers, "question": question}),
		},
	}

	fmt.Printf("============\n%s\n------------\n", messages[1].Content)

	// Create an instance of AiOutput to capture the response
	var aiOut AiOutput

	// Create the request for the OpenAI Chat API
	res, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:     openai.GPT4o,
		Messages:  messages,
		MaxTokens: 1000,
	}, &aiOut) // Pass the aiOut variable to capture the response

	// Handle any errors from the API request
	if err != nil {
		return "", fmt.Errorf("Failed to generate answer: %v", err)
	}

	fmt.Println()
	fmt.Printf("%s\n============\n", res.Choices[0].Message.Content)

	// Check if the response is empty and return it
	if aiOut.FinalAnswer == "" {
		return "", nil
	}

	return aiOut.FinalAnswer, nil
}

// GetInstructor initializes and returns a configured InstructorOpenAI client.
// It reads the OPENAI_API_KEY from the environment file and configures the client
// with JSON mode and 3 retry attempts. Panics if the API key is not found.
func GetInstructor() *instructor.InstructorOpenAI {
	var myEnv map[string]string
	myEnv, err := godotenv.Read()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	openaiApiKey, ok := myEnv["OPENAI_API_KEY"]
	if !ok {
		panic("OPENAI_API_KEY not found in .env")
	}
	oaiClient := openai.NewClient(openaiApiKey)
	instructorClient := instructor.FromOpenAI(
		oaiClient,
		instructor.WithMode(instructor.ModeJSON),
		instructor.WithMaxRetries(3),
	)
	return instructorClient
}

// ParseQuestions uses AI to extract individual questions from a string containing multiple questions.
// It takes a raw string with one or more questions and returns a slice of parsed question strings.
func ParseQuestions(questions string) ([]string, error) {

	type AiOutput struct {
		Questions []string `json:"questions"`
	}
	var aiOutput AiOutput

	client := GetInstructor()
	ctx := context.Background()

	_, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: openai.GPT4o,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    "system",
				Content: "You are a helpful assistant which can parse questions from a string of questions. You must answer in json format with the key being 'questions' and value an array of strings.",
			},
			{
				Role:    "user",
				Content: "<questions>" + questions + "</questions>",
			},
			{
				Role:    "user",
				Content: "You are a helpful assistant which can parse questions from a string of questions inside questions tag. You must answer in json format with the key being 'questions' and value an array of strings.",
			},
		},
		MaxTokens: 1000,
	}, &aiOutput)

	if err != nil {
		return nil, fmt.Errorf("Failed to parse questions: %v", err)
	}

	return aiOutput.Questions, nil

}
