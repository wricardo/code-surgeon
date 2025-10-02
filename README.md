# code-surgeon

Code Surgeon is:
- a library for generating code, parsing golang code
- a command line tool to parse golang code
- a stack trace analysis library for Go function parsing
- a Neo4j integration tool for code structure visualization
- an AI chatbot through grpc


## Dependencies

- https://comby.dev/docs/get-started

## Setup your env variables

- Create a `.env` file in the root of the project
- Add the following variables to the `.env` file
```
OPENAI_API_KEY=sk-....
NGROK_AUTH_TOKEN=your_ngrok_auth_token
NGROK_DOMAIN=example-domain.ngrok-free.app
NEO4j_DB_URI=neo4j://localhost
NEO4j_DB_USER=neo4j
NEO4j_DB_PASSWORD=neo4jneo4j
```

https://dashboard.ngrok.com/cloud-edge/domains

## Using chatbot


- Start infrastructure (terminal 1)
```
docker-compose up
```

- Run the chatbot server (terminal 2)
```
make run-server
```

- Run the chatbot cli client (terminal 3)
```
make new-chat
```

## Stack Trace Analysis

Code Surgeon includes a powerful stack trace parsing library in the `neo4j2` package that can analyze Go stack traces and extract function information.

### ParseReceiver Function

The `ParseReceiver` function parses Go function signatures from stack traces and returns the function name and receiver type.

```go
import "github.com/wricardo/code-surgeon/neo4j2"

// Parse simple method calls
function, receiver, _ := neo4j2.ParseReceiver("(*MyStruct).DoSomething")
// Returns: function="DoSomething", receiver="(*MyStruct)"

// Parse complex generic types
function, receiver, _ := neo4j2.ParseReceiver("(*Client[...]).CallUnary")
// Returns: function="CallUnary", receiver="(*Client[...])"

// Parse anonymous functions with middleware chains
function, receiver, _ := neo4j2.ParseReceiver("GrpcServer.DefaultClientInterceptors.NewInterceptor.func6.1")
// Returns: function="NewInterceptor", receiver="DefaultClientInterceptors"

// Parse simple functions (no receiver)
function, receiver, _ := neo4j2.ParseReceiver("ReportStacktrace")
// Returns: function="ReportStacktrace", receiver=""
```

### Features

- **Complex Type Support**: Handles generic types with `[...]` syntax
- **Anonymous Functions**: Parses `func1`, `func2.1` patterns correctly
- **Middleware Chains**: Understands complex call chains and method chaining
- **Pointer Notation**: Preserves `(*Type)` receiver notation
- **Edge Cases**: Robust handling of nested parentheses and brackets

### Use Cases

- **Debugging**: Extract clean function names from panic stack traces
- **Profiling**: Analyze call patterns and receiver types
- **Logging**: Clean up stack trace output for better readability
- **Static Analysis**: Parse function signatures from runtime traces

## Neo4j Integration & Testing

Code Surgeon provides comprehensive Neo4j integration for storing and analyzing Go code structures.

### Setup

1. **Start Neo4j Services**:
```bash
docker-compose up
```

2. **Run Tests**:
```bash
make test
```

### Testing Infrastructure

- **Isolated Tests**: Each test runs with a clean Neo4j database
- **Auto Infrastructure**: Tests automatically start/stop Docker containers
- **No Persistence**: Test database data is ephemeral
- **CI-Ready**: Skip integration tests with `SKIP_INTEGRATION_TESTS=true`

### Neo4j Services

- **Production**: `neo4j:7474/7687` (persistent data)
- **Testing**: `neo4j-test:7475/7688` (ephemeral data)

The test infrastructure automatically:
- Starts the `neo4j-test` container before tests
- Waits for Neo4j to be ready
- Clears data between test cases
- Stops and removes containers after tests
