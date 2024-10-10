package chatcli

import (
	"fmt"

	"github.com/sashabaranov/go-openai"
	"github.com/wricardo/code-surgeon/api"
)

var TRANSLATE_TO_SPANISH TMode = "translate_to_spanish"

func init() {
	RegisterMode(TRANSLATE_TO_SPANISH, NewTranslateToSpanishMode)
}

// NewTranslateToSpanishMode creates a new instance of the TranslateToSpanishMode
func NewTranslateToSpanishMode(chat *ChatImpl) *TranslateToSpanishMode {
	return &TranslateToSpanishMode{
		chat: chat,
	}
}

// TranslateToSpanishMode struct implements the Mode interface
type TranslateToSpanishMode struct {
	chat *ChatImpl
}

// Start initializes the TranslateToSpanishMode with a welcoming message
func (tsm *TranslateToSpanishMode) Start() (*api.Message, *api.Command, error) {
	return &api.Message{
		Text: "¡Bienvenido al Modo de Traducción al Español! Traduciré todo lo que digas al español.",
	}, MODE_START, nil
}

func (tsm *TranslateToSpanishMode) BestShot(msg *api.Message) (*api.Message, *api.Command, error) {
	message, _, err := tsm.HandleResponse(msg)
	if err != nil {
		return &api.Message{}, NOOP, err
	}
	message.Text = "Traducción a través de BestShot: " + message.Text
	return message, NOOP, nil
}

func (tsm *TranslateToSpanishMode) HandleIntent(userMessage *api.Message, intent Intent) (*api.Message, *api.Command, error) {
	return tsm.HandleResponse(userMessage)
}

// HandleResponse translates the user's message to Spanish using the chat LLM
func (tsm *TranslateToSpanishMode) HandleResponse(userMessage *api.Message) (*api.Message, *api.Command, error) {
	// Define the output structure for the translation
	type TranslationOutput struct {
		Translation string `json:"translation" jsonschema:"title=translation,description=The translation of the user's message to Spanish."`
	}

	var translationOut TranslationOutput

	// Use the chat LLM to perform the translation
	err := tsm.chat.Chat(&translationOut, []openai.ChatCompletionMessage{
		{
			Role:    "system",
			Content: "You are a helpful assistant that translates English text to Spanish.",
		},
		{
			Role:    "user",
			Content: fmt.Sprintf("Please translate the following text to Spanish: \"%s\"", userMessage.Text),
		},
	})
	if err != nil {
		return &api.Message{}, NOOP, err
	}

	// Return the translated message
	return &api.Message{
		Text: "ZTRANSLATE: " + translationOut.Translation,
	}, NOOP, nil
}

// Stop handles any cleanup logic when TranslateToSpanishMode is deactivated
func (tsm *TranslateToSpanishMode) Stop() error {
	return nil
}

func (tsm *TranslateToSpanishMode) Name() string {
	return "translate_to_spanish"
}
