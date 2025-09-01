package main

import (
	"context" // added for MySQL support
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	_ "github.com/go-sql-driver/mysql" // MySQL driver

	"connectrpc.com/connect"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/wricardo/code-surgeon/api"
	"github.com/wricardo/code-surgeon/api/apiconnect"
)

const CODE_SURGEON_ADDRESS = "http://localhost:8010"

var scratchpadFile string = "/Users/wricardo/go/src/bitbucket.org/zetaactions/trinity/ai-context.txt"

// MCPServer wraps the underlying MCP server and related dependencies.
type MCPServer struct {
	server *server.MCPServer
}

func NewMCPServer() *MCPServer {
	// Create an MCPServer with logging and resource capabilities.
	s := server.NewMCPServer("code-surgeon", "0.1.0",
		// server.WithResourceCapabilities(false, true),
		server.WithLogging(),
	)
	w := &MCPServer{server: s}

	// Register resources and tools.
	w.registerTools()
	return w
}

// registerTools sets up all available tools.
func (w *MCPServer) registerTools() {
	w.addSearchSimilarFunctionsTool()
	w.addThinkThroughTheProblemTool()
	w.addAddKnowledgeTool()
	w.addReadScratchpadTool()
	w.addWriteScratchpadTool()
	w.addExecuteNeo4jQueryTool()
	w.addAskQuestionsTool()
	w.addGetNeo4jSchemaTool()
}

func (w *MCPServer) addAskQuestionsTool() {
	tool := mcp.NewTool("askQuestions",
		mcp.WithDescription("Ask one or more questions to the experts."),
		mcp.WithString("questions", mcp.Description("list of questions to ask the experts. New line separated. Provide question with full context and details.")),
	)
	w.server.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.Params.Arguments
		problem, _ := args["problemStatement"].(string)
		contextStr, _ := args["context"].(string)
		goal, _ := args["goal"].(string)
		questions, _ := args["questions"].(string)

		client := apiconnect.NewGptServiceClient(http.DefaultClient, CODE_SURGEON_ADDRESS)
		req := &api.ThinkThroughProblemRequest{
			Goal:             goal,
			Context:          contextStr,
			ProblemStatement: problem,
			QuestionsString:  questions,
		}

		response, err := client.ThinkThroughProblem(ctx, connect.NewRequest(req))
		if err != nil {
			return toolError("Error: " + err.Error()), nil
		}
		return toolSuccess([]mcp.TextContent{{Type: "text", Text: response.Msg.String()}}), nil
	})
}

func (w *MCPServer) addSearchSimilarFunctionsTool() {
	tool := mcp.NewTool("searchSimilarFunctionsTool",
		mcp.WithDescription("Searches for similar functions based on the provided function description."),
		mcp.WithString("functionDescription", mcp.Description("The description of the function to search for similar functions."), mcp.Required()),
	)
	w.server.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		desc, _ := request.Params.Arguments["functionDescription"].(string)
		client := apiconnect.NewGptServiceClient(http.DefaultClient, CODE_SURGEON_ADDRESS)
		response, err := client.SearchSimilarFunctions(ctx, connect.NewRequest(&api.SearchSimilarFunctionsRequest{
			Objective: desc,
		}))
		if err != nil {
			return toolError("Error: " + err.Error()), nil
		}

		var txt strings.Builder
		for _, f := range response.Msg.Functions {
			txt.WriteString(fmt.Sprintf("%s\n", f.Code))
		}
		return toolSuccess([]mcp.TextContent{{Type: "text", Text: txt.String()}}), nil
	})
}

func (w *MCPServer) addThinkThroughTheProblemTool() {
	tool := mcp.NewTool("thinkThroughTheProblem",
		mcp.WithDescription("Think through the problem and provide a solution. Or just ask a question to the experts."),
		mcp.WithString("goal", mcp.Description("A possible goal to achieve.")),
		mcp.WithString("problemStatement", mcp.Description("The description of the problem to think through.")),
		mcp.WithString("questions", mcp.Description("list of questions to ask the experts. New line separated.")),
		mcp.WithString("context", mcp.Description("The context of the problem.")),
	)
	w.server.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.Params.Arguments
		problem, _ := args["problemStatement"].(string)
		contextStr, _ := args["context"].(string)
		goal, _ := args["goal"].(string)
		questions, _ := args["questions"].(string)

		client := apiconnect.NewGptServiceClient(http.DefaultClient, CODE_SURGEON_ADDRESS)
		req := &api.ThinkThroughProblemRequest{
			Goal:             goal,
			Context:          contextStr,
			ProblemStatement: problem,
			QuestionsString:  questions,
		}

		response, err := client.ThinkThroughProblem(ctx, connect.NewRequest(req))
		if err != nil {
			return toolError("Error: " + err.Error()), nil
		}
		return toolSuccess([]mcp.TextContent{{Type: "text", Text: response.Msg.String()}}), nil
	})
}

func (w *MCPServer) addAddKnowledgeTool() {
	tool := mcp.NewTool("addKnowledge",
		mcp.WithDescription("Adds knowledge to the system by storing question-answer pairs."),
		mcp.WithString("questionAnswers", mcp.Description("A JSON array of question-answer objects with 'question' and 'answer' fields."), mcp.Required()),
	)
	w.server.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		qaStr, _ := request.Params.Arguments["questionAnswers"].(string)
		var qas []struct {
			Question string `json:"question"`
			Answer   string `json:"answer"`
		}
		if err := json.Unmarshal([]byte(qaStr), &qas); err != nil {
			return toolError("Error parsing questionAnswers JSON: " + err.Error()), nil
		}

		var qaProtos []*api.QuestionAnswer
		for _, qa := range qas {
			qaProtos = append(qaProtos, &api.QuestionAnswer{Question: qa.Question, Answer: qa.Answer})
		}
		client := apiconnect.NewGptServiceClient(http.DefaultClient, CODE_SURGEON_ADDRESS)
		_, err := client.AddKnowledge(ctx, connect.NewRequest(&api.AddKnowledgeRequest{QuestionAnswer: qaProtos}))
		if err != nil {
			return toolError("Error calling AddKnowledge: " + err.Error()), nil
		}
		return toolSuccess([]mcp.TextContent{{Type: "text", Text: "Knowledge added successfully."}}), nil
	})
}

// New tool to read the scratchpad file.
func (w *MCPServer) addReadScratchpadTool() {
	tool := mcp.NewTool("readScratchpad",
		mcp.WithDescription("Reads the content of the scratchpad file."),
	)
	w.server.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data, err := os.ReadFile(scratchpadFile)
		if err != nil {
			return toolError("Error reading scratchpad file: " + err.Error()), nil
		}
		return toolSuccess([]mcp.TextContent{{Type: "text", Text: string(data)}}), nil
	})
}

// New tool to write to the scratchpad file.
func (w *MCPServer) addWriteScratchpadTool() {
	tool := mcp.NewTool("writeScratchpad",
		mcp.WithDescription("Append content to the scratchpad file."),
		mcp.WithString("content", mcp.Description("Content to append to the scratchpad."), mcp.Required()),
	)
	w.server.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.Params.Arguments
		content, _ := args["content"].(string)
		appendMode := true

		if appendMode {
			f, err := os.OpenFile(scratchpadFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
			if err != nil {
				return toolError("Error opening file for appending: " + err.Error()), nil
			}
			defer f.Close()
			if _, err := f.WriteString(content); err != nil {
				return toolError("Error appending to file: " + err.Error()), nil
			}
		} else {
			if err := os.WriteFile(scratchpadFile, []byte(content), 0644); err != nil {
				return toolError("Error writing to file: " + err.Error()), nil
			}
		}
		return toolSuccess([]mcp.TextContent{{Type: "text", Text: "Scratchpad updated successfully."}}), nil
	})
}

func (w *MCPServer) addExecuteNeo4jQueryTool() {
	tool := mcp.NewTool("executeNeo4jQuery",
		mcp.WithDescription("Executes a Neo4j cypher query using the CodeSurgeon API."),
		mcp.WithString("query", mcp.Description("The Neo4j cypher query string."), mcp.Required()),
	)
	w.server.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.Params.Arguments
		query, _ := args["query"].(string)
		client := apiconnect.NewGptServiceClient(http.DefaultClient, CODE_SURGEON_ADDRESS)
		response, err := client.ExecuteNeo4JQuery(ctx, connect.NewRequest(&api.ExecuteNeo4JQueryRequest{Query: query}))
		if err != nil {
			return toolError("Error executing Neo4j query: " + err.Error()), nil
		}
		// Assuming response.Msg.Result holds the query result.
		return toolSuccess([]mcp.TextContent{{Type: "text", Text: response.Msg.Result}}), nil
	})
}

// Adds a tool to fetch the Neo4j schema (converted from previous resource)
func (w *MCPServer) addGetNeo4jSchemaTool() {
	tool := mcp.NewTool("get_neo4j_schema",
		mcp.WithDescription("Fetches the Neo4j schema using the GetSchema RPC from the CodeSurgeon API."),
	)
	w.server.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		client := apiconnect.NewGptServiceClient(http.DefaultClient, CODE_SURGEON_ADDRESS)
		response, err := client.GetNeo4JSchema(ctx, connect.NewRequest(&api.GetNeo4JSchemaRequest{}))
		if err != nil {
			return toolError("Error fetching Neo4j schema: " + err.Error()), nil
		}

		schemaStr := string(response.Msg.Schema.String())
		return toolSuccess([]mcp.TextContent{{Type: "text", Text: schemaStr}}), nil
	})
}

// Helper to build a tool success response.
func toolSuccess(contents []mcp.TextContent) *mcp.CallToolResult {
	var iface []interface{}
	for _, c := range contents {
		iface = append(iface, c)
	}
	return &mcp.CallToolResult{Content: iface, IsError: false}
}

// Helper to build a tool error response.
func toolError(message string) *mcp.CallToolResult {
	return &mcp.CallToolResult{Content: []interface{}{mcp.NewTextContent(message)}, IsError: true}
}

func (w *MCPServer) ServeSSE(addr string) error {
	sseServer := server.NewSSEServer(w.server, fmt.Sprintf("http://localhost%s", addr))
	log.Printf("[wally] SSE server listening on %s\n", addr)
	return sseServer.Start(addr)
}

func (w *MCPServer) ServeStdio() error {
	return server.ServeStdio(w.server)
}
