# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Code Surgeon is a multi-purpose Go development tool that combines:
- Code analysis and parsing library for Go AST manipulation
- AI-powered chatbot with multiple interaction modes
- gRPC/Connect RPC server for remote access
- Neo4j graph database integration for code structure storage
- MCP (Model Context Protocol) server support

## Essential Commands

### Build and Install
```bash
make install          # Build and install code-surgeon binary
make build-delve      # Build with debugging symbols for Delve
goimports -w .        # Fix imports and formatting before building
```

### Running the Application
```bash
# Start infrastructure (required for full functionality)
docker-compose up

# Run as gRPC server
make run-server
# or
go run ./cmd/code-surgeon server

# Start interactive chat session
make new-chat
# or
go run ./cmd/code-surgeon chat

# Continue existing chat
make continue-chat CHAT_ID=<id>
```

### Testing
```bash
make test             # Run all tests
go test ./grpc/...    # Test gRPC package
go test ./chatcli/... # Test chat CLI package
```

### Protocol Buffers
```bash
make generate-proto   # Regenerate Go code from .proto files
```

## Architecture Overview

### Core Components

1. **AI Integration (`/ai/`)**: Unified interface for OpenAI and Anthropic APIs with instructor-go for structured outputs

2. **Chat CLI (`/chatcli/`)**: Rich terminal UI with multiple modes:
   - Code mode: AI-assisted code generation
   - Architect mode: System design assistance
   - Teacher mode: Interactive learning
   - Cypher mode: Neo4j query assistance
   - Question/Answer mode: General Q&A

3. **gRPC Server (`/grpc/`)**: Connect RPC server implementing services defined in `/api/codesurgeon.proto`

4. **Neo4j Integration (`/neo4j2/`)**: Converts Go code structures to graph format for advanced querying

5. **Go Tools (`/gotools/`)**: AST parsing, struct extraction, and code manipulation utilities

### Key Design Patterns

- **Repository Pattern**: Chat sessions stored with configurable backends
- **Mode-based Chat**: Different chat modes with specialized system prompts
- **Structured AI Outputs**: Uses instructor-go for type-safe AI responses
- **Graph-based Code Analysis**: Stores code relationships in Neo4j

### Dependencies

The project uses Connect RPC (not standard gRPC) for better HTTP/REST compatibility. Major dependencies include:
- `connectrpc.com/connect` - RPC framework
- `github.com/neo4j/neo4j-go-driver` - Graph database
- `github.com/charmbracelet/*` - Terminal UI components
- `github.com/instructor-ai/instructor-go` - Structured AI outputs

## Development Workflow

1. **Environment Setup**: Copy `.env.example` to `.env` and configure:
   - `OPENAI_API_KEY` - Required for AI features
   - `NEO4j_DB_*` - Database connection details
   - `NGROK_*` - Optional for exposing local server

2. **Code Changes**: After modifying Go files, always run `goimports -w .` before building

3. **Proto Changes**: Run `make generate-proto` after modifying `.proto` files

4. **Testing**: Write tests for new functionality, especially in `/grpc/` and `/chatcli/` packages

## Common Tasks

### Adding New Chat Mode
1. Create new mode file in `/chatcli/` (e.g., `my_mode.go`)
2. Implement the mode interface with custom system prompt
3. Register mode in chat instance initialization

### Extending gRPC API
1. Modify `/api/codesurgeon.proto`
2. Run `make generate-proto`
3. Implement new methods in `/grpc/server.go`

### Working with Neo4j
- Use `/neo4j2/toneo4j.go` for converting Go structures to graph format
- Query patterns available in cypher mode for reference

## Important Notes

- The project uses Go 1.23.0 with module support
- All generated code goes to `/api/` and `/api/apiconnect/`
- Infrastructure (Neo4j, PostgreSQL, Temporal) must be running for full functionality
- Chat sessions are persisted and can be resumed using chat IDs
- The `to-neo4j` command uses MERGE operations and does NOT erase existing Neo4j data
- Use `clear-neo4j` command to explicitly clear the database if needed

## Command Reference

For detailed documentation of all CLI commands, see [docs/COMMANDS.md](docs/COMMANDS.md)