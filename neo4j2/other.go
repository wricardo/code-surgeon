package neo4j2

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type Schema struct {
	Labels        []LabelSchema        `json:"labels"`
	Relationships []RelationshipSchema `json:"relationships"`
}

func (s *Schema) Format(format string) string {
	if format == "json" {
		encoded, _ := json.Marshal(s)
		return string(encoded)
	} else if format == "llm" {
		var output []string
		for _, label := range s.Labels {
			var props []string
			for _, prop := range label.Properties {
				props = append(props, prop.Property)
			}
			output = append(output, fmt.Sprintf("%s:[%s]", label.Label, strings.Join(props, ",")))
		}
		for _, rel := range s.Relationships {
			output = append(output, fmt.Sprintf("%s->%s->%s", rel.FromLabel, rel.Relationship, rel.ToLabel))
		}
		return strings.Join(output, ";")
	}

	return "unsupported schema format: " + format
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
