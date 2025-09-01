package codesurgeon

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/printer"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/mod/modfile"
)

// FS embeds OpenAPI and proto files for the codesurgeon package.
//
//go:embed api/codesurgeon.openapi.json
//go:embed api/codesurgeon.proto
var FS embed.FS

// ParsedInfo holds parsed information about Go packages.
type ParsedInfo struct {
	Modules   []Module  `json:"modules"`
	Packages  []Package `json:"packages"`  // Deprecated: use Modules instead
	Directory string    `json:"directory"` // if information was parsed from a directory. It's either a directory or a file
	File      string    `json:"file"`      // if information was parsed from a single file. It's either a directory or a file
}

// Module represents a Go module with its packages.
type Module struct {
	RootModuleName    string    `json:"root_module_name"`   // Name of the module as seen on the go.mod file of the project
	RelativeDirectory string    `json:"relative_directory"` // Relative (to go.mod) directory of the module / or /cmd/something
	FullName          string    `json:"full_name"`          // Full name of the module, including the relative directory should be RootModuleName/RelativeDirectory
	Packages          []Package `json:"packages"`
}

// Package represents a Go package with its components such as imports, structs, functions, etc.
type Package struct {
	Package    string      `json:"package"`     // Name of the package as seen in the package declaration (e.g., "main")
	ModuleName string      `json:"module_name"` // Name of the module as seen in the go.mod file
	Imports    []Import    `json:"imports,omitemity"`
	Structs    []Struct    `json:"structs,omitemity"`
	Functions  []Function  `json:"functions,omitemity"`
	Variables  []Variable  `json:"variables,omitemity"`
	Constants  []Constant  `json:"constants,omitemity"`
	Interfaces []Interface `json:"interfaces,omitemity"`

	PtrModule *Module `json:"-"` // Pointer to the module that this package belongs to
}

type Import struct {
	Name string `json:"name"` // the alias of the package as it's being imported
	Path string `json:"path"`

	PtrPackage *Package `json:"-"` // Pointer to the package that this import belongs to
}

// Interface represents a Go interface and its methods.
type Interface struct {
	Name       string   `json:"name"`
	Methods    []Method `json:"methods,omitemity"`
	Docs       []string `json:"docs,omitemity"`
	Definition string   `json:"definition,omitempty"` // Full Go code definition of the interface

	PtrPackage *Package `json:"-"` // Pointer to the package that this interface belongs to
}

// Struct represents a Go struct and its fields and methods.
type Struct struct {
	Name       string   `json:"name"`
	Fields     []Field  `json:"fields,omitemity"`
	Methods    []Method `json:"methods,omitemity"`
	Docs       []string `json:"docs,omitemity"`
	Definition string   `json:"definition,omitempty"` // Full Go code definition of the struct

	PtrPackage *Package `json:"-"` // Pointer to the package that this struct belongs to
}

// Method represents a method in a Go struct or interface.
type Method struct {
	Receiver   string   `json:"receiver,omitempty"` // Receiver type (e.g., "*MyStruct" or "MyStruct")
	Name       string   `json:"name"`
	Params     []Param  `json:"params,omitemity"`
	Returns    []Param  `json:"returns,omitemity"`
	Docs       []string `json:"docs,omitemity"`
	Signature  string   `json:"signature"`
	Body       string   `json:"body,omitempty"`       // New field for method body
	Definition string   `json:"definition,omitempty"` // Full Go code definition of the method

	PtrStruct *Struct `json:"-"` // Pointer to the struct that this method belongs to
}

// Function represents a Go function with its parameters, return types, and documentation.
type Function struct {
	Name       string   `json:"name"`
	Params     []Param  `json:"params,omitemity"`
	Returns    []Param  `json:"returns,omitemity"`
	Docs       []string `json:"docs,omitemity"`
	Signature  string   `json:"signature"`
	Body       string   `json:"body,omitempty"`       // New field for function body
	Definition string   `json:"definition,omitempty"` // Full Go code definition of the function
}

// Param represents a parameter or return value in a Go function or method.
type Param struct {
	Name        string      `json:"name"` // Name of the parameter or return value
	Type        string      `json:"type"` // Type (e.g., "int", "*string")
	TypeDetails TypeDetails `json:"type_details"`

	PtrMethod *Method   `json:"-"` // Pointer to the method that this parameter belongs to
	PtrFunc   *Function `json:"-"` // Pointer to the function that this parameter belongs to
}

// func (p *Param) FillTypeDetails() {
// 	td := GetTypeDetails(p.Type, nil, nil)
// 	p.TypeDetails = td
// }

// Field represents a field in a Go struct.
type Field struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"`
	TypeDetails TypeDetails `json:"type_details"`
	Tag         string      `json:"tag"`
	Private     bool        `json:"private"`
	// Pointer     bool        `json:"pointer"`
	// Slice       bool        `json:"slice"`
	Docs    []string `json:"docs,omitemity"`
	Comment string   `json:"comment,omitempty"`

	PtrStruct *Struct `json:"-"` // Pointer to the struct that this field belongs to
}

type TypeDetails struct {
	Package     *string // in the cases of external types. like pbhire.Person this would be "github.com/x/y/pbhire"
	PackageName *string // in the cases of external types. like pbhire.Person this would be "pbhire"
	Type        *string // in the cases of external types. like pbhire.Person this would be "Person"

	TypeName       string
	IsPointer      bool
	IsSlice        bool
	IsMap          bool
	IsBuiltin      bool // if string, int, etc
	IsExternal     bool // if the type is from another package
	TypeReferences []TypeReference
}

type TypeReference struct {
	Package     *string
	PackageName *string
	Name        string // the name of the type, example TypeReference or Person or int32 or string
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

	return extractParsedInfo(packages, "", "")
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

// getModulePath reads the module name from the go.mod file.
func getModulePath(path string) (*struct {
	Path string
	Dir  string
}, error) {
	dir := path
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
			if err != nil {
				return nil, fmt.Errorf("error reading go.mod: %w", err)
			}
			modFile, err := modfile.Parse("go.mod", data, nil)
			if err != nil {
				return nil, fmt.Errorf("error parsing go.mod: %w", err)
			}
			return &struct {
				Path string
				Dir  string
			}{
				Path: modFile.Module.Mod.Path,
				Dir:  dir,
			}, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return nil, fmt.Errorf("go.mod not found")
		}
		dir = parent
	}
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

	// Determine full module path
	modulePath, err := getModulePath(fileOrDirectory)
	if err != nil {
		return nil, fmt.Errorf("error retrieving module path: %w", err)
	}
	relPath, err := filepath.Rel(modulePath.Dir, fileOrDirectory)
	if err != nil {
		return nil, fmt.Errorf("error resolving relative path: %w", err)
	}

	parsedInfo, err := extractParsedInfo(packages, modulePath.Path, relPath)
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
func extractStructs(docPkg *doc.Package, ourPkg Package) ([]Struct, error) {
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

					PtrPackage: &ourPkg,
				}

				// Generate the full struct definition using AST
				var defBuf bytes.Buffer
				// Use go/format to properly format the struct definition
				defBuf.WriteString(fmt.Sprintf("type %s struct {\n", t.Name))
				for _, field := range structType.Fields.List {
					// Format each field properly
					defBuf.WriteString("\t")

					// Field names
					for i, name := range field.Names {
						if i > 0 {
							defBuf.WriteString(", ")
						}
						defBuf.WriteString(name.Name)
					}

					// If there are field names, add a space
					if len(field.Names) > 0 {
						defBuf.WriteString(" ")
					}

					// Field type
					defBuf.WriteString(exprToString(field.Type))

					// Field tag
					if field.Tag != nil {
						defBuf.WriteString(" ")
						defBuf.WriteString(field.Tag.Value)
					}

					// Field comment
					if field.Comment != nil && len(field.Comment.List) > 0 {
						defBuf.WriteString(" ")
						defBuf.WriteString(field.Comment.List[0].Text)
					}

					defBuf.WriteString("\n")
				}
				defBuf.WriteString("}")
				parsedStruct.Definition = defBuf.String()

				for _, fvalue := range structType.Fields.List {
					name := ""
					if len(fvalue.Names) > 0 {
						name = fvalue.Names[0].Obj.Name
					}

					field := Field{
						Name: name,
						Type: "",
						Tag:  "",

						TypeDetails: TypeDetails{},
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

					typeDetails, err := getFullType(fvalue.Type, ourPkg)
					if err != nil {
						return nil, err
					}
					field.TypeDetails = *typeDetails

					field.Type = typeDetails.TypeName

					parsedStruct.Fields = append(parsedStruct.Fields, field)
				}

				structs = append(structs, parsedStruct)
			}
		}
	}
	return structs, nil
}

// extractInterfaces extracts interfaces from the provided documentation package.
func extractInterfaces(docPkg *doc.Package, pkg Package) ([]Interface, error) {
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

					PtrPackage: &pkg,
				}

				// Generate the full interface definition
				var defBuf bytes.Buffer
				defBuf.WriteString(fmt.Sprintf("type %s interface {\n", t.Name))

				for _, m := range interfaceType.Methods.List {
					if funcType, ok := m.Type.(*ast.FuncType); ok {
						// Generate method definition
						methodDef := m.Names[0].Name + "(" + formatParams(funcType.Params, pkg) + ")"
						if funcType.Results != nil && len(funcType.Results.List) > 0 {
							returnStr := formatParams(funcType.Results, pkg)
							if len(funcType.Results.List) > 1 {
								methodDef += " (" + returnStr + ")"
							} else {
								methodDef += " " + returnStr
							}
						}

						method := Method{
							Name:    m.Names[0].Name,
							Params:  extractParams(funcType.Params, pkg),
							Returns: extractParams(funcType.Results, pkg),
							Docs:    getDocsForFieldAst(m.Doc),
							Signature: fmt.Sprintf("%s(%s) (%s)", m.Names[0].Name,
								formatParams(funcType.Params, pkg), formatParams(funcType.Results, pkg)),
							Definition: methodDef,
						}

						// Add method to definition buffer
						defBuf.WriteString("\t")
						defBuf.WriteString(m.Names[0].Name)
						defBuf.WriteString("(")
						defBuf.WriteString(formatParams(funcType.Params, pkg))
						defBuf.WriteString(")")

						// Add return types if any
						if funcType.Results != nil && len(funcType.Results.List) > 0 {
							defBuf.WriteString(" ")
							returnStr := formatParams(funcType.Results, pkg)
							if len(funcType.Results.List) > 1 {
								defBuf.WriteString("(")
								defBuf.WriteString(returnStr)
								defBuf.WriteString(")")
							} else {
								defBuf.WriteString(returnStr)
							}
						}
						defBuf.WriteString("\n")

						parsedInterface.Methods = append(parsedInterface.Methods, method)
					}
				}

				defBuf.WriteString("}")
				parsedInterface.Definition = defBuf.String()

				interfaces = append(interfaces, parsedInterface)
			}
		}
	}
	return interfaces, nil
}

// extractImports extracts unique imports from the provided package.
func extractImports(pkg *ast.Package) ([]Import, error) {
	importSet := make(map[string]struct{})
	for _, file := range pkg.Files {
		for _, importSpec := range file.Imports {
			importPath := strings.Trim(importSpec.Path.Value, "\"")
			importSet[importPath] = struct{}{}
		}
	}

	var imports []Import
	for imp := range importSet {
		imports = append(imports, Import{
			Path: imp,
		})
	}
	return imports, nil
}

// extractConstantsVariables extracts constants and variables from the provided package.
func extractConstantsVariables(pkg *ast.Package, ourPkg Package) ([]Constant, []Variable, error) {
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
							tmp, _ := getFullType(valSpec.Type, ourPkg)
							varType = tmp.TypeName
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

func extractParams(fieldList *ast.FieldList, pkg Package) []Param {
	if fieldList == nil {
		return nil
	}
	params := make([]Param, 0, len(fieldList.List))
	for _, field := range fieldList.List {
		paramType, err := getFullType(field.Type, pkg)
		// paramType, _, _, err := getType(field.Type)
		if err != nil {
			continue // Or handle the error properly
		}
		for _, name := range field.Names {
			params = append(params, Param{Name: name.Name, Type: paramType.TypeName, TypeDetails: *paramType})
		}
		// Handle anonymous parameters (e.g., func(int, string) without names)
		if len(field.Names) == 0 {
			params = append(params, Param{Name: "", Type: paramType.TypeName, TypeDetails: *paramType})
		}
	}
	return params
}

func formatParams(fields *ast.FieldList, pkg Package) string {
	if fields == nil {
		return ""
	}
	paramStrings := []string{}
	for _, param := range extractParams(fields, pkg) {
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

func void(_ ...interface{}) {}

// getFullType processes an AST expression and returns a comprehensive TypeReference.
func getFullType(expr ast.Expr, pkg Package) (*TypeDetails, error) {
	//TODO: need to know if is builtin, custom type on current package (needs current package), or external type

	coallesce := func(a, b *string) *string {
		if a != nil && *a != "" {
			return a
		}
		return b
	}
	var tr TypeDetails
	tr = TypeDetails{
		// Package:     nil,
		// PackageName: nil,
		// TypeName:    "",
		IsPointer:  false,
		IsSlice:    false,
		IsMap:      false,
		IsBuiltin:  false,
		IsExternal: false,
	}

	switch t := expr.(type) {
	case *ast.Ident:
		switch t.Name {
		case "string", "int", "int32", "int64", "float32", "float64", "bool", "byte", "rune", "error":
			tr.IsBuiltin = true
		default:
			// TODO: need to know if it's apackage or a Type that needs the PackageName from pkg
			if pkg.Package != "" {
				tr.PackageName = &pkg.Package
			}

			tr.IsBuiltin = false
		}
		tr.TypeName = t.Name
		tr.Type = &t.Name
		// Optionally, set IsBuiltin based on known built-in types
		// Example:
		// tr.IsBuiltin = isBuiltinType(t.Name)

	case *ast.SelectorExpr:
		tr.IsExternal = true
		xFullType, err := getFullType(t.X, pkg)
		if err != nil {
			return nil, err
		}
		tr.TypeName = xFullType.TypeName + "." + t.Sel.Name
		tr.PackageName = &xFullType.TypeName
		tr.Package = &xFullType.TypeName
		tr.Type = &t.Sel.Name
		tr.TypeReferences = []TypeReference{
			{
				Package:     coallesce(xFullType.Package, tr.Package),
				PackageName: coallesce(xFullType.PackageName, tr.PackageName),
				Name:        t.Sel.Name,
			},
		}

	case *ast.StarExpr:
		tr.IsPointer = true
		innerFullType, err := getFullType(t.X, pkg)
		if err != nil {
			return nil, err
		}

		// Propagate flags from the inner type
		tr.IsSlice = tr.IsSlice || innerFullType.IsSlice
		tr.IsMap = tr.IsMap || innerFullType.IsMap
		tr.TypeName = "*" + innerFullType.TypeName
		// tr.Type = innerFullType.Type
		tr.TypeReferences = append(tr.TypeReferences, innerFullType.TypeReferences...)
		tr.Package = coallesce(tr.Package, innerFullType.Package)
		tr.PackageName = coallesce(tr.PackageName, innerFullType.PackageName)
		tr.Type = coallesce(tr.Type, innerFullType.Type)

	case *ast.IndexExpr:
		// Handle single type parameter (legacy support)
		xFullType, err := getFullType(t.X, pkg)
		if err != nil {
			return nil, err
		}
		indexFullType, err := getFullType(t.Index, pkg)
		if err != nil {
			return nil, err
		}
		tr.TypeName = fmt.Sprintf("%s[%s]", xFullType.TypeName, indexFullType.TypeName)
		tr.Type = &xFullType.TypeName
		tr.Package = coallesce(tr.Package, indexFullType.Package)
		tr.PackageName = coallesce(tr.PackageName, indexFullType.PackageName)
		// Optionally, propagate other flags if necessary

	case *ast.IndexListExpr:
		// Handle multiple type parameters (generics)
		xFullType, err := getFullType(t.X, pkg)
		if err != nil {
			return nil, err
		}
		indices := []string{}
		for _, index := range t.Indices {
			indexFullType, err := getFullType(index, pkg)
			if err != nil {
				return nil, err
			}
			indices = append(indices, indexFullType.TypeName)
		}
		tr.TypeName = fmt.Sprintf("%s[%s]", xFullType.TypeName, strings.Join(indices, ", "))
		tr.Type = &xFullType.TypeName
		tr.Package = coallesce(tr.Package, xFullType.Package)
		tr.PackageName = coallesce(tr.PackageName, xFullType.PackageName)
		// Optionally, handle flags from generic parameters

	case *ast.ArrayType:
		tr.IsSlice = true
		eltFullType, err := getFullType(t.Elt, pkg)
		if err != nil {
			return nil, err
		}
		tr.TypeName = "[]" + eltFullType.TypeName
		tr.Type = &eltFullType.TypeName
		tr.Package = coallesce(tr.Package, eltFullType.Package)
		tr.PackageName = coallesce(tr.PackageName, eltFullType.PackageName)
		// Optionally, propagate other flags from element type

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
		valueFullType, err := getFullType(t.Value, pkg)
		if err != nil {
			return nil, err
		}
		tr.TypeName = fmt.Sprintf("%s%s", dir, valueFullType.TypeName)
		tr.Type = &valueFullType.TypeName
		tr.Package = coallesce(tr.Package, valueFullType.Package)
		tr.PackageName = coallesce(tr.PackageName, valueFullType.PackageName)
		// Optionally, propagate other flags from value type

	case *ast.MapType:
		keyFullType, err := getFullType(t.Key, pkg)
		if err != nil {
			return nil, err
		}
		valueFullType, err := getFullType(t.Value, pkg)
		if err != nil {
			return nil, err
		}
		tr.TypeName = fmt.Sprintf("map[%s]%s", keyFullType.TypeName, valueFullType.TypeName)
		tr.IsMap = true
		tr.Type = &valueFullType.TypeName
		tr.Package = coallesce(tr.Package, valueFullType.Package)
		tr.PackageName = coallesce(tr.PackageName, valueFullType.PackageName)
		// Optionally, propagate other flags from key/value types

	case *ast.FuncType:
		// Simplistic representation; expand as needed
		params, err := fieldListToString(t.Params, pkg)
		if err != nil {
			return nil, err
		}
		results, err := fieldListToString(t.Results, pkg)
		if err != nil {
			return nil, err
		}
		tr.TypeName = fmt.Sprintf("func(%s) (%s)", params, results)
		tr.Type = &tr.TypeName
		// Func types may have their own flags if needed

	case *ast.InterfaceType:
		methodsStr, err := fieldListToString(t.Methods, pkg)
		if err != nil {
			return nil, err
		}
		tr.TypeName = fmt.Sprintf("interface{%s}", methodsStr)
		tr.Type = &tr.TypeName
		// Optionally, handle embedded interfaces or other flags

	case *ast.StructType:
		fieldsStr, err := fieldListToString(t.Fields, pkg)
		if err != nil {
			return nil, err
		}
		tr.TypeName = fmt.Sprintf("struct{%s}", fieldsStr)
		tr.Type = &tr.TypeName
		// Optionally, handle embedded fields or other flags

	case *ast.Ellipsis:
		tmp := t // Direct type assertion
		eltFullType, err := getFullType(tmp.Elt, pkg)
		if err != nil {
			return nil, err
		}
		if eltFullType == nil {
			tr.TypeName = "..."
		} else {
			if eltFullType.PackageName == nil {
				tr.TypeName = "..." + eltFullType.TypeName
			} else {
				tr.TypeName = "..." + justTypeString(eltFullType.TypeName, eltFullType.IsPointer, eltFullType.IsSlice, &ast.Ident{Name: *eltFullType.PackageName})
			}
		}
		tr.Type = &eltFullType.TypeName
		// Adjust flags if necessary

	default:
		return nil, fmt.Errorf("unsupported type: %T", expr)
	}

	return &tr, nil
}

// fieldListToString converts an *ast.FieldList to its string representation.
// Implement this function based on your specific requirements.
func fieldListToString(fl *ast.FieldList, pkg Package) (string, error) {
	if fl == nil {
		return "", nil
	}
	fields := []string{}
	for _, field := range fl.List {
		typeRef, err := getFullType(field.Type, pkg)
		if err != nil {
			return "", err
		}
		if len(field.Names) == 0 {
			fields = append(fields, typeRef.TypeName)
		} else {
			for _, name := range field.Names {
				fields = append(fields, fmt.Sprintf("%s %s", name.Name, typeRef.TypeName))
			}
		}
	}
	return strings.Join(fields, ", "), nil
}

// justTypeString is a helper function to format type strings based on flags.
// Implement this function based on your specific requirements.
func justTypeString(typeName string, isPointer, isSlice bool, packageName *ast.Ident) string {
	// Example implementation:
	if isPointer {
		typeName = "*" + typeName
	}
	if isSlice {
		typeName = "[]" + typeName
	}
	if packageName != nil {
		typeName = packageName.Name + "." + typeName
	}
	return typeName
}
