# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Code Surgeon is a multi-purpose Go development tool that combines:
- Code analysis and parsing library for Go AST manipulation
- Neo4j graph database integration for code structure storage

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

# Run code parsing and analysis
go run ./cmd/code-surgeon parse [path]
```

### Testing
```bash
make test             # Run all tests
go test ./neo4j2/...  # Test Neo4j integration
```

## Architecture Overview

### Core Components

1. **AI Integration (`/ai/`)**: Unified interface for OpenAI and Anthropic APIs with instructor-go for structured outputs

2. **Neo4j Integration (`/neo4j2/`)**: Converts Go code structures to graph format for advanced querying

3. **Go Tools (`/gotools/`)**: AST parsing, struct extraction, and code manipulation utilities

### Key Design Patterns

- **Structured AI Outputs**: Uses instructor-go for type-safe AI responses
- **Graph-based Code Analysis**: Stores code relationships in Neo4j

### Dependencies

Major dependencies include:
- `github.com/neo4j/neo4j-go-driver` - Graph database
- `github.com/instructor-ai/instructor-go` - Structured AI outputs

## Development Workflow

1. **Environment Setup**: Copy `.env.example` to `.env` and configure:
   - `OPENAI_API_KEY` - Required for AI features
   - `NEO4j_DB_*` - Database connection details
   - `NGROK_*` - Optional for exposing local server

2. **Code Changes**: After modifying Go files, always run `goimports -w .` before building

3. **Testing**: Write tests for new functionality, especially in `/neo4j2/` packages

## Common Tasks

### Working with Neo4j
- Use `/neo4j2/toneo4j.go` for converting Go structures to graph format
- Query patterns available in cypher mode for reference

## Important Notes

- The project uses Go 1.24.0 with module support
- Infrastructure (Neo4j) must be running for graph functionality
- The `to-neo4j` command uses MERGE operations and does NOT erase existing Neo4j data
- Use `clear-neo4j` command to explicitly clear the database if needed

## Command Reference

For detailed documentation of all CLI commands, see [docs/COMMANDS.md](docs/COMMANDS.md)