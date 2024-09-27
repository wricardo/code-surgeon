package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func runCLI(chat *Chat, shutdownChan chan struct{}) {
	defer chat.PrintHistory()
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("ChatCLI - Type your message and press Enter. Type /help for commands.")
	for {
		select {
		case <-shutdownChan:
			return
		default:
		}

		mode := chat.GetModeText()
		if mode != "" {
			fmt.Printf("%s:\n", mode)
		} else {
			fmt.Print("🧓:\n")
		}
		userMessage, _ := reader.ReadString('\n')
		userMessage = strings.TrimSpace(userMessage)

		response, command, err := chat.HandleUserMessage(userMessage)
		if err != nil {
			fmt.Println("Error:", err)
			return
		} else if response != "" {
			fmt.Println("🤖:\n" + response)
		}
		switch command {
		case QUIT:
			return
		case NOOP:
			continue
		case MODE_QUIT:
			continue
		case MODE_START:
			continue
		case "":
			continue
		default:
			fmt.Printf("Command not recognized: %s\n", command)
		}
	}
}

func displayHelp() {
	fmt.Println("Available commands: /help, /exit, /code, /add_test")
}
