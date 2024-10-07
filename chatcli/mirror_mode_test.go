package chatcli

import (
	"testing"

	"github.com/wricardo/code-surgeon/api"
)

func TestMirrorMode_Start(t *testing.T) {
	chat := &ChatImpl{}
	mirrorMode := NewMirrorMode(chat)

	message, command, err := mirrorMode.Start()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expectedText := "Welcome to the Mirror Mode! I will repeat whatever you say."
	if message.Text != expectedText {
		t.Errorf("expected message text %q, got %q", expectedText, message.Text)
	}

	if command != MODE_START {
		t.Errorf("expected command %v, got %v", MODE_START, command)
	}
}

func TestMirrorMode_HandleIntent(t *testing.T) {
	chat := &ChatImpl{}
	mirrorMode := NewMirrorMode(chat)

	inputMessage := api.Message{Text: "Hello, Mirror!"}
	message, command, err := mirrorMode.HandleIntent(&inputMessage, Intent{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if message.Text != inputMessage.Text {
		t.Errorf("expected message text %q, got %q", inputMessage.Text, message.Text)
	}

	if command != NOOP {
		t.Errorf("expected command %v, got %v", NOOP, command)
	}
}

func TestMirrorMode_HandleResponse(t *testing.T) {
	chat := &ChatImpl{}
	mirrorMode := NewMirrorMode(chat)

	inputMessage := &api.Message{Text: "Echo this!"}
	message, command, err := mirrorMode.HandleResponse(inputMessage)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if message.Text != inputMessage.Text {
		t.Errorf("expected message text %q, got %q", inputMessage.Text, message.Text)
	}

	if command != NOOP {
		t.Errorf("expected command %v, got %v", NOOP, command)
	}
}

func TestMirrorMode_Stop(t *testing.T) {
	chat := &ChatImpl{}
	mirrorMode := NewMirrorMode(chat)

	err := mirrorMode.Stop()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}
