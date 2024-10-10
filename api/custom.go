package api

import "fmt"

func (m *Message) ChatString() string {
	if m.Form != nil && len(m.Form.Questions) > 0 {
		ret := ""
		for _, qa := range m.Form.Questions {
			ret += fmt.Sprintf("Q:\n%s\nA:\n%s\n\n", qa.Question, qa.Answer)
		}
		return ret
	}
	if m.Text != "" {
		return m.Text

	}

	return "No message or empty message"
}
