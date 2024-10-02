package main

import (
	"fmt"

	codesurgeon "github.com/wricardo/code-surgeon"
)

type ParseMode struct {
	chat *Chat
}

func NewParseMode(chat *Chat) *ParseMode {
	return &ParseMode{chat: chat}
}

func (m *ParseMode) Start() (Message, Command, error) {
	// Display a form to the user to get the directory or file path
	form := NewForm([]QuestionAnswer{
		{Question: "Enter the directory or file path to parse:", Answer: ""},
		{Question: "Select output format (only signatures, only names, full definition):", Answer: ""},
	})
	return Message{Form: form}, NOOP, nil
}

func (m *ParseMode) HandleIntent(msg Message) (Message, Command, error) {
	return m.HandleResponse(msg)
}

/*
type ParsedInfo struct {
	Packages []Package `json:"packages"`
}

// Package represents a Go package with its components such as imports, structs, functions, etc.
type Package struct {
	Package    string      `json:"package"`
	Imports    []string    `json:"imports,omitemity"`
	Structs    []Struct    `json:"structs,omitemity"`
	Functions  []Function  `json:"functions,omitemity"`
	Variables  []Variable  `json:"variables,omitemity"`
	Constants  []Constant  `json:"constants,omitemity"`
	Interfaces []Interface `json:"interfaces,omitemity"`
}

// Interface represents a Go interface and its methods.
type Interface struct {
	Name    string   `json:"name"`
	Methods []Method `json:"methods,omitemity"`
	Docs    []string `json:"docs,omitemity"`
}

// Struct represents a Go struct and its fields and methods.
type Struct struct {
	Name    string   `json:"name"`
	Fields  []Field  `json:"fields,omitemity"`
	Methods []Method `json:"methods,omitemity"`
	Docs    []string `json:"docs,omitemity"`
}

// Method represents a method in a Go struct or interface.
type Method struct {
	Receiver  string   `json:"receiver,omitempty"` // Receiver type (e.g., "*MyStruct" or "MyStruct")
	Name      string   `json:"name"`
	Params    []Param  `json:"params,omitemity"`
	Returns   []Param  `json:"returns,omitemity"`
	Docs      []string `json:"docs,omitemity"`
	Signature string   `json:"signature"`
	Body      string   `json:"body,omitempty"` // New field for method body
}

// Function represents a Go function with its parameters, return types, and documentation.
type Function struct {
	Name      string   `json:"name"`
	Params    []Param  `json:"params,omitemity"`
	Returns   []Param  `json:"returns,omitemity"`
	Docs      []string `json:"docs,omitemity"`
	Signature string   `json:"signature"`
	Body      string   `json:"body,omitempty"` // New field for function body
}

// Param represents a parameter or return value in a Go function or method.
type Param struct {
	Name string `json:"name"` // Name of the parameter or return value
	Type string `json:"type"` // Type (e.g., "int", "*string")
}

// Field represents a field in a Go struct.
type Field struct {
	Name    string   `json:"name"`
	Type    string   `json:"type"`
	Tag     string   `json:"tag"`
	Private bool     `json:"private"`
	Pointer bool     `json:"pointer"`
	Slice   bool     `json:"slice"`
	Docs    []string `json:"docs,omitemity"`
	Comment string   `json:"comment,omitempty"`
}

// Variable represents a global variable in a Go package.
type Variable struct {
	Name string   `json:"name"`
	Type string   `json:"type"`
	Docs []string `json:"docs,omitemity"`
}

// Constant represents a constant in a Go package.
type Constant struct {
	Name  string   `json:"name"`
	Value string   `json:"value"`
	Docs  []string `json:"docs,omitemity"`
}
*/

func (m *ParseMode) HandleResponse(input Message) (Message, Command, error) {
	if input.Form == nil || len(input.Form.Questions) == 0 {
		return Message{}, NOOP, fmt.Errorf("no input provided")
	}

	fileOrDirectory := input.Form.Questions[0].Answer
	outputFormat := input.Form.Questions[1].Answer

	parsedInfo, err := codesurgeon.ParseDirectory(fileOrDirectory)

	output := ""
	switch outputFormat {
	case "only signatures":
		// TODO: only print the signatures
	case "only names":
		// TODO: only print the names
	case "full definition":
		// TODO	:
	default:
		return Message{Text: "Invalid output format selected."}, NOOP, nil
	}
	if err != nil {
		return Message{Text: fmt.Sprintf("Error parsing: %v", err)}, NOOP, nil
	}

	// Convert parsedInfo to a string or JSON to display to the user
	return Message{Text: output}, MODE_QUIT, nil
}

func (m *ParseMode) Stop() error {
	return nil
}

func init() {
	RegisterMode("codeparser", NewParseMode)
}
