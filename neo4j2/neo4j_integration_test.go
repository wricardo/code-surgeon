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
		MATCH (s:Struct {name:"UnimplementedZivoAPIHandler", package:"test"})
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
		MATCH (s:Struct {name:"Person", package:"test"})
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
		MATCH (s:Struct {name:"Person", package:"test", packageName:"test"})
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
