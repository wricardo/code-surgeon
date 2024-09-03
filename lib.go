package codesurgeon

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/printer"
	"go/token"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/wricardo/structparser"
	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/imports"
)

type (
	FileChange struct {
		PackageName string
		File        string
		Fragments   []CodeFragment
	}

	CodeFragment struct {
		Content   string
		Overwrite bool
	}
)

func ApplyFileChanges(changes []FileChange) error {
	// Group changes by file
	implementationsMap := make(map[string][]CodeFragment)
	for _, change := range changes {
		implementationsMap[change.File] = change.Fragments
		// mkdir -p
		if err := os.MkdirAll(filepath.Dir(change.File), 0755); err != nil {
			return fmt.Errorf("Failed to create directory: %v", err)
		}
		// if file does not exist, create it
		if _, err := os.Stat(change.File); os.IsNotExist(err) {
			if f, err := os.Create(change.File); err != nil {
				return fmt.Errorf("Failed to create file: %v", err)
			} else {
				f.Write([]byte("package " + change.PackageName + "\n"))
				defer f.Close()
			}
		}
	}

	return InsertCodeFragments(implementationsMap)
}

func InsertCodeFragments(implementationsMap map[string][]CodeFragment) error {
	// Apply changes to each file
	for file, fragments := range implementationsMap {
		fset := token.NewFileSet()
		// if file does not exist, create it
		if _, err := os.Stat(file); os.IsNotExist(err) {
			if f, err := os.Create(file); err != nil {
				return fmt.Errorf("Failed to create file: %v", err)
			} else {
				f.Write([]byte("package main\n"))
				defer f.Close()
			}
		}
		node, err := parser.ParseFile(fset, file, nil, parser.AllErrors|parser.ParseComments)

		if err != nil {
			fmt.Printf("Failed to parse file: %v\n", err)
			continue
		}

		// // Process each change separately
		for _, fragment := range fragments {
			decls, err := parseDeclarations(fragment)
			if err != nil {
				fmt.Printf("Failed to parse change: %v\n", err)
				continue
			}

			for _, decl := range decls {
				upsertDeclaration(node, decl, fragment.Overwrite)
			}
		}

		err = writeChangesToFile(file, fset, node)
		if err != nil {
			fmt.Printf("Failed to write modified file: %v\n", err)
			return err
		}
	}
	return nil
}

// RenderTemplate is a helper function to render a template with the given data.
// It panics if the template is invalid.
func RenderTemplate(tmpl string, data interface{}) string {
	t, err := template.New("tpl").Parse(tmpl)
	if err != nil {
		panic(err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		panic(err)
	}

	return buf.String()
}

// FormatCodeAndFixImports applies gofmt and goimports to the modified files.
func FormatCodeAndFixImports(filePath string) error {
	// Read the file content
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}

	// Apply goimports to fix and organize imports
	processedContent, err := imports.Process(filePath, content, nil)
	if err != nil {
		return err
	}

	// Apply gofmt to format the code
	formattedContent, err := format.Source(processedContent)
	if err != nil {
		return err
	}

	// Write the formatted and import-fixed content back to the file
	if err := ioutil.WriteFile(filePath, formattedContent, 0644); err != nil {
		return err
	}

	return nil
}

// Parse multiple declarations from a string
func parseDeclarations(f CodeFragment) ([]ast.Decl, error) {
	code := strings.TrimSpace(f.Content)
	// check if no package is defined
	if !strings.HasPrefix(code, "package") {
		code = "package main\n\n" + code

	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", code, parser.AllErrors|parser.ParseComments)
	if err != nil {
		return nil, err
	}

	return file.Decls, nil
}

func upsertDeclaration(file *ast.File, newDecl ast.Decl, overwrite bool) {
	shouldAppend := true

	astutil.Apply(file, func(c *astutil.Cursor) bool {
		decl, ok := c.Node().(ast.Decl)
		if !ok {
			return true
		}

		switch existing := decl.(type) {
		case *ast.GenDecl:
			// Handle import declarations
			if existing.Tok == token.IMPORT {
				// If the new declaration is also an import, merge it
				if newGenDecl, ok := newDecl.(*ast.GenDecl); ok && newGenDecl.Tok == token.IMPORT {
					existing.Specs = append(existing.Specs, newGenDecl.Specs...)
					shouldAppend = false
					return false
				}
				return true
			}

			// Handle type declarations
			if ts, ok := existing.Specs[0].(*ast.TypeSpec); ok {
				if ts.Name.Name == getTypeName(newDecl) {
					if overwrite {
						c.Replace(newDecl)
					}
					shouldAppend = false
					return false
				}
			}

		case *ast.FuncDecl:
			// Handle function declarations
			newFunc, ok := newDecl.(*ast.FuncDecl)
			if !ok {
				return true
			}

			if existing.Name.Name == newFunc.Name.Name {
				// fmt.Printf("newDecl\n%s", spew.Sdump(newDecl))   // TODO: wallace debug
				// fmt.Printf("existing\n%s", spew.Sdump(existing)) // TODO: wallace debug
				// fmt.Printf("newFunc\n%s", spew.Sdump(newFunc))   // TODO: wallace debug

				existingRecv := getReceiverType(existing)
				newRecv := getReceiverType(newFunc)
				if existingRecv == newRecv {
					if overwrite {
						existing.Doc = newFunc.Doc
						c.Replace(newDecl)
					}
					shouldAppend = false
					return false
				}
			}
		}

		return true
	}, nil)

	// If the new declaration is an import and should be appended
	if shouldAppend {
		if newGenDecl, ok := newDecl.(*ast.GenDecl); ok && newGenDecl.Tok == token.IMPORT {
			// Try to insert with existing imports if any
			for _, decl := range file.Decls {
				if existingImport, ok := decl.(*ast.GenDecl); ok && existingImport.Tok == token.IMPORT {
					existingImport.Specs = append(existingImport.Specs, newGenDecl.Specs...)
					return
				}
			}
		}
		file.Decls = append(file.Decls, newDecl)
	}
}

func getReceiverType(funcDecl *ast.FuncDecl) string {
	if funcDecl.Recv != nil && len(funcDecl.Recv.List) > 0 {
		if starExpr, ok := funcDecl.Recv.List[0].Type.(*ast.StarExpr); ok {
			if ident, ok := starExpr.X.(*ast.Ident); ok {
				return ident.Name
			}
		} else if ident, ok := funcDecl.Recv.List[0].Type.(*ast.Ident); ok {
			return ident.Name
		}
	}
	return ""
}
func getTypeName(decl ast.Decl) string {
	if gd, ok := decl.(*ast.GenDecl); ok {
		if ts, ok := gd.Specs[0].(*ast.TypeSpec); ok {
			return ts.Name.Name
		}
	}
	return ""
}

func getFuncName(decl ast.Decl) string {
	if fd, ok := decl.(*ast.FuncDecl); ok {
		return fd.Name.Name
	}
	return ""
}

func writeChangesToFile(filePath string, fset *token.FileSet, node ast.Node) error {
	var buf bytes.Buffer
	if err := printer.Fprint(&buf, fset, node); err != nil {
		return err
	}

	// Apply gofmt to the generated code
	formattedCode, err := format.Source(buf.Bytes())
	if err != nil {
		return err
	}

	// Write the formatted code to the file
	return ioutil.WriteFile(filePath, formattedCode, 0644)
}

func renderModifiedNode(fset *token.FileSet, node ast.Node) (string, error) {
	var buf bytes.Buffer
	if err := printer.Fprint(&buf, fset, node); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func GenerateCypher(output structparser.Output) ([]string, error) {

	var cypherQueries []string

	// // Prepare inline data for batch operations
	// var structsData []string
	// var fieldsData []string
	// var methodsData []string
	// var functionsData []string
	// var paramsData []string
	// var returnsData []string

	// for _, p := range output.Packages {
	// 	// Collect data for structs
	// 	for _, s := range p.Structs {
	// 		structsData = append(structsData, fmt.Sprintf("{name: '%s'}", s.Name))

	// 		// Collect data for fields
	// 		for _, f := range s.Fields {
	// 			fieldsData = append(fieldsData, fmt.Sprintf(
	// 				"{structName: '%s', name: '%s', type: '%s', private: %t, pointer: %t, slice: %t, comment: '%s'}",
	// 				s.Name, f.Name, f.Type, f.Private, f.Pointer, f.Slice, f.Comment,
	// 			))
	// 		}

	// 		// Collect data for methods
	// 		for _, m := range s.Methods {
	// 			methodsData = append(methodsData, fmt.Sprintf(
	// 				"{structName: '%s', name: '%s', receiver: '%s', signature: '%s'}",
	// 				s.Name, m.Name, m.Receiver, m.Signature,
	// 			))
	// 		}
	// 	}

	// 	// Collect data for functions
	// 	for _, f := range p.Functions {
	// 		functionsData = append(functionsData, fmt.Sprintf("{name: '%s', signature: '%s'}", f.Name, f.Signature))

	// 		// Collect data for parameters
	// 		for _, p := range f.Params {
	// 			paramsData = append(paramsData, fmt.Sprintf(
	// 				"{functionName: '%s', name: '%s', type: '%s'}",
	// 				f.Name, p.Name, p.Type,
	// 			))
	// 		}

	// 		// Collect data for return values
	// 		for _, r := range f.Returns {
	// 			returnsData = append(returnsData, fmt.Sprintf(
	// 				"{functionName: '%s', name: '%s', type: '%s'}",
	// 				f.Name, r.Name, r.Type,
	// 			))
	// 		}
	// 	}
	// }

	// // Batch creation of Struct nodes
	// cypherQueries = append(cypherQueries, fmt.Sprintf(`
	// UNWIND [%s] AS structData
	// CREATE (:Struct {name: structData.name});
	// `, strings.Join(structsData, ", ")))

	// // Batch creation of Field nodes and relationships
	// cypherQueries = append(cypherQueries, fmt.Sprintf(`
	// UNWIND [%s] AS fieldData
	// MATCH (s:Struct {name: fieldData.structName})
	// CREATE (f:Field {name: fieldData.name, type: fieldData.type, private: fieldData.private, pointer: fieldData.pointer, slice: fieldData.slice, comment: fieldData.comment})
	// CREATE (s)-[:HAS_FIELD]->(f);
	// `, strings.Join(fieldsData, ", ")))

	// // Batch creation of Method nodes and relationships
	// cypherQueries = append(cypherQueries, fmt.Sprintf(`
	// UNWIND [%s] AS methodData
	// MATCH (s:Struct {name: methodData.structName})
	// CREATE (m:Method {name: methodData.name, receiver: methodData.receiver, signature: methodData.signature})
	// CREATE (s)-[:HAS_METHOD]->(m);
	// `, strings.Join(methodsData, ", ")))

	// // Batch creation of Function nodes
	// cypherQueries = append(cypherQueries, fmt.Sprintf(`
	// UNWIND [%s] AS functionData
	// CREATE (:Function {name: functionData.name, signature: functionData.signature});
	// `, strings.Join(functionsData, ", ")))

	// // Batch creation of Parameter nodes and relationships
	// cypherQueries = append(cypherQueries, fmt.Sprintf(`
	// UNWIND [%s] AS paramData
	// MATCH (f:Function {name: paramData.functionName})
	// CREATE (p:Param {name: paramData.name, type: paramData.type})
	// CREATE (f)-[:HAS_PARAM]->(p);
	// `, strings.Join(paramsData, ", ")))

	// // Batch creation of Return nodes and relationships
	// cypherQueries = append(cypherQueries, fmt.Sprintf(`
	// UNWIND [%s] AS returnData
	// MATCH (f:Function {name: returnData.functionName})
	// CREATE (r:Return {name: returnData.name, type: returnData.type})
	// CREATE (f)-[:HAS_RETURN]->(r);
	// `, strings.Join(returnsData, ", ")))

	return cypherQueries, nil
}

func EnsureFileExists(filename string, packageName string) error {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		if f, err := os.Create(filename); err != nil {
			return fmt.Errorf("Failed to create file: %v", err)
		} else {
			f.Write([]byte("package " + packageName + "\n"))
			defer f.Close()
		}
	}
	return nil
}

func ToSnakeCase(s string) string {
	var result []rune
	for i, c := range s {
		if i > 0 && 'A' <= c && c <= 'Z' {
			result = append(result, '_')
		}
		result = append(result, c)
	}
	return strings.ToLower(string(result))
}

func FormatWithGoImports(filename string) error {
	// Check if the file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", filename)
	}

	// Prepare the command to run `goimports`
	cmd := exec.Command("goimports", "-w", filename) // "-w" flag to write result to the file

	// Capture the output and error
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	// Run the command
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run goimports: %v, stderr: %s", err, stderr.String())
	}

	return nil
}
