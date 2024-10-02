package main

import (
	"html/template"
	"log"
	"net/http"
)

type Message struct {
	Author  string
	Content string
}

var messages []Message

func main() {
	http.HandleFunc("/", chatHandler)
	log.Println("Starting server on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func chatHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		author := r.FormValue("author")
		content := r.FormValue("content")
		if author != "" && content != "" {
			messages = append(messages, Message{Author: author, Content: content})
		}
	}

	tmpl := template.Must(template.New("chat").Parse(`
		<!DOCTYPE html>
		<html lang="en">
		<head>
			<meta charset="UTF-8">
			<meta name="viewport" content="width=device-width, initial-scale=1.0">
			<title>Chat UI</title>
		</head>
		<body>
			<h1>Chat Room</h1>
			<form method="POST" action="/">
				<input type="text" name="author" placeholder="Your name" required>
				<input type="text" name="content" placeholder="Your message" required>
				<button type="submit">Send</button>
			</form>
			<div id="chat">
				{{range .}}
					<p><strong>{{.Author}}:</strong> {{.Content}}</p>
				{{end}}
			</div>
		</body>
		</html>
	`))

	tmpl.Execute(w, messages)
}
