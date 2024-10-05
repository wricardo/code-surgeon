package chatcli

import . "github.com/wricardo/code-surgeon/api"

var MIRROR TMode = "mirror"

func init() {
	RegisterMode(MIRROR, NewMirrorMode)
}

// NewMirrorMode creates a new instance of the MirrorMode
func NewMirrorMode(chat *ChatImpl) *MirrorMode {
	return &MirrorMode{
		chat: chat,
	}
}

// MirrorMode struct implements the Mode interface
type MirrorMode struct {
	chat *ChatImpl
}

// Start initializes the MirrorMode with a welcoming message
func (mm *MirrorMode) Start() (*Message, *Command, error) {
	return &Message{
		Text: "Welcome to the Mirror Mode! I will repeat whatever you say.",
	}, MODE_START, nil
}


func (mm *MirrorMode) BestShot(msg *Message) (*Message, *Command, error) {
	message, _, err := mm.HandleResponse(msg)
	return message, NOOP, err
}

func (mm *MirrorMode) HandleIntent(userMessage *Message, intent Intent) (*Message, *Command, error) {
	return &Message{
		Text: userMessage.Text,
	}, NOOP, nil
}

// HandleResponse mirrors back the user's message
func (mm *MirrorMode) HandleResponse(userMessage *Message) (*Message, *Command, error) {
	return &Message{
		Text: userMessage.Text,
	}, NOOP, nil
}

// Stop handles any cleanup logic when MirrorMode is deactivated
func (mm *MirrorMode) Stop() error {
	return nil
}
