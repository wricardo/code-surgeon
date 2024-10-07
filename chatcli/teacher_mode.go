package chatcli

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/sashabaranov/go-openai"
	"github.com/wricardo/code-surgeon/ai"
	. "github.com/wricardo/code-surgeon/api"
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
	chat        *ChatImpl
	alreadySeen map[string]bool
}

func NewTeacherMode(chat *ChatImpl) *TeacherMode {
	return &TeacherMode{
		chat:        chat,
		alreadySeen: make(map[string]bool),
	}
}

func (m *TeacherMode) BestShot(msg *Message) (*Message, *Command, error) {
	message, _, err := m.HandleResponse(msg)
	return message, NOOP, err
}

func (ats *TeacherMode) HandleIntent(msg *Message, intent Intent) (*Message, *Command, error) {
	return ats.HandleResponse(msg)
}

func (ats *TeacherMode) Start() (*Message, *Command, error) {
	return TextMessage("Teach me by discussing topics. I'll keep track of the information and construct a Q&A for further reference."), MODE_START, nil
}

type QuestionAnswerLocal struct {
	Question string `json:"question" jsonschema:"title=question,description=the question."`
	Answer   string `json:"answer" jsonschema:"title=answer,description=the answer."`
}

func (ats *TeacherMode) HandleResponse(msg *Message) (*Message, *Command, error) {
	userMessage := msg.Text
	type ResponseWithQA struct {
		Response        string                `json:"response"`
		QuestionAnswers []QuestionAnswerLocal `json:"question_answers"`
	}

	type AiOutput struct {
		Response        string                `json:"response" jsonschema:"title=response,description=the assistant's response to the user."`
		QuestionAnswers []QuestionAnswerLocal `json:"q_and_as" jsonschema:"title=q_and_as,description=Question and answer, flashcard style."`
	}

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
		return &Message{}, NOOP, err
	}
	for _, qa := range aiOut.QuestionAnswers {
		if ats.alreadySeen[qa.Question] {
			continue
		}
		ats.alreadySeen[qa.Question] = true
		// save to knowledge base
		err := ats.SaveQuestionAndAnswer(context.Background(), []*QuestionAnswer{
			{
				Question: qa.Question,
				Answer:   qa.Answer,
			},
		})
		if err != nil {
			return &Message{}, NOOP, err
		}
	}

	response := &Message{Text: ""}
	if len(aiOut.QuestionAnswers) > 0 {
		response.Text += "Knowledge saved:\n"
		for _, qa := range aiOut.QuestionAnswers {
			response.Text += qa.Question + "\n" + qa.Answer + "\n"
		}
		response.Text += "I've stored these in my question answer database.\n\n"
	}
	response.Text += aiOut.Response

	return response, NOOP, nil
}

func (ats *TeacherMode) Stop() error {
	return nil
}

func (t *TeacherMode) SaveQuestionAndAnswer(ctx context.Context, qaPairs []*QuestionAnswer) error {
	for _, qa := range qaPairs {
		question := qa.Question
		answer := qa.Answer

		embedding, err := ai.EmbedQuestion(t.chat.instructor.Client, question)
		if err != nil {
			log.Printf("Error embedding question: %v", err)
			continue
		}

		err = neo4j2.CreateQuestionAndAnswers(ctx, *t.chat.driver, question, embedding, []string{answer})
		log.Info().Str("question", question).Str("answer", answer).Msg("Q&A saved to db")
		if err != nil {
			log.Printf("Error saving question and answer to database: %v", err)
			return err
		}
	}

	return nil
}
