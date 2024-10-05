package api

import "fmt"

func (m *Message) ChatString() string {
	if m.Form != nil {
		if len(m.Form.Questions) == 0 {
			return "Form: empty"
		}
		ret := ""
		for _, qa := range m.Form.Questions {
			ret += fmt.Sprintf("Q:\n%s\n\nA:\n%s\n", qa.Question, qa.Answer)
		}
		return ret
	}
	return m.Text
}
