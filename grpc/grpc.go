package grpc

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"

	"github.com/instructor-ai/instructor-go/pkg/instructor"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/rs/zerolog/log"

	"connectrpc.com/connect"
	"github.com/Jeffail/gabs"
	"github.com/davecgh/go-spew/spew"
	codesurgeon "github.com/wricardo/code-surgeon"
	"github.com/wricardo/code-surgeon/ai"
	"github.com/wricardo/code-surgeon/api"
	"github.com/wricardo/code-surgeon/api/apiconnect"
	"github.com/wricardo/code-surgeon/neo4j2" // Import the neo4j2 package
)

var _ apiconnect.GptServiceHandler = (*Handler)(nil)

type Handler struct {
	publicUrl string
	// chat      chatcli.IChat
	mu sync.Mutex // protects the chat
	// chatRepo chatcli.ChatRepository // TODO: Re-implement without chatcli

	driver           neo4j.DriverWithContext
	instructorClient *instructor.InstructorOpenAI
}

func NewHandler(
	publicUrl string,
	driver neo4j.DriverWithContext,
	instructorClient *instructor.InstructorOpenAI,
) *Handler {
	if instructorClient == nil {
		panic("instructorClient is required")
	}
	// TODO: Re-implement chat repository without chatcli
	// repo, err := chatcli.NewInMemoryChatRepository("chats.json", driver, instructorClient)
	// if err != nil {
	//	panic(err)
	// }
	h := &Handler{
		publicUrl: publicUrl,
		// chatRepo:         repo,
		driver:           driver,
		instructorClient: instructorClient,
	}
	return h
}

func (h *Handler) GetOpenAPI(ctx context.Context, req *connect.Request[api.GetOpenAPIRequest]) (*connect.Response[api.GetOpenAPIResponse], error) {
	// Read the embedded file using the embedded FS
	data, err := codesurgeon.FS.ReadFile("api/codesurgeon.openapi.json")
	if err != nil {
		return nil, err
	}

	parsed, err := gabs.ParseJSON(data)
	if err != nil {
		return nil, err
	}
	// https://chatgpt.com/gpts/editor/g-v09HRlzOu

	// add "server" field
	url := h.publicUrl
	url = strings.TrimSuffix("https://"+url, "/")
	log.Printf("url: %s", spew.Sdump(h.publicUrl))

	parsed.Array("servers")
	parsed.ArrayAppend(map[string]string{
		"url": url,
	}, "servers")

	//
	// Update "openapi" field to "3.1.0"
	parsed.Set("3.1.0", "openapi")

	// Paths to check
	paths, err := parsed.Path("paths").ChildrenMap()
	if err != nil {
		return nil, err
	}

	// Iterate over paths to update "operationId"
	for _, path := range paths {
		// Get the "post" object within each path
		post := path.Search("post")
		if post != nil {

			post.Set("false", "x-openai-isConsequential")

			// Get current "operationId"
			operationID, ok := post.Path("operationId").Data().(string)
			if ok {
				// Split the "operationId" by "."
				parts := strings.Split(operationID, ".")
				operationID = "operationId"
				// Get the last 2 parts of the "operationId" and join them with a "_"
				if len(parts) > 1 {
					operationID = strings.Join(parts[len(parts)-2:], "_")
				} else if len(parts) > 0 {
					operationID = parts[0]
				}
				operationID = strings.TrimPrefix(operationID, "GptService_")

				// Update "operationId"
				post.Set(operationID, "operationId")
			}
		}
	}

	return &connect.Response[api.GetOpenAPIResponse]{
		Msg: &api.GetOpenAPIResponse{
			Openapi: parsed.String(),
		},
	}, nil
}

// ParseCodebase handles the ParseCodebase gRPC method
func (h *Handler) ParseCodebase(ctx context.Context, req *connect.Request[api.ParseCodebaseRequest]) (*connect.Response[api.ParseCodebaseResponse], error) {
	// Extract the parameters from the request
	path := req.Msg.Path
	if path == "" {
		path = "." // Default to current directory if not provided
	}
	format := req.Msg.Format
	plainStructs := req.Msg.PlainStructs
	fieldsPlainStructs := req.Msg.FieldsPlainStructs
	structsWithMethod := req.Msg.StructsWithMethod
	fieldsStructsWithMethod := req.Msg.FieldsStructsWithMethod
	methods := req.Msg.Methods
	functions := req.Msg.Functions
	comments := req.Msg.Comments
	tags := req.Msg.Tags
	ignoreRule := req.Msg.IgnoreRule

	// Call the ParseDirectory function to parse the codebase with all flags
	parsedInfo, err := codesurgeon.ParseDirectoryRecursive(
		path,
	)
	if err != nil {
		log.Printf("Error parsing codebase: %v", err)
		return connect.NewResponse(&api.ParseCodebaseResponse{
			ParsedInfo: "",
		}), err
	}

	// Prepare the response
	response := &api.ParseCodebaseResponse{
		ParsedInfo: codesurgeon.PrettyPrint(parsedInfo, format, ignoreRule, plainStructs, fieldsPlainStructs, structsWithMethod, fieldsStructsWithMethod, methods, functions, comments, tags),
	}

	return connect.NewResponse(response), nil
}

func (h *Handler) SearchSimilarFunctions(ctx context.Context, req *connect.Request[api.SearchSimilarFunctionsRequest]) (*connect.Response[api.SearchSimilarFunctionsResponse], error) {
	embedding, err := ai.EmbedText(h.instructorClient.Client, req.Msg.Objective)
	if err != nil {
		log.Fatal().Err(err).Msg("Error embedding")
	}

	similarNodes, err := findSimilarNodesDirectly(ctx, h.driver, embedding)
	if err != nil {
		log.Fatal().Err(err).Msg("Error finding similar nodes")
	}

	res := &api.SearchSimilarFunctionsResponse{}
	for _, node := range similarNodes {
		res.Functions = append(res.Functions, &api.SearchSimilarFunctionsResponse_Function{
			Code: node.Documentation + "\n" + node.Body,
		})
	}

	return connect.NewResponse(res), nil
}

// findSimilarNodesDirectly finds nodes similar to the given embedding without creating a temporary node
func findSimilarNodesDirectly(ctx context.Context, driver neo4j.DriverWithContext, embedding []float32) ([]struct {
	ID            int64
	Documentation string
	Body          string
	Similarity    float64
}, error) {
	var results []struct {
		ID            int64
		Documentation string
		Body          string
		Similarity    float64
	}

	session := driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	_, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
        MATCH (n:Method)
        WHERE n.embedding IS NOT NULL
        WITH n, gds.similarity.cosine($embedding, n.embedding) AS similarity
        RETURN 
        id(n) AS id, 
        n.documentation AS documentation,
        n.body as body,  
        similarity
        ORDER BY similarity DESC
        LIMIT 10
        `

		result, err := tx.Run(ctx, query, map[string]any{"embedding": embedding})
		if err != nil {
			return nil, err
		}

		for result.Next(ctx) {
			record := result.Record()
			results = append(results, struct {
				ID            int64
				Documentation string
				Body          string
				Similarity    float64
			}{
				ID:            record.Values[0].(int64),
				Documentation: record.Values[1].(string),
				Body:          record.Values[2].(string),
				Similarity:    record.Values[3].(float64),
			})
		}
		return nil, result.Err()
	})

	return results, err
}

// GetNeo4JSchema retrieves the Neo4j schema from the database and returns it as a gRPC response.
// It returns a response containing the schema, which includes labels and relationships, or an error if retrieval fails.
func (h *Handler) GetNeo4JSchema(ctx context.Context, req *connect.Request[api.GetNeo4JSchemaRequest]) (*connect.Response[api.GetNeo4JSchemaResponse], error) {
	schema, err := neo4j2.GetSchema(ctx, h.driver)
	if err != nil {
		return nil, err
	}

	response := &api.GetNeo4JSchemaResponse{
		Schema: &api.Schema{
			Labels:        convertLabelsToProto(schema.Labels),
			Relationships: convertRelationshipsToProto(schema.Relationships),
		},
	}

	return &connect.Response[api.GetNeo4JSchemaResponse]{Msg: response}, nil
}

func convertLabelsToProto(labels []neo4j2.LabelSchema) []*api.LabelSchema {
	var protoLabels []*api.LabelSchema
	for _, label := range labels {
		protoLabel := &api.LabelSchema{
			Label:      label.Label,
			Properties: convertPropertiesToProto(label.Properties),
		}
		protoLabels = append(protoLabels, protoLabel)
	}
	return protoLabels
}

func convertPropertiesToProto(properties []neo4j2.PropertySchema) []*api.PropertySchema {
	var protoProperties []*api.PropertySchema
	for _, property := range properties {
		protoProperty := &api.PropertySchema{
			Property:            property.Property,
			Type:                property.Type,
			IsIndexed:           property.IsIndexed,
			UniqueConstraint:    property.UniqueConstraint,
			ExistenceConstraint: property.ExistenceConstraint,
		}
		protoProperties = append(protoProperties, protoProperty)
	}
	return protoProperties
}

func convertRelationshipsToProto(relationships []neo4j2.RelationshipSchema) []*api.RelationshipSchema {
	var protoRelationships []*api.RelationshipSchema
	for _, relationship := range relationships {
		protoRelationship := &api.RelationshipSchema{
			Relationship: relationship.Relationship,
			FromLabel:    relationship.FromLabel,
			ToLabel:      relationship.ToLabel,
		}
		protoRelationships = append(protoRelationships, protoRelationship)
	}
	return protoRelationships
}

// findSimilarQuestionAnswer finds the Question-Answer pair most similar to the given embedding.
// It uses only the embedding from the Question node.
func findSimilarQuestionAnswer(ctx context.Context, driver neo4j.DriverWithContext, embedding []float32) ([]struct {
	ID         int64
	Question   string
	Answer     string
	Similarity float64
}, error) {
	var results []struct {
		ID         int64
		Question   string
		Answer     string
		Similarity float64
	}

	session := driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	_, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			MATCH (q:Question)-[:HAS_ANSWER]->(a:Answer)
			WHERE q.embedding IS NOT NULL
			WITH q, a, gds.similarity.cosine($embedding, q.embedding) AS similarity
			RETURN id(q) AS id, q.content AS question, a.content AS answer, similarity
			ORDER BY similarity DESC
			LIMIT 5
		`
		result, err := tx.Run(ctx, query, map[string]any{"embedding": embedding})
		if err != nil {
			return nil, err
		}
		for result.Next(ctx) {
			record := result.Record()
			results = append(results, struct {
				ID         int64
				Question   string
				Answer     string
				Similarity float64
			}{
				ID:         record.Values[0].(int64),
				Question:   record.Values[1].(string),
				Answer:     record.Values[2].(string),
				Similarity: record.Values[3].(float64),
			})
		}
		return nil, result.Err()
	})
	return results, err
}

// ThinkThroughProblem handles a problem by finding a similar Question-Answer pair for each question using cosine similarity.
func (h *Handler) ThinkThroughProblem(ctx context.Context, req *connect.Request[api.ThinkThroughProblemRequest]) (*connect.Response[api.ThinkThroughProblemResponse], error) {
	if len(req.Msg.Questions) == 0 && req.Msg.QuestionsString != "" {
		var err error
		req.Msg.Questions, err = ai.ParseQuestions(req.Msg.QuestionsString)
		if err != nil {
			return nil, err
		}
	}

	if len(req.Msg.Questions) == 0 && req.Msg.ProblemStatement != "" {
		req.Msg.Questions = []string{req.Msg.ProblemStatement}
	}

	if len(req.Msg.Questions) == 0 && req.Msg.Goal != "" {
		req.Msg.Questions = []string{req.Msg.Goal}
	}

	similarQuestionsAnswers := make([]*api.QuestionAnswer, 0)
	for _, questionText := range req.Msg.Questions {
		// Embed the question text
		embedding, err := ai.EmbedText(h.instructorClient.Client, questionText)
		if err != nil {
			similarQuestionsAnswers = append(similarQuestionsAnswers, &api.QuestionAnswer{
				Question: questionText,
				Answer:   "Failed to embed question",
			})
			continue
		}
		// Find the most similar Question-Answer pair using the question embedding
		results, err := findSimilarQuestionAnswer(ctx, h.driver, embedding)
		if err != nil {
			similarQuestionsAnswers = append(similarQuestionsAnswers, &api.QuestionAnswer{
				Question: questionText,
				Answer:   "Failed to find similar question",
			})
			continue
		}

		for _, result := range results {
			similarQuestionsAnswers = append(similarQuestionsAnswers, &api.QuestionAnswer{
				Question: result.Question,
				Answer:   result.Answer,
			})
		}
	}

	res, err := ai.ThinkThroughProblem(h.instructorClient, req.Msg, similarQuestionsAnswers)
	if err != nil {
		return nil, err
	}

	return &connect.Response[api.ThinkThroughProblemResponse]{
		Msg: res,
	}, nil

}

// ExecuteNeo4JQuery executes a Neo4j query against the database.
// It takes a context and a request containing the query string.
// It returns a response containing the JSON-encoded result of the query, or an error if one occurred.
func (h *Handler) ExecuteNeo4JQuery(ctx context.Context, req *connect.Request[api.ExecuteNeo4JQueryRequest]) (*connect.Response[api.ExecuteNeo4JQueryResponse], error) {
	session := h.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)
	params := map[string]any{}

	result, err := session.Run(ctx, req.Msg.Query, params)
	if err != nil {
		return nil, err
	}

	records, err := result.Collect(ctx)
	if err != nil {
		return nil, err
	}

	encoded, err := json.Marshal(records)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&api.ExecuteNeo4JQueryResponse{
		Result: string(encoded),
	}), nil
}

// AddKnowledge handles the addition of new knowledge to the knowledge graph.
// It receives a request containing question-answer pairs, generates embeddings for the questions,
// and stores the knowledge in a Neo4j graph database.
//
// The function iterates through the provided question-answer pairs, validates that both question and answer are present,
// generates an embedding for the question using an AI model, and then creates nodes for the question and answer in Neo4j,
// linking them with a HAS_ANSWER relationship.
//
// If any error occurs during the process, such as missing question or answer, failure to generate an embedding,
// or failure to write to Neo4j, the error is logged, and the function continues with the next question-answer pair.
//
// Parameters:
//   - ctx: The context for the request.
//   - req: A connect.Request containing an AddKnowledgeRequest message, which includes a slice of question-answer pairs.
//
// Returns:
//   - A connect.Response containing an AddKnowledgeResponse message (currently empty).
//   - An error if any critical error occurs during the process.  Currently, only returns an error if no knowledge is provided.
func (h *Handler) AddKnowledge(ctx context.Context, req *connect.Request[api.AddKnowledgeRequest]) (*connect.Response[api.AddKnowledgeResponse], error) {
	if len(req.Msg.QuestionAnswer) == 0 {
		return nil, errors.New("no knowledge provided")
	}

	// Open a write session to Neo4j.
	session := h.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	// Iterate over each question-answer pair.
	for _, qa := range req.Msg.QuestionAnswer {
		// Validate that both question and answer are provided.
		if qa.Question == "" || qa.Answer == "" {
			log.Error().Msg("Both question and answer are required for each knowledge item")
			continue
		}

		// Compute the embedding for the question text.
		embedding, err := ai.EmbedText(h.instructorClient.Client, qa.Question)
		if err != nil {
			log.Error().Err(err).Msg("Failed to generate embedding for the question")
			continue
		}

		// Write a new Question node (with its embedding) and a connected Answer node into Neo4j.
		_, err = session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
			query := `
				MERGE (q:Question {content: $question, embedding: $embedding})
				MERGE (a:Answer {content: $answer})
				MERGE (q)-[:HAS_ANSWER]->(a)
				RETURN id(q) as qid
			`
			params := map[string]any{
				"question":  qa.Question,
				"answer":    qa.Answer,
				"embedding": embedding,
			}
			_, err := tx.Run(ctx, query, params)
			return nil, err
		})
		if err != nil {
			log.Error().Err(err).Msg("Failed to add a knowledge item")
			// Continue with next item or decide to return the error depending on your requirements.
		}
	}

	// Return an empty response (or add fields to the response if needed).
	return &connect.Response[api.AddKnowledgeResponse]{Msg: &api.AddKnowledgeResponse{}}, nil
}
