package neo4j2

import (
	"context"
	"fmt"
	"strings"

	codesurgeon "bitbucket.org/zetaactions/code-surgeon"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/rs/zerolog/log"
)

func MergePackage(ctx context.Context, alias string, mod codesurgeon.Module, pkg codesurgeon.Package) MergeQuery {
	return MergeQuery{
		NodeType: "Package",
		Alias:    alias,
		Properties: map[string]any{
			"module_full_name": mod.FullName,
			"name":             pkg.Package,
		},
	}
}

func MergeStruct(ctx context.Context, alias string, mod codesurgeon.Module, pkg codesurgeon.Package, strct codesurgeon.Struct) MergeQuery {
	return MergeQuery{
		NodeType: "Struct",
		Alias:    alias,
		Properties: map[string]any{
			"name":    strct.Name,
			"package": pkg.Package,
		},
		SetFields: map[string]string{
			"documentation": strings.Join(strct.Docs, "\n"),
			"packageName":   pkg.Package,
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

func UpsertPackage(ctx context.Context, driver neo4j.DriverWithContext, mod codesurgeon.Module, pkg codesurgeon.Package) error {
	session := driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(ctx)

	query := CypherQuery{}.
		Merge(MergePackage(ctx, "p", mod, pkg)).
		Merge(MergeModule(ctx, "m", mod)).
		MergeRel("p", "BELONGS_TO", "m", nil).
		Return("id(p) as nodeID")

	return query.Execute(ctx, session)
}

func UpsertStruct(ctx context.Context, driver neo4j.DriverWithContext, mod codesurgeon.Module, pkg codesurgeon.Package, strct codesurgeon.Struct) error {
	session := driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(ctx)

	query := CypherQuery{}.
		Merge(MergeBaseType(ctx, "bs", codesurgeon.TypeDetails{
			Package:        &pkg.Package,
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
func MergeField(ctx context.Context, alias string, pkg codesurgeon.Package, fieldName, fieldType, documentation string) MergeQuery {
	return MergeQuery{
		NodeType: "Field",
		Alias:    alias,
		Properties: map[string]any{
			"name":    fieldName,
			"package": pkg.Package,
		},
		SetFields: map[string]string{
			"documentation": documentation,
			"packageName":   pkg.Package,
			"type":          fieldType,
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
func MergeType(ctx context.Context, alias string, paramType codesurgeon.TypeDetails) MergeQuery {
	normalizedType := normalizeType(paramType.TypeName)
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
func MergeBaseType(ctx context.Context, alias string, td codesurgeon.TypeDetails) MergeQuery {
	packageName := ""
	baseType := "UNKNOWN"
	if td.Type != nil && len(*td.Type) > 0 {
		// baseType = field.TypeDetails.TypeReferences[0].Name
		baseType = *td.Type
		if len(td.TypeReferences) > 0 {
			packageName = *td.TypeReferences[0].PackageName
		}
	}
	if td.PackageName != nil {
		packageName = *td.PackageName
	}

	if packageName == "" {
		packageName = "builtin"
	}

	return MergeQuery{
		NodeType: "BaseType",
		Alias:    alias,
		Properties: map[string]any{
			"type":        baseType,
			"packageName": packageName,
		},
	}
}

// UpsertStructField creates or updates a field node in Neo4j and links it to its struct and package.
func UpsertStructField(ctx context.Context, driver neo4j.DriverWithContext, mod codesurgeon.Module, pkg codesurgeon.Package, strct codesurgeon.Struct, field codesurgeon.Field) error {
	session := driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(ctx)

	// packageName := ""
	// baseType := "UNKNOWN"
	// if field.TypeDetails.Type != nil && len(*field.TypeDetails.Type) > 0 {
	// 	// baseType = field.TypeDetails.TypeReferences[0].Name
	// 	baseType = *field.TypeDetails.Type
	// 	if len(field.TypeDetails.TypeReferences) > 0 {
	// 		packageName = *field.TypeDetails.TypeReferences[0].PackageName
	// 	}
	// }

	query := CypherQuery{}.
		Merge(MergeField(ctx, "f", pkg, field.Name, field.Type, strings.Join(field.Docs, "\n"))).
		Merge(MergeType(ctx, "t", field.TypeDetails)).
		Merge(MergeBaseType(ctx, "b", field.TypeDetails)).
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
			"name":     fn.Name,
			"receiver": fn.Receiver,
			"package":  pkg.Package,
		},
		SetFields: map[string]string{
			"documentation": strings.Join(fn.Docs, "\n"),
		},
	}
}

func UpsertMethod(ctx context.Context, driver neo4j.DriverWithContext, mod codesurgeon.Module, pkg codesurgeon.Package, method codesurgeon.Method, receiver codesurgeon.Struct) error {
	session := driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(ctx)

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
			"name":    fn.Name,
			"package": pkg.Package,
		},
		SetFields: map[string]string{
			"documentation": strings.Join(fn.Docs, "\n"),
		},
	}
}

func UpsertFunction(ctx context.Context, driver neo4j.DriverWithContext, mod codesurgeon.Module, pkg codesurgeon.Package, fn codesurgeon.Function) error {
	session := driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(ctx)

	query := CypherQuery{}.
		Merge(MergeFunction(ctx, "f", mod, pkg, fn)).
		Merge(MergePackage(ctx, "p", mod, pkg)).
		MergeRel("f", "BELONGS_TO", "p", nil).
		Return("id(f) as nodeID")

	return query.Execute(ctx, session)
}

func UpsertInterface(ctx context.Context, driver neo4j.DriverWithContext, mod codesurgeon.Module, pkg codesurgeon.Package, iface codesurgeon.Interface) error {
	session := driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(ctx)

	query := CypherQuery{}.
		Merge(MergeBaseType(ctx, "bs", codesurgeon.TypeDetails{
			Package:        &pkg.Package,
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
			"name":    iface.Name,
			"package": pkg.Package,
		},
		SetFields: map[string]string{
			"documentation": strings.Join(iface.Docs, "\n"),
		},
	}
}

func MergeReturn(ctx context.Context, alias string, ret codesurgeon.Param) MergeQuery {
	return MergeQuery{
		NodeType: "Return",
		Alias:    alias,
		Properties: map[string]any{
			"name": ret.Name,
			"type": ret.Type,
		},
	}
}

func UpsertFunctionReturn(ctx context.Context, driver neo4j.DriverWithContext, mod codesurgeon.Module, pkg codesurgeon.Package, fn codesurgeon.Function, ret codesurgeon.Param) error {
	session := driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(ctx)

	query := CypherQuery{}.
		Merge(MergeFunction(ctx, "f", mod, pkg, fn)).
		Merge(MergeReturn(ctx, "r", ret)).
		MergeRel("f", "RETURNS", "r", nil).
		Merge(MergeType(ctx, "t", ret.TypeDetails)).
		MergeRel("r", "OF_TYPE", "t", nil).
		Merge(MergeBaseType(ctx, "b", ret.TypeDetails)).
		MergeRel("t", "BASE_TYPE", "b", nil).
		Return("id(r) as nodeID")

	return query.Execute(ctx, session)
}

func UpsertMethodReturn(ctx context.Context, driver neo4j.DriverWithContext, mod codesurgeon.Module, pkg codesurgeon.Package, fn codesurgeon.Method, ret codesurgeon.Param) error {
	session := driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(ctx)

	query := CypherQuery{}.
		Merge(MergeMethod(ctx, "m", mod, pkg, fn)).
		Merge(MergeReturn(ctx, "r", ret)).
		MergeRel("m", "RETURNS", "r", nil).
		Merge(MergeType(ctx, "t", ret.TypeDetails)).
		MergeRel("r", "OF_TYPE", "t", nil).
		Merge(MergeBaseType(ctx, "b", ret.TypeDetails)).
		MergeRel("t", "BASE_TYPE", "b", nil).
		Return("id(r) as nodeID")

	return query.Execute(ctx, session)
}

func MergeFuncParam(ctx context.Context, alias string, pkg codesurgeon.Package, fn codesurgeon.Function, param codesurgeon.Param) MergeQuery {
	return MergeQuery{
		NodeType: "Param",
		Alias:    alias,
		Properties: map[string]interface{}{
			"name":       param.Name,
			"type":       param.Type,
			"package":    pkg.Package,
			"methodName": fn.Name,
		},
		SetFields: map[string]string{},
	}
}

func MergeInterfaceMethodParam(ctx context.Context, alias string, pkg codesurgeon.Package, iface codesurgeon.Interface, method codesurgeon.Method, param codesurgeon.Param) MergeQuery {
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
			"name": param.Name,
			"type": param.Type,
		},
	}
}

func MergeMethodParam(ctx context.Context, alias string, pkg codesurgeon.Package, method codesurgeon.Method, param codesurgeon.Param) MergeQuery {
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
			"name": param.Name,
			"type": param.Type,
		},
	}
}

func UpsertFunctionParam(ctx context.Context, driver neo4j.DriverWithContext, mod codesurgeon.Module, pkg codesurgeon.Package, fn codesurgeon.Function, param codesurgeon.Param) error {
	session := driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(ctx)

	query := CypherQuery{}.
		Merge(MergeFuncParam(ctx, "p", pkg, fn, param)).
		Merge(MergeFunction(ctx, "f", mod, pkg, fn)).
		Merge(MergeType(ctx, "t", param.TypeDetails)).
		Merge(MergeBaseType(ctx, "b", param.TypeDetails)).
		MergeRel("t", "BASE_TYPE", "b", nil).
		MergeRel("f", "HAS_PARAM", "p", nil).
		MergeRel("p", "OF_TYPE", "t", nil).
		Return("id(p) as nodeID")

	return query.Execute(ctx, session)
}

func UpsertMethodParam(ctx context.Context, driver neo4j.DriverWithContext, mod codesurgeon.Module, pkg codesurgeon.Package, strct codesurgeon.Struct, method codesurgeon.Method, param codesurgeon.Param) error {
	session := driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(ctx)

	query := CypherQuery{}.
		Merge(MergeMethodParam(ctx, "p", pkg, method, param)).
		Merge(MergeMethod(ctx, "m", mod, pkg, method)).
		Merge(MergeType(ctx, "t", param.TypeDetails)).
		Merge(MergeBaseType(ctx, "b", param.TypeDetails)).
		MergeRel("t", "BASE_TYPE", "b", nil).
		MergeRel("m", "HAS_PARAM", "p", nil).
		MergeRel("p", "OF_TYPE", "t", nil).
		Return("id(p) as nodeID")

	return query.Execute(ctx, session)
}

// For UpsertInterfaceMethodParam, we have interfaceName, methodName, etc.
// We can reuse MergeParam and define a special MergeFunction variant for interface methods.
func MergeInterfaceMethodFunction(ctx context.Context, alias string, pkg codesurgeon.Package, iface codesurgeon.Interface, method codesurgeon.Method) MergeQuery {
	return MergeQuery{
		NodeType: "Function",
		Alias:    alias,
		Properties: map[string]interface{}{
			"package":       pkg.ModuleName,
			"interfaceName": iface.Name,
			"name":          method.Name,
		},
		SetFields: map[string]string{
			"documentation": strings.Join(method.Docs, "\n"),
		},
	}
}

func UpsertInterfaceMethodParam(ctx context.Context, driver neo4j.DriverWithContext, mod codesurgeon.Module, pkg codesurgeon.Package, iface codesurgeon.Interface, method codesurgeon.Method, param codesurgeon.Param) error {
	session := driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(ctx)

	query := CypherQuery{}.
		Merge(MergeInterfaceMethodParam(ctx, "p", pkg, iface, method, param)).
		Merge(MergeInterfaceMethodFunction(ctx, "m", pkg, iface, method)).
		MergeRel("m", "HAS_PARAM", "p", nil).
		Merge(MergeType(ctx, "t", param.TypeDetails)).
		MergeRel("p", "OF_TYPE", "t", nil).
		Merge(MergeBaseType(ctx, "b", param.TypeDetails)).
		MergeRel("t", "BASE_TYPE", "b", nil).
		Return("id(p) as nodeID")

	return query.Execute(ctx, session)
}

func UpsertInterfaceMethodReturn(ctx context.Context, driver neo4j.DriverWithContext, mod codesurgeon.Module, pkg codesurgeon.Package, iface codesurgeon.Interface, method codesurgeon.Method, ret codesurgeon.Param) error {
	session := driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(ctx)

	query := CypherQuery{}.
		Merge(MergeInterfaceMethodFunction(ctx, "m", pkg, iface, method)).
		Merge(MergeReturn(ctx, "r", ret)).
		MergeRel("m", "RETURNS", "r", nil).
		Merge(MergeType(ctx, "t", ret.TypeDetails)).
		MergeRel("r", "OF_TYPE", "t", nil).
		Merge(MergeBaseType(ctx, "b", ret.TypeDetails)).
		MergeRel("t", "BASE_TYPE", "b", nil).
		Return("id(r) as nodeID")

	return query.Execute(ctx, session)
}

// UpsertInterfaceMethod creates or updates a method node in Neo4j and links it to its interface and package.
func UpsertInterfaceMethod(ctx context.Context, driver neo4j.DriverWithContext, mod codesurgeon.Module, pkg codesurgeon.Package, iface codesurgeon.Interface, method codesurgeon.Method) error {
	session := driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(ctx)

	query := CypherQuery{}.
		Merge(MergeInterfaceMethodFunction(ctx, "m", pkg, iface, method)).
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
