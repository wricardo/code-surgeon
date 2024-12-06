package chatcli

import (
	"github.com/wricardo/code-surgeon/chatlib"
	"github.com/wricardo/code-surgeon/api"
)

func HandleTopLevelResponseCommand(cmd *api.Command, chat *chatlib.ChatImpl, chatRepo ChatRepository) {
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
