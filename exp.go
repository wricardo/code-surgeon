package codesurgeon

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/doc"
	"go/format"
	"go/token"
	"strings"
)

// extractParsedInfo extracts parsed information from the AST.
// moduleName is the name of the module being parsed, as seen on the go.mod file.
func extractParsedInfo(packages map[string]*ast.Package, moduleName string, relModPath string) (*ParsedInfo, error) {
	output := &ParsedInfo{
		// Packages: make([]Package, 0, len(packages)),
		Modules: make([]Module, 0, len(packages)),
	}

	m := &Module{
		RootModuleName:    moduleName,
		RelativeDirectory: relModPath,
		FullName:          moduleName + "/" + strings.TrimLeft(relModPath, "./"),
		Packages:          make([]Package, 0, len(packages)),
	}

	for _, pkg := range packages {
		outPkg := Package{
			Structs:   make([]Struct, 0),
			Functions: make([]Function, 0),
			Variables: make([]Variable, 0),
			Constants: make([]Constant, 0),
			Imports:   make([]Import, 0),
		}

		docPkg := doc.New(pkg, "", doc.AllDecls|doc.AllMethods|doc.PreserveAST)
		outPkg.Package = pkg.Name // Set package name

		// Extract imports
		imports, err := extractImports(pkg)
		if err != nil {
			return nil, err
		}
		outPkg.Imports = imports

		// Extract types (structs and interfaces)
		structs, err := extractStructs(docPkg, outPkg)
		if err != nil {
			return nil, err
		}
		outPkg.Structs = append(outPkg.Structs, structs...)

		interfaces, err := extractInterfaces(docPkg, outPkg)
		if err != nil {
			return nil, err
		}
		outPkg.Interfaces = append(outPkg.Interfaces, interfaces...)

		// Build a map of structs for easy lookup
		structMap := make(map[string]*Struct)
		for i := range outPkg.Structs {
			structMap[outPkg.Structs[i].Name] = &outPkg.Structs[i]
		}

		// Extract functions and methods
		functions, methods, err := extractFunctionsAndMethods(pkg, outPkg)
		if err != nil {
			return nil, err
		}
		outPkg.Functions = append(outPkg.Functions, functions...)

		// Associate methods with structs
		for _, method := range methods {
			receiverName := strings.TrimPrefix(method.Receiver, "*")
			if structPtr, ok := structMap[receiverName]; ok {
				// Append to struct's methods
				method.PtrStruct = structPtr
				structPtr.Methods = append(structPtr.Methods, method)
			} else {
				// Optionally, handle methods for types not defined in this package
				// For now, we ignore them
			}
		}

		// Extract constants and variables
		constants, variables, err := extractConstantsVariables(pkg, outPkg)
		if err != nil {
			return nil, err
		}
		outPkg.Constants = append(outPkg.Constants, constants...)
		outPkg.Variables = append(outPkg.Variables, variables...)

		m.Packages = append(m.Packages, outPkg)
	}
	output.Modules = append(output.Modules, *m)
	output.Packages = m.Packages

	return output, nil
}

// parseFunctionDecl extracts function details from an *ast.FuncDecl node.
func parseFunctionDecl(funcDecl *ast.FuncDecl, docs string, pkg Package) (Function, error) {
	function := Function{
		Name: funcDecl.Name.Name,
		Docs: getDocsForField([]string{docs}),
	}

	// Parse function parameters
	params := []Param{}
	if funcDecl.Type.Params != nil {
		for _, param := range funcDecl.Type.Params.List {
			paramType, err := getFullType(param.Type, pkg)
			if err != nil {
				return function, err
			}

			if len(param.Names) > 0 {
				for _, name := range param.Names {
					p := &Param{
						Name:        name.Name,
						Type:        paramType.TypeName,
						TypeDetails: *paramType,

						PtrFunc: &function,
					}
					// p.FillTypeDetails()
					params = append(params, *p)

				}
			} else {
				// Handle anonymous parameters
				p := &Param{
					Type:        paramType.TypeName,
					TypeDetails: *paramType,
					PtrFunc:     &function,
				}
				// p.FillTypeDetails()
				params = append(params, *p)
			}
		}
	}
	function.Params = params

	// Parse return types
	returns := []Param{}
	if funcDecl.Type.Results != nil {
		for _, result := range funcDecl.Type.Results.List {
			returnType, err := getFullType(result.Type, pkg)
			if err != nil {
				return function, err
			}

			if len(result.Names) > 0 {
				for _, name := range result.Names {
					returns = append(returns, Param{
						Name: name.Name,
						Type: returnType.TypeName,

						PtrFunc: &function,
					})
				}
			} else {
				returns = append(returns, Param{
					Type: returnType.TypeName,

					PtrFunc: &function,
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
			return function, err
		}
		function.Body = bodyBuf.String()
	}

	// Construct the full function signature
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

	// Adjust the signature formatting
	signature := fmt.Sprintf("%s(%s)", function.Name, strings.Join(paramStrings, ", "))
	if len(returnStrings) > 0 {
		returnsStr := strings.Join(returnStrings, ", ")
		if len(function.Returns) > 1 {
			returnsStr = "(" + returnsStr + ")"
		}
		signature += fmt.Sprintf(" %s", returnsStr)
	}

	function.Signature = signature

	return function, nil
}

// parseMethodDecl extracts method details from an *ast.FuncDecl node.
func parseMethodDecl(funcDecl *ast.FuncDecl, docs string, ourPkg Package) (Method, error) {
	receiverType, _ := getFullType(funcDecl.Recv.List[0].Type, ourPkg)
	method := Method{
		Name:     funcDecl.Name.Name,
		Receiver: receiverType.TypeName,
		Docs:     getDocsForField([]string{docs}),
	}

	// Parse method parameters
	params := []Param{}
	if funcDecl.Type.Params != nil {
		for _, param := range funcDecl.Type.Params.List {
			paramType, err := getFullType(param.Type, ourPkg)
			if err != nil {
				return method, err
			}
			if (paramType.PackageName == nil || *paramType.PackageName == "") && ourPkg.Package != "" {
				paramType.PackageName = &ourPkg.Package
			}

			if len(param.Names) > 0 {
				for _, name := range param.Names {
					p := &Param{
						Name:        name.Name,
						Type:        paramType.TypeName,
						TypeDetails: *paramType,

						PtrMethod: &method,
					}
					// p.FillTypeDetails()
					params = append(params, *p)

				}
			} else {
				p := &Param{
					Name:        "",
					Type:        paramType.TypeName,
					TypeDetails: *paramType,

					PtrMethod: &method,
				}
				// p.FillTypeDetails()
				params = append(params, *p)
			}
		}
	}
	method.Params = params

	// Parse return types
	returns := []Param{}
	if funcDecl.Type.Results != nil {
		for _, result := range funcDecl.Type.Results.List {
			returnType, err := getFullType(result.Type, ourPkg)
			if err != nil {
				return method, err
			}

			if len(result.Names) > 0 {
				for _, name := range result.Names {
					p := &Param{
						Name:        name.Name,
						Type:        returnType.TypeName,
						TypeDetails: *returnType,

						PtrMethod: &method,
					}
					// p.FillTypeDetails()
					returns = append(returns, *p)
				}
			} else {
				p := &Param{
					Name:        "",
					Type:        returnType.TypeName,
					TypeDetails: *returnType,

					PtrMethod: &method,
				}
				// p.FillTypeDetails()
				returns = append(returns, *p)
			}
		}
	}
	method.Returns = returns

	// Extract the method body as a string
	var bodyBuf bytes.Buffer
	if funcDecl.Body != nil {
		err := format.Node(&bodyBuf, token.NewFileSet(), funcDecl.Body)
		if err != nil {
			return method, err
		}
		method.Body = bodyBuf.String()
	}

	// Construct the full method signature
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

	// Adjust the signature formatting
	signature := fmt.Sprintf("%s(%s)", method.Name, strings.Join(paramStrings, ", "))
	if len(returnStrings) > 0 {
		returnsStr := strings.Join(returnStrings, ", ")
		if len(method.Returns) > 1 {
			returnsStr = "(" + returnsStr + ")"
		}
		signature += fmt.Sprintf(" %s", returnsStr)
	}

	method.Signature = signature

	return method, nil
}

// extractFunctionsAndMethods traverses the AST to extract all functions and methods.
func extractFunctionsAndMethods(pkg *ast.Package, ourPkg Package) ([]Function, []Method, error) {
	var functions []Function
	var methods []Method

	for _, file := range pkg.Files {
		for _, decl := range file.Decls {
			if funcDecl, ok := decl.(*ast.FuncDecl); ok {
				if funcDecl.Recv == nil {
					// Package-level function
					function, err := parseFunctionDecl(funcDecl, funcDecl.Doc.Text(), ourPkg)
					if err != nil {
						return nil, nil, err
					}
					functions = append(functions, function)
				} else {
					// Method
					method, err := parseMethodDecl(funcDecl, funcDecl.Doc.Text(), ourPkg)
					if err != nil {
						return nil, nil, err
					}
					methods = append(methods, method)
				}
			}
		}
	}

	return functions, methods, nil
}
