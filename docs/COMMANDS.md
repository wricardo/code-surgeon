# Code Surgeon CLI Commands Reference

This document provides a comprehensive reference for all commands available in the code-surgeon CLI tool.

## Table of Contents

- [Core Commands](#core-commands)
  - [server](#server)
- [Code Analysis Commands](#code-analysis-commands)
  - [parse](#parse)
  - [document-functions](#document-functions)
- [Embedded Code Graph Commands](#embedded-code-graph-commands)
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
   
   # Optional for remote access
   NGROK_AUTH_TOKEN=your_token
   NGROK_DOMAIN=your-domain.ngrok-free.app
   ```

## Core Commands


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

## Embedded Code Graph Commands

`graph` builds a type-checked code graph with `go/packages` and stores it in a single bbolt file (default `.codesurgeon/graph.db`, git-ignored). No Docker or external service is needed. Queries open the file read-only; a `graph build` holds an exclusive lock while it writes, and each build atomically replaces the previous graph.

See [GRAPH.md](GRAPH.md) for a usage guide with recipes (understand a feature, check what a change affects, find dead code), pitfalls and limits.

```bash
code-surgeon graph build [--dir .] [--tests] [--if-stale] [--db FILE] [patterns...]   # default ./...
code-surgeon graph stats
code-surgeon graph find -k Struct ParsedInfo          # name-prefix search; -k Package|Struct|Interface|Type|Func|Method|Field
code-surgeon graph search "welcome message"           # by concept: name substring, doc comment or signature (--name/--doc/--sig)
code-surgeon graph types github.com/x/y              # structs, interfaces, named types of a package (also: short name, -k, --exported, -r)
code-surgeon graph decls github.com/x/y               # types, functions and methods with signatures (also -k, --exported, -r)
code-surgeon graph funcs --exported github.com/x/y    # just functions and methods
code-surgeon graph callers ParseDirectory             # direct callers
code-surgeon graph callers -d 0 ParseDirectory        # transitive callers (-d 0 = unlimited)
code-surgeon graph callers --refs Handler             # also count uses as a value (callbacks)
code-surgeon graph callees ParseFile
code-surgeon graph describe Server                     # everything about one node
code-surgeon graph usages Server                       # where a type is used
code-surgeon graph tree -d 4 --no-ext Handle           # call tree, crossing interface calls
code-surgeon graph roots queryDB                       # which entry points reach this?
code-surgeon graph path Handle queryDB                 # shortest call chain
code-surgeon graph unused                             # dead-code candidates
code-surgeon graph impact                             # what do my uncommitted changes affect?
code-surgeon graph implementers Shape                 # types implementing an interface
code-surgeon graph implements Rect                    # interfaces a type implements
code-surgeon graph neighbors --rel HAS_METHOD StackToGraph
code-surgeon graph neighbors --dir in --rel IMPORTS github.com/x/y
```

Arguments are an exact node id (`method:github.com/x/y.Calculator.Add`) or an exact name (`Add`, `Calculator.Add`); an ambiguous name queries the union and lists the matched ids on stderr. All query commands take `--format table|json`.

**Nodes:** `Package`, `Struct`, `Interface`, `Type`, `Func`, `Method`, `Field`. Declarations outside the loaded packages (stdlib, dependencies, `error`) appear as stub nodes marked `(external)`.

**Relations:** `BELONGS_TO`, `HAS_METHOD`, `HAS_FIELD`, `EMBEDS`, `FIELD_TYPE`, `PARAM_TYPE`, `RETURNS_TYPE`, `CALLS`, `REFERENCES`, `IMPLEMENTS`, `IMPORTS`.

**Limits:** `CALLS` records statically resolved calls. A call through an interface points at the interface method, not at implementations (join with `IMPLEMENTS`). Calls through function-typed variables have no static callee, but the function used as a value gets a `REFERENCES` edge. Calls and function values in package-level `var` initialisers are attributed to the `Package` node. `IMPLEMENTS` covers interfaces declared in the loaded packages and skips generic types.

**Freshness:** the graph is a snapshot. Query commands print a warning on stderr when a `.go` file, `go.mod` or `go.sum` under the build directory is newer than the build (deleted files are not detected). `graph build --if-stale` skips the work when the database was built with the same directory, patterns and `--tests` and nothing changed.

### search

```bash
code-surgeon graph search [-k KIND] [--name] [--doc] [--sig] [--no-ext] [-n 100] <text>
```
Finds nodes by concept when you don't know the exact name. It is a case-insensitive substring search over names, doc comments and signatures (all three by default; `--name`, `--doc` and `--sig` restrict it). Doc comments are matched with line breaks collapsed, so a phrase such as `welcome message` matches a doc that wraps between the two words. A signature search finds types by what functions take or return (`graph search --sig "*Subscription"`; the receiver of a method is part of its signature too).

- Each node is listed once, for its best match, in this order: exact name, name prefix, name substring, doc, signature. Ties list hand-written code before generated files (`generated`, `_gen.go`, `_templ.go`, `.pb.go`), and functions before methods, structs, interfaces, types and fields.
- The `MATCH` column shows `name`, `doc: …snippet…` or `sig: <signature>`. `-f json` returns `{node, match, rank, snippet}` objects.
- `-n` caps the output (default 100; `0` = all); a note on stderr gives the total. Narrow with `-k` (one kind) and `--no-ext`.
- Unlike `find` (indexed name prefix, instant), `search` scans every node: about 0.3s on a 68k-node graph. It only sees doc comments and signatures, not inline comments or string literals.

### types, decls, funcs

```bash
code-surgeon graph types [-k struct|interface|type] [--exported] [-r] <package>
code-surgeon graph decls [-k struct|interface|type|func|method] [--exported] [-r] <package>
code-surgeon graph funcs [--exported] [-r] <package>      # = decls -k func -k method
```
- **`types`** lists the structs, interfaces and named types a package declares.
- **`decls`** lists everything a package declares: types, functions and methods. Output columns are kind, name, location and signature (`-f json` has the full node, including `doc`). Struct fields are not listed; use `describe <struct>`. Interface method declarations appear as methods.
- **`funcs`** is `decls` restricted to functions and methods.
- The package is an import path or a short name (a short name shared by several packages matches all of them). `-r` includes sub-packages, and with an import path it also works on a directory that has no Go files of its own (every package under that path). Packages outside the loaded set (stdlib, dependencies) have no declarations in the graph.

### describe, usages, tree, roots, path

```bash
code-surgeon graph describe <name|id> [--all]          # location, signature, doc, neighbours grouped by relation
code-surgeon graph usages <type>                       # where a type is used: parameters, results, fields, embedding
code-surgeon graph tree [-d 3] [--up] [--refs] [--no-ext] [--no-dispatch] <func|method>
code-surgeon graph roots <func|method>                 # entry points that can reach it
code-surgeon graph path <from> <to> [--refs] [--no-dispatch]
```
- **`describe`** prints the node and every incoming (`←`) and outgoing (`→`) relation with its neighbours. Lists longer than 10 are truncated unless `--all` is given; `-f json` returns everything.
- **`usages`** answers "where is this type used": functions/methods taking it, returning it, fields holding it (through pointers, slices, maps, type arguments) and types embedding it. Only types (struct, interface, named type) are candidates, so an embedded field with the same short name does not make the name ambiguous.
- **`tree`** expands callees (or callers with `--up`) as an indented tree. A node is expanded where it first appears; later occurrences are marked `(*)`, so cycles and shared helpers do not repeat. Depth defaults to 3 (`-d 0` = unlimited); `[+N more]` marks children hidden by the depth limit. A call through an interface continues into the implementations, marked `[dispatch]` (`--no-dispatch` turns that off). `--no-ext` hides stdlib and dependency nodes.
- **`roots`** walks callers transitively (calls and references, across interface dispatch) and prints the nodes nothing uses: `main`, handlers wired by generated code, tests, unused code and package initialisers. It answers "which endpoints or jobs end up running this?". A pure cycle has no entry point and says so.
- **`path`** prints the shortest call chain from one function to another, with the relation of each step (`CALLS`, `REFERENCES` with `--refs`, `DISPATCH`). Paths are directed. A missing path usually means the link goes through a function value, HTTP/RPC or a workflow engine.

When a name matches both project code and external code (`ParseFile` exists locally and as `go/parser.ParseFile`), these commands and `callers`/`callees` use the project's declarations. Pass the node id to query the external one.

### unused

```bash
code-surgeon graph unused [--exported]
```
Lists declarations with no static caller and no reference as a value. A self-call does not count. Never reported: `main`, `init`, tests, interface method declarations, and methods reachable through interface dispatch (the receiver implements an interface that declares a method of the same name). Exported declarations are skipped unless `--exported` is given, since they may have callers outside the module. This is a candidate list: reflection, `//go:linkname` and groups of functions that only use each other are not detected.

### impact

```bash
code-surgeon graph build --tests
code-surgeon graph impact [--base HEAD] [-f json]
```
Runs `git diff -U0` against `--base` (default `HEAD`, so staged and unstaged edits), maps changed lines to declarations, then walks callers transitively over `CALLS` and `REFERENCES`, including interface dispatch (a changed method also affects callers of the interface methods it may be called through). A changed type affects its methods and the funcs/methods whose parameters or results mention it. Output sections: `CHANGED`, `AFFECTED`, and `AFFECTED TESTS` (needs `--tests` at build time). Untracked files are not in git's diff; use `git add -N <file>`.

From Go, use `codegraph.Open` and the `Query` builder: `s.Match("helper").In(codegraph.RelCalls).Where(...).Nodes()`.

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
  - Ask questions to domain experts

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
# Build the graph (type-checked, includes calls and interface implementations)
code-surgeon graph build

# Who is affected if ParseDirectory changes?
code-surgeon graph callers -d 0 ParseDirectory

# What can be used as a Shape?
code-surgeon graph implementers Shape
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
2. **AI Errors**: Verify OPENAI_API_KEY is set in .env file
3. **Import Errors**: Run `goimports -w .` before building
4. **Graph Errors**: Run `code-surgeon graph build` first; a build in progress holds an exclusive lock on the database file