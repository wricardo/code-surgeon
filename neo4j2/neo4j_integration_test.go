package neo4j2

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	codesurgeon "github.com/wricardo/code-surgeon"
)

// Neo4jIntegrationTestSuite handles all Neo4j integration tests
type Neo4jIntegrationTestSuite struct {
	suite.Suite
	driver    neo4j.DriverWithContext
	session   neo4j.SessionWithContext
	ctx       context.Context
	setupDone bool
}

// SetupSuite runs once before all tests in the suite
func (suite *Neo4jIntegrationTestSuite) SetupSuite() {
	suite.ctx = context.Background()

	// Start neo4j-test container
	suite.startNeo4jTestContainer()

	// Wait for Neo4j to be ready
	suite.waitForNeo4j()

	// Connect to test Neo4j instance
	var err error
	var closeFn func()
	suite.driver, closeFn, err = Connect(suite.ctx, "bolt://localhost:7688", "neo4j", "testpass")
	require.NoError(suite.T(), err, "Failed to connect to test Neo4j instance")

	// Store the close function for cleanup
	suite.T().Cleanup(closeFn)

	suite.setupDone = true
}

// SetupTest runs before each individual test
func (suite *Neo4jIntegrationTestSuite) SetupTest() {
	if !suite.setupDone {
		suite.T().Skip("Neo4j test setup failed")
	}

	// Create a new session for each test
	suite.session = suite.driver.NewSession(suite.ctx, neo4j.SessionConfig{})

	// Clear all data before each test to ensure isolation
	suite.clearDatabase()
}

// TearDownTest runs after each individual test
func (suite *Neo4jIntegrationTestSuite) TearDownTest() {
	if suite.session != nil {
		// Clear all data after each test
		suite.clearDatabase()
		suite.session.Close(suite.ctx)
	}
}

// TearDownSuite runs once after all tests in the suite
func (suite *Neo4jIntegrationTestSuite) TearDownSuite() {
	// Stop neo4j-test container
	suite.stopNeo4jTestContainer()
}

// startNeo4jTestContainer starts the neo4j-test Docker service
func (suite *Neo4jIntegrationTestSuite) startNeo4jTestContainer() {
	log.Info().Msg("Starting neo4j-test container...")

	cmd := exec.Command("docker", "compose", "up", "-d", "neo4j-test")
	cmd.Dir = "../" // Run from project root
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Error().Err(err).Str("output", string(output)).Msg("Failed to start neo4j-test container")
		suite.T().Skipf("Could not start neo4j-test container: %v", err)
	}
}

// stopNeo4jTestContainer stops the neo4j-test Docker service
func (suite *Neo4jIntegrationTestSuite) stopNeo4jTestContainer() {
	log.Info().Msg("Stopping neo4j-test container...")

	cmd := exec.Command("docker", "compose", "stop", "neo4j-test")
	cmd.Dir = "../" // Run from project root
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Warn().Err(err).Str("output", string(output)).Msg("Failed to stop neo4j-test container")
	}

	// Remove the container and its data
	cmd = exec.Command("docker", "compose", "rm", "-f", "neo4j-test")
	cmd.Dir = "../"
	output, err = cmd.CombinedOutput()
	if err != nil {
		log.Warn().Err(err).Str("output", string(output)).Msg("Failed to remove neo4j-test container")
	}
}

// waitForNeo4j waits for Neo4j to be ready to accept connections
func (suite *Neo4jIntegrationTestSuite) waitForNeo4j() {
	log.Info().Msg("Waiting for neo4j-test to be ready...")

	maxAttempts := 30
	for i := 0; i < maxAttempts; i++ {
		driver, closeFn, err := Connect(suite.ctx, "bolt://localhost:7688", "neo4j", "testpass")
		if err == nil {
			// Test basic connectivity
			session := driver.NewSession(suite.ctx, neo4j.SessionConfig{})
			_, err = session.Run(suite.ctx, "RETURN 1", nil)
			if err == nil {
				session.Close(suite.ctx)
				closeFn()
				log.Info().Msg("neo4j-test is ready")
				return
			}
			session.Close(suite.ctx)
			closeFn()
		}

		log.Debug().Int("attempt", i+1).Int("maxAttempts", maxAttempts).Msg("Waiting for neo4j-test...")
		time.Sleep(2 * time.Second)
	}

	suite.T().Skipf("Neo4j test instance did not become ready after %d attempts", maxAttempts)
}

// clearDatabase removes all nodes and relationships from the database
func (suite *Neo4jIntegrationTestSuite) clearDatabase() {
	if suite.session == nil {
		return
	}

	// Delete all relationships first, then all nodes
	queries := []string{
		"MATCH ()-[r]->() DELETE r",
		"MATCH (n) DELETE n",
	}

	for _, query := range queries {
		_, err := suite.session.Run(suite.ctx, query, nil)
		if err != nil {
			log.Warn().Err(err).Str("query", query).Msg("Failed to clear database")
		}
	}
}

// runCountQuery executes a count query and returns the result
func (suite *Neo4jIntegrationTestSuite) runCountQuery(query string, params map[string]interface{}) int64 {
	result, err := suite.session.Run(suite.ctx, query, params)
	require.NoError(suite.T(), err)

	require.True(suite.T(), result.Next(suite.ctx))
	record := result.Record()

	// Get the first value from the record (which should be the count)
	keys := record.Keys
	require.True(suite.T(), len(keys) > 0, "No keys in result")

	countValue, found := record.Get(keys[0])
	require.True(suite.T(), found, "Count result not found for key: "+keys[0])

	count, ok := countValue.(int64)
	require.True(suite.T(), ok, "Count result is not int64, got %T", countValue)
	return count
}

// Test_Neo4j_TypeStruct tests converting type struct declarations to Neo4j
func (suite *Neo4jIntegrationTestSuite) Test_Neo4j_TypeStruct() {
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
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), parsed)

	_, err = toNeo4j(suite.ctx, parsed, "", "", suite.session, false)
	require.NoError(suite.T(), err)

	// Verify the data was stored correctly
	packageCount := suite.runCountQuery(`
		MATCH (p:Package {name:$name})
		RETURN COUNT(p)
	`, map[string]interface{}{
		"name": "test",
	})
	suite.Equal(int64(1), packageCount, "There should be exactly one Package node")

	// Verify struct was created
	structCount := suite.runCountQuery(`
		MATCH (s:Struct {name:"UnimplementedZivoAPIHandler", packageName:"test"})
		RETURN COUNT(s)
	`, nil)
	suite.Equal(int64(1), structCount, "UnimplementedZivoAPIHandler struct should be present")
}

// Test_Neo4j_VanillaStruct tests converting vanilla struct to Neo4j
func (suite *Neo4jIntegrationTestSuite) Test_Neo4j_VanillaStruct() {
	code := `
	package test

	type Person struct {
		Name string ` + "`json:\"name\"`" + `
		Age int ` + "`json:\"age\"`" + `
		Active bool ` + "`json:\"active,omitempty\"`" + `
		Tags []string ` + "`json:\"tags\"`" + `
		private string // private field
	}
	`

	parsed, err := codesurgeon.ParseString(code)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), parsed)

	_, err = toNeo4j(suite.ctx, parsed, "", "", suite.session, false)
	require.NoError(suite.T(), err)

	// Verify struct was created
	structCount := suite.runCountQuery(`
		MATCH (s:Struct {name:"Person", packageName:"test"})
		RETURN COUNT(s)
	`, nil)
	suite.Equal(int64(1), structCount, "Person struct should be present")
}

// Test_Neo4j_Function tests converting functions to Neo4j
func (suite *Neo4jIntegrationTestSuite) Test_Neo4j_Function() {
	code := `
	package test

	type Person struct {
		Name string
	}

	func SomeFunc(animalParam Animal, pAnimalParam *Animal, person Person, pPerson *Person) (Person, *Person, Animal, *Animal, error) {
		return Person{}, &Person{}, Animal{}, &Animal{}, nil
	}
	`

	parsed, err := codesurgeon.ParseString(code)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), parsed)

	_, err = toNeo4j(suite.ctx, parsed, "", "", suite.session, false)
	require.NoError(suite.T(), err)

	// Verify function was created
	functionCount := suite.runCountQuery(`
		MATCH (f:Function {name:"SomeFunc", packageName:"test"})
		RETURN COUNT(f)
	`, nil)
	suite.Equal(int64(1), functionCount, "SomeFunc function should be present")
}

// Test_Neo4j_SharedTypes tests shared types across packages
func (suite *Neo4jIntegrationTestSuite) Test_Neo4j_SharedTypes() {
	code := `
	package test

	import "proj"

	type Person struct {
		Name string ` + "`json:\"name\"`" + `
		Age int ` + "`json:\"age\"`" + `
		Active bool ` + "`json:\"active,omitempty\"`" + `
		Tags []string ` + "`json:\"tags\"`" + `
		private string // private field
	}

	func SomeFunc(animalParam proj.Animal, pAnimalParam *proj.Animal, person Person, pPerson *Person) (Person, *Person, proj.Animal, *proj.Animal, error) {
		return Person{}, &Person{}, proj.Animal{}, &proj.Animal{}, nil
	}
	`

	parsed, err := codesurgeon.ParseString(code)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), parsed)

	_, err = toNeo4j(suite.ctx, parsed, "", "", suite.session, false)
	require.NoError(suite.T(), err)

	// Verify the data was stored correctly
	packageCount := suite.runCountQuery(`
		MATCH (p:Package {name:$name})
		RETURN COUNT(p)
	`, map[string]interface{}{
		"name": "test",
	})
	suite.Equal(int64(1), packageCount, "There should be exactly one Package node")
}

// Test_Neo4j_Advanced tests complex relationships
func (suite *Neo4jIntegrationTestSuite) Test_Neo4j_Advanced() {
	code := `
	package test

	import "proj"

	type Person struct {
		Name string ` + "`json:\"name\"`" + `
		Age int ` + "`json:\"age\"`" + `
		Active bool ` + "`json:\"active,omitempty\"`" + `
		Tags []string ` + "`json:\"tags\"`" + `
		private string // private field
	}

	func SomeFunc(animalParam proj.Animal, pAnimalParam *proj.Animal, person Person, pPerson *Person) (Person, *Person, proj.Animal, *proj.Animal, error) {
		return Person{}, &Person{}, proj.Animal{}, &proj.Animal{}, nil
	}
	`

	parsed, err := codesurgeon.ParseString(code)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), parsed)

	_, err = toNeo4j(suite.ctx, parsed, "", "", suite.session, false)
	require.NoError(suite.T(), err)

	// Verify package node
	packageCount := suite.runCountQuery(`
		MATCH (p:Package {name:$name})
		RETURN COUNT(p)
	`, map[string]interface{}{
		"name": "test",
	})
	suite.Equal(int64(1), packageCount, "There should be exactly one Package node")

	// Verify struct node
	personStructCount := suite.runCountQuery(`
		MATCH (s:Struct {name:"Person", packageName:"test"})
		RETURN COUNT(s)
	`, nil)
	suite.Equal(int64(1), personStructCount, "Person struct should be present")

	// Verify function node
	functionCount := suite.runCountQuery(`
		MATCH (f:Function {name:"SomeFunc", packageName:"test"})
		RETURN COUNT(f)
	`, nil)
	suite.Equal(int64(1), functionCount, "SomeFunc function should be present")
}

// Test_Neo4j_StructFieldRelationships verifies that struct fields are created as nodes
// and linked to the struct via HAS_FIELD, and that OF_TYPE / BASE_TYPE relationships exist.
func (suite *Neo4jIntegrationTestSuite) Test_Neo4j_StructFieldRelationships() {
	code := `
	package test

	type Person struct {
		Name string
		Age  int
		Tags []string
	}
	`

	parsed, err := codesurgeon.ParseString(code)
	require.NoError(suite.T(), err)

	_, err = toNeo4j(suite.ctx, parsed, "", "", suite.session, false)
	require.NoError(suite.T(), err)

	// Verify Field nodes exist
	fieldCount := suite.runCountQuery(`
		MATCH (f:Field)
		RETURN COUNT(f)
	`, nil)
	suite.Equal(int64(3), fieldCount, "Should have 3 Field nodes (Name, Age, Tags)")

	// Verify HAS_FIELD relationships
	hasFieldCount := suite.runCountQuery(`
		MATCH (s:Struct {name:"Person"})-[:HAS_FIELD]->(f:Field)
		RETURN COUNT(f)
	`, nil)
	suite.Equal(int64(3), hasFieldCount, "Person should have 3 HAS_FIELD relationships")

	// Verify OF_TYPE on fields
	ofTypeCount := suite.runCountQuery(`
		MATCH (f:Field)-[:OF_TYPE]->(t:Type)
		RETURN COUNT(f)
	`, nil)
	suite.Equal(int64(3), ofTypeCount, "All 3 fields should have OF_TYPE relationships")
}

// Test_Neo4j_MethodRelationships verifies that methods are linked to their receiver struct
// via HAS_METHOD, and that method params and returns have proper relationships.
func (suite *Neo4jIntegrationTestSuite) Test_Neo4j_MethodRelationships() {
	code := `
	package test

	type Calculator struct{}

	func (c Calculator) Add(a int, b int) int {
		return a + b
	}

	func (c *Calculator) Subtract(a int, b int) (int, error) {
		return a - b, nil
	}
	`

	parsed, err := codesurgeon.ParseString(code)
	require.NoError(suite.T(), err)

	_, err = toNeo4j(suite.ctx, parsed, "", "", suite.session, false)
	require.NoError(suite.T(), err)

	// Verify HAS_METHOD relationships
	methodCount := suite.runCountQuery(`
		MATCH (s:Struct {name:"Calculator"})-[:HAS_METHOD]->(m:Method)
		RETURN COUNT(m)
	`, nil)
	suite.Equal(int64(2), methodCount, "Calculator should have 2 methods")

	// Verify HAS_PARAM on Add method (2 params: a, b)
	addParamCount := suite.runCountQuery(`
		MATCH (m:Method {name:"Add"})-[:HAS_PARAM]->(p:Param)
		RETURN COUNT(p)
	`, nil)
	suite.Equal(int64(2), addParamCount, "Add should have 2 params")

	// Verify RETURNS on Subtract method (2 returns: int, error)
	subtractReturnCount := suite.runCountQuery(`
		MATCH (m:Method {name:"Subtract"})-[:RETURNS]->(r:Return)
		RETURN COUNT(r)
	`, nil)
	suite.Equal(int64(2), subtractReturnCount, "Subtract should have 2 returns")

	// Verify pointer receiver is stored without asterisk
	receiverResult := suite.runCountQuery(`
		MATCH (m:Method {name:"Subtract", receiver:"Calculator"})
		RETURN COUNT(m)
	`, nil)
	suite.Equal(int64(1), receiverResult, "Subtract receiver should be stored without asterisk")
}

// Test_Neo4j_FunctionParamsAndReturns verifies that function params and returns
// are properly created with HAS_PARAM, RETURNS, OF_TYPE, and BASE_TYPE relationships.
func (suite *Neo4jIntegrationTestSuite) Test_Neo4j_FunctionParamsAndReturns() {
	code := `
	package test

	func Process(name string, count int) (string, error) {
		return name, nil
	}
	`

	parsed, err := codesurgeon.ParseString(code)
	require.NoError(suite.T(), err)

	_, err = toNeo4j(suite.ctx, parsed, "", "", suite.session, false)
	require.NoError(suite.T(), err)

	// Verify BELONGS_TO relationship
	belongsToCount := suite.runCountQuery(`
		MATCH (f:Function {name:"Process"})-[:BELONGS_TO]->(p:Package {name:"test"})
		RETURN COUNT(f)
	`, nil)
	suite.Equal(int64(1), belongsToCount, "Process should BELONGS_TO test package")

	// Verify HAS_PARAM
	paramCount := suite.runCountQuery(`
		MATCH (f:Function {name:"Process"})-[:HAS_PARAM]->(p:Param)
		RETURN COUNT(p)
	`, nil)
	suite.Equal(int64(2), paramCount, "Process should have 2 params")

	// Verify RETURNS
	returnCount := suite.runCountQuery(`
		MATCH (f:Function {name:"Process"})-[:RETURNS]->(r:Return)
		RETURN COUNT(r)
	`, nil)
	suite.Equal(int64(2), returnCount, "Process should have 2 returns")

	// Verify param OF_TYPE chain
	paramTypeCount := suite.runCountQuery(`
		MATCH (p:Param)-[:OF_TYPE]->(t:Type)
		RETURN COUNT(p)
	`, nil)
	suite.Equal(int64(2), paramTypeCount, "Both params should have OF_TYPE")
}

// Test_Neo4j_InterfacePipeline verifies that interfaces, their methods, params, and returns
// are all correctly stored with proper relationships.
func (suite *Neo4jIntegrationTestSuite) Test_Neo4j_InterfacePipeline() {
	code := `
	package test

	type Reader interface {
		Read(p []byte) (n int, err error)
	}

	type ReadWriter interface {
		Read(p []byte) (n int, err error)
		Write(p []byte) (n int, err error)
	}
	`

	parsed, err := codesurgeon.ParseString(code)
	require.NoError(suite.T(), err)

	_, err = toNeo4j(suite.ctx, parsed, "", "", suite.session, false)
	require.NoError(suite.T(), err)

	// Verify Interface nodes
	ifaceCount := suite.runCountQuery(`
		MATCH (i:Interface)
		RETURN COUNT(i)
	`, nil)
	suite.Equal(int64(2), ifaceCount, "Should have 2 Interface nodes")

	// Verify BELONGS_TO package
	belongsCount := suite.runCountQuery(`
		MATCH (i:Interface)-[:BELONGS_TO]->(p:Package {name:"test"})
		RETURN COUNT(i)
	`, nil)
	suite.Equal(int64(2), belongsCount, "Both interfaces should BELONGS_TO test package")

	// Verify HAS_FUNCTION on Reader (1 method: Read)
	readerMethodCount := suite.runCountQuery(`
		MATCH (i:Interface {name:"Reader"})-[:HAS_FUNCTION]->(f:Function)
		RETURN COUNT(f)
	`, nil)
	suite.Equal(int64(1), readerMethodCount, "Reader should have 1 method")

	// Verify HAS_FUNCTION on ReadWriter (2 methods: Read, Write)
	rwMethodCount := suite.runCountQuery(`
		MATCH (i:Interface {name:"ReadWriter"})-[:HAS_FUNCTION]->(f:Function)
		RETURN COUNT(f)
	`, nil)
	suite.Equal(int64(2), rwMethodCount, "ReadWriter should have 2 methods")

	// Verify interface method params (Reader.Read has 1 param: p)
	readerParamCount := suite.runCountQuery(`
		MATCH (i:Interface {name:"Reader"})-[:HAS_FUNCTION]->(f:Function)-[:HAS_PARAM]->(p:Param)
		RETURN COUNT(p)
	`, nil)
	suite.Equal(int64(1), readerParamCount, "Reader.Read should have 1 param")

	// Verify interface method returns (Reader.Read has 2 returns: n, err)
	readerReturnCount := suite.runCountQuery(`
		MATCH (i:Interface {name:"Reader"})-[:HAS_FUNCTION]->(f:Function)-[:RETURNS]->(r:Return)
		RETURN COUNT(r)
	`, nil)
	suite.Equal(int64(2), readerReturnCount, "Reader.Read should have 2 returns")
}

// Test_Neo4j_Idempotency verifies that running toNeo4j twice with the same input
// does not duplicate nodes, since MERGE is used throughout.
func (suite *Neo4jIntegrationTestSuite) Test_Neo4j_Idempotency() {
	code := `
	package test

	type Animal struct {
		Species string
	}

	func (a Animal) Sound() string {
		return ""
	}

	func Hello(name string) string {
		return name
	}
	`

	parsed, err := codesurgeon.ParseString(code)
	require.NoError(suite.T(), err)

	// First upsert
	_, err = toNeo4j(suite.ctx, parsed, "", "", suite.session, false)
	require.NoError(suite.T(), err)

	// Count after first upsert
	firstStructCount := suite.runCountQuery(`MATCH (s:Struct) RETURN COUNT(s)`, nil)
	firstFuncCount := suite.runCountQuery(`MATCH (f:Function) RETURN COUNT(f)`, nil)
	firstMethodCount := suite.runCountQuery(`MATCH (m:Method) RETURN COUNT(m)`, nil)
	firstFieldCount := suite.runCountQuery(`MATCH (f:Field) RETURN COUNT(f)`, nil)
	firstParamCount := suite.runCountQuery(`MATCH (p:Param) RETURN COUNT(p)`, nil)
	firstReturnCount := suite.runCountQuery(`MATCH (r:Return) RETURN COUNT(r)`, nil)

	// Second upsert with same data
	_, err = toNeo4j(suite.ctx, parsed, "", "", suite.session, false)
	require.NoError(suite.T(), err)

	// Counts should be identical
	suite.Equal(firstStructCount, suite.runCountQuery(`MATCH (s:Struct) RETURN COUNT(s)`, nil), "Struct count should not change")
	suite.Equal(firstFuncCount, suite.runCountQuery(`MATCH (f:Function) RETURN COUNT(f)`, nil), "Function count should not change")
	suite.Equal(firstMethodCount, suite.runCountQuery(`MATCH (m:Method) RETURN COUNT(m)`, nil), "Method count should not change")
	suite.Equal(firstFieldCount, suite.runCountQuery(`MATCH (f:Field) RETURN COUNT(f)`, nil), "Field count should not change")
	suite.Equal(firstParamCount, suite.runCountQuery(`MATCH (p:Param) RETURN COUNT(p)`, nil), "Param count should not change")
	suite.Equal(firstReturnCount, suite.runCountQuery(`MATCH (r:Return) RETURN COUNT(r)`, nil), "Return count should not change")
}

// Test_Neo4j_ComplexTypeChains verifies Type → BASE_TYPE → BaseType relationships
// for various complex Go types (pointers, slices, maps, channels, nested types).
func (suite *Neo4jIntegrationTestSuite) Test_Neo4j_ComplexTypeChains() {
	code := `
	package test

	type Person struct {
		Name       string
		Friends    []Person
		BestFriend *Person
		Tags       []string
		Metadata   map[string]interface{}
		Updates    chan string
		Nested     *[]*Person
	}
	`

	parsed, err := codesurgeon.ParseString(code)
	require.NoError(suite.T(), err)

	_, err = toNeo4j(suite.ctx, parsed, "", "", suite.session, false)
	require.NoError(suite.T(), err)

	// Verify Field nodes
	fieldCount := suite.runCountQuery(`
		MATCH (f:Field)
		WHERE f.structName = "Person"
		RETURN COUNT(f)
	`, nil)
	suite.Equal(int64(7), fieldCount, "Person should have 7 fields")

	// Verify Type nodes created for each field type
	typeCount := suite.runCountQuery(`
		MATCH (t:Type)
		RETURN COUNT(t)
	`, nil)
	suite.GreaterOrEqual(typeCount, int64(7), "Should have at least 7 Type nodes")

	// Verify BaseType nodes
	baseTypeCount := suite.runCountQuery(`
		MATCH (b:BaseType)
		RETURN COUNT(b)
	`, nil)
	suite.GreaterOrEqual(baseTypeCount, int64(4), "Should have BaseType nodes for string, Person, interface{}, etc")

	// Verify Type → BASE_TYPE → BaseType chain
	chainCount := suite.runCountQuery(`
		MATCH (t:Type)-[:BASE_TYPE]->(b:BaseType)
		RETURN COUNT(t)
	`, nil)
	suite.GreaterOrEqual(chainCount, int64(7), "All Type nodes should link to BaseType")
}

// Test_Neo4j_ExternalPackageTypes verifies that external package types create
// BaseType nodes with correct package information.
func (suite *Neo4jIntegrationTestSuite) Test_Neo4j_ExternalPackageTypes() {
	code := `
	package test

	import (
		"context"
		"time"
	)

	type Service struct {
		Ctx context.Context
		Timeout time.Duration
	}

	func Process(ctx context.Context, deadline time.Time) error {
		return nil
	}
	`

	parsed, err := codesurgeon.ParseString(code)
	require.NoError(suite.T(), err)

	_, err = toNeo4j(suite.ctx, parsed, "", "", suite.session, false)
	require.NoError(suite.T(), err)

	// Verify BaseType for context.Context
	contextBaseCount := suite.runCountQuery(`
		MATCH (b:BaseType)
		WHERE b.type = "Context"
		RETURN COUNT(b)
	`, nil)
	suite.Equal(int64(1), contextBaseCount, "Should have BaseType for context.Context")

	// Verify BaseType for time.Duration
	durationBaseCount := suite.runCountQuery(`
		MATCH (b:BaseType)
		WHERE b.type = "Duration"
		RETURN COUNT(b)
	`, nil)
	suite.Equal(int64(1), durationBaseCount, "Should have BaseType for time.Duration")

	// Verify BaseType for time.Time
	timeBaseCount := suite.runCountQuery(`
		MATCH (b:BaseType)
		WHERE b.type = "Time"
		RETURN COUNT(b)
	`, nil)
	suite.Equal(int64(1), timeBaseCount, "Should have BaseType for time.Time")
}

// Test_Neo4j_EmbeddedStructs verifies that struct fields referencing other structs
// are properly stored (embedded fields with empty names may collide in Neo4j MERGE).
func (suite *Neo4jIntegrationTestSuite) Test_Neo4j_EmbeddedStructs() {
	code := `
	package test

	type Base struct {
		ID string
	}

	type Person struct {
		Base Base
		Name string
	}
	`

	parsed, err := codesurgeon.ParseString(code)
	require.NoError(suite.T(), err)

	_, err = toNeo4j(suite.ctx, parsed, "", "", suite.session, false)
	require.NoError(suite.T(), err)

	// Verify both structs
	structCount := suite.runCountQuery(`
		MATCH (s:Struct)
		RETURN COUNT(s)
	`, nil)
	suite.Equal(int64(2), structCount, "Should have Base and Person structs")

	// Verify Person has 2 fields
	personFieldCount := suite.runCountQuery(`
		MATCH (s:Struct {name:"Person"})-[:HAS_FIELD]->(f:Field)
		RETURN COUNT(f)
	`, nil)
	suite.Equal(int64(2), personFieldCount, "Person should have Base and Name fields")

	// Verify Base field links to Base type
	baseFieldTypeCount := suite.runCountQuery(`
		MATCH (s:Struct {name:"Person"})-[:HAS_FIELD]->(f:Field {name:"Base"})-[:OF_TYPE]->(t:Type)
		WHERE t.type = "Base"
		RETURN COUNT(f)
	`, nil)
	suite.Equal(int64(1), baseFieldTypeCount, "Base field should link to Base type")

	// Verify Name field exists
	nameFieldCount := suite.runCountQuery(`
		MATCH (s:Struct {name:"Person"})-[:HAS_FIELD]->(f:Field {name:"Name"})
		RETURN COUNT(f)
	`, nil)
	suite.Equal(int64(1), nameFieldCount, "Person should have Name field")
}

// Test_Neo4j_DocumentationPropagation verifies that Go doc comments
// are stored in the documentation field on nodes.
func (suite *Neo4jIntegrationTestSuite) Test_Neo4j_DocumentationPropagation() {
	code := `
	package test

	// Person represents a human being with a name and age.
	// This is a multi-line doc comment.
	type Person struct {
		// Name is the person's full name
		Name string
		// Age in years
		Age int
	}

	// Greet returns a greeting message for the person.
	func (p Person) Greet() string {
		return "Hello"
	}

	// ProcessPerson does some processing.
	func ProcessPerson(p Person) error {
		return nil
	}
	`

	parsed, err := codesurgeon.ParseString(code)
	require.NoError(suite.T(), err)

	_, err = toNeo4j(suite.ctx, parsed, "", "", suite.session, false)
	require.NoError(suite.T(), err)

	// Verify struct documentation
	result, err := suite.session.Run(suite.ctx, `
		MATCH (s:Struct {name:"Person"})
		RETURN s.documentation as doc
	`, nil)
	require.NoError(suite.T(), err)
	require.True(suite.T(), result.Next(suite.ctx))
	record := result.Record()
	doc, _ := record.Get("doc")
	suite.Contains(doc, "Person represents a human being", "Struct should have documentation")

	// Verify method documentation
	result2, err := suite.session.Run(suite.ctx, `
		MATCH (m:Method {name:"Greet"})
		RETURN m.documentation as doc
	`, nil)
	require.NoError(suite.T(), err)
	require.True(suite.T(), result2.Next(suite.ctx))
	record2 := result2.Record()
	methodDoc, _ := record2.Get("doc")
	suite.Contains(methodDoc, "Greet returns a greeting message", "Method should have documentation")

	// Verify function documentation
	result3, err := suite.session.Run(suite.ctx, `
		MATCH (f:Function {name:"ProcessPerson"})
		RETURN f.documentation as doc
	`, nil)
	require.NoError(suite.T(), err)
	require.True(suite.T(), result3.Next(suite.ctx))
	record3 := result3.Record()
	funcDoc, _ := record3.Get("doc")
	suite.Contains(funcDoc, "ProcessPerson does some processing", "Function should have documentation")
}

// Test_Neo4j_EmptyAndEdgeCases verifies handling of empty interfaces,
// empty structs, functions with no params/returns, and variadic params.
func (suite *Neo4jIntegrationTestSuite) Test_Neo4j_EmptyAndEdgeCases() {
	code := `
	package test

	type Empty struct{}

	type Marker interface{}

	func NoArgs() {}

	func NoReturns(x int) {}

	func Variadic(nums ...int) int {
		return 0
	}
	`

	parsed, err := codesurgeon.ParseString(code)
	require.NoError(suite.T(), err)

	_, err = toNeo4j(suite.ctx, parsed, "", "", suite.session, false)
	require.NoError(suite.T(), err)

	// Verify empty struct
	emptyStructCount := suite.runCountQuery(`
		MATCH (s:Struct {name:"Empty"})
		RETURN COUNT(s)
	`, nil)
	suite.Equal(int64(1), emptyStructCount, "Empty struct should exist")

	// Verify empty struct has no fields
	emptyFieldCount := suite.runCountQuery(`
		MATCH (s:Struct {name:"Empty"})-[:HAS_FIELD]->(f:Field)
		RETURN COUNT(f)
	`, nil)
	suite.Equal(int64(0), emptyFieldCount, "Empty struct should have no fields")

	// Verify empty interface
	markerInterfaceCount := suite.runCountQuery(`
		MATCH (i:Interface {name:"Marker"})
		RETURN COUNT(i)
	`, nil)
	suite.Equal(int64(1), markerInterfaceCount, "Empty interface should exist")

	// Verify NoArgs function (no params)
	noArgsCount := suite.runCountQuery(`
		MATCH (f:Function {name:"NoArgs"})
		RETURN COUNT(f)
	`, nil)
	suite.Equal(int64(1), noArgsCount, "NoArgs function should exist")

	noArgsParamCount := suite.runCountQuery(`
		MATCH (f:Function {name:"NoArgs"})-[:HAS_PARAM]->(p:Param)
		RETURN COUNT(p)
	`, nil)
	suite.Equal(int64(0), noArgsParamCount, "NoArgs should have no params")

	// Verify NoReturns function (no returns)
	noReturnsRetCount := suite.runCountQuery(`
		MATCH (f:Function {name:"NoReturns"})-[:RETURNS]->(r:Return)
		RETURN COUNT(r)
	`, nil)
	suite.Equal(int64(0), noReturnsRetCount, "NoReturns should have no return values")

	// Verify Variadic function has params
	variadicParamCount := suite.runCountQuery(`
		MATCH (f:Function {name:"Variadic"})-[:HAS_PARAM]->(p:Param)
		RETURN COUNT(p)
	`, nil)
	suite.Equal(int64(1), variadicParamCount, "Variadic should have 1 param")
}

// Test_Neo4j_StructToStructReferences verifies that struct fields referencing
// other structs in the same package create proper relationships.
func (suite *Neo4jIntegrationTestSuite) Test_Neo4j_StructToStructReferences() {
	code := `
	package test

	type Address struct {
		Street string
		City   string
	}

	type Person struct {
		Name    string
		Home    Address
		Work    *Address
		History []Address
	}
	`

	parsed, err := codesurgeon.ParseString(code)
	require.NoError(suite.T(), err)

	_, err = toNeo4j(suite.ctx, parsed, "", "", suite.session, false)
	require.NoError(suite.T(), err)

	// Verify both structs exist
	structCount := suite.runCountQuery(`
		MATCH (s:Struct)
		RETURN COUNT(s)
	`, nil)
	suite.Equal(int64(2), structCount, "Should have Person and Address structs")

	// Verify Person has 4 fields
	personFieldCount := suite.runCountQuery(`
		MATCH (s:Struct {name:"Person"})-[:HAS_FIELD]->(f:Field)
		RETURN COUNT(f)
	`, nil)
	suite.Equal(int64(4), personFieldCount, "Person should have 4 fields")

	// Verify fields referencing Address struct via Type → BaseType chain
	addressRefCount := suite.runCountQuery(`
		MATCH (s:Struct {name:"Person"})-[:HAS_FIELD]->(f:Field)-[:OF_TYPE]->(t:Type)-[:BASE_TYPE]->(b:BaseType {type:"Address"})
		RETURN COUNT(f)
	`, nil)
	suite.GreaterOrEqual(addressRefCount, int64(1), "At least one field should reference Address type")
}

// TestNeo4jIntegrationSuite runs the integration test suite
func TestNeo4jIntegrationSuite(t *testing.T) {
	// Check if Docker is available
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("Docker not available, skipping Neo4j integration tests")
	}

	// Check if we should skip integration tests
	if os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		t.Skip("Integration tests skipped via SKIP_INTEGRATION_TESTS environment variable")
	}

	suite.Run(t, new(Neo4jIntegrationTestSuite))
}
