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

func mustCreateChat(t *testing.T, client apiconnect.GptServiceClient) *api.NewChatResponse {
	newChatResponse, err := client.NewChat(context.Background(), connect.NewRequest(&api.NewChatRequest{}))
	require.NoError(t, err)
	require.NotNil(t, newChatResponse)
	require.NotNil(t, newChatResponse.Msg)
	require.NotNil(t, newChatResponse.Msg.Chat)
	require.NotEmpty(t, newChatResponse.Msg.Chat.Id)
	return newChatResponse.Msg

}

func requireAllGood[T any](t *testing.T, res *connect.Response[T], err error) {
	require.NoError(t, err)
	require.NotNil(t, res)
	require.NotNil(t, res.Msg)
}

// TestNewChat tests the creation of a new chat via the gRPC client.
func TestNewChat(t *testing.T) {
	ctx := context.Background()

	// Connect to the gRPC server
	client := apiconnect.NewGptServiceClient(http.DefaultClient, "http://localhost:8010")

	// Send a request to create a new chat
	newChatResponse, err := client.NewChat(ctx, connect.NewRequest(&api.NewChatRequest{}))
	requireAllGood(t, newChatResponse, err)

	// Ensure the Chat object in the response is not nil
	require.NotNil(t, newChatResponse.Msg.Chat)

	// Ensure the Chat ID is not empty
	require.NotEmpty(t, newChatResponse.Msg.Chat.Id)

	// Ensure the history of messages is empty for a new chat
	require.Empty(t, newChatResponse.Msg.Chat.Messages)
}

// TestSendMessage tests sending a message to the gRPC server, which should return a response message from gpt
func TestSendMessage(t *testing.T) {
	ctx := context.Background()

	// Connect to the gRPC server
	client := apiconnect.NewGptServiceClient(http.DefaultClient, "http://localhost:8010")

	// Create a new chat
	newChatRes := mustCreateChat(t, client)

	// Send a message to the new chat
	sendMessageRequest := &api.SendMessageRequest{
		ChatId: newChatRes.Chat.Id,
		Message: &api.Message{
			Text: "Hello, my name is Zyphir, can you say hello Zyphir?",
		},
	}
	sendMessageResponse, err := client.SendMessage(ctx, connect.NewRequest(sendMessageRequest))
	requireAllGood(t, sendMessageResponse, err)
	log.Printf("Response: %v", sendMessageResponse)

	// Ensure the response message text matches the sent message
	require.NotEmpty(t, sendMessageResponse.Msg.Message.Text)
	require.Contains(t, sendMessageResponse.Msg.Message.Text, "Zyphir")

	// Ensure the response command is not nil
	require.NotNil(t, sendMessageResponse.Msg.Command)

	// Ensure the response command is NOOP
	require.Equal(t, chatcli.NOOP.Name, sendMessageResponse.Msg.Command.Name)
}

// TestSendMessageDebug tests sending a "/debug" message to the gRPC server, which should switch the chat mode to debug.
// The response should contain the new mode name.
func TestSendMessageDebug(t *testing.T) {
	ctx := context.Background()

	// Connect to the gRPC server
	client := apiconnect.NewGptServiceClient(http.DefaultClient, "http://localhost:8010")

	// Create a new chat
	newChatRes := mustCreateChat(t, client)

	// Send a /debug message to the new chat
	sendMessageRequest := &api.SendMessageRequest{
		ChatId: newChatRes.Chat.Id,
		Message: &api.Message{
			Text: "/debug",
		},
	}
	sendMessageResponse, err := client.SendMessage(ctx, connect.NewRequest(sendMessageRequest))
	requireAllGood(t, sendMessageResponse, err)
	log.Printf("Response: %v", sendMessageResponse)

	// Ensure the mode has switched to debug
	require.NotNil(t, sendMessageResponse.Msg.Mode)
	require.Equal(t, "debug", sendMessageResponse.Msg.Mode.Name)
}

// TestSendMessageHistory tests sending a "/debug history" message to the gRPC server, which should return the chat history and remain in the main mode.
func TestSendMessageHistory(t *testing.T) {
	ctx := context.Background()

	// Connect to the gRPC server
	client := apiconnect.NewGptServiceClient(http.DefaultClient, "http://localhost:8010")

	// Create a new chat
	newChatResponse, err := client.NewChat(ctx, connect.NewRequest(&api.NewChatRequest{}))
	requireAllGood(t, newChatResponse, err)
	require.NotNil(t, newChatResponse.Msg.Chat)
	require.NotEmpty(t, newChatResponse.Msg.Chat.Id)

	// Send a /debug history message to the new chat
	sendMessageRequest := &api.SendMessageRequest{
		ChatId: newChatResponse.Msg.Chat.Id,
		Message: &api.Message{
			Text: "/debug history",
		},
	}
	sendMessageResponse, err := client.SendMessage(ctx, connect.NewRequest(sendMessageRequest))
	requireAllGood(t, sendMessageResponse, err)
	msg := sendMessageResponse.Msg.Message

	// Ensure the response contains the chat history
	require.NotNil(t, msg.Text)

	// Ensure the mode remains in main (empty mode name)
	require.NotNil(t, sendMessageResponse.Msg.Mode)
	require.Equal(t, "main", sendMessageResponse.Msg.Mode.Name)
}

// TestSendMessageMirror tests that after switching to mirror mode, the chat remains in mirror mode.
func TestSendMessageMirror(t *testing.T) {
	ctx := context.Background()

	// Connect to the gRPC server
	client := apiconnect.NewGptServiceClient(http.DefaultClient, "http://localhost:8010")

	// Create a new chat
	newChatRes := mustCreateChat(t, client)

	// Send a /mirror message to the new chat to switch to mirror mode
	sendMirrorModeRequest := &api.SendMessageRequest{
		ChatId: newChatRes.Chat.Id,
		Message: &api.Message{
			Text: "/mirror",
		},
	}
	mirrorModeResponse, err := client.SendMessage(ctx, connect.NewRequest(sendMirrorModeRequest))
	requireAllGood(t, mirrorModeResponse, err)
	log.Printf("Mirror Mode Response: %v", mirrorModeResponse)

	// Ensure the mode has switched to mirror
	require.NotNil(t, mirrorModeResponse.Msg.Mode)
	require.Equal(t, "mirror", mirrorModeResponse.Msg.Mode.Name)

	// Send a message in mirror mode
	sendMessageRequest := &api.SendMessageRequest{
		ChatId: newChatRes.Chat.Id,
		Message: &api.Message{
			Text: "This is a test message in mirror mode.",
		},
	}
	sendMessageResponse, err := client.SendMessage(ctx, connect.NewRequest(sendMessageRequest))
	requireAllGood(t, sendMessageResponse, err)
	log.Printf("First Message Response: %v", sendMessageResponse)

	// Ensure the mode remains mirror
	require.NotNil(t, sendMessageResponse.Msg.Mode)
	require.Equal(t, "mirror", sendMessageResponse.Msg.Mode.Name)

	// Ensure the response message mirrors the input message
	require.NotNil(t, sendMessageResponse.Msg.Message)
	require.Equal(t, sendMessageRequest.Message.Text, sendMessageResponse.Msg.Message.Text)

	// Send another message in mirror mode
	sendMessageRequest.Message.Text = "Another test message to confirm mirror mode persists."
	sendMessageResponse, err = client.SendMessage(ctx, connect.NewRequest(sendMessageRequest))
	requireAllGood(t, sendMessageResponse, err)
	log.Printf("Second Message Response: %v", sendMessageResponse)

	// Ensure the mode still remains mirror
	require.NotNil(t, sendMessageResponse.Msg.Mode)
	require.Equal(t, "mirror", sendMessageResponse.Msg.Mode.Name)

	// Ensure the response message mirrors the new input message
	require.NotNil(t, sendMessageResponse.Msg.Message)
	require.Equal(t, sendMessageRequest.Message.Text, sendMessageResponse.Msg.Message.Text)
}

// TestSendMessageClear tests the "/clear" command functionality.
// It ensures that after sending the "/clear" command, the chat history is emptied and persists.
func TestSendMessageClear(t *testing.T) {
	ctx := context.Background()

	// Connect to the gRPC server
	client := apiconnect.NewGptServiceClient(http.DefaultClient, "http://localhost:8010")

	// Create a new chat
	newChatRes := mustCreateChat(t, client)

	// Send the first message to the new chat
	sendMessageRequest1 := &api.SendMessageRequest{
		ChatId: newChatRes.Chat.Id,
		Message: &api.Message{
			Text: "Hello, I am Zanir, how are you?",
		},
	}
	sendMessageResponse1, err := client.SendMessage(ctx, connect.NewRequest(sendMessageRequest1))
	requireAllGood(t, sendMessageResponse1, err)
	log.Printf("First Message Response: %v", sendMessageResponse1)

	// Send the second message
	sendMessageRequest2 := &api.SendMessageRequest{
		ChatId: newChatRes.Chat.Id,
		Message: &api.Message{
			Text: "what is my name?",
		},
	}
	sendMessageResponse2, err := client.SendMessage(ctx, connect.NewRequest(sendMessageRequest2))
	requireAllGood(t, sendMessageResponse2, err)
	log.Printf("Second Message Response: %v", sendMessageResponse2)

	// Use "/debug history" to retrieve and verify the chat history contains the messages
	debugHistoryRequest := &api.SendMessageRequest{
		ChatId: newChatRes.Chat.Id,
		Message: &api.Message{
			Text: "/debug history",
		},
	}
	debugHistoryResponse, err := client.SendMessage(ctx, connect.NewRequest(debugHistoryRequest))
	requireAllGood(t, debugHistoryResponse, err)
	require.NotNil(t, debugHistoryResponse.Msg.Message)
	historyText := debugHistoryResponse.Msg.Message.Text

	// Ensure the history contains the initial messages
	require.Contains(t, historyText, "Hello, I am Zanir, how are you?")
	require.Contains(t, historyText, "what is my name?")

	// Send the "/clear" command to clear the chat history
	clearMessageRequest := &api.SendMessageRequest{
		ChatId: newChatRes.Chat.Id,
		Message: &api.Message{
			Text: "/clear",
		},
	}
	clearMessageResponse, err := client.SendMessage(ctx, connect.NewRequest(clearMessageRequest))
	requireAllGood(t, clearMessageResponse, err)
	log.Printf("Clear Command Response: %v", clearMessageResponse)

	// Use "/debug history" again to verify the chat history is now empty
	debugHistoryResponseAfterClear, err := client.SendMessage(ctx, connect.NewRequest(debugHistoryRequest))
	requireAllGood(t, debugHistoryResponseAfterClear, err)
	require.NotNil(t, debugHistoryResponseAfterClear.Msg.Message)
	historyTextAfterClear := debugHistoryResponseAfterClear.Msg.Message.Text

	// Ensure the history does not contain the previous messages
	require.NotContains(t, historyTextAfterClear, "Hello, I am Zanir, how are you?")
	require.NotContains(t, historyTextAfterClear, "what is my name?")

	// Optionally, ensure that the history only contains system messages if any
	// For example, check if the history only includes the "/clear" and "/debug history" commands
	// Adjust the assertions based on how your server handles system messages in history
}
