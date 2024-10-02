package chatcli

import (
	"fmt"
	"github.com/go-resty/resty/v2"
)

type RestyMode struct {
	client *resty.Client
}

func (rm *RestyMode) Start() (Message, error) {
	rm.client = resty.New()
	return TextMessage("Welcome to RestyMode! You can make HTTP requests by specifying the method and URL."), nil
}

func (rm *RestyMode) HandleIntent(msg Message) (Message, Command, error) {
	// Example: Parse user input to determine HTTP method and URL
	// This is a simplified example and should be expanded to handle more cases
	var method, url string
	fmt.Sscanf(msg.Text, "%s %s", &method, &url)

	var resp *resty.Response
	var err error

	switch method {
	case "GET":
		resp, err = rm.client.R().Get(url)
	case "POST":
		resp, err = rm.client.R().Post(url)
	default:
		return TextMessage("Unsupported HTTP method. Please use GET or POST."), NOOP, nil
	}

	if err != nil {
		return TextMessage(fmt.Sprintf("Error making request: %v", err)), NOOP, nil
	}

	return TextMessage(fmt.Sprintf("Response: %s", resp.String())), NOOP, nil
}

func (rm *RestyMode) HandleResponse(msg Message) (Message, Command, error) {
	// Handle any follow-up user inputs if needed
	return Message{}, NOOP, nil
}

func (rm *RestyMode) Stop() error {
	// Clean up resources if necessary
	return nil
}

func init() {
	RegisterMode("resty", &RestyMode{})
}
