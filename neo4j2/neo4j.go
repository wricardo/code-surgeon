package neo4j2

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/google/uuid"
	"github.com/instructor-ai/instructor-go/pkg/instructor"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/sashabaranov/go-openai"
)

func CreateQuestionAndAnswers(ctx context.Context, driver neo4j.DriverWithContext, questionText string, questionEmbedding []float32, answers []string) error { // CreateQuestionAndAnswers creates a question node and its corresponding answer nodes in Neo4j.

	session := driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(ctx)
	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		questionResult, err := tx.Run(ctx, "CREATE (q:Question {id: $id, text: $text, embedding: $embedding}) RETURN q.id", map[string]interface{}{
			"text":      questionText,
			"id":        uuid.New().String(),
			"embedding": questionEmbedding,
		})
		if err != nil {
			return nil, err
		}
		var questionId string
		if questionResult.Next(ctx) {
			questionId = questionResult.Record().Values[0].(string)
		}
		for _, answerText := range answers {
			answerResult, err := tx.Run(ctx, "CREATE (a:Answer {id: $id, text: $text}) RETURN a.id", map[string]interface{}{"text": answerText, "id": uuid.New().String()})
			if err != nil {
				return nil, err
			}
			var answerId string
			if answerResult.Next(ctx) {
				answerId = answerResult.Record().Values[0].(string)
			}
			_, err = tx.Run(ctx, "MATCH (q:Question {id: $questionId}), (a:Answer {id: $answerId}) CREATE (q)-[:ANSWERED_BY]->(a)", map[string]interface{}{"questionId": questionId, "answerId": answerId})
			if err != nil {
				return nil, err
			}
		}
		return nil, nil
	})
	return err
}

func GetTopAnswersForQuestions(ctx context.Context, driver neo4j.DriverWithContext, questionIds []string) ( // GetTopAnswersForQuestions retrieves the top answers for a list of question IDs from Neo4j.
	[]string, error) {
	session := driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(ctx)
	var allAnswers []string
	_, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		for _, questionId := range questionIds {
			result, err := tx.Run(ctx, "MATCH (q:Question {id: $questionId})-[:ANSWERED_BY]->(a:Answer) "+"RETURN a.text ORDER BY a.score DESC LIMIT 2", map[string]interface{}{"questionId": questionId})
			if err != nil {
				return nil, err
			}
			for result.Next(ctx) {
				allAnswers = append(allAnswers, result.Record().Values[0].(string))
			}
		}
		return nil, nil
	})
	if err != nil {
		return nil, err
	}
	return allAnswers, nil
}

func GenerateFinalAnswer(client *instructor.InstructorOpenAI, question string, answers []string) (string, error) {
	// Create a struct to capture the AI's response
	type AiOutput struct {
		Content string `json:"content"`
	}

	// Context for the OpenAI API call
	ctx := context.Background()

	// Construct the messages for the chat completion
	messages := []openai.ChatCompletionMessage{
		{
			Role:    "system",
			Content: "You are a helpful assistant.",
		},
		{
			Role:    "user",
			Content: fmt.Sprintf("Based on the following answers, formulate a comprehensive response to the question:\n\n%s\n\nQuestion: %s", strings.Join(answers, "\n\n"), question),
		},
	}

	// Create an instance of AiOutput to capture the response
	var aiOut AiOutput

	// Create the request for the OpenAI Chat API
	_, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:     openai.GPT4o,
		Messages:  messages,
		MaxTokens: 450,
	}, &aiOut) // Pass the aiOut variable to capture the response

	// Handle any errors from the API request
	if err != nil {
		return "", fmt.Errorf("Failed to generate answer: %v", err)
	}

	// Check if the response is empty and return it
	if aiOut.Content == "" {
		return "", nil
	}

	return aiOut.Content, nil
}

func VectorSearchQuestions(ctx context.Context, driver neo4j.DriverWithContext, userEmbedding []float32) ( // VectorSearchQuestions performs a vector search in Neo4j to find the top similar questions to the user's input.
	[]string, error) {
	session := driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(ctx)
	var questionIds []string
	_, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		result, err := tx.Run(ctx, "MATCH (q:Question) "+"RETURN q.id, gds.similarity.cosine(q.embedding, $userEmbedding) AS similarity "+"ORDER BY similarity DESC LIMIT 3", map[string]interface{}{"userEmbedding": userEmbedding})
		if err != nil {
			return nil, err
		}
		for result.Next(ctx) {
			questionIds = append(questionIds, result.Record().Values[0].(string))
		}
		return nil, nil
	})
	if err != nil {
		return nil, err
	}
	return questionIds, nil
}

func SaveConversationSummary(ctx context.Context, driver neo4j.DriverWithContext, conversationSummary, dateISO string) error {
	session := driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface {
	}, error) {
		query := `
			CREATE (cs:ConversationSummary {
				summary: $summary,
				date: $date
			})
		`
		_, err := tx.Run(ctx, query, map[string]interface{}{"summary": conversationSummary,

			"date": dateISO,
		})
		if err != nil {
			return nil, err
		}
		return nil, nil
	})
	if err != nil {
		log.
			Printf("Error saving conversation summary to database: %v",
				err)
		return err
	}
	return nil
}

// SaveConversationSummary saves a conversation summary and date to the Neo4j database.

// Open a new session to interact with the database

// Execute a write transaction to save the conversation summary
