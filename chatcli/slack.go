package chatcli

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog/log"
	"github.com/slack-go/slack"

	"connectrpc.com/connect"
	"github.com/wricardo/code-surgeon/api"
	"github.com/wricardo/code-surgeon/api/apiconnect"
)

type SlackBot struct {
	SlackClient *slack.Client
	GrpcClient  apiconnect.GptServiceClient
}

func NewSlackBot(slackToken string, grpcClient apiconnect.GptServiceClient) *SlackBot {
	return &SlackBot{
		SlackClient: slack.New(slackToken),
		GrpcClient:  grpcClient,
	}
}

func (bot *SlackBot) sendMessage(channelID, message string) error {
	_, _, err := bot.SlackClient.PostMessage(
		channelID,
		slack.MsgOptionText(message, false),
	)
	if err != nil {
		log.Error().Err(err).Msg("Error sending message to Slack")
		return err
	}
	return nil
}

func (bot *SlackBot) handleDirectMessage(ctx context.Context, event Event) {
	channelID := event.Channel
	incomingMessage := event.Text

	res, err := bot.GrpcClient.SendMessage(ctx, connect.NewRequest(&api.SendMessageRequest{
		ChatId: event.User,
		Message: &api.Message{
			Text:   incomingMessage,
			Form:   &api.FormMessage{},
			Sender: SenderYou, // Adjust according to your API
		},
	}))
	if err != nil {
		log.Error().Err(err).Msg("Error sending message to gRPC service")
		return
	}

	responseMessage := res.Msg.Message.ChatString()
	if err := bot.sendMessage(channelID, responseMessage); err != nil {
		log.Error().Err(err).Msg("Failed to send response message to Slack")
	} else {
		log.Debug().Msg("Response sent to Slack")
	}
}

func (bot *SlackBot) SlackMessageHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Debug().Msg("Received Slack message")

		var postBody SlackMessage
		if err := json.NewDecoder(r.Body).Decode(&postBody); err != nil {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			log.Error().Err(err).Msg("Failed to decode Slack message")
			return
		}

		// Handle Slack URL verification challenge
		if postBody.Challenge != "" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"challenge": postBody.Challenge})
			return
		}

		// Ignore messages from bots
		if postBody.Event.BotID != "" || postBody.Event.SubType == "bot_message" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Process direct messages
		if postBody.Event.Type == "message" && postBody.Event.ChannelType == "im" {
			go bot.handleDirectMessage(r.Context(), postBody.Event)
			w.WriteHeader(http.StatusOK)
			return
		}

		log.Warn().
			Str("event_type", postBody.Event.Type).
			Msg("Unhandled message type")

		w.WriteHeader(http.StatusOK)
	}
}

type SlackMessage struct {
	ContextEnterpriseID any     `json:"context_enterprise_id"`
	EventID             string  `json:"event_id"`
	IsExtSharedChannel  bool    `json:"is_ext_shared_channel"`
	EventContext        string  `json:"event_context"`
	ContextTeamID       string  `json:"context_team_id"`
	TeamID              string  `json:"team_id"`
	APIAppID            string  `json:"api_app_id"`
	Event               Event   `json:"event"`
	Type                string  `json:"type"`
	EventTime           float64 `json:"event_time"`
	Authorizations      []struct {
		IsEnterpriseInstall bool   `json:"is_enterprise_install"`
		EnterpriseID        any    `json:"enterprise_id"`
		TeamID              string `json:"team_id"`
		UserID              string `json:"user_id"`
		IsBot               bool   `json:"is_bot"`
	} `json:"authorizations"`
	Token     string `json:"token"`
	Challenge string `json:"challenge"`
}

type Event struct {
	BotID       string `json:"bot_id"`
	Channel     string `json:"channel"`
	SubType     string `json:"subtype"`
	ChannelType string `json:"channel_type"`
	Type        string `json:"type"`
	Ts          string `json:"ts"`
	ClientMsgID string `json:"client_msg_id"`
	Text        string `json:"text"`
	Team        string `json:"team"`
	User        string `json:"user"`
	Blocks      []struct {
		Elements []struct {
			Type     string `json:"type"`
			Elements []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"elements"`
		} `json:"elements"`
		Type    string `json:"type"`
		BlockID string `json:"block_id"`
	} `json:"blocks"`
	EventTs string `json:"event_ts"`
}
