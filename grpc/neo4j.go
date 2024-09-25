package grpc

import (
	"context"
	"encoding/json"

	"connectrpc.com/connect"
	"github.com/wricardo/code-surgeon/api"
	"github.com/wricardo/code-surgeon/neo4j2"
)

func (h *Handler) QueryNeo4J(ctx context.Context, req *connect.Request[api.QueryNeo4JRequest]) (*connect.Response[api.QueryNeo4JResponse], error) {
	// Execute the query
	result, err := neo4j2.QueryNeo4J(ctx, h.neo4jDriver, req.Msg.Cypher, nil)
	if err != nil {
		return nil, err
	}

	encoded, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	// Convert the result to the API response format
	response := &api.QueryNeo4JResponse{
		Output: string(encoded),
	}

	// Return the response
	return &connect.Response[api.QueryNeo4JResponse]{Msg: response}, nil
}

func (h *Handler) GetNeo4JSchema(ctx context.Context, req *connect.Request[api.GetNeo4JSchemaRequest]) (*connect.Response[api.GetNeo4JSchemaResponse], error) {
	schema, err := neo4j2.GetSchema(ctx, h.neo4jDriver)
	if err != nil {
		return nil, err
	}

	protoschema := &api.GetNeo4JSchemaResponse{}
	protoschema.Schema = &api.Schema{
		Labels: []*api.LabelSchema{},
	}
	for _, s := range schema.Labels {
		protoschema.Schema.Labels = append(protoschema.Schema.Labels, &api.LabelSchema{
			Label:      s.Label,
			Properties: []*api.PropertySchema{}})
		for _, p := range s.Properties {
			protoschema.Schema.Labels[len(protoschema.Schema.Labels)-1].Properties = append(protoschema.Schema.Labels[len(protoschema.Schema.Labels)-1].Properties, &api.PropertySchema{
				Property:            p.Property,
				Type:                p.Type,
				IsIndexed:           p.IsIndexed,
				UniqueConstraint:    p.UniqueConstraint,
				ExistenceConstraint: p.ExistenceConstraint,
			})
		}
	}

	for _, r := range schema.Relationships {
		protoschema.Schema.Relationships = append(protoschema.Schema.Relationships, &api.RelationshipSchema{
			Relationship: r.Relationship,
			FromLabel:    r.FromLabel,
			ToLabel:      r.ToLabel,
		})
	}

	return &connect.Response[api.GetNeo4JSchemaResponse]{
		Msg: protoschema,
	}, nil
}
