package codesurgeon

import (
	"bytes"
	"os"
	"testing"
)

// Helper function to create a temporary Go file with content
func createTempFile(t *testing.T, content string) *os.File {
	tmpfile, err := os.CreateTemp("", "example.go")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}
	return tmpfile
}

// Test adding new documentation to a function without existing comments
func TestUpsertDocumentationToFunction_AddNewDocumentation(t *testing.T) {
	content := `package main

func MyFunction() {
	fmt.Println("Hello, World!")
}
`
	tmpfile := createTempFile(t, content)
	defer os.Remove(tmpfile.Name()) // Clean up

	// Call the function to upsert documentation
	modified, err := UpsertDocumentationToFunction(tmpfile.Name(), "", "MyFunction", "// This is a new documentation")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !modified {
		t.Error("Expected documentation to be added, but it was not.")
	}

	// Read the modified file
	result, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}

	// Check if the new documentation was added correctly
	if !stringContains(result, "// This is a new documentation") {
		t.Errorf("Expected new documentation to be added, but it was not.\nGot:\n%s", result)
	}
}

// Test replacing existing documentation
func TestUpsertDocumentationToFunction_ReplaceExistingDocumentation(t *testing.T) {
	content := `package main

// Old documentation
func MyFunction() {
	fmt.Println("Hello, World!")
}
`
	tmpfile := createTempFile(t, content)
	defer os.Remove(tmpfile.Name()) // Clean up

	// Call the function to upsert documentation
	modified, err := UpsertDocumentationToFunction(tmpfile.Name(), "", "MyFunction", "// This is a new documentation")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !modified {
		t.Error("Expected documentation to be replaced, but it was not.")
	}

	// Read the modified file
	result, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}

	// Check if the old documentation was replaced with new documentation
	if stringContains(result, "// Old documentation") {
		t.Errorf("Expected old documentation to be replaced, but it was not.\nGot:\n%s", result)
	}

	if !stringContains(result, "// This is a new documentation") {
		t.Errorf("Expected new documentation to be added, but it was not.\nGot:\n%s", result)
	}
}

// Test adding new documentation to a method function (associated with a struct) without existing comments
func TestUpsertDocumentationToFunction_AddNewDocumentation_Method_Underscore(t *testing.T) {
	content := `package main

type MyStruct struct{}

func (_ *MyStruct) MyMethod() {
	fmt.Println("Hello from MyMethod!")
}
`
	tmpfile := createTempFile(t, content)
	defer os.Remove(tmpfile.Name()) // Clean up

	// Call the function to upsert documentation
	modified, err := UpsertDocumentationToFunction(tmpfile.Name(), "MyStruct", "MyMethod", "// This is a new documentation for MyMethod")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !modified {
		t.Error("Expected documentation to be added to the method, but it was not.")
	}

	// Read the modified file
	result, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}

	// Check if the new documentation was added correctly for the method
	if !stringContains(result, "// This is a new documentation for MyMethod") {
		t.Errorf("Expected new documentation to be added to the method, but it was not.\nGot:\n%s", result)
	}
}

// Test adding new documentation to a method function (associated with a struct) without existing comments
func TestUpsertDocumentationToFunction_AddNewDocumentation_Method_NotPointer(t *testing.T) {
	content := `package main

type MyStruct struct{}

func (m MyStruct) MyMethod() {
	fmt.Println("Hello from MyMethod!")
}
`
	tmpfile := createTempFile(t, content)
	defer os.Remove(tmpfile.Name()) // Clean up

	// Call the function to upsert documentation
	modified, err := UpsertDocumentationToFunction(tmpfile.Name(), "MyStruct", "MyMethod", "// This is a new documentation for MyMethod")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !modified {
		t.Error("Expected documentation to be added to the method, but it was not.")
	}

	// Read the modified file
	result, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}

	// Check if the new documentation was added correctly for the method
	if !stringContains(result, "// This is a new documentation for MyMethod") {
		t.Errorf("Expected new documentation to be added to the method, but it was not.\nGot:\n%s", result)
	}
}

// Test adding new documentation to a method function (associated with a struct) with existing comments
func TestUpsertDocumentationToFunction_ReplaceNewDocumentation_Method(t *testing.T) {
	content := `package main

type MyStruct struct{}

// MyMethod old comment
func (m *MyStruct) MyMethod() {
	fmt.Println("Hello from MyMethod!")
}
`
	tmpfile := createTempFile(t, content)
	defer os.Remove(tmpfile.Name()) // Clean up

	// Call the function to upsert documentation
	modified, err := UpsertDocumentationToFunction(tmpfile.Name(), "MyStruct", "MyMethod", "// This is a new documentation for MyMethod")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !modified {
		t.Error("Expected documentation to be added to the method, but it was not.")
	}

	// Read the modified file
	result, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}

	// Check if the new documentation was added correctly for the method
	if !stringContains(result, "// This is a new documentation for MyMethod") {
		t.Errorf("Expected new documentation to be added to the method, but it was not.\nGot:\n%s", result)
	}
}

// Test adding new documentation to a method function (associated with a struct) without existing comments
func TestUpsertDocumentationToFunction_AddNewDocumentation_Method(t *testing.T) {
	content := `package main

type MyStruct struct{}

func (m *MyStruct) MyMethod() {
	fmt.Println("Hello from MyMethod!")
}
`
	tmpfile := createTempFile(t, content)
	defer os.Remove(tmpfile.Name()) // Clean up

	// Call the function to upsert documentation
	modified, err := UpsertDocumentationToFunction(tmpfile.Name(), "MyStruct", "MyMethod", "// This is a new documentation for MyMethod")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !modified {
		t.Error("Expected documentation to be added to the method, but it was not.")
	}

	// Read the modified file
	result, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}

	// Check if the new documentation was added correctly for the method
	if !stringContains(result, "// This is a new documentation for MyMethod") {
		t.Errorf("Expected new documentation to be added to the method, but it was not.\nGot:\n%s", result)
	}
}

// Test replacing documentation for a standalone function when a method and function have the same name
func TestUpsertDocumentationToFunction_ReplaceDocumentation_FunctionOnly(t *testing.T) {
	content := `package main

type MyStruct struct{}

func (m *MyStruct) MyFunction() {
	fmt.Println("Hello from MyFunction!")
}

// Old documentation
func MyFunction() {
	fmt.Println("Hello, World!")
}
`
	tmpfile := createTempFile(t, content)
	defer os.Remove(tmpfile.Name()) // Clean up

	// Call the function to upsert documentation for the standalone function only
	modified, err := UpsertDocumentationToFunction(tmpfile.Name(), "", "MyFunction", "// This is a new documentation for the function")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !modified {
		t.Error("Expected documentation to be replaced for the standalone function, but it was not.")
	}

	// Read the modified file
	result, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}

	// Check if the documentation was replaced correctly for the standalone function
	if stringContains(result, "// Old documentation") {
		t.Errorf("Expected old documentation to be replaced for the standalone function, but it was not.\nGot:\n%s", result)
	}

	if !stringContains(result, "// This is a new documentation for the function") {
		t.Errorf("Expected new documentation to be added for the standalone function, but it was not.\nGot:\n%s", result)
	}

	// Ensure the method documentation is untouched
	if stringContains(result, "// This is a new documentation for MyFunction") {
		t.Errorf("Expected no changes for the method documentation, but changes were made.\nGot:\n%s", result)
	}
}

// Test replacing documentation for a method function when a method and function have the same name
func TestUpsertDocumentationToFunction_ReplaceDocumentation_MethodOnly(t *testing.T) {
	content := `package main

type MyStruct struct{}

// Old documentation
func (m *MyStruct) MyFunction() {
	fmt.Println("Hello from MyFunction!")
}

func MyFunction() {
	fmt.Println("Hello, World!")
}
`
	tmpfile := createTempFile(t, content)
	defer os.Remove(tmpfile.Name()) // Clean up

	// Call the function to upsert documentation for the method only
	modified, err := UpsertDocumentationToFunction(tmpfile.Name(), "MyStruct", "MyFunction", "// This is a new documentation for the method")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !modified {
		t.Error("Expected documentation to be replaced for the method, but it was not.")
	}

	// Read the modified file
	result, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}

	// Check if the documentation was replaced correctly for the method
	if stringContains(result, "// Old documentation") {
		t.Errorf("Expected old documentation to be replaced for the method, but it was not.\nGot:\n%s", result)
	}

	if !stringContains(result, "// This is a new documentation for the method") {
		t.Errorf("Expected new documentation to be added for the method, but it was not.\nGot:\n%s", result)
	}

	// Ensure the standalone function documentation is untouched
	if stringContains(result, "// This is a new documentation for the function") {
		t.Errorf("Expected no changes for the standalone function documentation, but changes were made.\nGot:\n%s", result)
	}
}

// Test replacing documentation for a method function with the same name in different structs
func TestUpsertDocumentationToFunction_ReplaceDocumentation_SameMethodDifferentStructs(t *testing.T) {
	content := `package main

type MyStruct1 struct{}

// Old documentation for MyStruct1
func (m *MyStruct1) MyFunction() {
	fmt.Println("Hello from MyStruct1.MyFunction!")
}

type MyStruct2 struct{}

// Old documentation for MyStruct2
func (m *MyStruct2) MyFunction() {
	fmt.Println("Hello from MyStruct2.MyFunction!")
}
`
	tmpfile := createTempFile(t, content)
	defer os.Remove(tmpfile.Name()) // Clean up

	// Call the function to upsert documentation for the method only in MyStruct1
	modified, err := UpsertDocumentationToFunction(tmpfile.Name(), "MyStruct1", "MyFunction", "// This is new documentation for MyStruct1.MyFunction")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !modified {
		t.Error("Expected documentation to be replaced for MyStruct1.MyFunction, but it was not.")
		t.FailNow()
	}

	// Read the modified file
	result, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}

	// Check if the documentation was replaced correctly for MyStruct1.MyFunction
	if stringContains(result, "// Old documentation for MyStruct1") {
		t.Errorf("Expected old documentation to be replaced for MyStruct1.MyFunction, but it was not.\nGot:\n%s", result)
		t.FailNow()
	}

	if !stringContains(result, "// This is new documentation for MyStruct1.MyFunction") {
		t.Errorf("Expected new documentation to be added for MyStruct1.MyFunction, but it was not.\nGot:\n%s", result)
		t.FailNow()
	}

	// Ensure the documentation for MyStruct2.MyFunction remains unchanged
	if !stringContains(result, "// Old documentation for MyStruct2") {
		t.Errorf("Expected old documentation for MyStruct2.MyFunction to remain unchanged, but it was not.\nGot:\n%s", result)
		t.FailNow()
	}

	if stringContains(result, "// This is new documentation for MyStruct2.MyFunction") {
		t.Errorf("Expected no changes for MyStruct2.MyFunction, but changes were made.\nGot:\n%s", result)
		t.FailNow()
	}
}

func stringContains(s []byte, substr string) bool {
	return bytes.Contains(s, []byte(substr))
}
