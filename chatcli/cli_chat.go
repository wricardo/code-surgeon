package chatcli

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"

	"connectrpc.com/connect"
	"github.com/charmbracelet/huh"
	"github.com/davecgh/go-spew/spew"
	"github.com/rs/zerolog/log"
	"github.com/wricardo/code-surgeon/api"
	"github.com/wricardo/code-surgeon/api/apiconnect"
	"github.com/wricardo/code-surgeon/log2"
)

type CliChat struct {
	client apiconnect.GptServiceClient
	mux    sync.Mutex
}

func NewCliChat(url string) *CliChat {
	client := apiconnect.NewGptServiceClient(http.DefaultClient, url) // replace with actual server URL
	return &CliChat{
		client: client,
	}
}

func (cli *CliChat) Start(shutdownChan chan struct{}, chatId string) error {
	// Initialize ConnectRPC client for the GptService
	ctx := context.Background()

	// Create a new chat session
	// newChatResp, err := client.NewChat(ctx, connect.NewRequest(&codesurgeon.NewChatRequest{}))
	// if err != nil {
	// 	fmt.Println("Error creating new chat:", err)
	// 	return
	// }
	// chatID := newChatResp.Msg.Chat.Id

	reader := bufio.NewReader(os.Stdin)
	fmt.Println("ChatCLI - Type your message and press Enter. Type /help for commands.")

	getChatRes, err := cli.client.GetChat(ctx, connect.NewRequest(&api.GetChatRequest{ChatId: chatId}))
	if err != nil {
		return fmt.Errorf("Error getting chat: %v", err)
	}
	spew.Dump(chatId)
	spew.Dump(getChatRes.Msg)
	for _, msg := range getChatRes.Msg.Chat.GetMessages() {
		if msg.Sender == "AI" {
			fmt.Printf("🤖: %s\n", msg.Text)
		} else {
			fmt.Printf("🧓: %s\n", msg.Text)
		}

	}

	currentModeName := ""
	for {
		select {
		case <-shutdownChan:
			return nil
		default:
		}

		// Display prompt and read user input
		if currentModeName == "" {
			fmt.Print("🧓: ")
		} else {
			fmt.Printf("🧓(%s): ", currentModeName)
		}
		userMessage, _ := reader.ReadString('\n')
		userMessage = strings.TrimSpace(userMessage)
		if userMessage == "" {
			continue
		}

		// Prepare and send the message to the server
		message := &api.Message{
			Text: userMessage}
		sendMsgReq := &api.SendMessageRequest{ChatId: chatId, Message: message}

		response, err := cli.client.SendMessage(ctx, connect.NewRequest(sendMsgReq))
		if err != nil {
			fmt.Println("Error:", err)
			return err
		}

		if response.Msg.Mode != nil {
			currentModeName = response.Msg.Mode.Name
		}
		// Handle the response from the server
		if response.Msg.Message != nil && response.Msg.Message.Text != "" {
			fmt.Printf("🤖(%s): %s\n", currentModeName, response.Msg.Message.Text)
		}

		// Handle form response if present
		if response.Msg.Message != nil && response.Msg.Message.Form != nil {
			formResponse, cmd, err := cli.handleForm(response.Msg.Message)
			if err != nil {
				fmt.Println("Error handling form:", err)
				return err
			}
			if cmd == QUIT {
				log.Debug().Msg("Exiting chat")
				return nil
			}

			// Send form response back to the server
			sendFormReq := &api.SendMessageRequest{ChatId: chatId, Message: formResponse}

			response, err = cli.client.SendMessage(ctx, connect.NewRequest(sendFormReq))
			if err != nil {
				fmt.Println("Error sending form response:", err)
				return err
			}

			if response.Msg.Message.Text != "" {
				fmt.Println("🤖:", response.Msg.Message.Text)
			} else if response.Msg.Message.Form != nil {
				fmt.Println("handling form")
				newresponse, _, err := cli.handleForm(response.Msg.Message)
				if err != nil {
					fmt.Println("Error:", err)
					return err
				}
				log2.Debugf("CLI handleForm response: \n%v", newresponse)
				sendFormReq.Message = newresponse
				// send the response back to the chat to process the form submission by the user
				continue
			}
			// formResponse, cmd, err := cli.handleForm(response.Msg.Message)
			// if err != nil {
			// 	fmt.Println("Error handling form:", err)
			// 	return
			// }
			// if cmd == QUIT {
			// 	log.Debug().Msg("Exiting chat")
			// 	return
			// }

			// // Send form response back to the server
			// sendFormReq := &api.SendMessageRequest{Message: formResponse}
			// response, err = client.SendMessage(ctx, connect.NewRequest(sendFormReq))
			// if err != nil {
			// 	fmt.Println("Error sending form response:", err)
			// 	return
			// }

			// fmt.Println("🤖:", response.Msg.Message.Text)
		}

		if response.Msg.Command != nil {
			switch response.Msg.Command.Name {
			case string(QUIT.Name):
				fmt.Println("Exiting chat")
				return nil
			}
		}

		// Process other commands based on the server response
		// (e.g., QUIT, MODE_START, etc.) if needed
	}
}

func (cli *CliChat) handleForm(msg *api.Message) (responseMsg *api.Message, responseCmd *api.Command, err error) {
	log.Debug().Any("msg", msg).Msg("CliChat.handleForm started.")
	defer func() {
		log.Debug().
			Any("responseMsg", responseMsg).
			Any("responseCmd", responseCmd).
			Any("err", err).
			Msg("CliChat.handleForm completed.")
	}()

	// Create a slice of inputs to collect responses
	var fields []huh.Field

	// Iterate over the questions in the form and add them to the group
	for k, qa := range msg.Form.Questions {
		if qa.Question == "" {
			continue
		}

		// Create a new input for each question
		input := huh.NewInput().
			Title(qa.Question).
			Value(&msg.Form.Questions[k].Answer)

		// Add the input field to the list of fields
		fields = append(fields, input)

	}
	if len(fields) == 0 {
		return &api.Message{Text: "warn: no questions in form"}, NOOP, nil
	}

	// Create a form group with all questions
	form := huh.NewForm(
		huh.NewGroup(fields...),
	)

	// Run the form group to get all responses
	err = form.Run()
	if err != nil {
		return &api.Message{}, NOOP, err // Return error if form fails
	}

	log.Debug().Any("msg", msg).Msg("CliChat.handleForm afterrun.")
	return msg, NOOP, nil
}
