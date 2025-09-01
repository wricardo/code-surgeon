package codesurgeon

import (
	"strings"
	"testing"
)

func TestStructDefinitionCapture(t *testing.T) {
	// Test Go code with a struct
	code := `
package example

// Person represents a person with name and age
type Person struct {
	Name    string   ` + "`json:\"name\"`" + `
	Age     int      ` + "`json:\"age\"`" + `
	Active  bool     ` + "`json:\"active,omitempty\"`" + `
	Tags    []string ` + "`json:\"tags\"`" + `
	private string   // private field
}
`

	// Parse the code string
	info, err := ParseString(code)
	if err != nil {
		t.Fatalf("Failed to parse code: %v", err)
	}

	// Find the Person struct
	var personStruct *Struct
	for _, mod := range info.Modules {
		for _, pkg := range mod.Packages {
			for _, strct := range pkg.Structs {
				if strct.Name == "Person" {
					personStruct = &strct
					break
				}
			}
		}
	}

	if personStruct == nil {
		t.Fatal("Person struct not found")
	}

	// Check that definition is not empty
	if personStruct.Definition == "" {
		t.Error("Struct definition is empty")
	}

	// Check that definition contains expected content
	if !strings.Contains(personStruct.Definition, "type Person struct") {
		t.Errorf("Definition doesn't contain 'type Person struct': %s", personStruct.Definition)
	}

	// Check that definition contains fields
	expectedFields := []string{"Name", "Age", "Active", "Tags", "private"}
	for _, field := range expectedFields {
		if !strings.Contains(personStruct.Definition, field) {
			t.Errorf("Definition doesn't contain field '%s': %s", field, personStruct.Definition)
		}
	}

	// Check that definition contains json tags
	if !strings.Contains(personStruct.Definition, `json:"name"`) {
		t.Errorf("Definition doesn't contain json tag for Name field: %s", personStruct.Definition)
	}

	t.Logf("Struct definition captured successfully:\n%s", personStruct.Definition)
}
