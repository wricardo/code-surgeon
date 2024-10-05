package codesurgeon

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"log"
	"testing"

	"github.com/stretchr/testify/require"
)

// Test for parseMultipleDeclarations
func TestParseMultipleDeclarations(t *testing.T) {
	code := `
	type Struct1 struct { Field1 string }
	type Struct2 struct { Field2 int }
	func Function1(param1 int) string { return "result" }
	`
	decls, err := parseDeclarationsFromCodeFrament(CodeFragment{Content: code})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(decls) != 3 {
		t.Fatalf("Expected 3 declarations, got %d", len(decls))
	}

	if getDeclName(decls[0]) != "Struct1" || getDeclName(decls[1]) != "Struct2" {
		t.Errorf("Type names do not match expected values")
	}

	if getDeclName(decls[2]) != "Function1" {
		t.Errorf("Function name does not match expected value")
	}
}

func TestReplaceOrAddDecl_dontoverwrite(t *testing.T) {
	originalCode := `
	package main
	type Struct1 struct { Field1 string }
	func Function1(param1 int) string { return "result" }
	`
	// Parse the original code
	file := parseCode(t, originalCode)

	// New declaration to replace the existing Struct1
	newDeclCode := `package main; 
	type Struct1 struct {
		Field1 string; 
		Field2 int
	}`
	newDecl := parseCode(t, newDeclCode).Decls[0]

	// Replace or add the new declaration
	upsertDeclaration(file, newDecl, false)

	// Format the modified code
	result := formatCode(t, file)

	// Verify that the new declaration was added correctly
	structs := getMapStructsFieldsType(result)
	require.Contains(t, structs, "Struct1", "Struct1 was not found")
	require.Len(t, structs["Struct1"], 1, "Expected 1 fields in Struct1")
	require.Contains(t, structs["Struct1"], "Field1", "Field1 not found in Struct1")
	require.NotContains(t, structs["Struct1"], "Field2", "Field2 found in Struct1")
	require.Equal(t, "string", structs["Struct1"]["Field1"], "Field1 should be of type string")
}

func TestReplaceOrAddDecl_overwrite(t *testing.T) {
	originalCode := `
	package main
	type Struct1 struct { Field1 string }
	func Function1(param1 int) string { return "result" }
	`
	// Parse the original code
	file := parseCode(t, originalCode)

	// New declaration to replace the existing Struct1
	newDeclCode := `package main; 
	type Struct1 struct {
		Field1 string; 
		Field2 int
	}`
	newDecl := parseCode(t, newDeclCode).Decls[0]

	// Replace or add the new declaration
	upsertDeclaration(file, newDecl, true)

	// Format the modified code
	result := formatCode(t, file)

	// Verify that the new declaration was added correctly
	structs := getMapStructsFieldsType(result)
	require.Contains(t, structs, "Struct1", "Struct1 was not found")
	require.Len(t, structs["Struct1"], 2, "Expected 2 fields in Struct1")
	require.Contains(t, structs["Struct1"], "Field1", "Field1 not found in Struct1")
	require.Contains(t, structs["Struct1"], "Field2", "Field2 not found in Struct1")
	require.Equal(t, "string", structs["Struct1"]["Field1"], "Field1 should be of type string")
	require.Equal(t, "int", structs["Struct1"]["Field2"], "Field2 should be of type int")
}

// Test for getTypeName
func TestGetTypeName(t *testing.T) {
	code := `type MyStruct struct { Field1 string }`
	decls, err := parseDeclarationsFromCodeFrament(CodeFragment{Content: code})
	if err != nil {
		t.Fatalf("Failed to parse declaration: %v", err)
	}

	typeName := getDeclName(decls[0])
	if typeName != "MyStruct" {
		t.Errorf("Expected 'MyStruct', got '%s'", typeName)
	}
}

// Test for getFuncName
func TestGetFuncName(t *testing.T) {
	code := `func MyFunction(param1 int) string { return "result" }`
	decls, err := parseDeclarationsFromCodeFrament(CodeFragment{Content: code})
	if err != nil {
		t.Fatalf("Failed to parse declaration: %v", err)
	}

	funcName := getDeclName(decls[0])
	if funcName != "MyFunction" {
		t.Errorf("Expected 'MyFunction', got '%s'", funcName)
	}
}

// Test for empty declarations in parseMultipleDeclarations
func TestParseMultipleDeclarations_EmptyCode(t *testing.T) {
	code := "" // Empty input
	decls, err := parseDeclarationsFromCodeFrament(CodeFragment{Content: code})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(decls) != 0 {
		t.Errorf("Expected no declarations, got %d", len(decls))
	}
}

// Test for invalid Go code in parseMultipleDeclarations
func TestParseMultipleDeclarations_InvalidCode(t *testing.T) {
	code := "invalid Go code"
	_, err := parseDeclarationsFromCodeFrament(CodeFragment{Content: code})
	if err == nil {
		t.Errorf("Expected an error for invalid code, but got none")
	}
}

// Test adding a new declaration without replacing an existing one with the same name
func TestReplaceOrAddDecl_AddNewDeclarationRepeatedName(t *testing.T) {
	originalCode := `
	package main
	type ExistingStruct struct { Field1 string }
	func (e *ExistingStruct) SomeMethod(arg1 string) {}
	`
	// Parse the original file
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", originalCode, parser.AllErrors)
	if err != nil {
		t.Fatalf("Failed to parse original code: %v", err)
	}

	// New declaration to add
	newDeclCode := `func SomeMethod(arg1 string) {}`
	decls, err := parseDeclarationsFromCodeFrament(CodeFragment{Content: newDeclCode})
	if err != nil {
		t.Fatalf("Failed to parse new declaration: %v", err)
	}

	// Add the new declaration
	upsertDeclaration(file, decls[0], true)

	rendered, err := renderModifiedNode(fset, file)
	if err != nil {
		panic(err)
	}

	fset = token.NewFileSet()
	file2, err := parser.ParseFile(fset, "", rendered, parser.AllErrors)

	countSomeMethod := 0
	for _, decl := range file2.Decls {
		if gd, ok := decl.(*ast.FuncDecl); ok {
			if gd.Name.Name == "SomeMethod" {
				countSomeMethod++
			}
		}
	}

	if countSomeMethod != 2 {
		t.Errorf("SomeMethod was not added correctly")
	}
}

// Test adding a new declaration without replacing an existing one
func TestReplaceOrAddDecl_AddNewDeclaration(t *testing.T) {
	originalCode := `
	package main
	type ExistingStruct struct { Field1 string }
	`
	// Parse the original file
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", originalCode, parser.AllErrors)
	if err != nil {
		t.Fatalf("Failed to parse original code: %v", err)
	}

	// New declaration to add
	newDeclCode := `type NewStruct struct { Field2 int }`
	decls, err := parseDeclarationsFromCodeFrament(CodeFragment{Content: newDeclCode})
	if err != nil {
		t.Fatalf("Failed to parse new declaration: %v", err)
	}

	// Add the new declaration
	upsertDeclaration(file, decls[0], true)

	rendered, err := renderModifiedNode(fset, file)
	if err != nil {
		panic(err)
	}

	fset = token.NewFileSet()
	file2, err := parser.ParseFile(fset, "", rendered, parser.AllErrors)
	// Check if the new declaration was added
	found := false
	for _, decl := range file2.Decls {
		if gd, ok := decl.(*ast.GenDecl); ok {
			if ts, ok := gd.Specs[0].(*ast.TypeSpec); ok {
				if ts.Name.Name == "NewStruct" {
					found = true
					break
				}
			}
		}
	}

	if !found {
		t.Errorf("NewStruct was not added correctly")
	}
}

// Test complex nested struct and method declarations
func TestReplaceOrAddDecl_ComplexDeclarations(t *testing.T) {
	originalCode := `
	package main
	type ExistingStruct struct { Field1 string }
	func (e *ExistingStruct) ExistingMethod() {}
	`
	// Parse the original file
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", originalCode, parser.AllErrors)
	if err != nil {
		t.Fatalf("Failed to parse original code: %v", err)
	}

	// New complex declaration
	newDeclCode := `
	type ComplexStruct struct {
		FieldA string
		FieldB int
		NestedStruct struct {
			SubField string
		}
	}
	func (c *ComplexStruct) ComplexMethod(param1 int) string {
		return "result"
	}`
	decls, err := parseDeclarationsFromCodeFrament(CodeFragment{Content: newDeclCode})
	if err != nil {
		t.Fatalf("Failed to parse complex declaration: %v", err)
	}

	// Add the complex declarations
	for _, decl := range decls {
		upsertDeclaration(file, decl, true)
	}

	// Check if ComplexStruct and ComplexMethod were added
	foundStruct := false
	foundMethod := false
	for _, decl := range file.Decls {
		switch decl := decl.(type) {
		case *ast.GenDecl:
			if ts, ok := decl.Specs[0].(*ast.TypeSpec); ok && ts.Name.Name == "ComplexStruct" {
				foundStruct = true
			}
		case *ast.FuncDecl:
			if decl.Name.Name == "ComplexMethod" {
				foundMethod = true
			}
		}
	}

	if !foundStruct || !foundMethod {
		t.Errorf("ComplexStruct or ComplexMethod was not added correctly")
	}
}

// Test for handling multiple changes in a single request
func TestMultipleChangesInSingleRequest(t *testing.T) {
	originalCode := `
	package main
	type ExistingStruct struct { Field1 string }
	`
	// Parse the original file
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", originalCode, parser.AllErrors)
	if err != nil {
		t.Fatalf("Failed to parse original code: %v", err)
	}

	// Multiple declarations
	newDeclCode := `
	type NewStruct1 struct { FieldA string }
	type NewStruct2 struct { FieldB int }
	`
	decls, err := parseDeclarationsFromCodeFrament(CodeFragment{Content: newDeclCode})
	if err != nil {
		t.Fatalf("Failed to parse new declarations: %v", err)
	}

	// Apply multiple changes
	for _, decl := range decls {
		upsertDeclaration(file, decl, true)
	}

	// Check if both NewStruct1 and NewStruct2 were added
	foundStruct1 := false
	foundStruct2 := false
	for _, decl := range file.Decls {
		if gd, ok := decl.(*ast.GenDecl); ok {
			if ts, ok := gd.Specs[0].(*ast.TypeSpec); ok {
				switch ts.Name.Name {
				case "NewStruct1":
					foundStruct1 = true
				case "NewStruct2":
					foundStruct2 = true
				}
			}
		}
	}

	if !foundStruct1 || !foundStruct2 {
		t.Errorf("NewStruct1 or NewStruct2 was not added correctly")
	}
}

// Test replacing an existing type when overwrite is true
func TestReplaceOrAddDecl_ReplaceExistingType(t *testing.T) {
	src := `
package main

import "fmt"

type ExistingType struct {
	Name string
}
`

	newDeclSrc := `
package main
type ExistingType struct {
	Id int
	Name string
	Age int
}
`
	file := parseCode(t, src)
	newDecl := parseCode(t, newDeclSrc).Decls[0]

	upsertDeclaration(file, newDecl, true)

	result := formatCode(t, file)
	structs := getMapStructsFieldsType(result)
	require.Len(t, structs, 1)
	require.Contains(t, structs, "ExistingType")
	require.Len(t, structs["ExistingType"], 3)
	require.Equal(t, "int", structs["ExistingType"]["Id"])
	require.Equal(t, "string", structs["ExistingType"]["Name"])
	require.Equal(t, "int", structs["ExistingType"]["Age"])
}

// Test not replacing an existing type when overwrite is false
func TestReplaceOrAddDecl_DoNotReplaceExistingType(t *testing.T) {
	src := `
package main

import "fmt"

type ExistingType struct {
	Name string
}
`

	newDeclSrc := `
package main
type ExistingType struct {
	Id int
	Name string
	Age int
}
`
	file := parseCode(t, src)
	newDecl := parseCode(t, newDeclSrc).Decls[0]

	upsertDeclaration(file, newDecl, false)

	result := formatCode(t, file)
	structs := getMapStructsFieldsType(result)
	require.Len(t, structs, 1)
	require.Contains(t, structs, "ExistingType")
	require.Len(t, structs["ExistingType"], 1)
	require.Equal(t, "string", structs["ExistingType"]["Name"])
}

// Test appending a new function when it does not exist
func TestReplaceOrAddDecl_AddNewFunction(t *testing.T) {
	src := `
package main

import "fmt"

func ExistingFunction() {}
`

	newDeclSrc := `
package main
func NewFunction() {
	fmt.Println("Hello from NewFunction")
}
`
	file := parseCode(t, src)
	newDecl := parseCode(t, newDeclSrc).Decls[0]

	upsertDeclaration(file, newDecl, false)

	result := formatCode(t, file)
	require.Contains(t, result, "func NewFunction()")
}

// Test replacing an existing function when overwrite is true
func TestReplaceOrAddDecl_ReplaceExistingFunction(t *testing.T) {
	src := `
package main

import "fmt"

func ExistingFunction() {
	fmt.Println("Original Function")
}
`

	newDeclSrc := `
package main
func ExistingFunction() {
	fmt.Println("Replaced Function")
}
`
	file := parseCode(t, src)
	newDecl := parseCode(t, newDeclSrc).Decls[0]

	upsertDeclaration(file, newDecl, true)

	result := formatCode(t, file)
	require.Contains(t, result, "fmt.Println(\"Replaced Function\")")
	require.NotContains(t, result, "fmt.Println(\"Original Function\")")
}

// Test not replacing an existing function when overwrite is false
func TestReplaceOrAddDecl_DoNotReplaceExistingFunction(t *testing.T) {
	src := `
package main

import "fmt"

func ExistingFunction() {
	fmt.Println("Original Function")
}
`

	newDeclSrc := `
package main
func ExistingFunction() {
	fmt.Println("Replaced Function")
}
`
	file := parseCode(t, src)
	newDecl := parseCode(t, newDeclSrc).Decls[0]

	upsertDeclaration(file, newDecl, false)

	result := formatCode(t, file)
	require.Contains(t, result, "fmt.Println(\"Original Function\")")
	require.NotContains(t, result, "fmt.Println(\"Replaced Function\")")
}

// Test appending a type when there are multiple types
func TestReplaceOrAddDecl_AddTypeWithMultipleExistingTypes(t *testing.T) {
	src := `
package main

import "fmt"

type ExistingType1 struct {
	Name string
}

type ExistingType2 struct {
	Name string
}
`

	newDeclSrc := `
package main
type NewType struct {
	Id int
	Name string
}
`
	file := parseCode(t, src)
	newDecl := parseCode(t, newDeclSrc).Decls[0]

	upsertDeclaration(file, newDecl, false)

	result := formatCode(t, file)
	structs := getMapStructsFieldsType(result)
	require.Len(t, structs, 3)
	require.Contains(t, structs, "NewType")
	require.Len(t, structs["NewType"], 2)
	require.Equal(t, "int", structs["NewType"]["Id"])
	require.Equal(t, "string", structs["NewType"]["Name"])
}

func TestReplaceOrAddDecl_SkipImportDeclarations(t *testing.T) {
	src := `
package main

import "fmt"

type ExistingType struct {
	Name string
}
`

	newDeclSrc := `
package main

import "strings"

func NewFunction() {
	strings.ToLower("Hello")
}
type NewType struct {
	Id int
}
`
	file := parseCode(t, src)
	newDecl := parseCode(t, newDeclSrc)

	for _, decl := range newDecl.Decls {
		upsertDeclaration(file, decl, false)
	}

	result := formatCode(t, file)
	require.NotContains(t, result, `import "strings"`)
}

// Helper function to parse code into an AST file
func parseCode(t *testing.T, src string) *ast.File {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", src, parser.AllErrors)
	if err != nil {
		t.Fatalf("Failed to parse code: %v", err)
	}
	return file
}

// returns a map with struct name as key and another map as value with field name as key and field type as value
func getMapStructsFieldsType(content string) map[string]map[string]string {
	structs := make(map[string]map[string]string)
	parsed, err := ParseString(content)
	if err != nil {
		log.Fatal(err)
	}

	for _, s := range parsed.Packages[0].Structs {
		structs[s.Name] = map[string]string{}
		for _, f := range s.Fields {
			structs[s.Name][f.Name] = f.Type
		}
	}
	return structs
}

func formatCode(t *testing.T, file *ast.File) string {
	var buf bytes.Buffer

	err := printer.Fprint(&buf, token.NewFileSet(), file)
	if err != nil {
		t.Fatalf("Failed to format code: %v", err)
	}
	return buf.String()
}
