package neo4j2

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/rs/zerolog/log"
)

func GetTopAnswersForQuestions(ctx context.Context, driver neo4j.DriverWithContext, questionIds []string) ([]QuestionAnswer, error) {
	session := driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(ctx)
	var allAnswers []QuestionAnswer
	_, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		for _, questionId := range questionIds {
			result, err := tx.Run(ctx, "MATCH (q:Question {id: $questionId})-[:ANSWERED_BY]->(a:Answer) RETURN q.text, a.text ORDER BY a.score DESC LIMIT 1", map[string]interface{}{"questionId": questionId})
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
				"WHERE q.embedding IS NOT NULL AND q.embedding <> [] "+
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

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		query := `
			CREATE (cs:ConversationSummary {
				summary: $summary,
				date: $date
			})
		`
		_, err := tx.Run(ctx, query, map[string]interface{}{
			"summary": conversationSummary,
			"date":    dateISO,
		})
		if err != nil {
			return nil, err
		}
		return nil, nil
	})
	if err != nil {
		log.Printf("Error saving conversation summary to database: %v", err)
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
	Similarity float64
	CreatedAt  string
}

type Schema struct {
	Labels        []LabelSchema        `json:"labels"`
	Relationships []RelationshipSchema `json:"relationships"`
}

type RelationshipSchema struct {
	Relationship string `json:"relationship"`
	FromLabel    string `json:"fromLabel"`
	ToLabel      string `json:"toLabel"`
}

type LabelSchema struct {
	Label      string           `json:"label"`
	Properties []PropertySchema `json:"properties"`
}

type PropertySchema struct {
	Property            string `json:"property"`
	Type                string `json:"type"`
	IsIndexed           bool   `json:"isIndexed"`
	UniqueConstraint    bool   `json:"uniqueConstraint"`
	ExistenceConstraint bool   `json:"existenceConstraint"`
}

// Schema related functions remain mostly unchanged as they already had abstraction.
func GetSchema(ctx context.Context, driver neo4j.DriverWithContext) (*Schema, error) {
	if driver == nil {
		return nil, fmt.Errorf("driver is nil")
	}

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

	const cypherQueryRelationships = `
        MATCH (start)-[r]->(end)
        UNWIND r AS rel
        WITH DISTINCT labels(start) AS startLabels, type(rel) AS relType, labels(end) AS endLabels
        RETURN startLabels, relType, endLabels
        ORDER BY startLabels, relType, endLabels
    `

	session := driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		labelRecords, err := tx.Run(ctx, cypherQueryLabels, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to execute schema query for labels: %w", err)
		}

		labelMap := make(map[string]*LabelSchema)
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

			if _, exists := labelMap[lbl]; !exists {
				labelMap[lbl] = &LabelSchema{
					Label:      lbl,
					Properties: []PropertySchema{},
				}
			}

			labelMap[lbl].Properties = append(labelMap[lbl].Properties, PropertySchema{
				Property:            prop,
				Type:                pType,
				IsIndexed:           idx,
				UniqueConstraint:    unique,
				ExistenceConstraint: existence,
			})
		}

		if err = labelRecords.Err(); err != nil {
			return nil, fmt.Errorf("error iterating label records: %w", err)
		}

		labels := make([]LabelSchema, 0, len(labelMap))
		for _, lblSchema := range labelMap {
			labels = append(labels, *lblSchema)
		}

		relRecords, err := tx.Run(ctx, cypherQueryRelationships, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to execute schema query for relationships: %w", err)
		}

		relationships := []RelationshipSchema{}
		for relRecords.Next(ctx) {
			record := relRecords.Record()
			startLabels, _ := record.Get("startLabels")
			relType, _ := record.Get("relType")
			endLabels, _ := record.Get("endLabels")

			startLbls := startLabels.([]interface{})
			relStr := relType.(string)
			endLbls := endLabels.([]interface{})

			fromLabel := labelsToString(startLbls)
			toLabel := labelsToString(endLbls)

			rel := RelationshipSchema{
				Relationship: relStr,
				FromLabel:    fromLabel,
				ToLabel:      toLabel,
			}

			relationships = append(relationships, rel)
		}

		if err = relRecords.Err(); err != nil {
			return nil, fmt.Errorf("error iterating relationship records: %w", err)
		}

		schema := &Schema{
			Labels:        labels,
			Relationships: relationships,
		}

		return schema, nil
	})
	if err != nil {
		return nil, err
	}

	schema, ok := result.(*Schema)
	if !ok {
		return nil, fmt.Errorf("unexpected result type")
	}
	return schema, nil
}

func labelsToString(labels []interface{}) string {
	labelStrings := make([]string, len(labels))
	for i, label := range labels {
		labelStrings[i] = label.(string)
	}
	return strings.Join(labelStrings, ",")
}

func CreateQuestionAndAnswers(ctx context.Context, driver neo4j.DriverWithContext, questionText string, questionEmbedding []float32, answers []string) error {
	session := driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(ctx)
	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		// We'll keep this logic as is, since it's more dynamic than the others:
		existingQuestionResult, err := tx.Run(ctx, "MATCH (q:Question {text: $text}) RETURN q.id", map[string]interface{}{
			"text": questionText,
		})
		if err != nil {
			return nil, err
		}

		var questionId string
		if existingQuestionResult.Next(ctx) {
			questionId = existingQuestionResult.Record().Values[0].(string)
		} else {
			newQuestionResult, err := tx.Run(ctx, "CREATE (q:Question {id: $id, text: $text, embedding: $embedding, created_at: datetime($createdAt)}) RETURN q.id", map[string]interface{}{
				"text":      questionText,
				"id":        uuid.New().String(),
				"embedding": questionEmbedding,
				"createdAt": time.Now().Format(time.RFC3339),
			})
			if err != nil {
				return nil, err
			}
			if newQuestionResult.Next(ctx) {
				questionId = newQuestionResult.Record().Values[0].(string)
			}
		}

		for _, answerText := range answers {
			answerResult, err := tx.Run(ctx, "CREATE (a:Answer {id: $id, text: $text, created_at: datetime($createdAt) }) RETURN a.id", map[string]interface{}{
				"text":      answerText,
				"id":        uuid.New().String(),
				"createdAt": time.Now().Format(time.RFC3339),
			})
			if err != nil {
				return nil, err
			}
			var answerId string
			if answerResult.Next(ctx) {
				answerId = answerResult.Record().Values[0].(string)
			}
			_, err = tx.Run(ctx, "MATCH (q:Question {id: $questionId}), (a:Answer {id: $answerId}) CREATE (q)-[:ANSWERED_BY]->(a)", map[string]interface{}{
				"questionId": questionId,
				"answerId":   answerId,
			})
			if err != nil {
				return nil, err
			}
		}
		log.Printf("Processed question with ID: %s", questionId)
		return nil, nil
	})
	return err
}

func PageQuestions(ctx context.Context, driver neo4j.DriverWithContext, page, limit int) ([]Question, error) {
	if limit == 0 {
		limit = 10
	}
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

func ListFileFromFunctions(ctx context.Context, driver neo4j.DriverWithContext) ([]string, error) {
	session := driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(ctx)

	var files []string
	_, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		result, err := tx.Run(ctx, "MATCH (f:Function) return distinct f.file", nil)
		if err != nil {
			return nil, err
		}
		for result.Next(ctx) {
			val := result.Record().Values[0]
			if valStr, ok := val.(string); ok {
				files = append(files, valStr)
			}
		}
		return nil, nil
	})
	if err != nil {
		return nil, err
	}

	return files, nil
}
