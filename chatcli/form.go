package chatcli

import (
	"github.com/wricardo/code-surgeon/chatlib"
)

type Form = chatlib.Form
type FormQuestion = chatlib.FormQuestion
type StringPromise = chatlib.StringPromise

var NewForm = chatlib.NewForm
var TryFillFormFromTextMessage = chatlib.TryFillFormFromTextMessage
