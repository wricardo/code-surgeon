package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"connectrpc.com/connect"
	"github.com/wricardo/code-surgeon/api"
	"github.com/wricardo/code-surgeon/api/apiconnect"
)

const (
	serverAddress = "localhost:8080" // Replace with your server address
	grpcAddress   = "localhost:8010" // Replace with your gRPC server address
)

func main() {
	http.HandleFunc("/", chatPageHandler)
	http.HandleFunc("/api/send", sendMessageHandler)
	http.HandleFunc("/api/messages", getMessagesHandler)
	log.Println("Starting server on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func getClient() apiconnect.GptServiceClient {
	return apiconnect.NewGptServiceClient(http.DefaultClient, "http://"+grpcAddress)
}

func chatPageHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "static-ui/index.html")
}

func sendMessageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var req struct {
			ChatID  string `json:"chat_id"`
			Content string `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}

		if req.ChatID == "" || req.Content == "" {
			http.Error(w, "chat_id and content are required", http.StatusBadRequest)
			return
		}

		client := getClient()
		message := &api.Message{
			Text: req.Content,
		}
		sendMsgReq := &api.SendMessageRequest{
			ChatId:  req.ChatID,
			Message: message,
		}

		resp, err := client.SendMessage(context.Background(), connect.NewRequest(sendMsgReq))
		if err != nil {
			log.Println("Error sending message:", err)
			http.Error(w, "Failed to send message", http.StatusInternalServerError)
			return
		}

		responseMsg := resp.Msg.Message
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseMsg)
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func getMessagesHandler(w http.ResponseWriter, r *http.Request) {
	chatID := r.URL.Query().Get("chat_id")
	if chatID == "" {
		http.Error(w, "chat_id is required", http.StatusBadRequest)
		return
	}

	client := getClient()
	getChatResp, err := client.GetChat(context.Background(), connect.NewRequest(&api.GetChatRequest{ChatId: chatID}))
	if err != nil {
		http.Error(w, "Failed to get chat messages", http.StatusInternalServerError)
		return
	}

	messages := getChatResp.Msg.Chat.Messages
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}
