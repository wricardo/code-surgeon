package chatlib

import "github.com/wricardo/code-surgeon/api"

type TMode string

var EXIT TMode = "exit"

// Keywords for detecting different modes
// modes are added here by init() functions in the mode files using the function RegisterMode
var modeKeywords = map[string]TMode{
	"/quit": EXIT,
	"/bye":  EXIT,
}

var modeRegistry = make(map[TMode]func(*ChatImpl) IMode)

// RegisterMode registers a mode constructor in the registry
func RegisterMode[T IMode](name TMode, constructor func(*ChatImpl) T) {
	// Explicitly convert the constructor to func(*Chat) Mode
	modeRegistry[name] = func(chat *ChatImpl) IMode {
		return constructor(chat) // Cast to Mode
	}
	if _, ok := modeKeywords[string(name)]; !ok {
		modeKeywords["/"+string(name)] = name
	}
}

// Mode is a chatbot specialized for a particular task, like coding or answering questions or playing a game o top of your data
type IMode interface {
	// Start is called when the mode is activated for interactive mode.
	Start() (*api.Message, *api.Command, error)
	// BestShot is called when the mode is activated for best-shot mode, has to give the best answer given only the api message and existing state.
	BestShot(msg *api.Message) (*api.Message, *api.Command, error)
	// HandleIntent is called when the mode is activated for intent mode, has to give the best answer given the api message and the intent.
	HandleIntent(msg *api.Message, intent Intent) (*api.Message, *api.Command, error)
	// HandleResponse is called when a message is send when the mode is in interactive mode.
	HandleResponse(input *api.Message) (*api.Message, *api.Command, error)
	// Stop is called when the mode is deactivated.
	Stop() error
	// Name returns the name of the mode
	Name() string
}

// ModeHandler is an interface for different types of modes
type ModeHandler interface {
	Start() (*api.Message, error)
	HandleResponse(msg *api.Message) (*api.Message, *api.Command, error)
}
