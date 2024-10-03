package neo4j2

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/google/uuid"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

func UpsertFunctionInfo(ctx context.Context, driver neo4j.DriverWithContext, file string, receiver, function string, documentation string, packageName string, package_ string) error {
	session := driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(ctx)
	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		name := fmt.Sprintf("%s.%s", receiver, function)
		if receiver == "" {
			name = function
		}
		result, err := tx.Run(ctx, `
		MERGE (f:Function {package: $package, receiver: $receiver, function: $function})
		SET f.documentation = $documentation, f.packageName = $packageName, f.file = $file, f.name = $name
		with f
		MERGE (p:Package {package: $package})
		SET p.name = $packageName, p.packageName = $packageName
		MERGE (f)-[:BELONGS_TO]->(p)
		RETURN id(f) as nodeID
		`, map[string]interface{}{
			"file":          file,
			"name":          name,
			"documentation": documentation,
			"receiver":      receiver,
			"function":      function,
			"packageName":   packageName,
			"package":       package_,
		})
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
	if err != nil {
		return err
	}
	return nil

}

func ListFileFromFunctions(ctx context.Context, driver neo4j.DriverWithContext) ([]string, error) {
	// MATCH (f:Function) return distinct f.file
	session := driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(ctx)
	var files []string
	_, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		result, err := tx.Run(ctx, "MATCH (f:Function) return distinct f.file", nil)
		if err != nil {
			return nil, err
		}
		for result.Next(ctx) {
			files = append(files, result.Record().Values[0].(string))
		}
		return nil, nil
	})

	if err != nil {
		return nil, err
	}

	return files, nil
}

// GetSchema retrieves the database schema, including labels, properties, and relationships.
func GetSchema(ctx context.Context, driver neo4j.DriverWithContext) (*Schema, error) {
	if driver == nil {
		return nil, fmt.Errorf("driver is nil")
	}

	// Define the Cypher query to retrieve the schema for labels and properties.
	const cypherQueryLabels = `
        CALL apoc.meta.schema() YIELD value as schemaMap
        UNWIND keys(schemaMap) as label
        WITH label, schemaMap[label] as data
        WHERE data.type = "node"
        UNWIND keys(data.properties) as property
        WITH label, property, data.properties[property] as propData
        RETURN label,
               property,
               propData.type as type,
               propData.indexed as isIndexed,
               propData.unique as uniqueConstraint,
               propData.existence as existenceConstraint
    `

	// Define the Cypher query to retrieve relationship schema using the new format.
	const cypherQueryRelationships = `
        MATCH (start)-[r]->(end)
        UNWIND r AS rel
        WITH DISTINCT labels(start) AS startLabels, type(rel) AS relType, labels(end) AS endLabels
        RETURN startLabels, relType, endLabels
        ORDER BY startLabels, relType, endLabels
    `

	// Initialize a new session.
	session := driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	// Execute the read transaction.
	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		// -------------------
		// Process Labels and Properties
		// -------------------
		labelRecords, err := tx.Run(ctx, cypherQueryLabels, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to execute schema query for labels: %w", err)
		}

		// Initialize a map to hold label schemas.
		labelMap := make(map[string]*LabelSchema)

		// Iterate over the label records.
		for labelRecords.Next(ctx) {
			record := labelRecords.Record()
			label, _ := record.Get("label")
			property, _ := record.Get("property")
			propType, _ := record.Get("type")
			isIndexed, _ := record.Get("isIndexed")
			uniqueConstraint, _ := record.Get("uniqueConstraint")
			existenceConstraint, _ := record.Get("existenceConstraint")

			lbl := label.(string)
			prop := property.(string)
			pType := propType.(string)
			idx := isIndexed.(bool)
			unique := uniqueConstraint.(bool)
			existence := existenceConstraint.(bool)

			// If the label is not yet in the map, add it.
			if _, exists := labelMap[lbl]; !exists {
				labelMap[lbl] = &LabelSchema{
					Label:      lbl,
					Properties: []PropertySchema{},
				}
			}

			// Append the property schema to the label.
			labelMap[lbl].Properties = append(labelMap[lbl].Properties, PropertySchema{
				Property:            prop,
				Type:                pType,
				IsIndexed:           idx,
				UniqueConstraint:    unique,
				ExistenceConstraint: existence,
			})
		}

		// Check for errors during label iteration.
		if err = labelRecords.Err(); err != nil {
			return nil, fmt.Errorf("error iterating label records: %w", err)
		}

		// Convert the label map to a slice.
		labels := make([]LabelSchema, 0, len(labelMap))
		for _, lblSchema := range labelMap {
			labels = append(labels, *lblSchema)
		}

		// -------------------
		// Process Relationships
		// -------------------
		relRecords, err := tx.Run(ctx, cypherQueryRelationships, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to execute schema query for relationships: %w", err)
		}

		// Initialize a slice to hold relationships.
		relationships := []RelationshipSchema{}

		// Iterate over the relationship records.
		for relRecords.Next(ctx) {
			record := relRecords.Record()
			startLabels, _ := record.Get("startLabels")
			relType, _ := record.Get("relType")
			endLabels, _ := record.Get("endLabels")

			startLbls, ok1 := startLabels.([]interface{})
			relStr, ok2 := relType.(string)
			endLbls, ok3 := endLabels.([]interface{})

			if !ok1 || !ok2 || !ok3 {
				return nil, fmt.Errorf("unexpected data types in relationship record")
			}

			// Convert labels from []interface{} to string
			fromLabel := labelsToString(startLbls)
			toLabel := labelsToString(endLbls)

			rel := RelationshipSchema{
				Relationship: relStr,
				FromLabel:    fromLabel,
				ToLabel:      toLabel,
			}

			relationships = append(relationships, rel)
		}

		// Check for errors during relationship iteration.
		if err = relRecords.Err(); err != nil {
			return nil, fmt.Errorf("error iterating relationship records: %w", err)
		}

		// -------------------
		// Construct and Return Schema
		// -------------------
		encodedLabels, err := json.MarshalIndent(labels, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal labels: %w", err)
		}
		log.Printf("Labels: %s", encodedLabels)

		encodedRelationships, err := json.MarshalIndent(relationships, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal relationships: %w", err)
		}
		log.Printf("Relationships: %s", encodedRelationships)

		return &Schema{
			Labels:        labels,
			Relationships: relationships,
		}, nil
	})

	if err != nil {
		return nil, err
	}

	// Type assertion to *Schema.
	schema, ok := result.(*Schema)
	if !ok {
		return nil, fmt.Errorf("unexpected result type")
	}

	encodedSchema, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal schema: %w", err)
	}
	log.Printf("Schema: %s", encodedSchema)
	return schema, nil
}

// labelsToString converts a slice of interface{} to a comma-separated string of labels.
func labelsToString(labels []interface{}) string {
	labelStrings := make([]string, len(labels))
	for i, label := range labels {
		labelStrings[i] = label.(string)
	}
	return strings.Join(labelStrings, ",")
}

/*
// GetSchema retrieves the database schema from Neo4j and returns it as a Schema struct.
func GetSchema(ctx context.Context, driver neo4j.DriverWithContext) (*Schema, error) {
	if driver == nil {
		return nil, fmt.Errorf("driver is nil")
	}
	// Define the Cypher query to retrieve the schema.
	const cypherQuery = `
        CALL apoc.meta.schema() YIELD value as schemaMap
        UNWIND keys(schemaMap) as label
        WITH label, schemaMap[label] as data
        WHERE data.type = "node"
        UNWIND keys(data.properties) as property
        WITH label, property, data.properties[property] as propData
        RETURN label,
               property,
               propData.type as type,
               propData.indexed as isIndexed,
               propData.unique as uniqueConstraint,
               propData.existence as existenceConstraint
    `

	// Initialize a new session.
	session := driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	// Execute the read transaction.
	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		// Run the Cypher query.
		records, err := tx.Run(ctx, cypherQuery, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to execute schema query: %w", err)
		}

		// Initialize a map to hold label schemas.
		labelMap := make(map[string]*LabelSchema)

		// Iterate over the results.
		for records.Next(ctx) {
			record := records.Record()
			label, _ := record.Get("label")
			property, _ := record.Get("property")
			propType, _ := record.Get("type")
			isIndexed, _ := record.Get("isIndexed")
			uniqueConstraint, _ := record.Get("uniqueConstraint")
			existenceConstraint, _ := record.Get("existenceConstraint")

			lbl := label.(string)
			prop := property.(string)
			pType := propType.(string)
			idx := isIndexed.(bool)
			unique := uniqueConstraint.(bool)
			existence := existenceConstraint.(bool)

			// If the label is not yet in the map, add it.
			if _, exists := labelMap[lbl]; !exists {
				labelMap[lbl] = &LabelSchema{
					Label:      lbl,
					Properties: []PropertySchema{},
				}
			}

			// Append the property schema to the label.
			labelMap[lbl].Properties = append(labelMap[lbl].Properties, PropertySchema{
				Property:            prop,
				Type:                pType,
				IsIndexed:           idx,
				UniqueConstraint:    unique,
				ExistenceConstraint: existence,
			})
		}

		// Check for errors during iteration.
		if err = records.Err(); err != nil {
			return nil, fmt.Errorf("error iterating schema records: %w", err)
		}

		// Convert the map to a slice.
		labels := make([]LabelSchema, 0, len(labelMap))
		for _, lblSchema := range labelMap {
			labels = append(labels, *lblSchema)
		}

		// Return the populated Schema.
		return &Schema{
			Labels: labels,
		}, nil
	})

	if err != nil {
		return nil, err
	}

	// Type assertion to *Schema.
	schema, ok := result.(*Schema)
	if !ok {
		return nil, fmt.Errorf("unexpected result type")
	}

	return schema, nil
}
*/

func CreateQuestionAndAnswers(ctx context.Context, driver neo4j.DriverWithContext, questionText string, questionEmbedding []float32, answers []string) error { // CreateQuestionAndAnswers creates a question node and its corresponding answer nodes in Neo4j.
	session := driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(ctx)
	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		questionResult, err := tx.Run(ctx, "CREATE (q:Question {id: $id, text: $text, embedding: $embedding, created_at: datetime($createdAt)}) RETURN q.id", map[string]interface{}{
			"text":      questionText,
			"id":        uuid.New().String(),
			"embedding": questionEmbedding,
			"createdAt": time.Now().Format(time.RFC3339),
		})
		if err != nil {
			return nil, err
		}
		var questionId string
		if questionResult.Next(ctx) {
			questionId = questionResult.Record().Values[0].(string)
		}
		for _, answerText := range answers {
			answerResult, err := tx.Run(ctx, "CREATE (a:Answer {id: $id, text: $text, created_at: datetime($createdAt) }) RETURN a.id", map[string]interface{}{"text": answerText, "id": uuid.New().String(), "createdAt": time.Now().Format(time.RFC3339)})
			if err != nil {
				return nil, err
			}
			var answerId string
			if answerResult.Next(ctx) {
				answerId = answerResult.Record().Values[0].(string)
			}
			_, err = tx.Run(ctx, "MATCH (q:Question {id: $questionId}), (a:Answer {id: $answerId}) CREATE (q)-[:ANSWERED_BY]->(a)", map[string]interface{}{"questionId": questionId, "answerId": answerId})
			if err != nil {
				return nil, err
			}
		}
		log.Printf("Created question with ID: %s", questionId)
		return nil, nil
	})
	return err
}

func PageQuestions(ctx context.Context, driver neo4j.DriverWithContext, page, limit int) ([]Question, error) {
	// set default limit to 10
	if limit == 0 {
		limit = 10
	}

	// set default page to 1
	if page == 0 {
		page = 1
	}

	session := driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(ctx)
	var questions []Question
	_, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		result, err := tx.Run(ctx, "MATCH (q:Question) RETURN q.id, q.text, q.embedding, q.created_at SKIP $skip LIMIT $limit", map[string]interface{}{"skip": (page - 1) * limit, "limit": limit})
		if err != nil {
			return nil, err
		}
		for result.Next(ctx) {
			record := result.Record()
			questions = append(questions, Question{
				ID:        toString(record.Values[0]),
				Text:      toString(record.Values[1]),
				CreatedAt: toString(record.Values[3]),
			})
		}
		return nil, nil
	})
	if err != nil {
		return nil, err
	}
	return questions, nil
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

// GetTopAnswersForQuestions retrieves the top answers for a list of question IDs from Neo4j.
func GetTopAnswersForQuestions(ctx context.Context, driver neo4j.DriverWithContext, questionIds []string) ([]QuestionAnswer, error) {
	session := driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(ctx)
	var allAnswers []QuestionAnswer
	_, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		for _, questionId := range questionIds {
			result, err := tx.Run(ctx, "MATCH (q:Question {id: $questionId})-[:ANSWERED_BY]->(a:Answer) "+"RETURN q.text, a.text ORDER BY a.score DESC LIMIT 1", map[string]interface{}{"questionId": questionId})
			if err != nil {
				return nil, err
			}
			for result.Next(ctx) {
				record := result.Record()
				allAnswers = append(allAnswers, QuestionAnswer{
					Question: toString(record.Values[0]),
					Answer:   toString(record.Values[1]),
				})
			}
		}
		return nil, nil
	})
	if err != nil {
		return nil, err
	}
	return allAnswers, nil
}

// VectorSearchQuestions performs a vector search in Neo4j to find the top similar questions to the user's input.
func VectorSearchQuestions(ctx context.Context, driver neo4j.DriverWithContext, userEmbedding []float32, limit int) ([]Question, error) {
	if len(userEmbedding) == 0 {
		return nil, fmt.Errorf("user embedding is empty")
	}
	session := driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(ctx)
	var questions []Question
	mapSeen := make(map[string]bool)
	_, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		result, err := tx.Run(ctx,
			"MATCH (q:Question) "+
				"WHERE q.embedding IS NOT NULL AND q.embedding <> []"+
				"RETURN q.id, q.text, gds.similarity.cosine(q.embedding, $userEmbedding) AS similarity "+
				"ORDER BY similarity DESC LIMIT $limit",
			map[string]interface{}{
				"userEmbedding": userEmbedding,
				"limit":         limit,
			})
		if err != nil {
			return nil, err
		}
		for result.Next(ctx) {
			record := result.Record()
			if _, seen := mapSeen[toString(record.Values[1])]; seen {
				continue
			}
			mapSeen[toString(record.Values[1])] = true
			questions = append(questions, Question{
				ID:         toString(record.Values[0]),
				Text:       toString(record.Values[1]),
				Similarity: toFloat64(record.Values[2]),
			})
		}
		return nil, nil
	})
	if err != nil {
		return nil, err
	}
	return questions, nil
}

func SaveConversationSummary(ctx context.Context, driver neo4j.DriverWithContext, conversationSummary, dateISO string) error {
	session := driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface {
	}, error) {
		query := `
			CREATE (cs:ConversationSummary {
				summary: $summary,
				date: $date
			})
		`
		_, err := tx.Run(ctx, query, map[string]interface{}{"summary": conversationSummary,

			"date": dateISO,
		})
		if err != nil {
			return nil, err
		}
		return nil, nil
	})
	if err != nil {
		log.
			Printf("Error saving conversation summary to database: %v",
				err)
		return err
	}
	return nil
}

type QuestionAnswer struct {
	Question string
	Answer   string
}

type Question struct {
	ID         string
	Text       string
	Embedding  []float32
	Similarity float64 // not part of the node
	CreatedAt  string
}

// Schema represents the entire database schema.
type Schema struct {
	Labels        []LabelSchema        `json:"labels"`
	Relationships []RelationshipSchema `json:"relationships"`
}

// RelationshipSchema represents the schema for a single relationship.
type RelationshipSchema struct {
	Relationship string `json:"relationship"`
	FromLabel    string `json:"fromLabel"`
	ToLabel      string `json:"toLabel"`
}

// LabelSchema represents the schema for a single label.
type LabelSchema struct {
	Label      string           `json:"label"`
	Properties []PropertySchema `json:"properties"`
}

// PropertySchema represents the schema for a single property.
type PropertySchema struct {
	Property            string `json:"property"`
	Type                string `json:"type"`
	IsIndexed           bool   `json:"isIndexed"`
	UniqueConstraint    bool   `json:"uniqueConstraint"`
	ExistenceConstraint bool   `json:"existenceConstraint"`
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
