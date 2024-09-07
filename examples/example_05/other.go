package main

import "fmt"

// Testing my automatic comment
func Testing() {
	fmt.Println("Testing")

}

// DoStuff my automatic comment
func DoStuff() {
	fmt.Println("Doing stuff")
}

type MyStruct struct {
}
// MyMethod my automatic comment for method for struct MyStruct
func (m *MyStruct) MyMethod() {
	fmt.Println("My method")
}

type OtherStruct struct {
}
// MyMethod my automatic comment for method for struct OtherStruct
func (m *OtherStruct) MyMethod() {
	fmt.Println("My method")
}
