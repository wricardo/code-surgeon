package neo4j2

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/rs/zerolog/log"
	"github.com/sashabaranov/go-openai"
	codesurgeon "github.com/wricardo/code-surgeon"
)

func ToNeo4j(ctx context.Context, path string, deep bool, myEnv map[string]string, recursive bool) error {
	neo4jDbUri, _ := myEnv["NEO4J_DB_URI"]
	neo4jDbUser, _ := myEnv["NEO4J_DB_USER"]
	neo4jDbPassword, _ := myEnv["NEO4J_DB_PASSWORD"]
	driver, closeFn, err := Connect(ctx, neo4jDbUri, neo4jDbUser, neo4jDbPassword)
	if err != nil {
		log.Info().Err(err).Msg("Error connecting to Neo4j (proceeding anyway)")
		return err
	} else {
		defer closeFn()
	}
	sess := driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer sess.Close(ctx)

	// Execute the command to list Go modules in JSON format
	cmd := exec.Command("go", "list", "-json", path)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to execute go list command: %w", err)
	}

	// fmt.Println(string(output))

	// Parse the JSON output and pretty-print
	decoder := json.NewDecoder(strings.NewReader(string(output)))

	for decoder.More() {
		fmt.Println("decoder.More()")
		var module codesurgeon.GoList
		if err := decoder.Decode(&module); err != nil {
			log.Printf("failed to decode module: %v", err)
			continue
		}

		if recursive {
			log.Info().Msgf("Parsed %s", module.Dir)
			infos, err := codesurgeon.ParseDirectoryRecursive(module.Dir)
			if err != nil {
				log.Info().Err(err).Msgf("Error parsing file %s", module.Dir)
				return err
			}
			for _, info := range infos {
				shouldContinue, err1 := toNeo4j(ctx, info, module.ImportPath, module.Dir, sess, deep)
				if err1 != nil {
					if shouldContinue {
						continue
					}

					return err1
				}
			}
		} else {
			log.Info().Msgf("Parsed %s", module.Dir)
			info, err := codesurgeon.ParseDirectory(module.Dir)
			if err != nil {
				log.Info().Err(err).Msgf("Error parsing file %s", module.Dir)
			}

			shouldContinue, err1 := toNeo4j(ctx, info, module.Dir, module.ImportPath, sess, deep)
			if err1 != nil {
				if shouldContinue {
					continue
				}
			}

		}

	}

	return nil
}

// toNeo4j processes parsed information and upserts it into Neo4j using the provided session.
func toNeo4j(ctx context.Context, info *codesurgeon.ParsedInfo, moduleDir, moduleImportPath string, session neo4j.SessionWithContext, deep bool) (bool, error) {
	var err error
	for _, mod := range info.Modules {
		for _, pkg := range mod.Packages {
			err = UpsertPackage(ctx, session, mod, pkg)
			if err != nil {
				log.Info().Err(err).Msgf("Error upserting package %s", pkg.Package)
				return true, err
			}

			for k, struct_ := range pkg.Structs {
				log.Info().Msgf("struct %d: %s", k, struct_.Name)
				if err = UpsertStruct(ctx, session, mod, info.Packages[0], struct_); err != nil {
					log.Info().Err(err).Msgf("Error upserting struct %s", struct_.Name)
					return true, err
				}
				// methods
				for k2, method := range struct_.Methods {
					funcFilePath := ""
					if deep {
						funcFilePath, err = codesurgeon.FindFunction(moduleDir, struct_.Name, method.Name)
						if err != nil {
							log.Info().Err(err).Msgf("Error finding function file %s", method.Name)
						} else {
							log.Info().Msgf("funcFilePath: %s", funcFilePath)
						}
					}
					err = UpsertMethod(ctx, session, mod, pkg, method, struct_)
					if err != nil {
						log.Info().Err(err).Msgf("Error upserting function %s", method.Name)
						return true, err
					}
					log.Info().Msgf("method %d %d: %s", k, k2, method.Name)
					for _, param := range method.Params {
						err = UpsertMethodParam(ctx, session, mod, pkg, struct_, method, param)
						if err != nil {
							log.Info().Err(err).Msgf("Error upserting function param %s", param.Name)
							return true, err
						}
					}
					for _, result := range method.Returns {
						err = UpsertMethodReturn(ctx, session, mod, pkg, method, result)
						if err != nil {
							log.Info().Err(err).Msgf("Error upserting function return %s", result.Name)
							return true, err
						}
					}

				}
				// fields
				for k2, field := range struct_.Fields {
					// Upsert each field of the struct into Neo4j
					err = UpsertStructField(ctx, session, mod, pkg, struct_, field)
					if err != nil {
						log.Info().Err(err).Msgf("Error upserting struct field %s", field.Name)
						return true, err
					}
					log.Info().Msgf("field %d %d: %s", k, k2, field.Name)
				}

			}
			fmt.Println("len functions", len(info.Packages[0].Functions))
			for k, function := range info.Packages[0].Functions {
				funcFilePath := ""
				if deep {
					funcFilePath, err = codesurgeon.FindFunction(moduleDir, "", function.Name)
					if err != nil {
						log.Info().Err(err).Msgf("Error finding function file %s", function.Name)
					} else {
						log.Info().Msgf("funcFilePath: %s", funcFilePath)
					}
				}
				err = UpsertFunction(ctx, session, mod, pkg, function)
				if err != nil {
					log.Info().Err(err).Msgf("Error upserting function %s", function.Name)
					return true, err
				}
				log.Info().Msgf("function %d: %s", k, function.Name)
				for _, param := range function.Params {
					err = UpsertFunctionParam(ctx, session, mod, pkg, function, param)
					if err != nil {
						log.Info().Err(err).Msgf("Error upserting function param %s", param.Name)
						return true, err
					}
				}
				for _, ret := range function.Returns {
					err = UpsertFunctionReturn(ctx, session, mod, pkg, function, ret)
					if err != nil {
						log.Info().Err(err).Msgf("Error upserting function return %s", ret.Name)
						return true, err
					}
				}
			}
			for _, interface_ := range info.Packages[0].Interfaces {
				log.Info().Msgf("interface: %s", interface_.Name)
				if err = UpsertInterface(ctx, session, mod, pkg, interface_); err != nil {
					log.Info().Err(err).Msgf("Error upserting interface %s", interface_.Name)
					return true, err
				}
				for _, method := range interface_.Methods {
					err = UpsertInterfaceMethod(ctx, session, mod, pkg, interface_, method)
					if err != nil {
						log.Info().Err(err).Msgf("Error upserting function %s", method.Name)
						return true, err
					}
					for _, param := range method.Params {
						err = UpsertInterfaceMethodParam(ctx, session, mod, pkg, interface_, method, param)
						if err != nil {
							log.Info().Err(err).Msgf("Error upserting function param %s", param.Name)
							return true, err
						}
					}
					for _, result := range method.Returns {
						err = UpsertInterfaceMethodReturn(ctx, session, mod, pkg, interface_, method, result)
						if err != nil {
							log.Info().Err(err).Msgf("Error upserting function return %s", result.Name)
							return true, err
						}
					}
				}
			}
		}
	}
	return false, nil
}

// GenerateEmbeddings fetches documentation nodes, computes embeddings, and updates Neo4j.
func GenerateEmbeddings(driver neo4j.DriverWithContext, openAIClient *openai.Client) error {

	var err error
	err = generateFunctionsAndMethodsEmbeddings(driver, openAIClient)
	if err != nil {
		log.Info().Err(err).Msg("Error generating embeddings")
		return err
	}

	err = generateQuestionsAndAnswersEmbeddings(driver, openAIClient)
	if err != nil {
		log.Info().Err(err).Msg("Error generating embeddings")
	}

	return nil
}

// generateQuestionsAndAnswersEmbeddings fetches Question nodes (only using Question.content),
// but only for questions that have a relationship with an Answer and do not already have an embedding,
// computes embeddings, and updates Neo4j.
func generateQuestionsAndAnswersEmbeddings(driver neo4j.DriverWithContext, openAIClient *openai.Client) error {
	// Define query to retrieve Question nodes with a HAS_ANSWER relationship and missing embedding
	query := `
MATCH (q:Question)-[:HAS_ANSWER]->(:Answer)
WHERE 
    q.content IS NOT NULL AND q.content <> "" AND 
    q.embedding IS NULL
RETURN 
    id(q) AS id, 
    q.content AS question
LIMIT 5000
`
	var nodes []TODORENAMENodeData
	ctx := context.Background()
	session := driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	// Execute a read transaction to fetch Question nodes
	_, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		result, err := tx.Run(ctx, query, nil)
		if err != nil {
			return nil, err
		}
		for result.Next(ctx) {
			record := result.Record()
			nodeID := record.Values[0].(int64)
			questionText := record.Values[1].(string)
			nodes = append(nodes, TODORENAMENodeData{ID: nodeID, Text: questionText})
		}
		return nil, result.Err()
	})
	if err != nil {
		log.Fatal().Msgf("Failed to fetch Question nodes: %v", err)
	}

	log.Printf("Fetched %d Question nodes to process.", len(nodes))
	// Loop through nodes to compute and update embeddings
	for _, node := range nodes {
		embedding, err := getEmbedding(openAIClient, node.Text)
		if err != nil {
			log.Printf("Failed to compute embedding for node ID %d: %v", node.ID, err)
			continue
		}
		node.Embedding = embedding
		err = updateNodeEmbedding(driver, node)
		if err != nil {
			return err
		}
		log.Printf("Updated embedding for Question node ID %d", node.ID)
		time.Sleep(100 * time.Millisecond) // Rate limiting
	}
	log.Print("Question Embedding generation completed.")
	return nil
}

// generateFunctionsAndMethodsEmbeddings fetches functions and methods with documentation, computes embeddings, and updates Neo4j
func generateFunctionsAndMethodsEmbeddings(driver neo4j.DriverWithContext, openAIClient *openai.Client) error {
	query := `
    // Retrieve Functions belonging directly to the specified Package
MATCH (f:Function)-[:BELONGS_TO]->(:Package)
WHERE 
    f.documentation IS NOT NULL AND 
    f.documentation <> "" AND
	f.embedding IS NULL
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
    m.documentation <> ""  AND
	m.embedding IS NULL
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
		log.Fatal().Msgf("Failed to fetch nodes: %v", err)
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
			return err
		} else {
			log.Printf("Updated embedding for node ID %d %s", node.ID, node.Text)
		}

		time.Sleep(100 * time.Millisecond) // Rate limiting
	}
	log.Print("Embedding generation completed.")
	return nil
}

// updateNodeEmbedding updates a Neo4j node with an embedding.
func updateNodeEmbedding(driver neo4j.DriverWithContext, node TODORENAMENodeData) error {
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

type TODORENAMENodeData struct {
	ID        int64
	Text      string
	Embedding []float32
}

// fetchNodes retrieves nodes with documentation.
func fetchNodes(driver neo4j.DriverWithContext, query string) ([]TODORENAMENodeData, error) {
	var nodes []TODORENAMENodeData
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

			nodes = append(nodes, TODORENAMENodeData{ID: id, Text: text})
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
