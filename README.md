# code-surgeon

Code Surgeon is:
- a library for generating code, parsing golang code
- a command line tool to parse golang code
- an embedded code graph (call graph, implementers, structure) queried from the CLI or Go
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
```

https://dashboard.ngrok.com/cloud-edge/domains

## Using chatbot


- Run the chatbot server (terminal 2)
```
make run-server
```

- Run the chatbot cli client (terminal 3)
```
make new-chat
```

## Code Graph

Build a type-checked graph of a code base into an embedded bbolt file (no external services) and query it from the CLI or from Go:

```bash
code-surgeon graph build --tests                 # once, and after code changes (--if-stale skips no-ops)
code-surgeon graph find -k Method Subscribe      # find entry points
code-surgeon graph search "welcome message"     # find by concept: name substring, doc comment or signature
code-surgeon graph types github.com/x/y/pkg     # structs, interfaces and named types of a package
code-surgeon graph funcs --exported pkg         # functions and methods of a package, with signatures
code-surgeon graph decls pkg                    # everything a package declares
code-surgeon graph describe Server              # everything about one node
code-surgeon graph usages Server                # where a type is used (params, results, fields)
code-surgeon graph tree -d 4 --no-ext Handle    # call tree, crossing interface calls
code-surgeon graph roots queryDB                # which entry points reach this function?
code-surgeon graph path Handle queryDB          # shortest call chain between two functions
code-surgeon graph callers -d 0 ParseDirectory   # everything that transitively calls it
code-surgeon graph callees SomeFunc              # what it calls
code-surgeon graph implementers Shape            # who implements an interface
code-surgeon graph unused                        # dead-code candidates
code-surgeon graph impact                        # functions and tests affected by uncommitted changes
```

- [docs/GRAPH.md](docs/GRAPH.md): usage guide with recipes (understand a feature, check what a change affects, find dead code, script with JSON, use from Go), pitfalls and limits.
- [docs/COMMANDS.md](docs/COMMANDS.md#embedded-code-graph-commands): command reference, node kinds and relations.
