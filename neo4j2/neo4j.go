package neo4j2

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/rs/zerolog/log"
	codesurgeon "github.com/wricardo/code-surgeon"
)

func MergePackage(ctx context.Context, alias string, mod codesurgeon.Module, pkg codesurgeon.Package) MergeQuery {
	return MergeQuery{
		NodeType: "Package",
		Alias:    alias,
		Properties: map[string]any{
			"rootPackageFullName": mod.RootModuleName,
			"module_full_name":    mod.FullName,
			"name":                pkg.Package,
		},
	}
}

func MergeStruct(ctx context.Context, alias string, mod codesurgeon.Module, pkg codesurgeon.Package, strct codesurgeon.Struct) MergeQuery {
	return MergeQuery{
		NodeType: "Struct",
		Alias:    alias,
		Properties: map[string]any{
			"name":                strct.Name,
			"packageName":         pkg.Package,
			"packageFullName":     mod.FullName,
			"rootPackageFullName": mod.RootModuleName,
		},
		SetFields: map[string]string{
			"documentation": strings.Join(strct.Docs, "\n"),
			"definition":    strct.Definition,
			"isExported":    fmt.Sprintf("%t", strct.IsExported),
		},
	}
}

func MergeModule(ctx context.Context, alias string, mod codesurgeon.Module) MergeQuery {
	return MergeQuery{
		NodeType: "Module",
		Alias:    alias,
		Properties: map[string]any{
			"name": mod.RootModuleName,
		},
	}
}

// UpsertPackage creates or updates a package node in Neo4j and links it to its module.
func UpsertPackage(ctx context.Context, session neo4j.SessionWithContext, mod codesurgeon.Module, pkg codesurgeon.Package) error {
	query := CypherQuery{}.
		Merge(MergePackage(ctx, "p", mod, pkg)).
		Merge(MergeModule(ctx, "m", mod)).
		MergeRel("p", "BELONGS_TO", "m", nil).
		Return("id(p) as nodeID")

	return query.Execute(ctx, session)
}

// UpsertStruct creates or updates a struct node in Neo4j and links it to its package and base type.
func UpsertStruct(ctx context.Context, session neo4j.SessionWithContext, mod codesurgeon.Module, pkg codesurgeon.Package, strct codesurgeon.Struct) error {
	query := CypherQuery{}.
		Merge(MergeBaseType(ctx, "bs", strct.Name, pkg.Package, mod.FullName, codesurgeon.TypeDetails{
			Package:        &mod.FullName,
			PackageName:    &pkg.Package,
			Type:           &strct.Name,
			TypeName:       strct.Name,
			IsPointer:      false,
			IsSlice:        false,
			IsMap:          false,
			IsBuiltin:      false,
			IsExternal:     false,
			TypeReferences: []codesurgeon.TypeReference{},
		})).
		Merge(MergeStruct(ctx, "f", mod, pkg, strct)).
		Merge(MergePackage(ctx, "p", mod, pkg)).
		MergeRel("f", "BELONGS_TO", "p", nil).
		MergeRel("f", "OF_TYPE", "bs", nil).
		Return("id(f) as nodeID")

	return query.Execute(ctx, session)
}

// Helper function to merge a Field node
func MergeField(ctx context.Context, alias string, mod codesurgeon.Module, pkg codesurgeon.Package, field codesurgeon.Field) MergeQuery {
	return MergeQuery{
		NodeType: "Field",
		Alias:    alias,
		Properties: map[string]any{
			"name":    field.Name,
			"package": pkg.Package,
		},
		SetFields: map[string]string{
			"documentation":   strings.Join(field.Docs, "\n"),
			"packageName":     pkg.Package,
			"packageFullName": mod.FullName,
			"type":            field.Type,
			"isExported":      fmt.Sprintf("%t", !field.Private),
		},
	}
}

func MergeType2(ctx context.Context, alias string, paramType string) MergeQuery {
	normalizedType := normalizeType(paramType)
	return MergeQuery{
		NodeType: "Type",
		Alias:    alias,
		Properties: map[string]any{
			"type": normalizedType,
		},
	}
}

// Helper to merge a Type node
func MergeType(ctx context.Context, alias string, typeString string, paramType codesurgeon.TypeDetails) MergeQuery {
	normalizedType := normalizeType(paramType.TypeName)
	// Fallback to TypeDetails.Type if TypeName is empty
	if normalizedType == "" && paramType.Type != nil {
		normalizedType = *paramType.Type
	}
	// Final fallback to the type string from Param.Type
	if normalizedType == "" {
		normalizedType = typeString
	}
	return MergeQuery{
		NodeType: "Type",
		Alias:    alias,
		Properties: map[string]any{
			"type": normalizedType,
		},
	}
}

func normalizeType(n string) string {
	return n
}

// A simpler BaseType merge for when we only have a "type" property
func MergeBaseType(ctx context.Context, alias string, typeString string, currentPackage string, currentPackageFullName string, td codesurgeon.TypeDetails) MergeQuery {
	packageName := ""
	packageFullName := ""
	baseType := "UNKNOWN"

	// Try to get from TypeDetails first
	if td.Type != nil && len(*td.Type) > 0 {
		baseType = *td.Type
		if len(td.TypeReferences) > 0 {
			if td.TypeReferences[0].PackageName != nil {
				packageName = *td.TypeReferences[0].PackageName
			}
			if td.TypeReferences[0].Package != nil {
				packageFullName = *td.TypeReferences[0].Package
			}
		}
	}
	if td.PackageName != nil {
		packageName = *td.PackageName
	}
	if td.Package != nil {
		packageFullName = *td.Package
	}

	// If we have packageName but no packageFullName, and it matches current package, use currentPackageFullName
	if packageName != "" && packageFullName == "" && packageName == currentPackage {
		packageFullName = currentPackageFullName
	}

	// Fallback: parse the type string if TypeDetails is empty
	if baseType == "UNKNOWN" && typeString != "" {
		// Strip pointer prefix
		cleanType := strings.TrimPrefix(typeString, "*")
		// Strip slice prefix
		cleanType = strings.TrimPrefix(cleanType, "[]")

		// Check if it has a package prefix (e.g., "model.Person")
		if strings.Contains(cleanType, ".") {
			parts := strings.Split(cleanType, ".")
			if len(parts) == 2 {
				packageName = parts[0]
				baseType = parts[1]
				// If this package matches current package, we can infer full name
				if packageName == currentPackage {
					packageFullName = currentPackageFullName
				}
			}
		} else {
			// No package prefix, use current package
			baseType = cleanType
			if packageName == "" {
				packageName = currentPackage
				packageFullName = currentPackageFullName
			}
		}
	}

	// Final check: if we still have packageName but no packageFullName, and it matches current package
	if packageName != "" && packageFullName == "" && packageName == currentPackage {
		packageFullName = currentPackageFullName
	}

	// Override package for known builtin types
	if isBuiltinType(baseType) {
		packageName = "builtin"
		packageFullName = "builtin"
	}

	if packageName == "" {
		packageName = "builtin"
		packageFullName = "builtin"
	}

	return MergeQuery{
		NodeType: "BaseType",
		Alias:    alias,
		Properties: map[string]any{
			"type":            baseType,
			"packageName":     packageName,
			"packageFullName": packageFullName,
		},
	}
}

func isBuiltinType(typeName string) bool {
	builtins := map[string]bool{
		"bool":       true,
		"byte":       true,
		"complex64":  true,
		"complex128": true,
		"error":      true,
		"float32":    true,
		"float64":    true,
		"int":        true,
		"int8":       true,
		"int16":      true,
		"int32":      true,
		"int64":      true,
		"rune":       true,
		"string":     true,
		"uint":       true,
		"uint8":      true,
		"uint16":     true,
		"uint32":     true,
		"uint64":     true,
		"uintptr":    true,
	}
	return builtins[typeName]
}

// UpsertStructField creates or updates a field node in Neo4j and links it to its struct and package.
// UpsertStructField creates or updates a field node in Neo4j and links it to its struct and package.
func UpsertStructField(ctx context.Context, session neo4j.SessionWithContext, mod codesurgeon.Module, pkg codesurgeon.Package, strct codesurgeon.Struct, field codesurgeon.Field) error {
	query := CypherQuery{}.
		Merge(MergeField(ctx, "f", mod, pkg, field)).
		Merge(MergeType(ctx, "t", field.Type, field.TypeDetails)).
		Merge(MergeBaseType(ctx, "b", field.Type, pkg.Package, mod.FullName, field.TypeDetails)).
		Merge(MergeStruct(ctx, "s", mod, pkg, strct)).
		Merge(MergePackage(ctx, "p", mod, pkg)).
		MergeRel("f", "OF_TYPE", "t", nil).
		MergeRel("t", "BASE_TYPE", "b", nil).
		MergeRel("s", "HAS_FIELD", "f", nil).
		Return("id(f) as nodeID")

	return query.Execute(ctx, session)
}

// We need to handle UpsertMethodParam and others similarly:
// For UpsertMethodParam, we have a Function node with receiver. Let's define MergeFunctionWithReceiver.
// func MergeMethod(ctx context.Context, alias string, package_, receiver, methodName, documentation, packageName string) MergeQuery {
// 	return MergeQuery{
// 		NodeType: "Method",
// 		Alias:    alias,
// 		Properties: map[string]interface{}{
// 			"package":  package_,
// 			"receiver": receiver,
// 			"function": methodName,
// 		},
// 		SetFields: map[string]string{
// 			"name":          methodName,
// 			"documentation": documentation,
// 			"packageName":   packageName,
// 		},
// 	}
// }

func MergeMethod(ctx context.Context, alias string, mod codesurgeon.Module, pkg codesurgeon.Package, fn codesurgeon.Method) MergeQuery {
	return MergeQuery{
		NodeType: "Method",
		Alias:    alias,
		Properties: map[string]any{
			"name":                fn.Name,
			"receiver":            strings.Replace(fn.Receiver, "*", "", 1),
			"packageName":         pkg.Package,        //short name of the package like "xyz"
			"packageFullName":     mod.FullName,       //full name of the package like "github.com/wricardo/code-surgeon/examples/xyz"
			"rootPackageFullName": mod.RootModuleName, //full name of the package like "github.com/wricardo/code-surgeon"
		},
		SetFields: map[string]string{
			"documentation": strings.Join(fn.Docs, "\n"),
			"definition":    fn.Definition,
			"isExported":    fmt.Sprintf("%t", fn.IsExported),
			"isTest":        fmt.Sprintf("%t", fn.IsTest),
			"isBenchmark":   fmt.Sprintf("%t", fn.IsBenchmark),
		},
	}
}

// UpsertMethod creates or updates a method node in Neo4j and links it to its receiver Struct.
func UpsertMethod(ctx context.Context, session neo4j.SessionWithContext, mod codesurgeon.Module, pkg codesurgeon.Package, method codesurgeon.Method, receiver codesurgeon.Struct) error {
	query := CypherQuery{}.
		Merge(MergeMethod(ctx, "m", mod, pkg, method)).
		Merge(MergeStruct(ctx, "s", mod, pkg, receiver)). // Ensure the receiver Struct is merged
		MergeRel("s", "HAS_METHOD", "m", nil).            // Create a relationship between the receiver Struct and the Method
		Return("id(m) as nodeID")

	return query.Execute(ctx, session)
}

func MergeFunction(ctx context.Context, alias string, mod codesurgeon.Module, pkg codesurgeon.Package, fn codesurgeon.Function) MergeQuery {
	return MergeQuery{
		NodeType: "Function",
		Alias:    alias,
		Properties: map[string]any{
			"name":                fn.Name,
			"packageName":         pkg.Package,
			"packageFullName":     mod.FullName,
			"rootPackageFullName": mod.RootModuleName,
		},
		SetFields: map[string]string{
			"documentation": strings.Join(fn.Docs, "\n"),
			"definition":    fn.Definition,
			"isExported":    fmt.Sprintf("%t", fn.IsExported),
			"isTest":        fmt.Sprintf("%t", fn.IsTest),
			"isBenchmark":   fmt.Sprintf("%t", fn.IsBenchmark),
		},
	}
}

// UpsertFunction creates or updates a function node in Neo4j and links it to its package.
func UpsertFunction(ctx context.Context, session neo4j.SessionWithContext, mod codesurgeon.Module, pkg codesurgeon.Package, fn codesurgeon.Function) error {
	query := CypherQuery{}.
		Merge(MergeFunction(ctx, "f", mod, pkg, fn)).
		Merge(MergePackage(ctx, "p", mod, pkg)).
		MergeRel("f", "BELONGS_TO", "p", nil).
		Return("id(f) as nodeID")

	return query.Execute(ctx, session)
}

// UpsertInterface creates or updates an interface node in Neo4j and links it to its package.
func UpsertInterface(ctx context.Context, session neo4j.SessionWithContext, mod codesurgeon.Module, pkg codesurgeon.Package, iface codesurgeon.Interface) error {
	query := CypherQuery{}.
		Merge(MergeBaseType(ctx, "bs", iface.Name, pkg.Package, mod.FullName, codesurgeon.TypeDetails{
			Package:        &mod.FullName,
			PackageName:    &pkg.Package,
			Type:           &iface.Name,
			TypeName:       iface.Name,
			IsPointer:      false,
			IsSlice:        false,
			IsMap:          false,
			IsBuiltin:      false,
			IsExternal:     false,
			TypeReferences: []codesurgeon.TypeReference{},
		})).
		Merge(MergeInterface(ctx, "f", mod, pkg, iface)).
		Merge(MergePackage(ctx, "p", mod, pkg)).
		MergeRel("f", "BELONGS_TO", "p", nil).
		MergeRel("f", "OF_TYPE", "bs", nil).
		Return("id(f) as nodeID")

	return query.Execute(ctx, session)
}

func MergeInterface(ctx context.Context, alias string, mod codesurgeon.Module, pkg codesurgeon.Package, iface codesurgeon.Interface) MergeQuery {
	return MergeQuery{
		NodeType: "Interface",
		Alias:    alias,
		Properties: map[string]any{
			"name":                iface.Name,
			"packageName":         pkg.Package,
			"packageFullName":     mod.FullName,
			"rootPackageFullName": mod.RootModuleName,
		},
		SetFields: map[string]string{
			"documentation": strings.Join(iface.Docs, "\n"),
			"definition":    iface.Definition,
			"isExported":    fmt.Sprintf("%t", iface.IsExported),
		},
	}
}

func MergeReturn(ctx context.Context, alias string, mod codesurgeon.Module, pkg codesurgeon.Package, ret codesurgeon.Param) MergeQuery {
	return MergeQuery{
		NodeType: "Return",
		Alias:    alias,
		Properties: map[string]any{
			"name": ret.Name,
			"type": ret.Type,
		},
		SetFields: map[string]string{
			"packageName":     pkg.Package,
			"packageFullName": mod.FullName,
		},
	}
}

// UpsertFunctionReturn creates or updates a function return node in Neo4j and links it appropriately.
func UpsertFunctionReturn(ctx context.Context, session neo4j.SessionWithContext, mod codesurgeon.Module, pkg codesurgeon.Package, fn codesurgeon.Function, ret codesurgeon.Param) error {
	query := CypherQuery{}.
		Merge(MergeFunction(ctx, "f", mod, pkg, fn)).
		Merge(MergeReturn(ctx, "r", mod, pkg, ret)).
		MergeRel("f", "RETURNS", "r", nil).
		Merge(MergeType(ctx, "t", ret.Type, ret.TypeDetails)).
		MergeRel("r", "OF_TYPE", "t", nil).
		Merge(MergeBaseType(ctx, "b", ret.Type, pkg.Package, mod.FullName, ret.TypeDetails)).
		MergeRel("t", "BASE_TYPE", "b", nil).
		Return("id(r) as nodeID")

	return query.Execute(ctx, session)
}

// UpsertMethodReturn creates or updates a method return node in Neo4j and links it appropriately.
func UpsertMethodReturn(ctx context.Context, session neo4j.SessionWithContext, mod codesurgeon.Module, pkg codesurgeon.Package, fn codesurgeon.Method, ret codesurgeon.Param) error {
	query := CypherQuery{}.
		Merge(MergeMethod(ctx, "m", mod, pkg, fn)).
		Merge(MergeReturn(ctx, "r", mod, pkg, ret)).
		MergeRel("m", "RETURNS", "r", nil).
		Merge(MergeType(ctx, "t", ret.Type, ret.TypeDetails)).
		MergeRel("r", "OF_TYPE", "t", nil).
		Merge(MergeBaseType(ctx, "b", ret.Type, pkg.Package, mod.FullName, ret.TypeDetails)).
		MergeRel("t", "BASE_TYPE", "b", nil).
		Return("id(r) as nodeID")

	return query.Execute(ctx, session)
}

func MergeFuncParam(ctx context.Context, alias string, mod codesurgeon.Module, pkg codesurgeon.Package, fn codesurgeon.Function, param codesurgeon.Param) MergeQuery {
	return MergeQuery{
		NodeType: "Param",
		Alias:    alias,
		Properties: map[string]interface{}{
			"name":       param.Name,
			"type":       param.Type,
			"package":    pkg.Package,
			"methodName": fn.Name,
		},
		SetFields: map[string]string{
			"packageName":     pkg.Package,
			"packageFullName": mod.FullName,
		},
	}
}

func MergeInterfaceMethodParam(ctx context.Context, alias string, mod codesurgeon.Module, pkg codesurgeon.Package, iface codesurgeon.Interface, method codesurgeon.Method, param codesurgeon.Param) MergeQuery {
	return MergeQuery{
		NodeType: "Param",
		Alias:    alias,
		Properties: map[string]interface{}{
			"name":    param.Name,
			"type":    param.Type,
			"package": pkg.Package,
			"iface":   iface.Name,
			"method":  method.Name,
		},
		SetFields: map[string]string{
			"name":            param.Name,
			"type":            param.Type,
			"packageName":     pkg.Package,
			"packageFullName": mod.FullName,
		},
	}
}

func MergeMethodParam(ctx context.Context, alias string, mod codesurgeon.Module, pkg codesurgeon.Package, method codesurgeon.Method, param codesurgeon.Param) MergeQuery {
	return MergeQuery{
		NodeType: "Param",
		Alias:    alias,
		Properties: map[string]interface{}{
			"name":       param.Name,
			"type":       param.Type,
			"package":    pkg.Package,
			"methodName": method.Name,
		},
		SetFields: map[string]string{
			"name":            param.Name,
			"type":            param.Type,
			"packageName":     pkg.Package,
			"packageFullName": mod.FullName,
		},
	}
}

func UpsertFunctionParam(ctx context.Context, session neo4j.SessionWithContext, mod codesurgeon.Module, pkg codesurgeon.Package, fn codesurgeon.Function, param codesurgeon.Param) error {
	query := CypherQuery{}.
		Merge(MergeFuncParam(ctx, "p", mod, pkg, fn, param)).
		Merge(MergeFunction(ctx, "f", mod, pkg, fn)).
		Merge(MergeType(ctx, "t", param.Type, param.TypeDetails)).
		Merge(MergeBaseType(ctx, "b", param.Type, pkg.Package, mod.FullName, param.TypeDetails)).
		MergeRel("t", "BASE_TYPE", "b", nil).
		MergeRel("f", "HAS_PARAM", "p", nil).
		MergeRel("p", "OF_TYPE", "t", nil).
		Return("id(p) as nodeID")

	return query.Execute(ctx, session)
}

func UpsertMethodParam(ctx context.Context, session neo4j.SessionWithContext, mod codesurgeon.Module, pkg codesurgeon.Package, strct codesurgeon.Struct, method codesurgeon.Method, param codesurgeon.Param) error {
	query := CypherQuery{}.
		Merge(MergeMethodParam(ctx, "p", mod, pkg, method, param)).
		Merge(MergeMethod(ctx, "m", mod, pkg, method)).
		Merge(MergeType(ctx, "t", param.Type, param.TypeDetails)).
		Merge(MergeBaseType(ctx, "b", param.Type, pkg.Package, mod.FullName, param.TypeDetails)).
		MergeRel("t", "BASE_TYPE", "b", nil).
		MergeRel("m", "HAS_PARAM", "p", nil).
		MergeRel("p", "OF_TYPE", "t", nil).
		Return("id(p) as nodeID")

	return query.Execute(ctx, session)
}

// For UpsertInterfaceMethodParam, we have interfaceName, methodName, etc.
// We can reuse MergeParam and define a special MergeFunction variant for interface methods.
func MergeInterfaceMethodFunction(ctx context.Context, alias string, mod codesurgeon.Module, pkg codesurgeon.Package, iface codesurgeon.Interface, method codesurgeon.Method) MergeQuery {
	return MergeQuery{
		NodeType: "Function",
		Alias:    alias,
		Properties: map[string]interface{}{
			"packageName":         pkg.Package,
			"packageFullName":     mod.FullName,
			"rootPackageFullName": mod.RootModuleName,
			"interfaceName":       iface.Name,
			"name":                method.Name,
		},
		SetFields: map[string]string{
			"documentation": strings.Join(method.Docs, "\n"),
			"definition":    method.Definition,
			"isExported":    fmt.Sprintf("%t", method.IsExported),
			"isTest":        fmt.Sprintf("%t", method.IsTest),
			"isBenchmark":   fmt.Sprintf("%t", method.IsBenchmark),
		},
	}
}

func UpsertInterfaceMethodParam(ctx context.Context, session neo4j.SessionWithContext, mod codesurgeon.Module, pkg codesurgeon.Package, iface codesurgeon.Interface, method codesurgeon.Method, param codesurgeon.Param) error {
	query := CypherQuery{}.
		Merge(MergeInterfaceMethodParam(ctx, "p", mod, pkg, iface, method, param)).
		Merge(MergeInterfaceMethodFunction(ctx, "m", mod, pkg, iface, method)).
		MergeRel("m", "HAS_PARAM", "p", nil).
		Merge(MergeType(ctx, "t", param.Type, param.TypeDetails)).
		MergeRel("p", "OF_TYPE", "t", nil).
		Merge(MergeBaseType(ctx, "b", param.Type, pkg.Package, mod.FullName, param.TypeDetails)).
		MergeRel("t", "BASE_TYPE", "b", nil).
		Return("id(p) as nodeID")

	return query.Execute(ctx, session)
}

func UpsertInterfaceMethodReturn(ctx context.Context, session neo4j.SessionWithContext, mod codesurgeon.Module, pkg codesurgeon.Package, iface codesurgeon.Interface, method codesurgeon.Method, ret codesurgeon.Param) error {
	query := CypherQuery{}.
		Merge(MergeInterfaceMethodFunction(ctx, "m", mod, pkg, iface, method)).
		Merge(MergeReturn(ctx, "r", mod, pkg, ret)).
		MergeRel("m", "RETURNS", "r", nil).
		Merge(MergeType(ctx, "t", ret.Type, ret.TypeDetails)).
		MergeRel("r", "OF_TYPE", "t", nil).
		Merge(MergeBaseType(ctx, "b", ret.Type, pkg.Package, mod.FullName, ret.TypeDetails)).
		MergeRel("t", "BASE_TYPE", "b", nil).
		Return("id(r) as nodeID")

	return query.Execute(ctx, session)
}

// UpsertInterfaceMethod creates or updates a method node in Neo4j and links it to its interface and package.
func UpsertInterfaceMethod(ctx context.Context, session neo4j.SessionWithContext, mod codesurgeon.Module, pkg codesurgeon.Package, iface codesurgeon.Interface, method codesurgeon.Method) error {
	query := CypherQuery{}.
		Merge(MergeInterfaceMethodFunction(ctx, "m", mod, pkg, iface, method)).
		Merge(MergeInterface(ctx, "i", mod, pkg, iface)).
		Merge(MergePackage(ctx, "p", mod, pkg)).
		MergeRel("i", "HAS_FUNCTION", "m", nil).
		Return("id(m) as nodeID")

	return query.Execute(ctx, session)
}

func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	str, ok := v.(string)
	if !ok {
		return ""
	}
	return str
}

func toFloat32Slice(v interface{}) []float32 {
	if v == nil {
		return nil
	}
	floats, ok := v.([]float32)
	if !ok {
		return nil
	}
	return floats
}

func toFloat64(v interface{}) float64 {
	if v == nil {
		return 0
	}
	f, ok := v.(float64)
	if !ok {
		return 0
	}
	return f
}

func QueryNeo4J(ctx context.Context, driver neo4j.DriverWithContext, query string, params map[string]interface{}) ([][]interface{}, error) {
	session := driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(ctx)
	var results [][]interface{}
	_, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		result, err := tx.Run(ctx, query, params)
		if err != nil {
			return nil, err
		}
		for result.Next(ctx) {
			values := result.Record().Values
			results = append(results, values)
		}
		return nil, nil
	})
	if err != nil {
		return nil, err
	}

	return results, nil
}

func Connect(ctx context.Context, uri, user, password string) (neo4j.DriverWithContext, func(), error) {
	driver, err := neo4j.NewDriverWithContext(
		uri,
		neo4j.BasicAuth(user, password, ""))
	var closefn func()
	if err == nil && driver != nil {
		closefn = func() {
			driver.Close(ctx)
		}

		err = driver.VerifyConnectivity(ctx)
		if err != nil {
			log.Print("Error connecting to Neo4j (proceeding anyway):", err)
			return nil, nil, err
		}
	} else {
		log.Print("Error connecting to Neo4j (proceeding anyway):", err)
		return nil, nil, err
	}

	return driver, closefn, nil
}

func ClearAll(ctx context.Context, driver neo4j.DriverWithContext) error {
	session := driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		_, err := tx.Run(ctx, "MATCH (n) DETACH DELETE n", nil)
		if err != nil {
			return nil, err
		}
		return nil, nil
	})
	if err != nil {
		return err
	}

	return nil
}

type MatchQuery struct {
	NodeType   string
	Alias      string
	Properties map[string]interface{}
}

func BuildOptionalMatchQuery(cq MatchQuery) QueryFragment {
	if cq.Alias == "" {
		cq.Alias = "n"
	}

	matchParts := make([]string, 0, len(cq.Properties))
	params := make(map[string]interface{})
	for key, value := range cq.Properties {
		matchParts = append(matchParts, fmt.Sprintf("%s: $%s%s", key, cq.Alias, key))
		params[cq.Alias+key] = value
	}

	query := fmt.Sprintf("OPTIONAL MATCH (%s:%s {%s})", cq.Alias, cq.NodeType, strings.Join(matchParts, ", "))
	return QueryFragment{
		Fragment: query,
		Params:   params,
	}
}

func BuildMatchQuery(cq MatchQuery) QueryFragment {
	if cq.Alias == "" {
		cq.Alias = "n"
	}

	matchParts := make([]string, 0, len(cq.Properties))
	params := make(map[string]interface{})
	for key, value := range cq.Properties {
		matchParts = append(matchParts, fmt.Sprintf("%s: $%s%s", key, cq.Alias, key))
		params[cq.Alias+key] = value
	}

	query := fmt.Sprintf("MATCH (%s:%s {%s})", cq.Alias, cq.NodeType, strings.Join(matchParts, ", "))
	return QueryFragment{
		Fragment: query,
		Params:   params,
	}
}

type MergeQuery struct {
	NodeType   string
	Alias      string
	Properties map[string]interface{}
	SetFields  map[string]string
}

func BuildMergeQuery(cq MergeQuery) QueryFragment {
	if cq.Alias == "" {
		cq.Alias = "n"
	}

	matchParts := make([]string, 0, len(cq.Properties))
	params := make(map[string]interface{})
	for key, value := range cq.Properties {
		matchParts = append(matchParts, fmt.Sprintf("%s: $%s%s", key, cq.Alias, key))
		params[cq.Alias+key] = value
	}

	query := fmt.Sprintf("MERGE (%s:%s {%s})", cq.Alias, cq.NodeType, strings.Join(matchParts, ", "))

	if len(cq.SetFields) > 0 {
		setParts := make([]string, 0, len(cq.SetFields))
		for field, value := range cq.SetFields {
			setParts = append(setParts, fmt.Sprintf("%s.%s = $%s%s", cq.Alias, field, cq.Alias, field))
			params[cq.Alias+field] = value
		}
		query += "\nSET " + strings.Join(setParts, ", ")
	}

	return QueryFragment{
		Fragment: query,
		Params:   params,
	}
}

type QueryFragment struct {
	Fragment string
	Params   map[string]interface{}
}

type CypherQuery struct {
	query []string
	Args  map[string]interface{}
}

func (cq CypherQuery) Unwind(param string, alias string) CypherQuery {
	if cq.query == nil {
		cq.query = []string{}
	}
	cq.query = append(cq.query, fmt.Sprintf("UNWIND $%s as %s", param, alias))
	return cq
}

func (cq CypherQuery) With(ws ...string) CypherQuery {
	if cq.query == nil {
		cq.query = []string{}
	}
	cq.query = append(cq.query, "WITH "+strings.Join(ws, ", "))
	return cq
}

func (cq CypherQuery) Raw(q string) CypherQuery {
	if cq.query == nil {
		cq.query = []string{}
	}
	cq.query = append(cq.query, q)
	return cq
}

func (cq CypherQuery) Where(ws ...string) CypherQuery {
	if cq.query == nil {
		cq.query = []string{}
	}
	cq.query = append(cq.query, "WHERE "+strings.Join(ws, " AND "))
	return cq
}

func (cq CypherQuery) OptionalMatch(qf MatchQuery) CypherQuery {
	if cq.Args == nil {
		cq.Args = make(map[string]interface{})
	}

	bqf := BuildOptionalMatchQuery(qf)
	for key, value := range bqf.Params {
		cq.Args[key] = value
	}
	if cq.query == nil {
		cq.query = []string{}
	}
	cq.query = append(cq.query, bqf.Fragment)
	return cq
}

func (cq CypherQuery) Match(qf MatchQuery) CypherQuery {
	if cq.Args == nil {
		cq.Args = make(map[string]interface{})
	}

	bqf := BuildMatchQuery(qf)
	for key, value := range bqf.Params {
		cq.Args[key] = value
	}
	if cq.query == nil {
		cq.query = []string{}
	}
	cq.query = append(cq.query, bqf.Fragment)
	return cq
}

func (cq CypherQuery) Merge(qf MergeQuery) CypherQuery {
	if cq.Args == nil {
		cq.Args = make(map[string]interface{})
	}

	bqf := BuildMergeQuery(qf)
	for key, value := range bqf.Params {
		cq.Args[key] = value
	}
	if cq.query == nil {
		cq.query = []string{}
	}
	cq.query = append(cq.query, bqf.Fragment)
	return cq
}

func (cq CypherQuery) MergeRel(from, rel, to string, properties map[string]interface{}) CypherQuery {
	if properties != nil && len(properties) > 0 {
		var propsList []string
		for k, v := range properties {
			paramKey := fmt.Sprintf("%s_%s_%s", from, rel, k)
			cq.Args[paramKey] = v
			propsList = append(propsList, fmt.Sprintf("%s: $%s", k, paramKey))
		}
		cq.query = append(cq.query, fmt.Sprintf("MERGE (%s)-[:%s {%s}]->(%s)", from, rel, strings.Join(propsList, ", "), to))
	} else {
		cq.query = append(cq.query, fmt.Sprintf("MERGE (%s)-[:%s]->(%s)", from, rel, to))
	}
	return cq
}

func (cq CypherQuery) Return(fields ...string) CypherQuery {
	if cq.query == nil {
		cq.query = []string{}
	}
	if len(fields) == 0 {
		cq.query = append(cq.query, "RETURN 1")
	} else {
		cq.query = append(cq.query, "RETURN "+strings.Join(fields, ", "))
	}
	return cq
}

func (cq CypherQuery) Execute(ctx context.Context, session neo4j.SessionWithContext) error {
	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		result, err := tx.Run(ctx, strings.Join(cq.query, "\n"), cq.Args)
		if err != nil {
			return nil, err
		}

		records, err := result.Collect(ctx)
		if err != nil {
			return nil, err
		}
		if len(records) == 0 {
			return nil, fmt.Errorf("no records returned")
		}
		return nil, nil
	})
	return err
}

func (cq CypherQuery) ExecuteSession(ctx context.Context, session neo4j.Session) ([]*neo4j.Record, error) {
	result, err := session.Run(strings.Join(cq.query, "\n"), cq.Args)
	if err != nil {
		return nil, err
	}

	records, err := result.Collect()
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("no records returned")
	}
	return records, nil

}

// FormatQueryResults formats and prints the query results in the specified format
func FormatQueryResults(results [][]interface{}, format string) error {
	if len(results) == 0 {
		fmt.Println("No results found.")
		return nil
	}

	switch format {
	case "json":
		return formatAsJSON(results)
	case "table":
		return formatAsTable(results)
	case "raw":
		return formatAsRaw(results)
	default:
		return fmt.Errorf("unsupported format: %s (supported: json, table, raw)", format)
	}
}

func formatAsJSON(results [][]interface{}) error {
	jsonResults := make([]map[string]interface{}, len(results))
	for i, row := range results {
		rowMap := make(map[string]interface{})
		for j, value := range row {
			rowMap[fmt.Sprintf("column_%d", j)] = value
		}
		jsonResults[i] = rowMap
	}

	output, err := json.MarshalIndent(jsonResults, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(output))
	return nil
}

func formatAsTable(results [][]interface{}) error {
	if len(results) == 0 {
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	// Print header
	numCols := len(results[0])
	for i := 0; i < numCols; i++ {
		fmt.Fprintf(w, "Column_%d\t", i)
	}
	fmt.Fprintln(w)

	// Print separator
	for i := 0; i < numCols; i++ {
		fmt.Fprint(w, "--------\t")
	}
	fmt.Fprintln(w)

	// Print data rows
	for _, row := range results {
		for _, value := range row {
			fmt.Fprintf(w, "%v\t", value)
		}
		fmt.Fprintln(w)
	}

	return w.Flush()
}

func formatAsRaw(results [][]interface{}) error {
	for i, row := range results {
		fmt.Printf("Row %d: %v\n", i, row)
	}
	return nil
}
