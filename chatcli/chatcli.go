package chatcli

import (
	"github.com/rs/zerolog/log"

	"github.com/sashabaranov/go-openai"
	"github.com/wricardo/code-surgeon/api"
)

// INTENT_DATA is the data used to train the AI model to detect intents
var INTENT_DATA string = `
g: "exit" i: "exit"
g: "write some code" i: "code"
g: "i want to write some code" i: "code"
g: "I want to query neo4j" i: "cypher"
g: "what can you do" i: "help"
g: "consult the knowledge base" i: "question_answer"
g: "add a question and answer to knowledge base" i: "teacher"
g: "I want to query postgres" i: "postgres"
g: "I want to fetch data from postgres" i: "postgres"
g: "I want run some bash script" i: "bash"
g: "I want to know information about a local golang package, directory or file." i: "codeparser"
g: "I want to know the signature/info of function XYZ from folder X" i: "codeparser"
g: "I want to know the signature/info of function XYZ from file X" i: "codeparser"
g: "I want to parse the code in ./xyz directory" i: "codeparser"
g: "I want to parse the code in ./xyz.go file" i: "codeparser"
g: "I want to see the summary of the conversation" i: "debug"
g: "I want to see the history of the conversation" i: "debug"
g: "I want to save information to the knowledge base(KB)" i: "teacher"
g: "I want to make an http request" i: "resty"
g: "I want to translate text to spanish" i: "translate"
g: "I want to create a zasper workflow" i: "zasper"
`

const (
	SenderAI  = "AI"
	SenderYou = "You"
)

var NOOP *api.Command = &api.Command{Name: "noop"}
var QUIT *api.Command = &api.Command{Name: "quit"}
var MODE_QUIT *api.Command = &api.Command{Name: "mode_quit"}
var SILENT_MODE_QUIT_CLEAR *api.Command = &api.Command{Name: "silent_mode_quit_clear"} // not save to history, quit mode, persistance will clear history
var SILENT_CLEAR *api.Command = &api.Command{Name: "silent_clear"}                     // not save to history, quit mode, persistance will clear history
var SILENT_MODE_QUIT *api.Command = &api.Command{Name: "silent_mode_quit"}
var MODE_START *api.Command = &api.Command{Name: "mode_start"}
var SILENT *api.Command = &api.Command{Name: "silent"} // this will not save to history

func ModeFromString(mode string) TMode {
	mode = "/" + mode
	if m, ok := modeKeywords[mode]; ok {
		return m
	}
	return ""
}

func TextMessage(text string) *api.Message {
	return &api.Message{
		Text: text,
	}
}

type LLMChat interface {
	Chat(aiOut interface{}, msgs []openai.ChatCompletionMessage) error
}

// MessagePayload represents a single chat message
type MessagePayload struct {
	Sender  string
	Message *api.Message
}

type Intent struct {
	Mode                   IMode
	TMode                  TMode
	ParsedIntentAttributes map[string]string
}

// RequestResponseChat is a chat that uses requests and response pattern for communication with the user.
type RequestResponseChat struct {
	shutdownChan chan struct{}
	chat         IChat
}

func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	str, ok := v.(string)
	if !ok {
		return ""
	}
	return str
}

func toFloat32Slice(v interface{}) []float32 {
	if v == nil {
		return nil
	}
	floats, ok := v.([]float32)
	if !ok {
		return nil
	}
	return floats
}

func toFloat64(v interface{}) float64 {
	if v == nil {
		return 0
	}
	f, ok := v.(float64)
	if !ok {
		return 0
	}
	return f
}

func fnValidateNotEmpty[T comparable](s T) bool {
	var z T
	return s != z
}

func HandleTopLevelResponseCommand(cmd *api.Command, chat *ChatImpl, chatRepo ChatRepository) {
	log.Debug().Any("cmd", cmd).Msg("HandleTopLevelResponseCommand started.")
	if cmd == nil {
		return
	}
	switch cmd.Name {
	case "mode_quit_clear":
		fetched, err := chatRepo.GetChat(chat.Id)
		if err != nil {
			log.Error().Err(err).Msg("Failed to fetch chat")
			return
		}
		if fetched == nil {
			log.Warn().Msg("Chat not found")
			return
		}
		fetched.History = []*api.Message{}
		if err := chatRepo.SaveChat(chat.Id, fetched); err != nil {
			log.Error().Err(err).Msg("Failed to save chat")
			return
		}
	default:
		if err := chatRepo.SaveToDisk(); err != nil {
			log.Error().Err(err).Msg("Failed to save to disk")
		}

	}
}
