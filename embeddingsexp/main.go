package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	openai "github.com/sashabaranov/go-openai"
)

// NodeData represents a Neo4j node with text and embedding.
type NodeData struct {
	ID        int64
	Text      string
	Embedding []float32
}

func main() {
	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found or error loading .env")
	}

	// Parse flags for mode selection
	mode := flag.String("mode", "generate-embeddings", "Mode: generate-embeddings or query")
	flag.Parse()

	// Load environment variables
	neo4jURI := "bolt://localhost:7687"
	neo4jUser := "neo4j"
	neo4jPassword := "neo4jneo4j"
	openAIKey := os.Getenv("OPENAI_API_KEY")

	if neo4jURI == "" || neo4jUser == "" || neo4jPassword == "" || openAIKey == "" {
		log.Fatal("Environment variables NEO4J_URI, NEO4J_USER, NEO4J_PASSWORD, and OPENAI_API_KEY must be set.")
	}

	// Initialize Neo4j driver
	driver, err := neo4j.NewDriverWithContext(neo4jURI, neo4j.BasicAuth(neo4jUser, neo4jPassword, ""))
	if err != nil {
		log.Fatalf("Failed to create Neo4j driver: %v", err)
	}
	defer driver.Close(context.Background())

	// Initialize OpenAI client
	openAIClient := openai.NewClient(openAIKey)

	switch *mode {
	case "generate-embeddings":
		generateEmbeddings(driver, openAIClient)
	case "generate-question-embeddings":
		generateQuestionEmbeddings(driver, openAIClient)
	case "query":
		runQueryMode(driver, openAIClient)
	default:
		log.Fatalf("Invalid mode: %s. Use 'generate-embeddings' or 'query'.", *mode)
	}
}

func generateQuestionEmbeddings(driver neo4j.DriverWithContext, openAIClient *openai.Client) {
	query := `
MATCH (q:Question)
WHERE q.text IS NOT NULL AND q.text <> ""
RETURN id(q) AS id, q.text AS text
`

	nodes, err := fetchNodes(driver, query)
	if err != nil {
		log.Fatalf("Failed to fetch question nodes: %v", err)
	}

	log.Printf("Fetched %d question nodes to process.", len(nodes))

	for _, node := range nodes {
		embedding, err := getEmbedding(openAIClient, node.Text)
		if err != nil {
			log.Printf("Failed to compute embedding for question node ID %d: %v", node.ID, err)
			continue
		}

		node.Embedding = embedding
		err = updateNodeEmbedding(driver, node)
		if err != nil {
			log.Printf("Failed to update embedding for question node ID %d: %v", node.ID, err)
		} else {
			log.Printf("Updated embedding for question node ID %d %s", node.ID, node.Text)
		}

		time.Sleep(100 * time.Millisecond) // Rate limiting
	}
	log.Println("Question embedding generation completed.")
}

// generateEmbeddings fetches documentation nodes, computes embeddings, and updates Neo4j.
func generateEmbeddings(driver neo4j.DriverWithContext, openAIClient *openai.Client) {
	query := `
// Retrieve Functions belonging directly to the specified Package
MATCH (f:Function)-[:BELONGS_TO]->(:Package)
WHERE 
    f.documentation IS NOT NULL AND 
    f.documentation <> "" 
RETURN 
    id(f) AS id, 
	f.packageFullName as packageFullName,
	f.name as name,
    f.documentation AS documentation,
	f.body as body

UNION

// Retrieve Methods belonging to Structs, which in turn belong to the specified Package
MATCH (m:Method)<-[:HAS_METHOD]-(:Struct)-[:BELONGS_TO]->(:Package)
WHERE 
    m.documentation IS NOT NULL AND 
    m.documentation <> "" 
RETURN 
    id(m) AS id, 
	m.packageFullName as packageFullName,
	m.receiver +'.'+ m.name as name,
    m.documentation AS documentation,
	m.body as body

LIMIT 5000
`

	nodes, err := fetchNodes(driver, query)
	if err != nil {
		log.Fatalf("Failed to fetch nodes: %v", err)
	}

	log.Printf("Fetched %d nodes to process.", len(nodes))

	for _, node := range nodes {
		embedding, err := getEmbedding(openAIClient, node.Text)
		if err != nil {
			log.Printf("Failed to compute embedding for node ID %d: %v", node.ID, err)
			continue
		}

		node.Embedding = embedding
		err = updateNodeEmbedding(driver, node)
		if err != nil {
			log.Printf("Failed to update embedding for node ID %d: %v", node.ID, err)
		} else {
			log.Printf("Updated embedding for node ID %d %s", node.ID, node.Text)
		}

		time.Sleep(100 * time.Millisecond) // Rate limiting
	}
	log.Println("Embedding generation completed.")
}

// runQueryMode takes user input, computes embedding, and finds similar nodes using Neo4j's cosine similarity.
func runQueryMode(driver neo4j.DriverWithContext, openAIClient *openai.Client) {
	reader := bufio.NewReader(os.Stdin)

	// Prompt for user input
	fmt.Print("Enter documentation text to search for similar functions: ")
	userInput, _ := reader.ReadString('\n')
	userInput = strings.TrimSpace(userInput)

	// Compute embedding for user input
	embedding, err := getEmbedding(openAIClient, userInput)
	if err != nil {
		log.Fatalf("Failed to compute embedding for input: %v", err)
	}

	// Create a temporary node in Neo4j with the input embedding
	tempNodeID, err := createTemporaryNode(driver, embedding)
	if err != nil {
		log.Fatalf("Failed to create temporary node: %v", err)
	}
	defer deleteTemporaryNode(driver, tempNodeID)

	// Perform cosine similarity search using GDS
	log.Println("Calculating cosine similarity...")
	similarNodes, err := findSimilarNodes(driver, tempNodeID)
	if err != nil {
		log.Fatalf("Failed to calculate similarity: %v", err)
	}

	// Print the most similar node
	if len(similarNodes) > 0 {
		for _, node := range similarNodes {
			fmt.Printf("- ID: %d\n- Documentation: %s\n- Similarity: %.4f\n\n", node.ID, node.Text, node.Similarity)
		}

	} else {
		fmt.Println("No similar functions found.")
	}
}

// fetchNodes retrieves nodes with documentation.
func fetchNodes(driver neo4j.DriverWithContext, query string) ([]NodeData, error) {
	var nodes []NodeData
	ctx := context.Background()

	session := driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	_, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		result, err := tx.Run(ctx, query, nil)
		if err != nil {
			return nil, err
		}

		for result.Next(ctx) {
			record := result.Record()
			id := record.Values[0].(int64)
			text := record.Values[3].(string) + record.Values[4].(string)

			nodes = append(nodes, NodeData{ID: id, Text: text})
		}
		return nil, result.Err()
	})

	return nodes, err
}

// getEmbedding computes embeddings using OpenAI.
func getEmbedding(client *openai.Client, text string) ([]float32, error) {
	ctx := context.Background()
	resp, err := client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Model: openai.AdaEmbeddingV2,
		Input: text,
	})
	if err != nil {
		return nil, err
	}

	embedding := resp.Data[0].Embedding
	embeddingFloat32 := make([]float32, len(embedding))
	for i, v := range embedding {
		embeddingFloat32[i] = float32(v)
	}

	return embeddingFloat32, nil
}

// updateNodeEmbedding updates a Neo4j node with an embedding.
func updateNodeEmbedding(driver neo4j.DriverWithContext, node NodeData) error {
	ctx := context.Background()
	session := driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `MATCH (n) WHERE id(n) = $id SET n.embedding = $embedding`
		params := map[string]any{"id": node.ID, "embedding": node.Embedding}
		_, err := tx.Run(ctx, query, params)
		return nil, err
	})
	return err
}

// createTemporaryNode adds a temporary node to Neo4j for querying.
func createTemporaryNode(driver neo4j.DriverWithContext, embedding []float32) (int64, error) {
	ctx := context.Background()
	var nodeID int64

	session := driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `CREATE (n:TempEmbedding {embedding: $embedding}) RETURN id(n)`
		result, err := tx.Run(ctx, query, map[string]any{"embedding": embedding})
		if err != nil {
			return nil, err
		}
		if result.Next(ctx) {
			nodeID = result.Record().Values[0].(int64)
		}
		return nil, result.Err()
	})

	return nodeID, err
}

// deleteTemporaryNode removes the temporary node.
func deleteTemporaryNode(driver neo4j.DriverWithContext, nodeID int64) {
	ctx := context.Background()
	session := driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		_, err := tx.Run(ctx, `MATCH (n:TempEmbedding) WHERE id(n) = $id DELETE n`, map[string]any{"id": nodeID})
		return nil, err
	})
}

// findSimilarNodes finds nodes similar to the temporary node.
func findSimilarNodes(driver neo4j.DriverWithContext, tempNodeID int64) ([]struct {
	ID         int64
	Text       string
	Similarity float64
}, error) {
	ctx := context.Background()
	var results []struct {
		ID         int64
		Text       string
		Similarity float64
	}

	session := driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	_, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
		MATCH (temp:TempEmbedding), (n:Method)
		WHERE n.embedding IS NOT NULL
		WITH n, gds.similarity.cosine(temp.embedding, n.embedding) AS similarity
		RETURN id(n) AS id, n.documentation AS documentation, similarity
		ORDER BY similarity DESC
		LIMIT 10
		`

		result, err := tx.Run(ctx, query, map[string]any{"tempNodeID": tempNodeID})
		if err != nil {
			return nil, err
		}

		for result.Next(ctx) {
			record := result.Record()
			results = append(results, struct {
				ID         int64
				Text       string
				Similarity float64
			}{
				ID:         record.Values[0].(int64),
				Text:       record.Values[1].(string),
				Similarity: record.Values[2].(float64),
			})
		}
		return nil, result.Err()
	})

	return results, err
}
