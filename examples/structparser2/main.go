package structparser2

import (
	"github.com/sashabaranov/go-openai"
	"github.com/wricardo/code-surgeon/examples/structparser2/db"
	"github.com/wricardo/code-surgeon/neo4j2"
)

// Person is a struct that represents a person.
// It has basic fields like Name.
type Person struct {
	Name string
}

// FromDb converts a db.Person to a Person
func (_ *Person) FromDb(p db.Person) Person {
	return Person{Name: p.Name}
}

// Controller is a struct that contains a Person and a ChatCompletionRequest
type Controller struct {
	Anna    Person
	Req     openai.ChatCompletionRequest
	Pointer *openai.ChatCompletionRequest
}

// GetPerson returns a Person by the name
func (c *Controller) GetPerson(name string) Person {
	dpPerson := db.GetPerson(name)

	neo4j2.ReportStacktrace()
	return (&Person{}).FromDb(dpPerson)
}

// GetPointer returns a pointer to a Person
func (c *Controller) GetPointer(p *Person) *Person {
	return &Person{Name: "pointer"}
}

// GetPointer2 returns a pointer to a Person
func GlobalFuncWithParam(p Person) error {
	neo4j2.ReportStacktrace()
	return nil
}

// WithPointer returns a pointer to a Person
func WithPointer(p *Person) *Person {
	return p
}
