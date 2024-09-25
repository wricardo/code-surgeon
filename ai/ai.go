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
	"github.com/sashabaranov/go-openai"
	codesurgeon "github.com/wricardo/code-surgeon"
	"github.com/wricardo/code-surgeon/neo4j2"
)

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

You are a helpful and experienced golang developer that can follow the instructions and produce the desired output. 
You should use the actions defined to call functions that find information about type definitions, specially functions and methods on the project we are working on.
Call the API with the operation you want see that the user want you to execute, if any.
You may sometimes execute bash commands to fulfill the user request.
`, m)
}

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

type GenerateDocumentationRequest struct {
	Path              string
	OverwriteExisting bool
	ReceiverName      string
	FunctionName      string
}

func GenerateDocumentation(instructorClient *instructor.InstructorOpenAI, req GenerateDocumentationRequest) (bool, error) {
	receiverName := strings.Replace(req.ReceiverName, "*", "", -1)

	parsedInfo, err := codesurgeon.ParseDirectory(req.Path)
	if err != nil {
		return false, err
	}
	hasModifiedAny := false
	for _, pkg := range parsedInfo.Packages {
		for _, fn := range pkg.Functions {
			// if we're trying to add to a specific method, skip all functions
			if receiverName != "" {
				continue
			}
			if req.FunctionName != "" && fn.Name != req.FunctionName {
				continue
			}
			if len(fn.Docs) > 0 && !req.OverwriteExisting {
				fmt.Printf("Function %s already has documentation\n", fn.Name)
				continue
			}
			filename, err := codesurgeon.FindFunction(req.Path, "", fn.Name)
			if err != nil {
				return false, err
			}
			if filename == "" {
				fmt.Printf("Function %s not found in any file\n", fn.Name)
				continue
			}
			documentation, err := documentFunction(instructorClient, &fn, nil)
			if err != nil {
				fmt.Println(err)
				continue
			}
			if documentation == "" {
				fmt.Println("No documentation generated by AI")
				continue
			}

			modified, err := codesurgeon.UpsertDocumentationToFunction(filename, "", fn.Name, documentation)
			if err != nil {
				return false, err
			}
			if modified {
				hasModifiedAny = true
			}
		}

		for _, st := range pkg.Structs {
			if receiverName == "" {
				continue
			} else if st.Name != receiverName {
				continue
			}
			for _, fn := range st.Methods {
				if req.FunctionName != "" && fn.Name != req.FunctionName {
					continue
				}
				if len(fn.Docs) > 0 {
					continue
				}
				filename, err := codesurgeon.FindFunction(req.Path, st.Name, fn.Name)
				if err != nil {
					return false, err
				}
				if filename == "" {
					fmt.Printf("Function %s not found in any file\n", fn.Name)
					continue
				}
				documentation, err := documentFunction(instructorClient, nil, &fn)
				if err != nil {
					fmt.Println(err)
					continue
				}
				if documentation == "" {
					fmt.Println("No documentation generated by AI")
					continue
				}

				modified, err := codesurgeon.UpsertDocumentationToFunction(filename, st.Name, fn.Name, documentation)
				if err != nil {
					return false, err
				}
				if modified {
					hasModifiedAny = true
				}
			}

		}
	}
	return hasModifiedAny, nil
}

func documentFunction(client *instructor.InstructorOpenAI, sf *codesurgeon.Function, meth *codesurgeon.Method) (string, error) {
	type AiOutput struct {
		Documentation string `json:"documentation" jsonschema:"title=documentation,description=the few lines max of documentation that goes above a golang function. Each line starts with //."`
	}
	// client := instructor.FromOpenAI(
	// 	openai.NewClient(os.Getenv("OPENAI_API_KEY")),
	// 	instructor.WithMode(instructor.ModeJSON),
	// 	instructor.WithMaxRetries(3),
	// )

	ctx := context.Background()
	var aiOut AiOutput
	if sf != nil {
		_, err := client.CreateChatCompletion(
			ctx,
			openai.ChatCompletionRequest{
				Model: openai.GPT4o,
				Messages: []openai.ChatCompletionMessage{
					{
						Role: openai.ChatMessageRoleUser,
						Content: Render(`
write the documentation for this golang function in json format with "documentation" being the object key . Be thorough and detailed, but concise since this will go in the function documentation above the signature. Just output the string with the comment.
{{.sf}}
`, map[string]interface{}{"sf": fmt.Sprintf("%v", sf)}),
					},
				},
			},
			&aiOut,
		)
		if err != nil {
			return "", fmt.Errorf("Failed to generate documentation: %v", err)
		}
	} else if meth != nil {
		_, err := client.CreateChatCompletion(
			ctx,
			openai.ChatCompletionRequest{
				Model: openai.GPT4o,
				Messages: []openai.ChatCompletionMessage{
					{
						Role: openai.ChatMessageRoleUser,
						Content: Render(`
						write the documentation for this golang method in json format with "documentation" being the object key . Be thorough and detailed, but concise since this will go in the function documentation above the signature. Just output the string with the comment.err != nil {
							{{.meth}}
						}
						`, map[string]interface{}{"meth": fmt.Sprintf("%v", meth)}),
					},
				},
			},
			&aiOut,
		)

		if err != nil {
			return "", fmt.Errorf("Failed to generate documentation: %v", err)
		}
	}

	if aiOut.Documentation == "" {
		return "", nil
	}

	if !strings.HasPrefix(aiOut.Documentation, "//") {
		aiOut.Documentation = "// " + aiOut.Documentation
	}

	return aiOut.Documentation, nil
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

// EmbedQuestion embeds a user's question into a vector representation using a predefined embedding model.
func EmbedQuestion(client *openai.Client, question string) ([]float32, error) {
	resp, err := client.CreateEmbeddings(context.Background(), openai.
		EmbeddingRequest{Input: []string{question}, Model: openai.AdaEmbeddingV2})
	if err != nil {
		return nil, err
	}
	if len(resp.Data) == 0 {
		return nil, errors.New("no embedding found in response")
	}
	return resp.Data[0].Embedding, nil
}

func GenerateFinalAnswer(client *instructor.InstructorOpenAI, question string, questionsAnswers []neo4j2.QuestionAnswer) (string, error) {
	// Create a struct to capture the AI's response
	type AiOutput struct {
		FinalAnswer string `json:"final_answer" jsonschema:"title=final_answer,description=the final answer to the user's question.`
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
