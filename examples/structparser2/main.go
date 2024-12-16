package structparser2

import (
	"github.com/sashabaranov/go-openai"
)

type Person struct {
	Name string
}

type Controller struct {
	Anna    Person
	Req     openai.ChatCompletionRequest
	Pointer *openai.ChatCompletionRequest
}

func (c *Controller) GetPerson(id string) Person {
	return Person{Name: id}
}

func (c *Controller) GetPointer(p *Person) *Person {
	return &Person{Name: "pointer"}
}

func GlobalFuncWithParam(p Person) error {
	return nil
}

func WithPointer(p *Person) *Person {
	return p
}
