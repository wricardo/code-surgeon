package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	codesurgeon "github.com/wricardo/code-surgeon"
)

func main() {
	smsProviders := []struct {
		Id   int
		Name string
	}{
		{1, "Twilio"},
		{2, "Nexmo"},
	}

	changes := []codesurgeon.FileChange{
		{
			PackageName: "main",
			File:        "assets/dynamic.go",
			Fragments: []codesurgeon.CodeFragment{
				{
					Content: codesurgeon.RenderTemplateNoError(`
			import "fmt"
			{{- range .}}
			type {{.Name}}Provider struct {
				Id   int
				Name string
			}

			func (p *{{.Name}}Provider) SendSMS(to, body string) error {
				fmt.Printf("Sending SMS to %s using %s provider\n", to, p.Name)
				return nil
			}
			{{- end}}
			`, smsProviders),
					Overwrite: true,
				},
				{
					Content: `type Person struct{
					Name string
				}`,
					Overwrite: false,
				},
			},
		},
	}
	jsonChanges, _ := json.Marshal(changes)
	fmt.Println(strings.Replace(string(jsonChanges), "\\t", "", -1))

	codesurgeon.ApplyFileChanges(changes)
}

// Get user input for request
func getUserRequest() string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Please enter your modification request: ")
	request, _ := reader.ReadString('\n')
	return strings.TrimSpace(request)
}
