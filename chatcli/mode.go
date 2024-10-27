package chatcli

import "github.com/wricardo/code-surgeon/chatlib"

type TMode = chatlib.TMode

var EXIT TMode = "exit"

// Keywords for detecting different modes
// modes are added here by init() functions in the mode files using the function RegisterMode
var modeKeywords = map[string]TMode{
	"/quit": EXIT,
	"/bye":  EXIT,
}

var modeRegistry = chatlib.ModeRegistry

// RegisterMode registers a mode constructor in the registry
func RegisterMode[T chatlib.IMode](name TMode, constructor func(*chatlib.ChatImpl) T) {
	chatlib.RegisterMode(name, constructor)
}

// Mode is a chatbot specialized for a particular task, like coding or answering questions or playing a game o top of your data
type IMode = chatlib.IMode

// ModeHandler is an interface for different types of modes
type ModeHandler = chatlib.ModeHandler
