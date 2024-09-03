package main

import "fmt"

type Person struct {
	Id   string
	Name string
}

func (p Person) GetID() string {
	return p.Id
}
func (p Person) SetID(id string) {
	p.Id = id
}
func (_ Person) GetLabel() string {
	return "Person"
}

// not in the pattern
func (p Person) GetProps() map[string]any {
	return map[string]any{
		"Name": p.Name,
	}
}

// SetProps my manual comment
func (p Person) SetProps(props map[string]any) {
	if val, ok := props["Name"]; ok {
		p.Name = val.(string)
	}
}

func Testing() {
	fmt.Println(
		"Testing")
}

// DoStuff my manual comment
func DoStuff() {
	fmt.Println("Doing stuff")
}
