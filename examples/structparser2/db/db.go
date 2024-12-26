package db

import "github.com/wricardo/code-surgeon/neo4j2"

// Person is a struct that represents a person in mongodb
type Person struct {
	Name string
}

// GetPerson returns a Person by the name from mongodb
func GetPerson(name string) Person {
	neo4j2.ReportStacktrace()
	return Person{Name: name}
}
