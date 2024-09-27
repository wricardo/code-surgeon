package main

type AddTestMode struct {
	questionAnswerMap map[string]string
	questions         []string
	questionIndex     int
}

func NewAddTestMode(chat *Chat) *AddTestMode {
	return &AddTestMode{
		questionAnswerMap: make(map[string]string),
		questions: []string{
			"Which function would you like to test?",
			"In which file is this function located?",
			"Where should the test file be saved?",
			"Are there any specific edge cases you want to cover?",
		},
		questionIndex: 0,
	}
}

func (ats *AddTestMode) Start() (string, Command, error) {
	message := "Starting add test mode. I will ask you some questions to generate a test function."
	question, _ := ats.AskNextQuestion()
	return message + "\n" + question, MODE_START, nil
}

func (ats *AddTestMode) HandleResponse(userMessage string) (string, Command, error) {
	ats.questionAnswerMap[ats.questions[ats.questionIndex]] = userMessage
	ats.questionIndex++
	if ats.questionIndex < len(ats.questions) {
		question, _ := ats.AskNextQuestion()
		return question, NOOP, nil
	} else {
		response, _ := ats.GenerateTestCode()
		return response, MODE_QUIT, nil
	}
}

func (ats *AddTestMode) AskNextQuestion() (string, error) {
	if ats.questionIndex >= len(ats.questions) {
		response, _ := ats.GenerateTestCode()
		return response, nil
	}
	question := ats.questions[ats.questionIndex]
	return question, nil
}

func (ats *AddTestMode) GenerateTestCode() (string, error) {
	// Generate test code based on user inputs
	testCode := "<<GENERATED TEST CODE>>"
	return "Generating test code based on your inputs...\n" + testCode, nil
}

func (ats *AddTestMode) Stop() error {
	return nil
}