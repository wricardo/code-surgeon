package main

import (
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

	fragments := map[string][]codesurgeon.CodeFragment{
		"dynamic.go": {
			codesurgeon.CodeFragment{
				Content: codesurgeon.RenderTemplate(`
					{{- range .}}
					type {{.Name}}Provider struct {
						Id   int
						Name string
					}

					{{- end}}
				`, smsProviders),
				Overwrite: true,
			},
			codesurgeon.CodeFragment{
				Content: codesurgeon.RenderTemplate(`
				{{- range .}}
				func (p *{{.Name}}Provider) SendSMS(to, body string) error {
					fmt.Printf("Sending SMS to %s using %s provider\n", to, p.Name)
					return nil
				}
				{{- end}}
				`, smsProviders),
				Overwrite: false,
			},
			codesurgeon.CodeFragment{
				Content: `type Person struct{
					Name string
				}`,
				Overwrite: false,
			},
		},
	}

	codesurgeon.InsertCodeFragments(fragments)
}
