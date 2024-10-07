package chatcli

import (
	"fmt"
	"strings"

	"github.com/sashabaranov/go-openai"
	codesurgeon "github.com/wricardo/code-surgeon"
	"github.com/wricardo/code-surgeon/api"
	. "github.com/wricardo/code-surgeon/api"
)

func init() {
	RegisterMode("codeparser", NewParseMode)
}

type ParseMode struct {
	chat *ChatImpl
	form *Form
}

func NewParseMode(chat *ChatImpl) *ParseMode {
	return &ParseMode{chat: chat}
}

func (m *ParseMode) init() {
	qas := []*QuestionAnswer{
		{Question: "Enter the directory or file path to parse:", Answer: ""},
		{Question: "Select output format (signatures, names, full definition, docs):", Answer: "signatures"},
	}
	codeForm := NewForm()
	for _, qa := range qas {
		codeForm.AddQuestion(qa.Question, true, fnValidateNotEmpty, qa.Answer)
	}
	m.form = codeForm
}

func (m *ParseMode) Start() (*Message, *Command, error) {
	m.init()

	// Display a form to the user to get the directory or file path
	return &Message{Form: m.form.MakeFormMessage()}, NOOP, nil
}

func (m *ParseMode) BestShot(msg *Message) (*Message, *Command, error) {
	message, _, err := m.HandleResponse(msg)
	return message, NOOP, err
}

func (m *ParseMode) HandleIntent(msg *Message, intent Intent) (*Message, *Command, error) {
	m.init()

	m.chat.FillFormWithIntent(m.form, intent, msg)

	res, cmd, err := m.HandleResponse(msg)
	if err != nil {
		return res, cmd, err
	}
	// force exit after handling response from intent
	cmd = MODE_QUIT
	return res, cmd, err
}

func (c *ChatImpl) IsTesting() bool {
	return c.test
}

func (c *ChatImpl) FillFormWithIntent(form *Form, intent Intent, msg *Message) error {
	qas := form.Questions
	type AiOutput struct {
		Questions []*QuestionAnswer `json:"questions" jsonschema:"title=questions,description=the questions to be asked to the user."`
	}
	var aiOut AiOutput

	userMsg := `Identify which the questions are answered by the attributes. Questions:\n`
	for _, question := range qas {
		userMsg += fmt.Sprintf("\nQ:%s\nA:%s", question.Question, question.Answer)
	}
	userMsg += "\n"
	userMsg += "Attributes:" + fmt.Sprintf("\n%v", intent.ParsedIntentAttributes)
	userMsg += "\nAnswer in json format, with questions being the key to an array with an object question and answer inside. Example: {\"questions\":[{\"question\":\"What is the sky color?\",\"answer\":\"blue\"}]}.\n"
	userMsg += "UserMessage:" + msg.Text + "\n"
	userMsg += "You must only answer the questions that are not already answered.Include all given Questions in your response, even if does not have an answer. You should print only the questions that were given.Return all questions, even if no answer could be found on attributes, which should have an empty answer.\n"

	err := c.Chat(&aiOut, []openai.ChatCompletionMessage{
		{
			Role:    "user",
			Content: "Identify which the questions are answered by the attributes. Questions:\nQ:What's the color of HIVOLERI\nA:\nAttributes: map[What's the color of HIVOLERI:blue]\nAnswer in json format.",
		},
		{
			Role:    "assistant",
			Content: `{"questions":[{"question":"What's the color of HIVOLERI","answer":"blue"}]`,
		},
		{
			Role:    "user",
			Content: userMsg,
		},
	})
	if err != nil {
		return err
	}
	for _, qa := range qas {
		for _, q := range aiOut.Questions {
			if q.Question == qa.Question {
				form.Answer(qa.Question, q.Answer)
			}
		}
	}
	return nil
}

func (m *ParseMode) HandleResponse(input *Message) (*Message, *Command, error) {
	if m.form == nil {
		return &Message{}, NOOP, fmt.Errorf("form is nil")
	}
	if input.Form == nil || !m.form.IsFilled() {
		if input.Form != nil {
			for _, qa := range input.Form.Questions {
				m.form.Answer(qa.Question, qa.Answer)
			}
		}
		if !m.form.IsFilled() {
			return &Message{Form: m.form.MakeFormMessage()}, NOOP, nil
		}

	}
	m.chat.AddMessageYou(&Message{Form: m.form.MakeFormMessage()})

	fileOrDirectory := m.form.Questions[0].Answer
	outputFormat := m.form.Questions[1].Answer

	if fileOrDirectory == "" {
		m.form.ClearAnswers()
		return &Message{
			Text: "Please provide a directory or file path to parse.",
		}, MODE_QUIT, nil
	}

	if m.chat.IsTesting() {
		return TextMessage("TEST_MODE"), NOOP, nil
	}

	parsedInfo, err := codesurgeon.ParseDirectory(fileOrDirectory)
	if err != nil {
		return &Message{Text: fmt.Sprintf("Error parsing: %v", err)}, MODE_QUIT, nil
	} else if parsedInfo == nil {
		return &Message{Text: "parsedInfo=nil"}, MODE_QUIT, nil
	}

	output := ""
	switch outputFormat {
	case "signatures":
		output = formatOnlySignatures(*parsedInfo)
	case "names":
		output = formatOnlyNames(*parsedInfo)
	case "full definition":
		output = formatFullDefinition(*parsedInfo)
	case "docs":
		output = formatDocs(*parsedInfo)
	default:
		output = formatOnlySignatures(*parsedInfo)
	}

	// Convert parsedInfo to a string or JSON to display to the user
	return &api.Message{Text: output}, MODE_QUIT, nil
}

func (m *ParseMode) Stop() error {
	return nil
}

func formatOnlySignatures(parsedInfo codesurgeon.ParsedInfo) string {
	var result []string

	for _, pkg := range parsedInfo.Packages {
		result = append(result, fmt.Sprintf("Package: %s", pkg.Package))

		for _, strct := range pkg.Structs {
			result = append(result, fmt.Sprintf("Struct: %s", strct.Name))
			for _, method := range strct.Methods {
				result = append(result, fmt.Sprintf("  Method: %s", method.Signature))
			}
		}

		for _, iface := range pkg.Interfaces {
			result = append(result, fmt.Sprintf("Interface: %s", iface.Name))
			for _, method := range iface.Methods {
				result = append(result, fmt.Sprintf("  Method: %s", method.Signature))
			}
		}

		for _, function := range pkg.Functions {
			result = append(result, fmt.Sprintf("Function: %s", function.Signature))
		}

		for _, variable := range pkg.Variables {
			result = append(result, fmt.Sprintf("Variable: %s %s", variable.Name, variable.Type))
		}

		for _, constant := range pkg.Constants {
			result = append(result, fmt.Sprintf("Constant: %s = %s", constant.Name, constant.Value))
		}
	}

	return strings.Join(result, "\n")
}

func formatOnlyNames(parsedInfo codesurgeon.ParsedInfo) string {
	var result []string

	for _, pkg := range parsedInfo.Packages {
		result = append(result, fmt.Sprintf("Package: %s", pkg.Package))

		for _, strct := range pkg.Structs {
			result = append(result, fmt.Sprintf("Struct: %s", strct.Name))
			for _, method := range strct.Methods {
				result = append(result, fmt.Sprintf("  Method: %s", method.Name))
			}
		}

		for _, iface := range pkg.Interfaces {
			result = append(result, fmt.Sprintf("Interface: %s", iface.Name))
			for _, method := range iface.Methods {
				result = append(result, fmt.Sprintf("  Method: %s", method.Name))
			}
		}

		for _, function := range pkg.Functions {
			result = append(result, fmt.Sprintf("Function: %s", function.Name))
		}

		for _, variable := range pkg.Variables {
			result = append(result, fmt.Sprintf("Variable: %s", variable.Name))
		}

		for _, constant := range pkg.Constants {
			result = append(result, fmt.Sprintf("Constant: %s", constant.Name))
		}
	}

	return strings.Join(result, "\n")
}

func formatFullDefinition(parsedInfo codesurgeon.ParsedInfo) string {
	var result []string

	for _, pkg := range parsedInfo.Packages {
		result = append(result, fmt.Sprintf("Package: %s", pkg.Package))

		for _, strct := range pkg.Structs {
			result = append(result, fmt.Sprintf("Struct: %s", strct.Name))
			for _, field := range strct.Fields {
				result = append(result, fmt.Sprintf("  Field: %s %s", field.Name, field.Type))
			}
			for _, method := range strct.Methods {
				result = append(result, fmt.Sprintf("  Method: %s", method.Signature))
				result = append(result, fmt.Sprintf("    Body: %s", method.Body))
			}
		}

		for _, iface := range pkg.Interfaces {
			result = append(result, fmt.Sprintf("Interface: %s", iface.Name))
			for _, method := range iface.Methods {
				result = append(result, fmt.Sprintf("  Method: %s", method.Signature))
			}
		}

		for _, function := range pkg.Functions {
			result = append(result, fmt.Sprintf("Function: %s", function.Signature))
			result = append(result, fmt.Sprintf("  Body: %s", function.Body))
		}

		for _, variable := range pkg.Variables {
			result = append(result, fmt.Sprintf("Variable: %s %s", variable.Name, variable.Type))
		}

		for _, constant := range pkg.Constants {
			result = append(result, fmt.Sprintf("Constant: %s = %s", constant.Name, constant.Value))
		}
	}

	return strings.Join(result, "\n")
}

func formatDocs(parsedInfo codesurgeon.ParsedInfo) string {
	var result []string

	for _, pkg := range parsedInfo.Packages {
		result = append(result, fmt.Sprintf("Package: %s", pkg.Package))

		for _, strct := range pkg.Structs {
			result = append(result, fmt.Sprintf("Struct: %s", strct.Name))
			for _, doc := range strct.Docs {
				result = append(result, fmt.Sprintf("  Doc: %s", doc))
			}
			for _, method := range strct.Methods {
				result = append(result, fmt.Sprintf("  Method: %s", method.Signature))
				for _, doc := range method.Docs {
					result = append(result, fmt.Sprintf("    Doc: %s", doc))
				}
			}
		}

		for _, iface := range pkg.Interfaces {
			result = append(result, fmt.Sprintf("Interface: %s", iface.Name))
			for _, doc := range iface.Docs {
				result = append(result, fmt.Sprintf("  Doc: %s", doc))
			}
			for _, method := range iface.Methods {
				result = append(result, fmt.Sprintf("  Method: %s", method.Signature))
				for _, doc := range method.Docs {
					result = append(result, fmt.Sprintf("    Doc: %s", doc))
				}
			}
		}

		for _, function := range pkg.Functions {
			result = append(result, fmt.Sprintf("Function: %s", function.Signature))
			for _, doc := range function.Docs {
				result = append(result, fmt.Sprintf("  Doc: %s", doc))
			}
		}
	}

	return strings.Join(result, "\n")
}
