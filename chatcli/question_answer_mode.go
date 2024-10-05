package chatcli

import (
	"context"
	"fmt"

	"github.com/sashabaranov/go-openai"
	"github.com/wricardo/code-surgeon/ai"
	. "github.com/wricardo/code-surgeon/api"
	"github.com/wricardo/code-surgeon/neo4j2"
)

var QUESTION_ANSWER TMode = "question_answer"

func init() {
	RegisterMode(QUESTION_ANSWER, NewQuestionAnswerMode)
}

type QuestionAnswerMode struct {
	chat *ChatImpl
}

func NewQuestionAnswerMode(chat *ChatImpl) *QuestionAnswerMode {
	return &QuestionAnswerMode{
		chat: chat,
	}
}

func (ats *QuestionAnswerMode) Start() (*Message, *Command, error) {
	return TextMessage("Ask away!"), MODE_START, nil
}

func (m *QuestionAnswerMode) BestShot(msg *Message) (*Message, *Command, error) {
	message, _, err := m.HandleResponse(msg)
	return message, NOOP, err
}

func (m *QuestionAnswerMode) HandleIntent(msg *Message, intent Intent) (*Message, *Command, error) {
	msg, _, err := m.HandleResponse(msg)
	// if comming from intent, we quit the mode after answering
	return msg, MODE_QUIT, err
}

func (m *QuestionAnswerMode) HandleResponse(msg *Message) (*Message, *Command, error) {
	userMessage := msg.Text

	userEmbedding, err := ai.EmbedQuestion(m.chat.instructor.Client, msg.Text)
	if err != nil {
		return &Message{}, NOOP, err
	}
	if len(userEmbedding) == 0 {
		return &Message{}, NOOP, fmt.Errorf("embedding is empty") // TODO: should return an error message instead of an error
	}

	// search for similar questions in neo4j
	similarQuestions, err := neo4j2.VectorSearchQuestions(context.Background(), *m.chat.driver, userEmbedding, 3)
	if err != nil {
		return &Message{}, NOOP, err
	}

	// Fetch top answers for similar questions
	topQuestionIds := make([]string, len(similarQuestions))
	for i, question := range similarQuestions {
		topQuestionIds[i] = question.ID
	}

	topAnswers, err := neo4j2.GetTopAnswersForQuestions(context.Background(), *m.chat.driver, topQuestionIds)
	if err != nil {
		return &Message{}, NOOP, err
	}

	// Generate final answer using AI
	type FinalAnswerOutput struct {
		FinalAnswer string `json:"final_answer" jsonschema:"title=final_answer,description=the final answer to the user's question."`
	}

	var finalAnswerOut FinalAnswerOutput
	err = m.chat.Chat(&finalAnswerOut, []openai.ChatCompletionMessage{
		{
			Role:    "system",
			Content: "You are a helpful assistant that can generate a final answer based on similar questions and answers.",
		},
		{
			Role:    "user",
			Content: fmt.Sprintf("User question: %s\nSimilar questions and answers: %v", userMessage, topAnswers),
		},
	})
	if err != nil {
		return &Message{}, NOOP, err
	}

	// Construct response text
	responseText := "Similar Questions:\n"
	for _, ta := range topAnswers {
		responseText += fmt.Sprintf("Q:%s\nA:%s\n", ta.Question, ta.Answer)
	}
	responseText += "\nFinal Answer:\n" + finalAnswerOut.FinalAnswer

	return TextMessage(responseText), NOOP, nil
}

func (ats *QuestionAnswerMode) Stop() error {
	return nil
}
