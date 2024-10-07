package grpc

import (
	"context"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	log "github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"
	"github.com/wricardo/code-surgeon/api"
	"github.com/wricardo/code-surgeon/api/apiconnect"
	"github.com/wricardo/code-surgeon/chatcli"
)

// TestNewChat tests the creation of a new chat via the gRPC client.
func TestNewChat(t *testing.T) {
	ctx := context.Background()

	// Connect to the gRPC server
	client := apiconnect.NewGptServiceClient(http.DefaultClient, "http://localhost:8010")

	// Send a request to create a new chat
	newChatResponse, err := client.NewChat(ctx, connect.NewRequest(&api.NewChatRequest{}))

	// Ensure no error occurred
	require.NoError(t, err)

	// Ensure the response is not nil
	require.NotNil(t, newChatResponse)

	// Ensure the response message is not nil
	require.NotNil(t, newChatResponse.Msg)

	// Ensure the Chat object in the response is not nil
	require.NotNil(t, newChatResponse.Msg.Chat)

	// Ensure the Chat ID is not empty
	require.NotEmpty(t, newChatResponse.Msg.Chat.Id)

	// Ensure the history of messages is empty for a new chat
	require.Empty(t, newChatResponse.Msg.Chat.Messages)
}

// TestSendMessageDebug tests sending a "/debug" message to the gRPC server, which should switch the chat mode to debug.
// The response should contain the new mode name.
func TestSendMessageDebug(t *testing.T) {
	ctx := context.Background()

	// Connect to the gRPC server
	client := apiconnect.NewGptServiceClient(http.DefaultClient, "http://localhost:8010")

	// Create a new chat
	newChatResponse, err := client.NewChat(ctx, connect.NewRequest(&api.NewChatRequest{}))
	require.NoError(t, err)
	require.NotNil(t, newChatResponse)
	require.NotNil(t, newChatResponse.Msg)
	require.NotNil(t, newChatResponse.Msg.Chat)
	require.NotEmpty(t, newChatResponse.Msg.Chat.Id)

	// Send a /debug message to the new chat
	sendMessageRequest := &api.SendMessageRequest{
		Message: &api.Message{
			ChatId: newChatResponse.Msg.Chat.Id,
			Text:   "/debug",
		},
	}
	sendMessageResponse, err := client.SendMessage(ctx, connect.NewRequest(sendMessageRequest))
	log.Printf("Response: %v", sendMessageResponse)

	// Ensure no error occurred
	require.NoError(t, err)

	// Ensure the response is not nil
	require.NotNil(t, sendMessageResponse)

	// Ensure the response message is not nil
	require.NotNil(t, sendMessageResponse.Msg)

	// Ensure the mode has switched to debug
	require.NotNil(t, sendMessageResponse.Msg.Mode)
	require.Equal(t, "debug", sendMessageResponse.Msg.Mode.Name)
}

func TestSendMessage(t *testing.T) {
	ctx := context.Background()

	// Connect to the gRPC server
	client := apiconnect.NewGptServiceClient(http.DefaultClient, "http://localhost:8010")

	// Create a new chat
	newChatResponse, err := client.NewChat(ctx, connect.NewRequest(&api.NewChatRequest{}))
	require.NoError(t, err)
	require.NotNil(t, newChatResponse)
	require.NotNil(t, newChatResponse.Msg)
	require.NotNil(t, newChatResponse.Msg.Chat)
	require.NotEmpty(t, newChatResponse.Msg.Chat.Id)

	// Send a message to the new chat
	sendMessageRequest := &api.SendMessageRequest{
		Message: &api.Message{
			ChatId: newChatResponse.Msg.Chat.Id,
			Text:   "Hello, my name is Zyphir, can you say hello Zyphir?",
		},
	}
	sendMessageResponse, err := client.SendMessage(ctx, connect.NewRequest(sendMessageRequest))
	log.Printf("Response: %v", sendMessageResponse)
	require.NoError(t, err)
	require.NotNil(t, sendMessageResponse)
	require.NotNil(t, sendMessageResponse.Msg)

	// Ensure the response message text matches the sent message
	require.NotEmpty(t, sendMessageResponse.Msg.Message.Text)
	require.Contains(t, sendMessageResponse.Msg.Message.Text, "Zyphir")

	// Ensure the response command is not nil
	require.NotNil(t, sendMessageResponse.Msg.Command)

	// Ensure the response command is NOOP
	require.Equal(t, chatcli.NOOP.Name, sendMessageResponse.Msg.Command.Name)
}

// TestSendMessageHistory tests sending a "/debug history" message to the gRPC server, which should return the chat history and remain in the main mode.
func TestSendMessageHistory(t *testing.T) {
	ctx := context.Background()

	// Connect to the gRPC server
	client := apiconnect.NewGptServiceClient(http.DefaultClient, "http://localhost:8010")

	// Create a new chat
	newChatResponse, err := client.NewChat(ctx, connect.NewRequest(&api.NewChatRequest{}))
	require.NoError(t, err)
	require.NotNil(t, newChatResponse)
	require.NotNil(t, newChatResponse.Msg)
	require.NotNil(t, newChatResponse.Msg.Chat)
	require.NotEmpty(t, newChatResponse.Msg.Chat.Id)

	// Send a /debug history message to the new chat
	sendMessageRequest := &api.SendMessageRequest{
		Message: &api.Message{
			ChatId: newChatResponse.Msg.Chat.Id,
			Text:   "/debug history",
		},
	}
	sendMessageResponse, err := client.SendMessage(ctx, connect.NewRequest(sendMessageRequest))
	require.NoError(t, err)
	require.NotNil(t, sendMessageResponse)
	require.NotNil(t, sendMessageResponse.Msg)
	msg := sendMessageResponse.Msg.Message

	// Ensure the response contains the chat history
	require.NotNil(t, msg.Text)

	// Ensure the mode remains in main (empty mode name)
	require.NotNil(t, sendMessageResponse.Msg.Mode)
	require.Equal(t, "", sendMessageResponse.Msg.Mode.Name)
}
