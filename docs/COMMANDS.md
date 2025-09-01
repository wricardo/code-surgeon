# Code Surgeon CLI Commands Reference

This document provides a comprehensive reference for all commands available in the code-surgeon CLI tool.

## Table of Contents

- [Core Commands](#core-commands)
  - [chat](#chat)
  - [server](#server)
  - [message](#message)
  - [new-chat](#new-chat)
- [Code Analysis Commands](#code-analysis-commands)
  - [parse](#parse)
  - [document-functions](#document-functions)
- [Neo4j Graph Database Commands](#neo4j-graph-database-commands)
  - [to-neo4j](#to-neo4j)
  - [clear-neo4j](#clear-neo4j)
  - [get-schema](#get-schema)
  - [generate-embeddings](#generate-embeddings)
- [AI & Knowledge Base Commands](#ai--knowledge-base-commands)
  - [ingest-knowledgebase](#ingest-knowledgebase)
  - [mcp-server](#mcp-server)
- [API Documentation Commands](#api-documentation-commands)
  - [openapi-json](#openapi-json)
  - [introduction](#introduction)

## Prerequisites

Before using the CLI commands, ensure you have:

1. **Environment Variables** configured in a `.env` file:
   ```bash
   # Required for AI features
   OPENAI_API_KEY=sk-...
   
   # Required for Neo4j commands
   NEO4J_DB_URI=neo4j://localhost
   NEO4J_DB_USER=neo4j
   NEO4J_DB_PASSWORD=neo4jneo4j
   
   # Optional for remote access
   NGROK_AUTH_TOKEN=your_token
   NGROK_DOMAIN=your-domain.ngrok-free.app
   ```

2. **Infrastructure** running (for full functionality):
   ```bash
   docker-compose up
   ```

## Core Commands

### chat

Start an interactive chat session with the AI assistant.

```bash
code-surgeon chat [options]
```

**Options:**
- `--chat-id` - Resume an existing chat session by ID (optional)

**Description:**
- Creates a new chat session if no ID is provided
- Connects to the gRPC server
- Provides an interactive terminal UI with rich formatting
- Supports multiple chat modes (code, architect, teacher, cypher, Q&A)
- Handles graceful shutdown on SIGINT/SIGTERM

**Example:**
```bash
# Start a new chat
code-surgeon chat

# Resume existing chat
code-surgeon chat --chat-id=550e8400-e29b-41d4-a716-446655440000
```

### server

Run the gRPC/Connect RPC server that handles all API requests.

```bash
code-surgeon server [options]
```

**Options:**
- `--port`, `-p` - Port number to listen on (default: 8010)

**Description:**
- Starts the main server that handles chat sessions, API requests, and integrations
- Provides Connect RPC endpoints (HTTP/REST compatible)
- Required for chat, message, and other client commands to work

**Example:**
```bash
# Run on default port
code-surgeon server

# Run on custom port
code-surgeon server --port 9000
```

### message

Send a message to an existing chat session via gRPC.

```bash
code-surgeon message --chat-id=<chat-id> < message.txt
```

**Options:**
- `--chat-id` - The ID of the chat session (required)

**Description:**
- Reads message content from stdin
- Sends the message to the specified chat session
- Prints the AI response
- Useful for scripting and automation

**Example:**
```bash
echo "What is the purpose of the parse command?" | code-surgeon message --chat-id=550e8400-e29b-41d4-a716-446655440000
```

### new-chat

Create a new chat session programmatically.

```bash
code-surgeon new-chat
```

**Description:**
- Connects to the gRPC server
- Creates a new chat session
- Returns the new chat ID
- Useful for scripting when you need to create sessions programmatically

**Example:**
```bash
# Create new chat and save ID
CHAT_ID=$(code-surgeon new-chat)
echo "New chat created: $CHAT_ID"
```

## Code Analysis Commands

### parse

Parse Go code to extract structural information.

```bash
code-surgeon parse [options]
```

**Alias:** `p`

**Options:**
- `--path`, `-f` - Path to file or directory to parse (default: ".")
- `--recursive`, `-r` - Recursively parse directories
- `--format` - Output format: `llm`, `text_short`, `text_long`, `json` (default: "llm")
- `--plain-structs` - Print plain structs (default: true)
- `--fields-plain-structs` - Print fields of plain structs (default: true)
- `--structs-with-method` - Print structs with methods (default: true)
- `--fields-structs-with-method` - Print fields of structs with methods (default: true)
- `--methods` - Print methods (default: true)
- `--functions` - Print functions (default: true)
- `--comments` - Print comments (default: true)
- `--tags` - Print struct field tags (default: true)
- `--ignore-rule` - Ignore files/directories matching the rule (can be specified multiple times)

**Description:**
- Parses Go code to extract packages, structs, interfaces, functions, and methods
- Supports multiple output formats for different use cases
- Can filter what elements to include in the output
- Useful for code analysis, documentation, and AI training

**Examples:**
```bash
# Parse current directory with default settings
code-surgeon parse

# Parse specific file with JSON output
code-surgeon parse --path main.go --format json

# Recursively parse, excluding test files
code-surgeon parse --recursive --ignore-rule "*_test.go"

# Parse only functions and methods
code-surgeon parse --plain-structs=false --structs-with-method=false --functions=true --methods=true
```

### document-functions

Generate AI-powered documentation for Go functions and methods.

```bash
code-surgeon document-functions [options]
```

**Options:**
- `--path`, `-p` - Path to Go file or folder (default: ".")
- `--overwrite` - Overwrite existing documentation (default: false)
- `--receiver`, `-r` - Document only methods with specific receiver
- `--function`, `-f` - Document only specific function by name

**Description:**
- Uses AI (OpenAI) to generate documentation for Go code
- Adds doc comments above functions and methods
- Preserves existing documentation unless --overwrite is used
- Can target specific functions or process entire directories

**Examples:**
```bash
# Document all functions in current directory
code-surgeon document-functions

# Document specific function
code-surgeon document-functions --function ParseStruct

# Document all methods of a specific receiver
code-surgeon document-functions --receiver Parser

# Overwrite existing documentation
code-surgeon document-functions --overwrite
```

## Neo4j Graph Database Commands

### to-neo4j

Parse Go code and store it as a graph in Neo4j database.

```bash
code-surgeon to-neo4j [options]
```

**Alias:** `tn`

**Options:**
- `--path`, `-p` - Path to parse (default: "./.")
- `--deep` - Enable deep parsing with file path resolution (default: false)
- `--recursive` - Enable recursive directory parsing (default: false)

**Environment Variables Required:**
- `NEO4J_DB_URI` - Neo4j connection URI (e.g., `neo4j://localhost`)
- `NEO4J_DB_USER` - Database username
- `NEO4J_DB_PASSWORD` - Database password

**Description:**
- Parses Go modules and creates a knowledge graph in Neo4j
- Creates nodes for: modules, packages, structs, fields, methods, functions, interfaces, types
- Creates relationships: BELONGS_TO, HAS_FIELD, HAS_METHOD, OF_TYPE, BASE_TYPE
- Enables powerful graph queries for code analysis

**Graph Structure Created:**
```
Module 
  └─ BELONGS_TO → Package
                     ├─ BELONGS_TO → Struct (with 'definition' property containing full Go struct code)
                     │                 ├─ HAS_FIELD → Field → OF_TYPE → Type
                     │                 └─ HAS_METHOD → Method (with 'definition' property)
                     ├─ BELONGS_TO → Function (with 'definition' property containing full signature)
                     └─ BELONGS_TO → Interface (with 'definition' property containing full interface code)
                                       └─ HAS_METHOD → Method (with 'definition' property)
```

**Node Properties Include:**
- **Struct nodes**: `definition` property contains the complete Go struct definition
- **Function nodes**: `definition` property contains the full function signature (e.g., `func FunctionName(param Type) ReturnType`)
- **Method nodes**: `definition` property contains the full method signature (e.g., `func (r ReceiverType) MethodName(param Type) ReturnType`)
- **Interface nodes**: `definition` property contains the complete interface definition with all methods

**Examples:**
```bash
# Parse current directory
code-surgeon to-neo4j

# Parse specific project recursively
code-surgeon to-neo4j --path /path/to/project --recursive

# Deep parsing with file path resolution
code-surgeon to-neo4j --deep
```

### clear-neo4j

Clear all nodes and relationships from the Neo4j database.

```bash
code-surgeon clear-neo4j
```

**Description:**
- Removes ALL data from the Neo4j database
- Use with caution - this operation cannot be undone
- Useful for resetting the database before re-importing

**Example:**
```bash
# Clear the database
code-surgeon clear-neo4j
```

### get-schema

Retrieve and display the Neo4j database schema.

```bash
code-surgeon get-schema [options]
```

**Options:**
- `--format`, `-f` - Output format: `json` or `llm` (default: "json")

**Description:**
- Connects to Neo4j and retrieves the current schema
- Shows node labels, relationship types, and properties
- `llm` format is optimized for AI consumption

**Examples:**
```bash
# Get schema in JSON format
code-surgeon get-schema

# Get schema formatted for LLM
code-surgeon get-schema --format llm
```

### generate-embeddings

Generate vector embeddings for code elements in Neo4j.

```bash
code-surgeon generate-embeddings
```

**Alias:** `ge`

**Description:**
- Generates OpenAI embeddings for functions and methods with documentation
- Stores embeddings in Neo4j for semantic search
- Requires OpenAI API key in environment

**Example:**
```bash
# Generate embeddings for all documented code
code-surgeon generate-embeddings
```

## AI & Knowledge Base Commands

### ingest-knowledgebase

Import Q&A pairs from text files into the knowledge base.

```bash
code-surgeon ingest-knowledgebase [options]
```

**Options:**
- `--path`, `-p` - Path to knowledge base folder (default: "./knowledgebase")

**Description:**
- Reads all .txt files from the specified directory
- First non-empty line is treated as the question
- Remaining content becomes the answer
- Sends Q&A pairs to the server for storage

**File Format:**
```txt
How do I implement error handling in Go?

In Go, error handling is done through explicit error returns...
[rest of the answer]
```

**Example:**
```bash
# Ingest from default location
code-surgeon ingest-knowledgebase

# Ingest from custom location
code-surgeon ingest-knowledgebase --path /path/to/kb
```

### mcp-server

Start a Model Context Protocol (MCP) server.

```bash
code-surgeon mcp-server
```

**Description:**
- Starts an MCP server that communicates via stdio
- Provides tools for AI assistants to interact with the codebase
- Available tools:
  - Search similar functions
  - Think through problems step-by-step
  - Add knowledge to the system
  - Read/write scratchpad for temporary storage
  - Execute Neo4j Cypher queries
  - Ask questions to domain experts
  - Get Neo4j schema information

**Example:**
```bash
# Start MCP server
code-surgeon mcp-server
```

## API Documentation Commands

### openapi-json

Generate and display the OpenAPI specification.

```bash
code-surgeon openapi-json [options]
```

**Options:**
- `--url`, `-u` - Server URL (default: "http://localhost:8010")

**Description:**
- Connects to the server and retrieves OpenAPI spec
- Outputs JSON specification to stdout
- Useful for API documentation and client generation

**Example:**
```bash
# Get OpenAPI spec from local server
code-surgeon openapi-json

# Get from remote server
code-surgeon openapi-json --url https://api.example.com
```

### introduction

Generate introduction text for LLM context.

```bash
code-surgeon introduction [options]
```

**Options:**
- `--url`, `-u` - Server URL (default: "http://localhost:8010")

**Description:**
- Retrieves OpenAPI spec and generates introduction text
- Formatted specifically for LLM consumption
- Provides context about available APIs and capabilities

**Example:**
```bash
# Generate introduction for local server
code-surgeon introduction
```

## Common Workflows

### 1. Analyzing a Go Project

```bash
# Parse and store in Neo4j
code-surgeon to-neo4j --path /my/project --recursive

# Generate embeddings for semantic search
code-surgeon generate-embeddings

# Start interactive chat to query the code
code-surgeon chat
```

### 2. Documenting Code

```bash
# Generate AI documentation
code-surgeon document-functions --path ./pkg

# Parse to verify structure
code-surgeon parse --format json > structure.json
```

### 3. Running as a Service

```bash
# Terminal 1: Start infrastructure
docker-compose up

# Terminal 2: Start server
code-surgeon server

# Terminal 3: Use the service
code-surgeon chat
```

### 4. Batch Processing

```bash
# Create new chat
CHAT_ID=$(code-surgeon new-chat)

# Send multiple messages
echo "Analyze the main function" | code-surgeon message --chat-id=$CHAT_ID
echo "What design patterns are used?" | code-surgeon message --chat-id=$CHAT_ID
```

## Troubleshooting

1. **Connection Errors**: Ensure the server is running (`code-surgeon server`)
2. **Neo4j Errors**: Check that Docker containers are running and environment variables are set
3. **AI Errors**: Verify OPENAI_API_KEY is set in .env file
4. **Import Errors**: Run `goimports -w .` before building