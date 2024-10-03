package main

import (
	"context"
	"net/http"

	"connectrpc.com/connect"
	"github.com/rs/zerolog/log"
	"github.com/sashabaranov/go-openai"
	"github.com/wricardo/code-surgeon/ai"
	"github.com/wricardo/code-surgeon/api"
	"github.com/wricardo/code-surgeon/api/apiconnect"
	"github.com/wricardo/code-surgeon/neo4j2"
)

type ResponseWithQA struct {
    Response        string           `json:"response"`
    QuestionAnswers []QuestionAnswer `json:"question_answers"`
}

var TEACHER TMode = "teacher"

func init() {
	RegisterMode(TEACHER, NewTeacherMode)
}

type TeacherMode struct {
	chat        *Chat
	alreadySeen map[string]bool
}

func NewTeacherMode(chat *Chat) *TeacherMode {
	return &TeacherMode{
		chat:        chat,
		alreadySeen: make(map[string]bool),
	}
}

func (m *TeacherMode) HandleIntent(msg Message) (Message, Command, error) {
	return m.HandleResponse(msg)
}

func (ats *TeacherMode) Start() (Message, Command, error) {
	return TextMessage("Teach me by discussing topics. I'll keep track of the information and construct a Q&A for further reference."), MODE_START, nil
}

type QuestionAnswer struct {
	Question string `json:"question" jsonschema:"title=question,description=the question."`
	Answer   string `json:"answer" jsonschema:"title=answer,description=the answer."`
}

func (ats *TeacherMode) HandleResponse(msg Message) (Message, Command, error) {
	userMessage := msg.Text

	type AiOutput struct {
		Response        string           `json:"response" jsonschema:"title=response,description=the assistant's response to the user."`
		QuestionAnswers []QuestionAnswer `json:"q_and_as" jsonschema:"title=q_and_as,description=Question and answer, flashcard style."`
	}

	client := apiconnect.NewGptServiceClient(http.DefaultClient, "http://localhost:8010")
	ctx := context.Background()

	var aiOut AiOutput
	err := ats.chat.Chat(&aiOut, []openai.ChatCompletionMessage{
		{
			Role:    "system",
			Content: "You are a helpful assistant which can respond to user's input in the most helpful form. Any information that you find it useful to remember for future conversations, write them in the json on the q_and_as key. The json should contain a list of objects with the keys question and answer. The question is the question that goes in the flashcard about important information in this conversation. Above all you should be helpful and informative, writing your response to the user under the response key.",
		},

		{
			Role:    "user",
			Content: userMessage,
		},
	})
	if err != nil {
		return Message{}, NOOP, err
	}
	for _, qa := range aiOut.QuestionAnswers {
		if ats.alreadySeen[qa.Question] {
			continue
		}
		ats.alreadySeen[qa.Question] = true
		qas := make([]*api.Answer, 0, len(aiOut.QuestionAnswers))
		for _, qa := range aiOut.QuestionAnswers {
			qas = append(qas, &api.Answer{
				Question: qa.Question,
				Answer:   qa.Answer,
			})
		}
		// save to knowledge base
		client.SaveQuestionAndAnswer(ctx, &connect.Request[api.SaveQuestionAndAnswerRequest]{
			Msg: &api.SaveQuestionAndAnswerRequest{
				QuestionAndAnswer: qas,
			},
		})
	}

	responseWithQA := ResponseWithQA{
		Response:        aiOut.Response,
		QuestionAnswers: aiOut.QuestionAnswers,
	}

	return Message{Text: responseWithQA}, NOOP, nil
}

func (ats *TeacherMode) Stop() error {
	return nil
}

func (t *TeacherMode) SaveQuestionAndAnswer(ctx context.Context, qaPairs []QuestionAnswer) error {
	for _, qa := range qaPairs {
		question := qa.Question
		answer := qa.Answer

		embedding, err := ai.EmbedQuestion(t.chat.instructor.Client, question)
		if err != nil {
			log.Printf("Error embedding question: %v", err)
			continue
		}

		err = neo4j2.CreateQuestionAndAnswers(ctx, *t.chat.driver, question, embedding, []string{answer})
		if err != nil {
			log.Printf("Error saving question and answer to database: %v", err)
			return err
		}
	}

	return nil
}
