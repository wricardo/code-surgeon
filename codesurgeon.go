package codesurgeon

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/printer"
	"go/token"
	"io/fs"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/imports"
)

var STATICFS, _ = fs.Sub(FS, "api")

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
			decls, err := parseDeclarationsFromCodeFrament(fragment)
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

// MustRenderTemplate is a helper function to render a template with the given data.
// It panics if the template is invalid.
func MustRenderTemplate(tmpl string, data interface{}) string {
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

func RenderTemplate(tmpl string, data interface{}) (string, error) {
	t, err := template.New("tpl").Parse(tmpl)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
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

func parseDeclarationsFromCodeFrament(f CodeFragment) ([]ast.Decl, error) {
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
				if ts.Name.Name == getDeclName(newDecl) {
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
					// Add a default comment if none exists
					// AddDocumentationToFunc(file, existing.Name.Name, fmt.Sprintf("%s auto-generated documentation", existing.Name.Name))
					if overwrite {
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

func getDeclName(decl ast.Decl) string {
	if fd, ok := decl.(*ast.FuncDecl); ok {
		return fd.Name.Name
	}
	if gd, ok := decl.(*ast.GenDecl); ok {
		if ts, ok := gd.Specs[0].(*ast.TypeSpec); ok {
			return ts.Name.Name
		}
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

// FindFunction uses Comby to find a function in a directory.
// returns the file path and nil error if found
// returns empty string and nil error if not found
// returns empty string and error if there was an error
func FindFunction(directory, receiver, functionName string) (foundFilePath string, err error) {
	// Walk through the directory to find Go files
	err = filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Check if the file is a Go file
		if !info.IsDir() && filepath.Ext(path) == ".go" {
			found, err := findFunctionInFile(path, receiver, functionName)
			if err != nil {
				return err
			}
			if found {
				foundFilePath = path
				return filepath.SkipDir // Stop searching further as we found the function
			}
		}
		return nil
	})
	if err != nil && err != filepath.SkipDir {
		return "", err
	}
	return foundFilePath, nil
}

func findFunctionInFile(filePath, receiver, functionName string) (bool, error) {
	type matchRewrite struct {
		Match    string
		Rewrite  string
		Rule     string
		Continue bool
	}

	mrs := []matchRewrite{}

	if receiver != "" {
		// Patterns for methods associated with a struct
		mrs = append(mrs,
			// Add new documentation to a method without documentation
			matchRewrite{
				Match:   fmt.Sprintf("func (:[receiver]%s) %s(:[c])", receiver, functionName),
				Rewrite: "nop",
			},
			// Add new documentation to a method without documentation with Pointer receiver
			matchRewrite{
				Match:   fmt.Sprintf("func (:[receiver]*%s) %s(:[c])", receiver, functionName),
				Rewrite: "nop",
			},
		)
	} else {
		// Updated Match, Rewrite, and Rule
		mrs = append(mrs,
			matchRewrite{
				Match:   fmt.Sprintf("func %s(:[c])", functionName),
				Rewrite: "nop",
			},
		)
	}

	for _, mr := range mrs {
		args := []string{mr.Match, mr.Rewrite, filePath, "-json-lines", "-match-newline-at-toplevel", "-matcher", ".go"}
		if mr.Rule != "" {
			args = append(args, "-rule", mr.Rule)
		}
		cmd := exec.Command("comby", args...)

		output, err := cmd.CombinedOutput()
		outputStr := string(output)
		if err != nil {
			return false, fmt.Errorf("failed to run comby: %w, output: %s", err, string(output))
		}

		if outputStr != "" {
			expectedOutput := &struct {
				RewrittenSource string `json:"rewritten_source"`
			}{}
			if err := json.Unmarshal(output, expectedOutput); err != nil {
				return false, fmt.Errorf("failed to unmarshal comby output: %w", err)
			}
			if expectedOutput.RewrittenSource == "" {
				continue
			}
			return true, nil
		}
	}

	return false, nil
}

// writeFile writes the given content to the specified file path.
// If the file does not exist, it creates a new one. If it exists, it overwrites the file.
func writeFile(filePath, content string) error {
	// Create or open the file with write permissions.
	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Write content to the file.
	_, err = file.WriteString(content)
	if err != nil {
		return fmt.Errorf("failed to write content to file: %w", err)
	}

	return nil
}

type GoList struct {
	Dir         string       `json:"Dir"`
	ImportPath  string       `json:"ImportPath"`
	Name        string       `json:"Name"`
	Target      string       `json:"Target"`
	Root        string       `json:"Root"`
	Module      GoListModule `json:"Module"`
	Match       []string     `json:"Match"`
	Stale       bool         `json:"Stale"`
	StaleReason string       `json:"StaleReason"`
	GoFiles     []string     `json:"GoFiles"`
	Imports     []string     `json:"Imports"`
	Deps        []string     `json:"Deps"`
}

type GoListModule struct {
	Path      string `json:"Path"`
	Main      bool   `json:"Main"`
	Dir       string `json:"Dir"`
	GoMod     string `json:"GoMod"`
	GoVersion string `json:"GoVersion"`
}
