package chatcli

import (
	"testing"

	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"
	"github.com/wricardo/code-surgeon/api"
	"github.com/wricardo/code-surgeon/log2"
)

func init() {
	log2.Configure()
}

// Mock or setup any necessary dependencies here
func TestParseMode_HandleResponse_ValidInput(t *testing.T) {
	chat := newTestChat()
	parseMode := NewParseMode(chat)
	msg, _, err := parseMode.Start()
	require.NotNil(t, msg.Form)
	require.NoError(t, err)
	require.LessOrEqual(t, 0, len(msg.Form.Questions))
	require.NotEmpty(t, msg.Form.Questions[0].Question)

	// Execute
	msg, cmd, err := parseMode.HandleIntent(&api.Message{
		Text: "I want to parse the directory ./tmp/, output format is names",
	}, Intent{
		TMode: "codeparser",
		ParsedIntentAttributes: map[string]string{
			"directory": "./tmp/",
		},
	})

	// require
	require.NoError(t, err)
	require.Equal(t, MODE_QUIT, cmd)
	log.Debug().Msgf("msg: %v", msg)
	require.True(t, parseMode.form.IsFilled(), "Form should be filled")
	require.Equal(t, "Enter the directory or file path to parse:", parseMode.form.Questions[0].Question)
	require.Equal(t, "./tmp/", parseMode.form.Questions[0].Answer)
	require.Equal(t, "names", parseMode.form.Questions[1].Answer) // this may fail

}
