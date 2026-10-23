# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Code Surgeon is a multi-purpose Go development tool that combines:
- Code analysis and parsing library for Go AST manipulation
- Embedded code graph (bbolt) for call/implements/structure queries

## Essential Commands

### Build and Install
```bash
make install          # Build and install code-surgeon binary
make build-delve      # Build with debugging symbols for Delve
goimports -w .        # Fix imports and formatting before building
```

### Running the Application
```bash
# Build and query the code graph
go run ./cmd/code-surgeon graph build

# Run code parsing and analysis
go run ./cmd/code-surgeon parse [path]
```

### Testing
```bash
make test             # Run all tests
go test ./codegraph/... # Test the code graph
```

## Architecture Overview

### Core Components

1. **AI Integration (`/ai/`)**: Unified interface for OpenAI and Anthropic APIs with instructor-go for structured outputs

2. **Code Graph (`/codegraph/`)**: Builds a type-checked graph of Go code (calls, implements, imports) in bbolt for traversal queries

3. **Go Tools (`/gotools/`)**: AST parsing, struct extraction, and code manipulation utilities

### Key Design Patterns

- **Structured AI Outputs**: Uses instructor-go for type-safe AI responses
- **Graph-based Code Analysis**: Stores code relationships in an embedded bbolt graph

### Dependencies

Major dependencies include:
- `go.etcd.io/bbolt` - Embedded key-value store backing the code graph
- `golang.org/x/tools` - go/packages type loading
- `github.com/instructor-ai/instructor-go` - Structured AI outputs

## Development Workflow

1. **Environment Setup**: create a `.env` and configure:
   - `OPENAI_API_KEY` - Required for AI features
   - `NGROK_*` - Optional for exposing local server

2. **Code Changes**: After modifying Go files, always run `goimports -w .` before building

3. **Testing**: Write tests for new functionality, especially in `/codegraph/` packages

## Common Tasks

### Working with the code graph
- `codegraph/build.go` extracts nodes/edges with go/packages + go/types; `codegraph/store.go` persists them in bbolt; `codegraph/query.go` is the traversal builder.
- Query from the CLI: `code-surgeon graph build`, then `graph callers|callees|implementers|neighbors|find|stats`.

## Important Notes

- The project uses Go 1.24.0 with module support
- `graph build` replaces the whole database file `.codesurgeon/graph.db`; re-run it after code changes

## Command Reference

For detailed documentation of all CLI commands, see [docs/COMMANDS.md](docs/COMMANDS.md)