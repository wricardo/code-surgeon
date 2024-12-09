package codesurgeon

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"go/ast"
	"go/doc"
	"go/format"
	"go/parser"
	"go/printer"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// FS embeds OpenAPI and proto files for the codesurgeon package.
//
//go:embed api/codesurgeon.openapi.json
//go:embed api/codesurgeon.proto
var FS embed.FS

// ParsedInfo holds parsed information about Go packages.
type ParsedInfo struct {
	Packages  []Package `json:"packages"`
	Directory string    `json:"directory"` // if information was parsed from a directory. It's either a directory or a file
	File      string    `json:"file"`      // if information was parsed from a single file. It's either a directory or a file
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

// ParseFile parses a Go file or directory and returns the parsed information.
func ParseFile(fileOrDirectory string) (*ParsedInfo, error) {
	return ParseDirectory(fileOrDirectory)
}

// ParseDirectory parses a directory containing Go files and returns the parsed information.
func ParseDirectory(fileOrDirectory string) (*ParsedInfo, error) {
	return ParseDirectoryWithFilter(fileOrDirectory, nil)
}

// ParseString parses Go source code provided as a string and returns the parsed information.
func ParseString(fileContent string) (*ParsedInfo, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", fileContent, parser.ParseComments|parser.AllErrors|parser.DeclarationErrors)
	if err != nil {
		return nil, err
	}

	packages := map[string]*ast.Package{
		"": {
			Name:  file.Name.Name,
			Files: map[string]*ast.File{"": file},
		},
	}

	return extractParsedInfo(packages)
}

func augment(m map[string]interface{}, n map[string]interface{}) map[string]interface{} {
	copy_ := make(map[string]interface{}, len(m)+len(n))
	for k, v := range m {
		copy_[k] = v
	}
	for k, v := range n {
		copy_[k] = v
	}
	return copy_
}

// ParseDirectoryRecursive parses a directory recursively and returns the parsed information.
func ParseDirectoryRecursive(path string) ([]*ParsedInfo, error) {
	var results []*ParsedInfo

	err := filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() { // Process only directories
			if strings.Contains(p, ".git") {
				return nil
			}
			parsed, err := ParseDirectoryWithFilter(p, func(info fs.FileInfo) bool {
				return true
			}) // Assuming this function parses a single file
			if err != nil {
				return err
			}
			results = append(results, parsed)
		}
		return nil
	})
	return results, err
}

// ParseDirectoryWithFilter parses a directory with an optional filter function to include specific files.
func ParseDirectoryWithFilter(fileOrDirectory string, filter func(fs.FileInfo) bool) (*ParsedInfo, error) {
	fi, err := os.Stat(fileOrDirectory)
	if err != nil {
		return nil, err
	}

	var packages map[string]*ast.Package
	fset := token.NewFileSet()

	isDir := true
	switch mode := fi.Mode(); {
	case mode.IsDir():
		packages, err = parser.ParseDir(fset, fileOrDirectory, filter, parser.ParseComments|parser.AllErrors|parser.DeclarationErrors)
		if err != nil {
			return nil, err
		}
	case mode.IsRegular():
		isDir = false
		file, err := parser.ParseFile(fset, fileOrDirectory, nil, parser.ParseComments|parser.AllErrors|parser.DeclarationErrors)
		if err != nil {
			return nil, err
		}
		packages = map[string]*ast.Package{
			fileOrDirectory: {
				Name:  file.Name.Name,
				Files: map[string]*ast.File{fileOrDirectory: file},
			},
		}
	}

	parsedInfo, err := extractParsedInfo(packages)
	if err != nil {
		return nil, err
	}
	if isDir {
		parsedInfo.Directory = fileOrDirectory
		if abs, err := filepath.Abs(fileOrDirectory); err == nil {
			parsedInfo.Directory = abs
		}
	} else {
		parsedInfo.File = fileOrDirectory
		if abs, err := filepath.Abs(fileOrDirectory); err == nil {
			parsedInfo.File = abs
		}
	}
	return parsedInfo, nil
}

// extractStructs extracts structs from the provided documentation package.
func extractStructs(docPkg *doc.Package) ([]Struct, error) {
	var structs []Struct
	for _, t := range docPkg.Types {
		if t == nil || t.Decl == nil {
			return nil, errors.New("t or t.Decl is nil")
		}

		for _, spec := range t.Decl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				return nil, errors.New("not a *ast.TypeSpec")
			}

			structType, ok := typeSpec.Type.(*ast.StructType)
			if ok {
				parsedStruct := Struct{
					Name:    t.Name,
					Fields:  make([]Field, 0, len(structType.Fields.List)),
					Docs:    getDocsForStruct(t.Doc),
					Methods: make([]Method, 0),
				}

				for _, fvalue := range structType.Fields.List {
					name := ""
					if len(fvalue.Names) > 0 {
						name = fvalue.Names[0].Obj.Name
					}

					field := Field{
						Name:    name,
						Type:    "",
						Tag:     "",
						Pointer: false,
						Slice:   false,
					}

					if len(field.Name) > 0 {
						field.Private = strings.ToLower(string(field.Name[0])) == string(field.Name[0])
					}

					if fvalue.Doc != nil {
						field.Docs = getDocsForFieldAst(fvalue.Doc)
					}

					if fvalue.Comment != nil {
						field.Comment = cleanDocText(fvalue.Comment.Text())
					}

					if fvalue.Tag != nil {
						field.Tag = strings.Trim(fvalue.Tag.Value, "`")
					}

					var err error
					field.Type, field.Pointer, field.Slice, err = getType(fvalue.Type)
					if err != nil {
						return nil, err
					}

					parsedStruct.Fields = append(parsedStruct.Fields, field)
				}

				structs = append(structs, parsedStruct)
			}
		}
	}
	return structs, nil
}

// extractInterfaces extracts interfaces from the provided documentation package.
func extractInterfaces(docPkg *doc.Package) ([]Interface, error) {
	var interfaces []Interface
	for _, t := range docPkg.Types {
		if t == nil || t.Decl == nil {
			return nil, errors.New("t or t.Decl is nil")
		}

		for _, spec := range t.Decl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				return nil, errors.New("not a *ast.TypeSpec")
			}

			interfaceType, ok := typeSpec.Type.(*ast.InterfaceType)
			if ok {
				parsedInterface := Interface{
					Name:    t.Name,
					Methods: make([]Method, 0),
					Docs:    getDocsForStruct(t.Doc),
				}

				for _, m := range interfaceType.Methods.List {
					if funcType, ok := m.Type.(*ast.FuncType); ok {
						method := Method{
							Name:    m.Names[0].Name,
							Params:  extractParams(funcType.Params),
							Returns: extractParams(funcType.Results),
							Docs:    getDocsForFieldAst(m.Doc),
							Signature: fmt.Sprintf("%s(%s) (%s)", m.Names[0].Name,
								formatParams(funcType.Params), formatParams(funcType.Results)),
						}
						parsedInterface.Methods = append(parsedInterface.Methods, method)
					}
				}

				interfaces = append(interfaces, parsedInterface)
			}
		}
	}
	return interfaces, nil
}

// extractMethods extracts methods associated with the provided structs.
func extractMethods(docPkg *doc.Package, structs []Struct) error {
	for _, t := range docPkg.Types {
		if t == nil || t.Decl == nil {
			return errors.New("t or t.Decl is nil")
		}

		for _, m := range t.Methods {
			funcDecl := m.Decl
			receiverType, _, _, _ := getType(funcDecl.Recv.List[0].Type)

			method := Method{
				Name:     funcDecl.Name.Name,
				Receiver: receiverType,
				Docs:     getDocsForField([]string{m.Doc}),
			}

			// Parse function parameters
			params := []Param{}
			for _, param := range funcDecl.Type.Params.List {
				paramType, _, _, err := getType(param.Type)
				if err != nil {
					return err
				}

				for _, name := range param.Names {
					params = append(params, Param{
						Name: name.Name,
						Type: paramType,
					})
				}
			}
			method.Params = params

			// Parse return types
			returns := []Param{}
			if funcDecl.Type.Results != nil {
				for _, result := range funcDecl.Type.Results.List {
					returnType, _, _, err := getType(result.Type)
					if err != nil {
						return err
					}

					if len(result.Names) > 0 {
						for _, name := range result.Names {
							returns = append(returns, Param{
								Name: name.Name,
								Type: returnType,
							})
						}
					} else {
						returns = append(returns, Param{
							Name: "",
							Type: returnType,
						})
					}
				}
			}
			method.Returns = returns

			// Extract the function body as a string
			var bodyBuf bytes.Buffer
			if funcDecl.Body != nil {
				err := format.Node(&bodyBuf, token.NewFileSet(), funcDecl.Body)
				if err != nil {
					return err
				}
				method.Body = bodyBuf.String()
			}

			// Construct the full method signature for easy comparison
			paramStrings := []string{}
			for _, param := range method.Params {
				if param.Name != "" {
					paramStrings = append(paramStrings, param.Name+" "+param.Type)
				} else {
					paramStrings = append(paramStrings, param.Type)
				}
			}

			returnStrings := []string{}
			for _, ret := range method.Returns {
				if ret.Name != "" {
					returnStrings = append(returnStrings, ret.Name+" "+ret.Type)
				} else {
					returnStrings = append(returnStrings, ret.Type)
				}
			}

			method.Signature = fmt.Sprintf("%s(%s) (%s)",
				method.Name,
				strings.Join(paramStrings, ", "),
				strings.Join(returnStrings, ", "),
			)

			// Find and update the corresponding struct
			for i := range structs {
				if structs[i].Name == strings.TrimPrefix(receiverType, "*") {
					structs[i].Methods = append(structs[i].Methods, method)
					break
				}
			}
		}
	}
	return nil
}

// extractFunctions extracts functions from the provided documentation package.
func extractFunctions(docPkg *doc.Package) ([]Function, error) {
	var functions []Function
	for _, t := range docPkg.Funcs {
		if t == nil || t.Decl == nil {
			return nil, errors.New("t or t.Decl is nil")
		}

		funcDecl := t.Decl
		function := Function{
			Name: t.Name,
			Docs: getDocsForField([]string{t.Doc}),
		}

		// Parse function parameters
		params := []Param{}
		for _, param := range funcDecl.Type.Params.List {
			paramType, _, _, err := getType(param.Type)
			if err != nil {
				return nil, err
			}

			for _, name := range param.Names {
				params = append(params, Param{
					Name: name.Name,
					Type: paramType,
				})
			}
		}
		function.Params = params

		// Parse return types
		returns := []Param{}
		if funcDecl.Type.Results != nil {
			for _, result := range funcDecl.Type.Results.List {
				returnType, _, _, err := getType(result.Type)
				if err != nil {
					return nil, err
				}

				if len(result.Names) > 0 {
					for _, name := range result.Names {
						returns = append(returns, Param{
							Name: name.Name,
							Type: returnType,
						})
					}
				} else {
					returns = append(returns, Param{
						Name: "",
						Type: returnType,
					})
				}
			}
		}
		function.Returns = returns

		// Extract the function body as a string
		var bodyBuf bytes.Buffer
		if funcDecl.Body != nil {
			err := format.Node(&bodyBuf, token.NewFileSet(), funcDecl.Body)
			if err != nil {
				return nil, err
			}
			function.Body = bodyBuf.String()
		}

		// Construct the full function signature for easy comparison
		paramStrings := []string{}
		for _, param := range function.Params {
			if param.Name != "" {
				paramStrings = append(paramStrings, param.Name+" "+param.Type)
			} else {
				paramStrings = append(paramStrings, param.Type)
			}
		}

		returnStrings := []string{}
		for _, ret := range function.Returns {
			if ret.Name != "" {
				returnStrings = append(returnStrings, ret.Name+" "+ret.Type)
			} else {
				returnStrings = append(returnStrings, ret.Type)
			}
		}

		function.Signature = fmt.Sprintf("%s(%s) (%s)",
			function.Name,
			strings.Join(paramStrings, ", "),
			strings.Join(returnStrings, ", "),
		)

		functions = append(functions, function)
	}
	return functions, nil
}

// extractImports extracts unique imports from the provided package.
func extractImports(pkg *ast.Package) ([]string, error) {
	importSet := make(map[string]struct{})
	for _, file := range pkg.Files {
		for _, importSpec := range file.Imports {
			importPath := strings.Trim(importSpec.Path.Value, "\"")
			importSet[importPath] = struct{}{}
		}
	}

	var imports []string
	for imp := range importSet {
		imports = append(imports, imp)
	}
	return imports, nil
}

// extractConstantsVariables extracts constants and variables from the provided package.
func extractConstantsVariables(pkg *ast.Package) ([]Constant, []Variable, error) {
	var constants []Constant
	var variables []Variable

	for _, file := range pkg.Files {
		for _, decl := range file.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}

			switch genDecl.Tok {
			case token.CONST:
				for _, spec := range genDecl.Specs {
					valSpec, ok := spec.(*ast.ValueSpec)
					if !ok {
						continue
					}
					for i, name := range valSpec.Names {
						constant := Constant{
							Name:  name.Name,
							Value: "",
							Docs:  getDocsForFieldAst(valSpec.Doc),
						}
						if i < len(valSpec.Values) {
							constant.Value = exprToString(valSpec.Values[i])
						}
						constants = append(constants, constant)
					}
				}
			case token.VAR:
				for _, spec := range genDecl.Specs {
					valSpec, ok := spec.(*ast.ValueSpec)
					if !ok {
						continue
					}
					for _, name := range valSpec.Names {
						varType := ""
						if valSpec.Type != nil {
							varType, _, _, _ = getType(valSpec.Type)
						}
						variable := Variable{
							Name: name.Name,
							Type: varType,
							Docs: getDocsForFieldAst(valSpec.Doc),
						}
						variables = append(variables, variable)
					}
				}
			}
		}
	}

	return constants, variables, nil
}

func extractParams(fieldList *ast.FieldList) []Param {
	if fieldList == nil {
		return nil
	}
	params := make([]Param, 0, len(fieldList.List))
	for _, field := range fieldList.List {
		paramType, _, _, err := getType(field.Type)
		if err != nil {
			continue // Or handle the error properly
		}
		for _, name := range field.Names {
			params = append(params, Param{Name: name.Name, Type: paramType})
		}
		// Handle anonymous parameters (e.g., func(int, string) without names)
		if len(field.Names) == 0 {
			params = append(params, Param{Name: "", Type: paramType})
		}
	}
	return params
}

func formatParams(fields *ast.FieldList) string {
	if fields == nil {
		return ""
	}
	paramStrings := []string{}
	for _, param := range extractParams(fields) {
		if param.Name != "" {
			paramStrings = append(paramStrings, fmt.Sprintf("%s %s", param.Name, param.Type))
		} else {
			paramStrings = append(paramStrings, param.Type)
		}
	}
	return strings.Join(paramStrings, ", ")
}

func exprToString(expr ast.Expr) string {
	var buf bytes.Buffer
	err := printer.Fprint(&buf, token.NewFileSet(), expr)
	if err != nil {
		return "<err>"
	}
	return buf.String()
}

func exprToStringoriginal(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.BasicLit:
		return e.Value
	case *ast.Ident:
		return e.Name
	case *ast.BinaryExpr:
		return exprToString(e.X) + " " + e.Op.String() + " " + exprToString(e.Y)
	case *ast.CallExpr:
		return fmt.Sprintf("%s(%s)", exprToString(e.Fun), exprToString(e.Args[0]))
		// Add more cases as needed
	}
	return ""
}

func getDocsForStruct(doc string) []string {
	trimmed := strings.Trim(doc, "\n")
	if trimmed == "" {
		return []string{}
	}
	tmp := strings.Split(trimmed, "\n")

	docs := make([]string, 0, len(tmp))
	for _, v := range tmp {
		clean := cleanDocText(v)
		if clean == "" {
			continue
		}
		docs = append(docs, clean)
	}
	return docs
}

func getDocsForFieldAst(cg *ast.CommentGroup) []string {
	if cg == nil {
		return []string{}
	}
	docs := make([]string, 0, len(cg.List))
	for _, v := range cg.List {
		docs = append(docs, cleanDocText(v.Text))
	}
	return docs
}

func getDocsForField(list []string) []string {
	docs := make([]string, 0, len(list))
	for _, v := range list {
		clean := cleanDocText(v)
		if clean == "" {
			continue
		}
		docs = append(docs, clean)
	}
	return docs
}

func cleanDocText(doc string) string {
	reverseString := func(s string) string {
		runes := []rune(s)
		for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
			runes[i], runes[j] = runes[j], runes[i]
		}
		return string(runes)
	}

	if strings.HasPrefix(doc, "// ") {
		doc = strings.Replace(doc, "// ", "", 1)
	} else if strings.HasPrefix(doc, "//") {
		doc = strings.Replace(doc, "//", "", 1)
	} else if strings.HasPrefix(doc, "/*") {
		doc = strings.Replace(doc, "/*", "", 1)
	}
	if strings.HasSuffix(doc, "*/") {
		doc = reverseString(strings.Replace(reverseString(doc), "/*", "", 1))
	}
	return strings.Trim(strings.Trim(doc, " "), "\n")
}

// nolint: unusedparams
func justTypeString(a string, b, c bool, err error) string {
	void(a, b, c, err)
	return a
}

func void(_ ...interface{}) {}

func fieldListToString(fl *ast.FieldList) (string, error) {
	if fl == nil {
		return "", nil
	}
	parts := []string{}
	for _, field := range fl.List {
		typ, _, _, err := getType(field.Type)
		if err != nil {
			return "", err
		}
		if len(field.Names) == 0 {
			parts = append(parts, typ)
		} else {
			names := []string{}
			for _, name := range field.Names {
				names = append(names, name.Name)
			}
			parts = append(parts, fmt.Sprintf("%s %s", strings.Join(names, ", "), typ))
		}
	}
	return strings.Join(parts, ", "), nil
}

// TODO: solve for: unknown type for &ast.InterfaceType{Interface:552, Methods:(*ast.FieldList)(0x14000112a50), Incomplete:false}
// getType returns the type as a string, and two booleans indicating
// whether the type is a pointer and whether it's a slice, along with any error encountered.
func getType(expr ast.Expr) (string, bool, bool, error) {
	var typeStr string
	var isPointer, isSlice bool
	switch t := expr.(type) {
	case *ast.Ident:
		typeStr = t.Name
	case *ast.SelectorExpr:
		x, _, _, err := getType(t.X)
		if err != nil {
			return "", false, false, err
		}
		typeStr = x + "." + t.Sel.Name
	case *ast.StarExpr:
		isPointer = true
		innerType, p, s, err := getType(t.X)
		if err != nil {
			return "", false, false, err
		}
		// Propagate pointer and slice flags
		isPointer = isPointer || p
		isSlice = isSlice || s
		typeStr = "*" + innerType
	case *ast.IndexExpr:
		// Handle single type parameter (legacy support)
		x, _, _, err := getType(t.X)
		if err != nil {
			return "", false, false, err
		}
		index, _, _, err := getType(t.Index)
		if err != nil {
			return "", false, false, err
		}
		typeStr = fmt.Sprintf("%s[%s]", x, index)
	case *ast.IndexListExpr:
		// Handle multiple type parameters (generics)
		x, _, _, err := getType(t.X)
		if err != nil {
			return "", false, false, err
		}
		indices := []string{}
		for _, index := range t.Indices {
			indexStr, _, _, err := getType(index)
			if err != nil {
				return "", false, false, err
			}
			indices = append(indices, indexStr)
		}
		typeStr = fmt.Sprintf("%s[%s]", x, strings.Join(indices, ", "))
	case *ast.ArrayType:
		isSlice = true
		eltType, _, _, err := getType(t.Elt)
		if err != nil {
			return "", false, false, err
		}
		typeStr = "[]" + eltType
	case *ast.ChanType:
		// Handle channel types
		dir := ""
		switch t.Dir {
		case ast.RECV:
			dir = "<-chan "
		case ast.SEND:
			dir = "chan<- "
		default:
			dir = "chan "
		}
		elemType, _, _, err := getType(t.Value)
		if err != nil {
			return "", false, false, err
		}
		typeStr = fmt.Sprintf("%s%s", dir, elemType)
	case *ast.MapType:
		keyType, _, _, err := getType(t.Key)
		if err != nil {
			return "", false, false, err
		}
		valueType, _, _, err := getType(t.Value)
		if err != nil {
			return "", false, false, err
		}
		typeStr = fmt.Sprintf("map[%s]%s", keyType, valueType)
	case *ast.FuncType:
		// Simplistic representation; expand as needed
		params, err := fieldListToString(t.Params)
		if err != nil {
			return "", false, false, err
		}
		results, err := fieldListToString(t.Results)
		if err != nil {
			return "", false, false, err
		}
		typeStr = fmt.Sprintf("func(%s) (%s)", params, results)
	case *ast.InterfaceType:
		methodsStr, err := fieldListToString(t.Methods)
		if err != nil {
			return "", false, false, err
		}
		typeStr = fmt.Sprintf("interface{%s}", methodsStr)
	case *ast.StructType:
		fieldsStr, err := fieldListToString(t.Fields)
		if err != nil {
			return "", false, false, err
		}
		typeStr = fmt.Sprintf("struct{%s}", fieldsStr)
	case *ast.Ellipsis:
		tmp := expr.(*ast.Ellipsis)
		eltType, _, _, err := getType(tmp.Elt)
		if err != nil {
			return "", false, false, err
		}
		typeStr = "..." + justTypeString(eltType, false, false, nil)
	default:
		return "", false, false, fmt.Errorf("unsupported type: %T", expr)
	}
	return typeStr, isPointer, isSlice, nil
}
