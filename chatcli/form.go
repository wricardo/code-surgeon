package chatcli

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/sashabaranov/go-openai"
	"github.com/wricardo/code-surgeon/api"
)

type Form struct {
	Questions []FormQuestion
}

type FormQuestion struct {
	Question   string
	Answer     string
	Required   bool
	Validation func(string) bool
}

func NewForm() *Form {
	return &Form{
		Questions: []FormQuestion{},
	}
}

func (pf *Form) ClearAnswers() {
	for i := range pf.Questions {
		pf.Questions[i].Answer = ""
	}
}

func (pf *Form) AddQuestion(question string, required bool, validation func(string) bool, defaultValue string) StringPromise {
	q := FormQuestion{
		Question:   question,
		Required:   required,
		Validation: validation,
		Answer:     defaultValue,
	}
	pf.Questions = append(pf.Questions, q)
	return func() string {
		for _, q := range pf.Questions {
			if q.Question == question {
				return q.Answer
			}
		}
		return ""
	}
}

func (pf *Form) IsQuestionFilled(q FormQuestion) bool {
	if q.Required && q.Answer == "" {
		return false
	} else if q.Answer != "" {
		if q.Validation != nil && !q.Validation(q.Answer) {
			return false
		}
	}
	return true
}

func (pf *Form) IsFilled() bool {
	if pf == nil || len(pf.Questions) == 0 {
		return true
	}
	for _, q := range pf.Questions {
		if !pf.IsQuestionFilled(q) {
			return false
		}
	}
	return true
}

func (pf *Form) Answer(question, answer string) {
	if answer == "" {
		return
	}
	log.Debug().Str("question", question).Str("answer", answer).Msg("Form.Answer")
	if pf == nil || len(pf.Questions) == 0 {
		log.Warn().Msg("Form is nil or empty here")
		return
	}
	for i, q := range pf.Questions {
		if q.Question == question {
			pf.Questions[i].Answer = answer
			log.Warn().Str("question", question).Str("answer", answer).Msg("Form.Answer answered for real")
			return
		}
	}
	log.Warn().Str("question", question).Str("answer", answer).Msg("Form.Answer question NOT FOUND")
}

func TryFillFormFromTextMessage(msg *api.Message, form *Form, chat LLMChat) *Form {
	if form == nil {
		return form
	}
	if msg == nil {
		return form
	}
	if msg.Form != nil && len(msg.Form.Questions) > 0 {
		for _, q := range msg.Form.Questions {
			form.Answer(q.Question, q.Answer)
		}
		return form
	} else if msg.Text != "" {
		// Define AiOutput
		type AiOutput struct {
			Answers map[string]string `json:"answers"`
		}
		var aiOutput AiOutput

		// Build the prompt
		// Include the questions in the form
		var questionsList string
		for _, q := range form.Questions {
			questionsList += fmt.Sprintf("- %s\n", q.Question)
		}

		prompt := fmt.Sprintf(`
Given the user message: "%s"
and the following form questions:
%s
Please provide answers to the questions based on the user message.
Output the answers in JSON format like:
{
    "answers": {
        "Question1": "Answer1",
        "Question2": "Answer2",
        ...
    }
}
important: If an answer cannot be determined, you can leave it empty.
`, msg.Text, questionsList)

		err := chat.Chat(&aiOutput, []openai.ChatCompletionMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		})
		if err != nil {
			log.Warn().Err(err).Msg("Failed to generate AI response on TryFillFormFromTextMessage")
			return form
		}

		// Fill the form from aiOutput
		for question, answer := range aiOutput.Answers {
			form.Answer(question, answer)
		}
	}
	return form
}

func (pf *Form) MakeFormMessage() *api.FormMessage {
	if pf == nil || len(pf.Questions) == 0 {
		log.Warn().Msg("Form is nil or empty here222")
		return &api.FormMessage{
			Questions: []*api.QuestionAnswer{},
		}
	}
	form_ := &api.FormMessage{
		Questions: []*api.QuestionAnswer{},
	}
	for _, q := range pf.Questions {
		form_.Questions = append(form_.Questions, &api.QuestionAnswer{
			Question: q.Question,
			Answer:   q.Answer,
		})
	}
	return form_

}

type StringPromise func() string
