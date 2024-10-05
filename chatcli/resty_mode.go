package chatcli

import (
	"fmt"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/sashabaranov/go-openai"
	. "github.com/wricardo/code-surgeon/api"
)

type RestyMode struct {
	client *resty.Client
	chat   *ChatImpl
	form   *Form
	ask    StringPromise
}

func NewRestyMode(chat *ChatImpl) *RestyMode {
	restyForm := NewForm()

	return &RestyMode{
		ask:    restyForm.AddQuestion("What HTTP request would you like to make?", true, fnValidateNotEmpty, ""),
		form:   restyForm,
		client: resty.New(),
		chat:   chat,
	}
}
func init() {
	RegisterMode("resty", NewRestyMode)
}

func (rm *RestyMode) Start() (*Message, *Command, error) {
	rm.client = resty.New()
	return &Message{Form: rm.form.MakeFormMessage()}, MODE_START, nil
}

func (rm *RestyMode) BestShot(msg *Message) (*Message, *Command, error) {
	message, _, err := rm.HandleResponse(msg)
	return message, NOOP, err
}

func (rm *RestyMode) HandleIntent(msg *Message, intent Intent) (*Message, *Command, error) {
	return rm.HandleResponse(msg)
}

func (rm *RestyMode) HandleResponse(msg *Message) (*Message, *Command, error) {
	if msg.Form != nil || !rm.form.IsFilled() {
		if msg.Form != nil {
			for _, qa := range msg.Form.Questions {
				rm.form.Answer(qa.Question, qa.Answer)
			}
		}
		if !rm.form.IsFilled() {
			return &Message{Form: rm.form.MakeFormMessage()}, NOOP, nil
		}
	}

	// Process the user's request through chat.Chat(...)
	type AiOutput struct {
		Method  string            `json:"method" jsonschema:"title=method,description=the HTTP method to use"`
		URL     string            `json:"url" jsonschema:"title=url,description=the URL to request"`
		Headers map[string]string `json:"headers" jsonschema:"title=headers,description=the headers to include in the request"`
		Body    string            `json:"body" jsonschema:"title=body,description=the body to include in the request, usually json"`
	}

	var aiOut AiOutput
	err := rm.chat.Chat(&aiOut, []openai.ChatCompletionMessage{
		{
			Role:    "system",
			Content: "Determine the HTTP method, URL, headers, json body post from the user's request.",
		},
		{
			Role:    "user",
			Content: rm.ask(),
		},
	})
	if err != nil {
		return TextMessage(fmt.Sprintf("Error processing request: %v", err)), NOOP, err
	}

	// Prepare the HTTP request using Resty
	request := rm.client.R()
	request = request.SetDebug(true)

	// Set headers if provided
	if len(aiOut.Headers) > 0 {
		request.SetHeaders(aiOut.Headers)
	}

	// Set body if provided and method supports a body
	methodsWithBody := map[string]bool{
		"POST":   true,
		"PUT":    true,
		"PATCH":  true,
		"DELETE": true, // DELETE can have a body, though it's not common
	}
	method := strings.ToUpper(aiOut.Method)
	if aiOut.Body != "" && methodsWithBody[method] {
		request.SetBody(aiOut.Body)
	}

	// Execute the HTTP request using the specified method and URL
	resp, err := request.Execute(method, aiOut.URL)
	if err != nil {
		return TextMessage(fmt.Sprintf("Error making request: %v", err)), NOOP, err
	}

	return TextMessage(fmt.Sprintf("Response: %s", resp.String())), NOOP, nil
}

func (rm *RestyMode) Stop() error {
	// Clean up resources if necessary
	return nil
}
