package neo4j2

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	codesurgeon "bitbucket.org/zetaactions/code-surgeon"
	"github.com/davecgh/go-spew/spew"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"
)

// getAllData retrieves all nodes and relationships from the Neo4j database
func getAllData(ctx context.Context, session neo4j.SessionWithContext) map[string]interface{} {
	data := make(map[string]interface{})

	// Retrieve all nodes
	nodesQuery := "MATCH (n) RETURN n"
	nodesResult, err := session.Run(ctx, nodesQuery, nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to retrieve nodes")
		return data
	}

	var nodes []interface{}
	for nodesResult.Next(ctx) {
		record := nodesResult.Record()
		node, _ := record.Get("n")
		nodes = append(nodes, node)
	}
	data["nodes"] = nodes

	// Retrieve all relationships
	relationshipsQuery := "MATCH ()-[r]->() RETURN r"
	relationshipsResult, err := session.Run(ctx, relationshipsQuery, nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to retrieve relationships")
		return data
	}

	var relationships []interface{}
	for relationshipsResult.Next(ctx) {
		record := relationshipsResult.Record()
		relationship, _ := record.Get("r")
		relationships = append(relationships, relationship)
	}
	data["relationships"] = relationships

	return data
}

/*
{
  "labels": [
    {
      "label": "Function",
      "properties": [
        {
          "property": "package",
          "type": "STRING",
          "isIndexed": false,
          "uniqueConstraint": false,
          "existenceConstraint": false
        },
        {
          "property": "file",
          "type": "STRING",
          "isIndexed": false,
          "uniqueConstraint": false,
          "existenceConstraint": false
        },
        {
          "property": "receiver",
          "type": "STRING",
          "isIndexed": false,
          "uniqueConstraint": false,
          "existenceConstraint": false
        },
        {
          "property": "documentation",
          "type": "STRING",
          "isIndexed": false,
          "uniqueConstraint": false,
          "existenceConstraint": false
        },
        {
          "property": "function",
          "type": "STRING",
          "isIndexed": false,
          "uniqueConstraint": false,
          "existenceConstraint": false
        },
        {
          "property": "name",
          "type": "STRING",
          "isIndexed": false,
          "uniqueConstraint": false,
          "existenceConstraint": false
        },
        {
          "property": "packageName",
          "type": "STRING",
          "isIndexed": false,
          "uniqueConstraint": false,
          "existenceConstraint": false
        }
      ]
    },
    {
      "label": "Type",
      "properties": [
        {
          "property": "type",
          "type": "STRING",
          "isIndexed": false,
          "uniqueConstraint": false,
          "existenceConstraint": false
        }
      ]
    },
    {
      "label": "Return",
      "properties": [
        {
          "property": "name",
          "type": "STRING",
          "isIndexed": false,
          "uniqueConstraint": false,
          "existenceConstraint": false
        },
        {
          "property": "methodName",
          "type": "STRING",
          "isIndexed": false,
          "uniqueConstraint": false,
          "existenceConstraint": false
        },
        {
          "property": "package",
          "type": "STRING",
          "isIndexed": false,
          "uniqueConstraint": false,
          "existenceConstraint": false
        },
        {
          "property": "type",
          "type": "STRING",
          "isIndexed": false,
          "uniqueConstraint": false,
          "existenceConstraint": false
        }
      ]
    },
    {
      "label": "BaseType",
      "properties": [
        {
          "property": "type",
          "type": "STRING",
          "isIndexed": false,
          "uniqueConstraint": false,
          "existenceConstraint": false
        },
        {
          "property": "package",
          "type": "STRING",
          "isIndexed": false,
          "uniqueConstraint": false,
          "existenceConstraint": false
        }
      ]
    },
    {
      "label": "Param",
      "properties": [
        {
          "property": "name",
          "type": "STRING",
          "isIndexed": false,
          "uniqueConstraint": false,
          "existenceConstraint": false
        },
        {
          "property": "methodName",
          "type": "STRING",
          "isIndexed": false,
          "uniqueConstraint": false,
          "existenceConstraint": false
        },
        {
          "property": "package",
          "type": "STRING",
          "isIndexed": false,
          "uniqueConstraint": false,
          "existenceConstraint": false
        },
        {
          "property": "type",
          "type": "STRING",
          "isIndexed": false,
          "uniqueConstraint": false,
          "existenceConstraint": false
        }
      ]
    },
    {
      "label": "Package",
      "properties": [
        {
          "property": "name",
          "type": "STRING",
          "isIndexed": false,
          "uniqueConstraint": false,
          "existenceConstraint": false
        },
        {
          "property": "package",
          "type": "STRING",
          "isIndexed": false,
          "uniqueConstraint": false,
          "existenceConstraint": false
        },
        {
          "property": "packageName",
          "type": "STRING",
          "isIndexed": false,
          "uniqueConstraint": false,
          "existenceConstraint": false
        }
      ]
    },
    {
      "label": "Struct",
      "properties": [
        {
          "property": "name",
          "type": "STRING",
          "isIndexed": false,
          "uniqueConstraint": false,
          "existenceConstraint": false
        },
        {
          "property": "package",
          "type": "STRING",
          "isIndexed": false,
          "uniqueConstraint": false,
          "existenceConstraint": false
        },
        {
          "property": "packageName",
          "type": "STRING",
          "isIndexed": false,
          "uniqueConstraint": false,
          "existenceConstraint": false
        },
        {
          "property": "documentation",
          "type": "STRING",
          "isIndexed": false,
          "uniqueConstraint": false,
          "existenceConstraint": false
        }
      ]
    }
  ],
  "relationships": [
    {
      "relationship": "RECEIVER",
      "fromLabel": "Function",
      "toLabel": "Struct"
    },
    {
      "relationship": "BELONGS_TO",
      "fromLabel": "Param",
      "toLabel": "Function"
    },
    {
      "relationship": "OF_TYPE",
      "fromLabel": "Param",
      "toLabel": "Type"
    },
    {
      "relationship": "BELONGS_TO",
      "fromLabel": "Return",
      "toLabel": "Function"
    },
    {
      "relationship": "OF_TYPE",
      "fromLabel": "Return",
      "toLabel": "Type"
    },
    {
      "relationship": "BELONGS_TO",
      "fromLabel": "Struct",
      "toLabel": "Package"
    },
    {
      "relationship": "OF_TYPE",
      "fromLabel": "Struct",
      "toLabel": "BaseType"
    }
  ]
}
*/

// TestToNeo4j_type_struct
// - assert that UnimplementedZivoAPIHandler struct is inserted into Neo4j
// - assert that SendSms function is inserted into Neo4j
// - assert that SendSms function has two parameters: ctx and r
// - assert that SendSms function has two return types: *connect.Response[FirstStruct] and error
// - assert that relationships are created between the nodes
// - assert that types are associated with params and returns
// - assert that struct has OF_TYPE relationship to BaseType if applicable
func TestToNeo4j_type_struct(t *testing.T) {
	code := `
	package test

	import (
		"context"
		"errors"
		"connect"
	)

	// UnimplementedZivoAPIHandler returns CodeUnimplemented from all methods.
	type UnimplementedZivoAPIHandler struct{}

	func (UnimplementedZivoAPIHandler) SendSms(ctx context.Context, r *connect.Request[FirstStruct]) (*connect.Response[FirstStruct], error) {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("trinity.zivo.ZivoAPI.SendSms is not implemented"))
	}
	`

	parsed, err := codesurgeon.ParseString(code)
	require.NoError(t, err)
	require.NotNil(t, parsed)
	spew.Dump(parsed)

	ctx := context.Background()
	driver, closeFn, err := Connect(ctx, "bolt://localhost:7687", "neo4j", "neo4jneo4j")
	if err != nil {
		log.Info().Err(err).Msg("Error connecting to Neo4j (proceeding anyway)")
	} else {
		defer closeFn()
	}

	if driver == nil {
		t.Fatal("Driver is nil")
	}

	session := driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	// Clear out the database before inserting new data
	_, err = session.Run(ctx, "MATCH (n) DETACH DELETE n", nil)
	require.NoError(t, err, "Failed to clear out database")

	// Insert the parsed AST into Neo4j
	_, err = toNeo4j(ctx, parsed, "test", "test", driver, false)
	require.NoError(t, err, "Failed to insert nodes into Neo4j")

	// Helper function to run a query that returns a single integer count
	runCountQuery := func(query string, params map[string]interface{}) int64 {
		res, err := session.Run(ctx, query, params)
		require.NoError(t, err, "Failed to run query: %s", query)
		require.True(t, res.Next(ctx), "No result returned for query: %s", query)
		countVal, ok := res.Record().Values[0].(int64)
		require.True(t, ok, "Result was not an int64 for query: %s", query)
		require.False(t, res.Next(ctx), "More than one record returned for query: %s", query)
		return countVal
	}

	// Assert there is a package node
	packageCount := runCountQuery("MATCH (p:Package {package:$pkg, packageName:$pkgName}) RETURN COUNT(p)", map[string]interface{}{
		"pkg":     "test",
		"pkgName": "test",
	})
	require.Equal(t, int64(1), packageCount, "There should be exactly one Package node")

	// Assert there is a struct node: UnimplementedZivoAPIHandler
	structCount := runCountQuery("MATCH (s:Struct {name:$name, package:$pkg, packageName:$pkgName}) RETURN COUNT(s)", map[string]interface{}{
		"name":    "UnimplementedZivoAPIHandler",
		"pkg":     "test",
		"pkgName": "test",
	})
	require.Equal(t, int64(1), structCount, "There should be exactly one Struct node")

	// Check documentation for the struct
	structDocCount := runCountQuery(`
		MATCH (s:Struct {name:$name, package:$pkg})
		WHERE s.documentation = "UnimplementedZivoAPIHandler returns CodeUnimplemented from all methods."
		RETURN COUNT(s)
	`, map[string]interface{}{
		"name": "UnimplementedZivoAPIHandler",
		"pkg":  "test",
	})
	require.Equal(t, int64(1), structDocCount, "Struct should have the correct documentation")

	// Assert there is a function node: SendSms
	functionCount := runCountQuery("MATCH (f:Function {name:$name, package:$pkg, packageName:$pkgName}) RETURN COUNT(f)", map[string]interface{}{
		"name":    "SendSms",
		"pkg":     "test",
		"pkgName": "test",
	})
	require.Equal(t, int64(1), functionCount, "There should be exactly one Function node")

	// Check documentation for the function (in this case, it's empty)
	functionDocCount := runCountQuery(`
		MATCH (f:Function {name:$name, package:$pkg, packageName:$pkgName})
		WHERE f.documentation = ""
		RETURN COUNT(f)
	`, map[string]interface{}{
		"name":    "SendSms",
		"pkg":     "test",
		"pkgName": "test",
	})
	require.Equal(t, int64(1), functionDocCount, "Function should have the correct (empty) documentation")

	// Assert there are param nodes for the SendSms function
	paramCount := runCountQuery("MATCH (param:Param {methodName:$methodName, package:$pkg}) RETURN COUNT(param)", map[string]interface{}{
		"methodName": "SendSms",
		"pkg":        "test",
	})
	require.Equal(t, int64(2), paramCount, "There should be exactly two Param nodes")

	// Assert there are return nodes for the SendSms function
	returnCount := runCountQuery("MATCH (r:Return {methodName:$methodName, package:$pkg}) RETURN COUNT(r)", map[string]interface{}{
		"methodName": "SendSms",
		"pkg":        "test",
	})
	require.Equal(t, int64(2), returnCount, "There should be exactly two Return nodes")

	// Now assert relationships

	// Struct belongs to Package
	structBelongsToPackageCount := runCountQuery(`
			MATCH (s:Struct {name:$structName})-[:BELONGS_TO]->(p:Package {package:$pkg})
			RETURN COUNT(s)
		`, map[string]interface{}{
		"structName": "UnimplementedZivoAPIHandler",
		"pkg":        "test",
	})
	require.Equal(t, int64(1), structBelongsToPackageCount, "Struct should belong to Package")

	// Function has a RECEIVER relationship to Struct
	functionReceiverCount := runCountQuery(`
			MATCH (f:Function {name:$funcName})-[:RECEIVER]->(s:Struct {name:$structName})
			RETURN COUNT(f)
		`, map[string]interface{}{
		"funcName":   "SendSms",
		"structName": "UnimplementedZivoAPIHandler",
	})
	require.Equal(t, int64(1), functionReceiverCount, "Function should have RECEIVER relationship to Struct")

	// Param belongs to Function
	paramBelongsToFunctionCount := runCountQuery(`
			MATCH (param:Param)<-[:HAS_PARAM]-(f:Function {name:$funcName})
			RETURN COUNT(param)
		`, map[string]interface{}{
		"funcName": "SendSms",
	})
	require.Equal(t, int64(2), paramBelongsToFunctionCount, "Params should belong to the Function")

	// Returns belong to Function
	returnBelongsToFunctionCount := runCountQuery(`
			MATCH (r:Return {methodName:$methodName})<-[:RETURNS]-(f:Function {name:$funcName})
			RETURN COUNT(r)
		`, map[string]interface{}{
		"methodName": "SendSms",
		"funcName":   "SendSms",
	})
	require.Equal(t, int64(2), returnBelongsToFunctionCount, "Returns should belong to the Function")

	// Check OF_TYPE relationships for Params
	paramOfTypeCount := runCountQuery(`
			MATCH (param:Param)-[:OF_TYPE]->(t:Type)
			WHERE param.methodName = $methodName
			RETURN COUNT(param)
		`, map[string]interface{}{
		"methodName": "SendSms",
	})
	require.Equal(t, int64(2), paramOfTypeCount, "All Params should have OF_TYPE relationship to a Type")

	// Check OF_TYPE relationships for Returns
	returnOfTypeCount := runCountQuery(`
			MATCH (r:Return)-[:OF_TYPE]->(t:Type)
			WHERE r.methodName = $methodName
			RETURN COUNT(r)
		`, map[string]interface{}{
		"methodName": "SendSms",
	})
	require.Equal(t, int64(2), returnOfTypeCount, "All Returns should have OF_TYPE relationship to a Type")

	// Verify the Param and Return types are as expected
	// For the params: ctx is "context.Context", r is "*connect.Request[FirstStruct]"
	ctxParamTypeCount := runCountQuery(`
		MATCH (param:Param {name:"ctx", methodName:$methodName})-[:OF_TYPE]->(t:Type {type:"context.Context"})
		RETURN COUNT(param)
	`, map[string]interface{}{
		"methodName": "SendSms",
	})
	require.Equal(t, int64(1), ctxParamTypeCount, "ctx param should have correct type 'context.Context'")

	rParamTypeCount := runCountQuery(`
		MATCH (param:Param {name:"r", methodName:$methodName})-[:OF_TYPE]->(t:Type {type:"*connect.Request[FirstStruct]"})
		RETURN COUNT(param)
	`, map[string]interface{}{
		"methodName": "SendSms",
	})
	require.Equal(t, int64(1), rParamTypeCount, "r param should have type '*connect.Request[FirstStruct]'")

	// For the returns: one is "*connect.Response[FirstStruct]" and the other is "error"
	responseReturnTypeCount := runCountQuery(`
		MATCH (r:Return {methodName:$methodName})-[:OF_TYPE]->(t:Type {type:"*connect.Response[FirstStruct]"})
		RETURN COUNT(r)
	`, map[string]interface{}{
		"methodName": "SendSms",
	})
	require.Equal(t, int64(1), responseReturnTypeCount, "One return should have type '*connect.Response[FirstStruct]'")

	errorReturnTypeCount := runCountQuery(`
		MATCH (r:Return {methodName:$methodName})-[:OF_TYPE]->(t:Type {type:"error"})
		RETURN COUNT(r)
	`, map[string]interface{}{
		"methodName": "SendSms",
	})
	require.Equal(t, int64(1), errorReturnTypeCount, "One return should have type 'error'")

	// Struct should have OF_TYPE relationship to BaseType if applicable
	structOfTypeCount := runCountQuery(`
			MATCH (s:Struct {name:$structName})-[:OF_TYPE]->(b:BaseType)
			RETURN COUNT(b)
		`, map[string]interface{}{
		"structName": "UnimplementedZivoAPIHandler",
	})
	require.True(t, structOfTypeCount > 0, "Struct should have OF_TYPE relationship to BaseType")

	log.Info().Msgf("Number of BaseType relationships for UnimplementedZivoAPIHandler: %d", structOfTypeCount)

	// Optional: Check total counts of known labels to ensure no unexpected nodes
	// (Adjust these expected counts if your toNeo4j logic changes)
	totalStructs := runCountQuery("MATCH (s:Struct) RETURN COUNT(s)", nil)
	require.Equal(t, int64(1), totalStructs, "Only one struct expected")

	totalFunctions := runCountQuery("MATCH (f:Function) RETURN COUNT(f)", nil)
	require.Equal(t, int64(1), totalFunctions, "Only one function expected")

	totalParams := runCountQuery("MATCH (p:Param) RETURN COUNT(p)", nil)
	require.Equal(t, int64(2), totalParams, "Only two params expected")

	totalReturns := runCountQuery("MATCH (r:Return) RETURN COUNT(r)", nil)
	require.Equal(t, int64(2), totalReturns, "Only two returns expected")

	totalTypes := runCountQuery("MATCH (t:Type) RETURN COUNT(t)", nil)
	require.True(t, totalTypes >= 4, "At least 4 Type nodes expected: context.Context, *connect.Request[FirstStruct], *connect.Response[FirstStruct], error")

	totalBaseTypes := runCountQuery("MATCH (b:BaseType) RETURN COUNT(b)", nil)
	require.True(t, totalBaseTypes >= 1, "At least one BaseType node expected")

	// If we reach this point, all assertions have passed successfully.
}

/*
package test

// VanillaStruct is a simple struct with all builtin types

	type VanillaStruct struct {
		Name          string
		Age           int
		IsAdmin       bool
		privateField  string
		PointerString *string
		StringSlice   []string
	}
*/
func TestToNeo4j_VanillaStruct(t *testing.T) {
	code := `
	package test

	// VanillaStruct is a simple struct with all builtin types
	type VanillaStruct struct {
		Name          string
		Age           int
		IsAdmin       bool
		privateField  string
		PointerString *string
		StringSlice   []string
	}
	`

	parsed, err := codesurgeon.ParseString(code)
	require.NoError(t, err)
	require.NotNil(t, parsed)
	spew.Dump(parsed)

	ctx := context.Background()
	driver, closeFn, err := Connect(ctx, "bolt://localhost:7687", "neo4j", "neo4jneo4j")
	if err != nil {
		log.Info().Err(err).Msg("Error connecting to Neo4j (proceeding anyway)")
	} else {
		defer closeFn()
	}

	if driver == nil {
		t.Fatal("Driver is nil")
	}

	session := driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	// Clear out the database before inserting new data
	_, err = session.Run(ctx, "MATCH (n) DETACH DELETE n", nil)
	require.NoError(t, err, "Failed to clear out database")

	// Insert the parsed AST into Neo4j
	_, err = toNeo4j(ctx, parsed, "test", "test", driver, false)
	require.NoError(t, err, "Failed to insert nodes into Neo4j")

	// Helper function to run a query that returns a single integer count
	runCountQuery := func(query string, params map[string]interface{}) int64 {
		res, err := session.Run(ctx, query, params)
		require.NoError(t, err, "Failed to run query: %s", query)
		require.True(t, res.Next(ctx), "No result returned for query: %s", query)
		countVal, ok := res.Record().Values[0].(int64)
		require.True(t, ok, "Result was not an int64 for query: %s", query)
		require.False(t, res.Next(ctx), "More than one record returned for query: %s", query)
		return countVal
	}

	// Check that the struct node exists
	structCount := runCountQuery("MATCH (s:Struct {name:$name, package:$pkg, packageName:$pkgName}) RETURN COUNT(s)", map[string]interface{}{
		"name":    "VanillaStruct",
		"pkg":     "test",
		"pkgName": "test",
	})
	require.Equal(t, int64(1), structCount, "There should be exactly one Struct node for VanillaStruct")

	// Check the documentation for VanillaStruct
	structDocCount := runCountQuery(`
		MATCH (s:Struct {name:$name, package:$pkg})
		WHERE s.documentation = "VanillaStruct is a simple struct with all builtin types"
		RETURN COUNT(s)
	`, map[string]interface{}{
		"name": "VanillaStruct",
		"pkg":  "test",
	})
	require.Equal(t, int64(1), structDocCount, "Struct should have the correct documentation")

	// Verify fields
	// We have 6 fields: Name, Age, IsAdmin, privateField, PointerString, StringSlice
	fields := []string{"Name", "Age", "IsAdmin", "privateField", "PointerString", "StringSlice"}

	for _, fieldName := range fields {
		fieldCount := runCountQuery(`
			MATCH (f:Field {name:$fieldName, package:$pkg}) 
			RETURN COUNT(f)
		`, map[string]interface{}{
			"fieldName": fieldName,
			"pkg":       "test",
		})
		require.Equal(t, int64(1), fieldCount, fmt.Sprintf("Field %s should exist", fieldName))
	}

	// Verify that the struct has a HAS_FIELD relationship to each field
	for _, fieldName := range fields {
		fieldRelCount := runCountQuery(`
			MATCH (s:Struct {name:$structName})-[:HAS_FIELD]->(f:Field {name:$fieldName})
			RETURN COUNT(f)
		`, map[string]interface{}{
			"structName": "VanillaStruct",
			"fieldName":  fieldName,
		})
		require.Equal(t, int64(1), fieldRelCount, fmt.Sprintf("Struct should have HAS_FIELD relationship to %s", fieldName))
	}

	// Verify that each field has an OF_TYPE relationship to a Type node
	// Check each field's expected type:
	fieldTypes := map[string]string{
		"Name":          "string",
		"Age":           "int",
		"IsAdmin":       "bool",
		"privateField":  "string",
		"PointerString": "string",
		"StringSlice":   "string",
	}

	for fname, ftype := range fieldTypes {
		fieldTypeCount := runCountQuery(`
			MATCH (f:Field {name:$fieldName})-[:OF_TYPE]->(t:BaseType {type:$typeName})
			RETURN COUNT(f)
		`, map[string]interface{}{
			"fieldName": fname,
			"typeName":  ftype,
		})
		require.Equal(t, int64(1), fieldTypeCount, fmt.Sprintf("Field %s should have OF_TYPE relationship to BaseType %s", fname, ftype))
	}

	// Optionally, check overall counts to ensure no extra nodes or relationships
	totalStructs := runCountQuery("MATCH (s:Struct) RETURN COUNT(s)", nil)
	require.Equal(t, int64(1), totalStructs, "Only one struct expected")

	totalFields := runCountQuery("MATCH (f:Field) RETURN COUNT(f)", nil)
	require.Equal(t, int64(6), totalFields, "Six fields expected")

	// Check that we have the expected types (at least one Type node for each type)
	// We know we have these types: string, int, bool, *string, []string
	// This count might be larger if your logic creates additional Type nodes for packages.
	expectedTypes := []string{"string", "int", "bool"}
	for _, tname := range expectedTypes {
		typeCount := runCountQuery(`
			MATCH (t:BaseType {type:$typeName})
			RETURN COUNT(t)
		`, map[string]interface{}{
			"typeName": tname,
		})
		require.Equal(t, int64(1), typeCount, fmt.Sprintf("Type %s should exist", tname))
	}

	// If your code creates a BaseType for the struct:
	structOfTypeCount := runCountQuery(`
		MATCH (s:Struct {name:$structName})-[:OF_TYPE]->(b:BaseType)
		RETURN COUNT(b)
	`, map[string]interface{}{
		"structName": "VanillaStruct",
	})
	require.True(t, structOfTypeCount > 0, "VanillaStruct should have OF_TYPE relationship to BaseType")

	log.Info().Msg("TestToNeo4j_VanillaStruct passed all checks.")
}

func TestToNeo4j_Func(t *testing.T) {
	code := `
	package test

	type Person struct {
		Name string
	}

	func SomeFunc(strParam string, strPointer *string, p Person) (error, *Person) {
		return nil, nil
	}
	`

	parsed, err := codesurgeon.ParseString(code)
	require.NoError(t, err)
	require.NotNil(t, parsed)
	spew.Dump(parsed)

	ctx := context.Background()
	driver, closeFn, err := Connect(ctx, "bolt://localhost:7687", "neo4j", "neo4jneo4j")
	if err != nil {
		log.Info().Err(err).Msg("Error connecting to Neo4j (proceeding anyway)")
	} else {
		defer closeFn()
	}

	if driver == nil {
		t.Fatal("Driver is nil")
	}

	session := driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	// Clear out the database before inserting new data
	_, err = session.Run(ctx, "MATCH (n) DETACH DELETE n", nil)
	require.NoError(t, err, "Failed to clear out database")

	// Insert the parsed AST into Neo4j
	_, err = toNeo4j(ctx, parsed, "test", "test", driver, false)
	require.NoError(t, err, "Failed to insert nodes into Neo4j")

	allData := getAllData(ctx, session)
	allDataEncoded, _ := json.MarshalIndent(allData, "", "  ")
	fmt.Println("allData", string(allDataEncoded))

	// Helper function to run a query that returns a single integer count
	runCountQuery := func(query string, params map[string]interface{}) int64 {
		log.Print("runCountQuery", query, params)
		res, err := session.Run(ctx, query, params)
		require.NoError(t, err, "Failed to run query: %s", query)
		require.True(t, res.Next(ctx), "No result returned for query: %s", query)
		countVal, ok := res.Record().Values[0].(int64)
		require.True(t, ok, "Result was not an int64 for query: %s", query)
		require.False(t, res.Next(ctx), "More than one record returned for query: %s", query)
		return countVal
	}

	// Check that the Person struct node exists
	structCount := runCountQuery("MATCH (s:Struct {name:$name, package:$pkg, packageName:$pkgName}) RETURN COUNT(s)", map[string]interface{}{
		"name":    "Person",
		"pkg":     "test",
		"pkgName": "test",
	})
	require.Equal(t, int64(1), structCount, "There should be exactly one Struct node for Person")

	// Check Person struct fields
	fields := []string{"Name"}
	for _, fieldName := range fields {
		fieldCount := runCountQuery(`
			MATCH (f:Field {name:$fieldName, package:$pkg}) 
			RETURN COUNT(f)
		`, map[string]interface{}{
			"fieldName": fieldName,
			"pkg":       "test",
		})
		require.Equal(t, int64(1), fieldCount, fmt.Sprintf("Field %s should exist", fieldName))

		fieldRelCount := runCountQuery(`
			MATCH (s:Struct {name:$structName})-[:HAS_FIELD]->(f:Field {name:$fieldName})
			RETURN COUNT(f)
		`, map[string]interface{}{
			"structName": "Person",
			"fieldName":  fieldName,
		})
		require.Equal(t, int64(1), fieldRelCount, fmt.Sprintf("Struct Person should have HAS_FIELD relationship to %s", fieldName))

		fieldTypeCount := runCountQuery(`
			MATCH (f:Field {name:$fieldName})-[:OF_TYPE]->(b:BaseType {type:"string"})
			RETURN COUNT(f)
		`, map[string]interface{}{
			"fieldName": fieldName,
		})
		require.Equal(t, int64(1), fieldTypeCount, fmt.Sprintf("Field %s should have OF_TYPE relationship to BaseType string", fieldName))
	}

	// Check SomeFunc function node
	functionCount := runCountQuery("MATCH (f:Function {name:$name, package:$pkg, packageName:$pkgName}) RETURN COUNT(f)", map[string]interface{}{
		"name":    "SomeFunc",
		"pkg":     "test",
		"pkgName": "test",
	})
	require.Equal(t, int64(1), functionCount, "There should be exactly one Function node for SomeFunc")

	// SomeFunc has 3 parameters: strParam, strPointer, p
	params := []struct {
		Name string
		Type string
	}{
		{"strParam", "string"},
		{"strPointer", "*string"},
		{"p", "Person"},
	}

	for _, param := range params {
		paramCount := runCountQuery(`
			MATCH (param:Param {name:$paramName, package:$pkg, methodName:$methodName}) RETURN COUNT(param)
		`, map[string]interface{}{
			"paramName":  param.Name,
			"pkg":        "test",
			"methodName": "SomeFunc",
		})
		require.Equal(t, int64(1), paramCount, fmt.Sprintf("Parameter %s should exist", param.Name))

		paramTypeCount := runCountQuery(`
			MATCH (param:Param {name:$paramName, methodName:$methodName})-[:OF_TYPE]->(t:Type {type:$typeName})
			RETURN COUNT(param)
		`, map[string]interface{}{
			"paramName":  param.Name,
			"methodName": "SomeFunc",
			"typeName":   param.Type,
		})
		require.Equal(t, int64(1), paramTypeCount, fmt.Sprintf("Parameter %s should have type %s", param.Name, param.Type))
	}

	// SomeFunc returns (error, *Person)
	returns := []string{"error", "*Person"}
	for _, ret := range returns {
		returnCount := runCountQuery(`
			MATCH (f:Function {name:$methodName})-[:RETURNS]->(r:Return {type:$typeName, package:$pkg})-[:OF_TYPE]->(t:Type {type:$typeName})
			RETURN COUNT(r)
		`, map[string]interface{}{
			"methodName": "SomeFunc",
			"pkg":        "test",
			"typeName":   ret,
		})
		require.Equal(t, int64(1), returnCount, fmt.Sprintf("SomeFunc should return %s", ret))
	}

	// Verify relationships
	// Function belongs to package
	funcBelongsToPackageCount := runCountQuery(`
		MATCH (f:Function {name:$funcName})-[:BELONGS_TO]->(p:Package {package:$pkg})
		RETURN COUNT(f)
	`, map[string]interface{}{
		"funcName": "SomeFunc",
		"pkg":      "test",
	})
	require.Equal(t, int64(1), funcBelongsToPackageCount, "Function should belong to Package")

	// Params belong to Function
	paramCount := runCountQuery(`
		MATCH (f:Function {name:$funcName})-[:HAS_PARAM]->(param:Param)
		RETURN COUNT(param)
	`, map[string]interface{}{
		"funcName": "SomeFunc",
	})
	require.Equal(t, int64(3), paramCount, "SomeFunc should have three parameters")

	// Returns belong to Function
	returnCount := runCountQuery(`
		MATCH (f:Function {name:$funcName})-[:RETURNS]->(r:Return)
		RETURN COUNT(r)
	`, map[string]interface{}{
		"funcName": "SomeFunc",
	})
	require.Equal(t, int64(2), returnCount, "SomeFunc should have two returns")

	// Struct (Person) should have OF_TYPE relationship to BaseType if applicable
	structOfTypeCount := runCountQuery(`
		MATCH (s:Struct {name:"Person"})-[:OF_TYPE]->(b:BaseType)
		RETURN COUNT(b)
	`, nil)
	require.True(t, structOfTypeCount > 0, "Person should have OF_TYPE relationship to BaseType")

	// Define base types for verification
	baseTypes := map[string]string{
		"string":  "string",
		"error":   "error",
		"Person":  "Person",
		"*string": "string",
		"*Person": "Person",
	}

	// Check that each Return has an OF_TYPE relationship to a Type node
	for _, ret := range returns {
		// Verify the OF_TYPE relationship from Return to Type
		returnOfTypeCount := runCountQuery(`
			MATCH (f:Function {name:$funcName, package:$pkg, packageName:$pkgName})-[:RETURNS]->(r:Return {type:$typeName, package:$pkg})-[:OF_TYPE]->(t:Type {type:$typeName})
			RETURN COUNT(r)
		`, map[string]interface{}{
			"funcName": "SomeFunc",
			"pkg":      "test",
			"pkgName":  "test",
			"typeName": ret,
		})
		require.Equal(t, int64(1), returnOfTypeCount, fmt.Sprintf("Return type '%s' should have an OF_TYPE relationship to Type node", ret))

		// If the type is a base type, verify the BASE_TYPE relationship to BaseType
		if baseType, exists := baseTypes[ret]; exists {
			baseTypeCount := runCountQuery(`
				MATCH (t:Type {type:$typeName})-[:BASE_TYPE]->(b:BaseType {type:$baseType})
				RETURN COUNT(b)
			`, map[string]interface{}{
				"typeName": ret,
				"baseType": baseType,
			})
			require.Equal(t, int64(1), baseTypeCount, fmt.Sprintf("Type '%s' should have a BASE_TYPE relationship to BaseType node", ret))
		} else {
			// If it's not a base type, optionally verify that no BASE_TYPE relationship exists
			// This ensures that user-defined types do not incorrectly link to BaseType
			noBaseTypeCount := runCountQuery(`
				MATCH (t:Type {type:$typeName})-[:BASE_TYPE]->(b:BaseType)
				RETURN COUNT(b)
			`, map[string]interface{}{
				"typeName": ret,
			})
			require.Equal(t, int64(0), noBaseTypeCount, fmt.Sprintf("Type '%s' should NOT have a BASE_TYPE relationship to any BaseType node", ret))
		}
	}

	log.Info().Msg("TestToNeo4j_Func passed all checks.")

}

func TestToNeo4j_SharedTypes(t *testing.T) {
	code := `
	package test

	type Person struct {
		Name string
		Birth BirthInfo
	}

	type Animal struct {
		Name string
		Info BirthInfo
	}

	func SomeFunc(p *Person) (*Animal, error) {
		return nil, nil
	}

	func AnotherFunc(a Animal)  {
	}
	`

	parsed, err := codesurgeon.ParseString(code)
	require.NoError(t, err)
	require.NotNil(t, parsed)
	spew.Dump(parsed)

	ctx := context.Background()
	driver, closeFn, err := Connect(ctx, "bolt://localhost:7687", "neo4j", "neo4jneo4j")
	if err != nil {
		log.Info().Err(err).Msg("Error connecting to Neo4j (proceeding anyway)")
	} else {
		defer closeFn()
	}

	if driver == nil {
		t.Fatal("Driver is nil")
	}

	session := driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	// Clear out the database before inserting new data
	_, err = session.Run(ctx, "MATCH (n) DETACH DELETE n", nil)
	require.NoError(t, err, "Failed to clear out database")

	// Insert the parsed AST into Neo4j
	_, err = toNeo4j(ctx, parsed, "test", "test", driver, false)
	require.NoError(t, err, "Failed to insert nodes into Neo4j")

	allData := getAllData(ctx, session)
	allDataEncoded, _ := json.MarshalIndent(allData, "", "  ")
	fmt.Println("allData", string(allDataEncoded))

	// // Helper function to run a query that returns a single integer count
	// runCountQuery := func(query string, params map[string]interface{}) int64 {
	// 	res, err := session.Run(ctx, query, params)
	// 	require.NoError(t, err, "Failed to run query: %s", query)
	// 	require.True(t, res.Next(ctx), "No result returned for query: %s", query)
	// 	countVal, ok := res.Record().Values[0].(int64)
	// 	require.True(t, ok, "Result was not an int64 for query: %s", query)
	// 	require.False(t, res.Next(ctx), "More than one record returned for query: %s", query)
	// 	return countVal
	// }

}

func TestToNeo4j_Advanced(t *testing.T) {
	code := `
	package test

	import "github.com/company/proj"

	type Person struct {
		Name string
	}

	func SomeFunc(animalParam proj.Animal, pAnimalParam *proj.Animal, person Person, pPerson *Person) (Person, *Person, proj.Animal, *proj.Animal, error) {
		return Person{}, &Person{}, proj.Animal{}, &proj.Animal{}, nil
	}
	`

	parsed, err := codesurgeon.ParseString(code)
	require.NoError(t, err)
	require.NotNil(t, parsed)
	spew.Dump(parsed)

	ctx := context.Background()
	driver, closeFn, err := Connect(ctx, "bolt://localhost:7687", "neo4j", "neo4jneo4j")
	if err != nil {
		log.Info().Err(err).Msg("Error connecting to Neo4j (proceeding anyway)")
	} else {
		defer closeFn()
	}

	if driver == nil {
		t.Fatal("Driver is nil")
	}

	session := driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	// Clear out the database before inserting new data
	_, err = session.Run(ctx, "MATCH (n) DETACH DELETE n", nil)
	require.NoError(t, err, "Failed to clear out database")

	// Insert the parsed AST into Neo4j
	_, err = toNeo4j(ctx, parsed, "test", "test", driver, false)
	require.NoError(t, err, "Failed to insert nodes into Neo4j")

	// Helper function to run a query that returns a single integer count
	runCountQuery := func(query string, params map[string]interface{}) int64 {
		log.Print("runCountQuery", query, params)
		res, err := session.Run(ctx, query, params)
		require.NoError(t, err, "Failed to run query: %s", query)
		require.True(t, res.Next(ctx), "No result returned for query: %s", query)
		countVal, ok := res.Record().Values[0].(int64)
		require.True(t, ok, "Result was not an int64 for query: %s", query)
		require.False(t, res.Next(ctx), "More than one record returned for query: %s", query)
		return countVal
	}

	// Assert Package node
	packageCount := runCountQuery(`
		MATCH (p:Package {package:$pkg, packageName:$pkgName})
		RETURN COUNT(p)
	`, map[string]interface{}{
		"pkg":     "test",
		"pkgName": "test",
	})
	require.Equal(t, int64(1), packageCount, "There should be exactly one Package node")

	// Assert Struct node: Person
	personStructCount := runCountQuery(`
		MATCH (s:Struct {name:"Person", package:"test", packageName:"test"}) 
		RETURN COUNT(s)
	`, nil)
	require.Equal(t, int64(1), personStructCount, "Person struct should be present")

	// Check that Person has a field: Name
	nameFieldCount := runCountQuery(`
		MATCH (s:Struct {name:"Person"})-[:HAS_FIELD]->(f:Field {name:"Name"})
		RETURN COUNT(f)
	`, nil)
	require.Equal(t, int64(1), nameFieldCount, "Person struct should have a field 'Name'")

	// Check that the field 'Name' has OF_TYPE relationship to BaseType string
	nameFieldTypeCount := runCountQuery(`
		MATCH (f:Field {name:"Name"})-[:OF_TYPE]->(b:BaseType {type:"string"})
		RETURN COUNT(f)
	`, nil)
	require.Equal(t, int64(1), nameFieldTypeCount, "Name field should be of type string")

	// Assert Function node: SomeFunc
	funcCount := runCountQuery(`
		MATCH (f:Function {name:"SomeFunc", package:"test", packageName:"test"})
		RETURN COUNT(f)
	`, nil)
	require.Equal(t, int64(1), funcCount, "SomeFunc function node should be present")

	// Verify parameters of SomeFunc
	params := []struct {
		Name string
		Type string
	}{
		{"animalParam", "proj.Animal"},
		{"pAnimalParam", "*proj.Animal"},
		{"person", "Person"},
		{"pPerson", "*Person"},
	}

	for _, param := range params {
		paramCount := runCountQuery(`
			MATCH (param:Param {name:$paramName, package:$pkg, methodName:$methodName}) 
			RETURN COUNT(param)
		`, map[string]interface{}{
			"paramName":  param.Name,
			"pkg":        "test",
			"methodName": "SomeFunc",
		})
		require.Equal(t, int64(1), paramCount, fmt.Sprintf("Parameter %s should exist", param.Name))

		paramTypeCount := runCountQuery(`
			MATCH (param:Param {name:$paramName, methodName:$methodName})-[:OF_TYPE]->(t:Type {type:$typeName})
			RETURN COUNT(param)
		`, map[string]interface{}{
			"paramName":  param.Name,
			"methodName": "SomeFunc",
			"typeName":   param.Type,
		})
		require.Equal(t, int64(1), paramTypeCount, fmt.Sprintf("Parameter %s should have type %s", param.Name, param.Type))
	}

	// Verify returns of SomeFunc: (Person, *Person, proj.Animal, *proj.Animal, error)
	returnTypes := []string{
		"Person",
		"*Person",
		"proj.Animal",
		"*proj.Animal",
		"error",
	}

	for _, ret := range returnTypes {
		returnCount := runCountQuery(`
			MATCH (f:Function {name:$methodName, package:$pkg})-[:RETURNS]->(r:Return {type:$typeName})-[:OF_TYPE]->(t:Type {type:$typeName})
			RETURN COUNT(r)
		`, map[string]interface{}{
			"methodName": "SomeFunc",
			"pkg":        "test",
			"typeName":   ret,
		})
		require.Equal(t, int64(1), returnCount, fmt.Sprintf("SomeFunc should return %s", ret))
	}

	// Check relationships: Function belongs to package
	funcBelongsToPackageCount := runCountQuery(`
		MATCH (f:Function {name:"SomeFunc"})-[:BELONGS_TO]->(p:Package {package:"test"})
		RETURN COUNT(f)
	`, nil)
	require.Equal(t, int64(1), funcBelongsToPackageCount, "Function should belong to Package")

	// Params belong to Function
	paramCount := runCountQuery(`
		MATCH (f:Function {name:"SomeFunc"})-[:HAS_PARAM]->(param:Param)
		RETURN COUNT(param)
	`, nil)
	require.Equal(t, int64(len(params)), paramCount, "SomeFunc should have all parameters")

	// Returns belong to Function
	someFuncReturnCount := runCountQuery(`
		MATCH (f:Function {name:"SomeFunc"})-[:RETURNS]->(r:Return)
		RETURN COUNT(r)
	`, nil)
	require.Equal(t, int64(len(returnTypes)), someFuncReturnCount, "SomeFunc should have all expected returns")

	// Check OF_TYPE and BASE_TYPE relationships
	// For simplicity, let's define what we consider base types here.
	// Adjust as needed based on your toNeo4j implementation.
	baseTypes := map[string]string{
		"string":       "string",
		"error":        "error",
		"Person":       "Person",
		"proj.Animal":  "proj.Animal",
		"*string":      "string",
		"*Person":      "Person",
		"*proj.Animal": "proj.Animal",
	}
	// Person and proj.Animal are user-defined (non-base) types.
	// However, if your logic maps them to base types or custom logic, you can adjust accordingly.

	// Check Returns' OF_TYPE and BASE_TYPE
	for _, ret := range returnTypes {
		// OF_TYPE relationship tested above. Now check BASE_TYPE if applicable.
		if baseT, isBase := baseTypes[ret]; isBase {
			baseTypeCount := runCountQuery(`
				MATCH (t:Type {type:$typeName})-[:BASE_TYPE]->(b:BaseType {type:$baseType})
				RETURN COUNT(b)
			`, map[string]interface{}{
				"typeName": ret,
				"baseType": baseT,
			})
			require.Equal(t, int64(1), baseTypeCount, fmt.Sprintf("Type '%s' should have a BASE_TYPE relationship to BaseType %s", ret, baseT))
		} else {
			// Non-base types should not have a BASE_TYPE relationship (unless your logic defines otherwise)
			noBaseTypeCount := runCountQuery(`
				MATCH (t:Type {type:$typeName})-[:BASE_TYPE]->(b:BaseType)
				RETURN COUNT(b)
			`, map[string]interface{}{
				"typeName": ret,
			})
			require.Equal(t, int64(0), noBaseTypeCount, fmt.Sprintf("Type '%s' should not have a BASE_TYPE relationship", ret))
		}
	}

	// Ensure Person struct still has an OF_TYPE to BaseType if applicable
	personStructOfTypeCount := runCountQuery(`
		MATCH (s:Struct {name:"Person"})-[:OF_TYPE]->(b:BaseType)
		RETURN COUNT(b)
	`, nil)
	require.True(t, personStructOfTypeCount > 0, "Person struct should have OF_TYPE relationship to BaseType")

	log.Info().Msg("TestToNeo4j_Advanced passed all checks.")
}
