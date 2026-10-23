# Repository Guidelines

## Project Overview
Code Surgeon (`github.com/wricardo/code-surgeon`, Go 1.24) is a Go AST analysis/rewriting library plus a `urfave/cli/v2` CLI. It parses Go code into a structured model, can rewrite code, and builds an embedded, type-checked code graph (calls, implements, imports) stored in bbolt. `ai/` wraps OpenAI for structured outputs and embeddings.

## Architecture & Data Flow
```
Go source ─┬─► root pkg `codesurgeon` (go/ast) ──► ParsedInfo{Modules→Packages→Struct/Interface/Method/Function/...}
           │        ├─► PrettyPrint (llm | text_short | text_long | json)   [parse, list-tests, describe-test]
           │        └─► ApplyFileChanges / InsertCodeFragments (rewrite code)
           └─► codegraph.Build (go/packages + go/types) ──► nodes + edges ──► bbolt file (.codesurgeon/graph.db)
                                                              └─► Store.Match/Find ... Query.Out/In/Transitive/Where ──► `graph` CLI
```
- The two pipelines are independent: `codesurgeon` is syntax-only (`go/ast`); `codegraph` is type-aware (`go/types`) and does not use `ParsedInfo`.
- No DI framework. Dependencies (store, options) are passed as arguments. Global state is limited to zerolog config.
- `cmd/code-surgeon/main.go` registers inline-closure commands; `graph.go` builds the `graph` command family via `graphCommand()`.

## Key Directories
- `/` (package `codesurgeon`): `structparser.go` (model + `Parse*` entry points), `exp.go` (unexported AST extraction), `parser_pretty_print.go`, `codesurgeon.go` (rewrite API, templates), `helper.go`.
- `cmd/code-surgeon/`: CLI entry point (`main.go`: `parse`, `list-tests`, `describe-test`; `graph.go`: `graph ...`).
- `codegraph/`: `build.go` (nodes/edges extraction), `store.go` (bbolt buckets `nodes`/`out`/`in`/`byname`/`bykind`/`meta`; key parts joined by `\x00`), `query.go` (chainable traversal), `walk.go` (`step` with interface-dispatch bridging, `Tree`, `Path`, `Roots`, `upwardClosure` shared with `Impact`), `inspect.go` (`Usages`, `Describe`), `types.go` (`Types`), `unused.go`, `impact.go` (git diff ranges → affected set), `stale.go` (staleness, `UpToDate`), `graph.go` (Node/Edge types, kind and relation constants).
- `ai/`: OpenAI wrapper via instructor-go (`ai.go`); used by `examples/example_02`.
- `log2/`: zerolog setup.
- `examples/`, `assets/`: sample Go sources used as parser fixtures (compiled by `./...`; not product code).
- `docs/COMMANDS.md`: CLI reference (partly stale for `server`, `chat`, `mcp-server` and similar; `main.go` is authoritative).
- `docs/GRAPH.md`: graph usage guide (recipes, pitfalls, limits). Keep it in sync when `graph` commands or relations change.

## Development Commands
```bash
goimports -w .                      # REQUIRED after editing Go files, before build
make install                        # go install ./cmd/code-surgeon
make build-delve                    # ./bin/code-surgeon with debug flags
make test                           # build check + go test -v ./...
go test ./codegraph/...
go run ./cmd/code-surgeon parse --path . -r --format llm --ignore-rule "*_test.go"
go run ./cmd/code-surgeon graph build && go run ./cmd/code-surgeon graph callers -d 0 ParseDirectory
```
`graph` subcommands: `build [--dir D] [--tests] [--if-stale] [patterns...]`, `find -k KIND prefix`, `types [-k struct|interface|type] [--exported] [-r] <pkg>`, `callers|callees [-d N] [--refs]`, `implementers`, `implements`, `neighbors [--rel R --dir in|out -d N]`, `describe [--all]`, `usages <type>`, `tree [-d N] [--up] [--no-ext] [--no-dispatch]`, `roots`, `path <from> <to>`, `unused [--exported]`, `impact [--base REV]` (needs a `--tests` build for affected tests), `stats`; all queries take `--db` and `--format table|json` and warn on stderr when sources are newer than the graph. No lint target exists.
`decls [-k struct|interface|type|func|method] [--exported] [-r] <pkg>` and `funcs` (= decls restricted to func/method) list a package's declarations with signatures; `types` is the types-only view. All three go through `Store.Decls` (`codegraph/types.go`); `-r` with an import path also matches directories without Go files by path prefix.
`search [-k KIND] [--name|--doc|--sig] [--no-ext] [-n N] <text>` (`codegraph/search.go`) scans every node for a case-insensitive substring of name, doc or signature; one hit per node for its strongest match; `find` stays the indexed name-prefix lookup with a stable node-array JSON schema, which is why search is a separate command.

## Code Conventions & Common Patterns
- Format with `gofmt`/`goimports`. `CONVENTIONS.md`: comment above every function saying what it does; descriptive variable names; modular, reusable code. Use `for i := range n` over classic counting loops (Go 1.22+).
- `codesurgeon` model structs carry snake_case `json` tags; `codegraph.Node`/`Edge` use lowercase `json` tags.
- Node ids are stable strings: `pkg:<path>`, `struct:<path>.<T>`, `iface:`, `type:`, `func:<path>.<F>`, `method:<path>.<T>.<M>`, `field:<path>.<T>.<F>`. Methods/fields have `Name="T.M"` and `Short="M"`; both are indexed by name.
- Referenced-but-undeclared objects (stdlib, deps, `error`) become stub nodes with `Ext=true`, created in `builder.addStubs`; never override a locally declared node.
- `CALLS` is static: interface calls target the interface method (join with `IMPLEMENTS`); uses in func literals are attributed to the enclosing declaration; uses in package-level `var` initialisers are attributed to the `pkg:` node. Functions used as values get `REFERENCES` (callers/unused/impact treat both as uses).
- Node `Line`..`EndLine` span a whole func/method/type declaration (from `collectDecls`); `Impact` relies on it to map git hunks to declarations.
- `codegraph.Build` MUST keep `packages.NeedDeps`: loading dependencies from export data hit `log.Fatalf` ("package ... without types") under Go 1.27 + x/tools v0.36.
- Errors: return `error` normally; wrap with `%w`. `MustRenderTemplate` panics and `ai.GetInstructor()` panics without an API key; do not spread these patterns into new library code.
- `ai.GetInstructor()` reads `.env` via godotenv and panics when the file or key is missing.

## Important Files
- `cmd/code-surgeon/main.go`, `cmd/code-surgeon/graph.go`, `structparser.go`, `codesurgeon.go`, `codegraph/build.go`, `codegraph/store.go`, `ai/ai.go`
- `Makefile`, `CLAUDE.md`, `CONVENTIONS.md`
- `.golangci.yml` (if present) is an unedited template with no effective linters; treat as non-authoritative.
- `go.mod` has a self-referential `replace` to `./`.
- Stray root artifacts (`parsed.json`, file named `false`) are not used.

## Runtime/Tooling Preferences
- Go 1.24 (toolchain 1.24.4+), Go modules; `goimports` required. No Docker or external service is needed.
- Key deps: `go.etcd.io/bbolt`, `golang.org/x/tools` (go/packages), `urfave/cli/v2`, `rs/zerolog`, `instructor-go` + `go-openai` (ai only).
- A writer (`graph build`) holds an exclusive bbolt file lock; queries open read-only.
- `.gitignore`: `.env`, `bin/*`, `*.log`, `chats.json`, `.codesurgeon/`.

## Testing & QA
- `testing` + `testify/require`. White-box tests in the same package; no mocks, no build tags, no `testdata/`, no `t.Parallel()`.
- Naming: `TestFunc`, `TestFunc_Scenario`, `t.Run("lowercase desc")`, table-driven with `input`/`expected`. Use `t.TempDir()` for filesystem work.
- `codegraph/codegraph_test.go` (shared `fixture` const, `buildFixture`/`buildFixtureDir`) and `features_test.go` write a throwaway module into `t.TempDir()`, run `Build`, store into a temp bbolt file and assert structure, `CALLS`/`REFERENCES`, `IMPLEMENTS`, cycles, `Replace`, `Unused`, `Impact`, `ParseDiffRanges`, `Stale`/`UpToDate`. Adding declarations to `fixture` changes expectations in several tests; node lists come back sorted by kind then name, so use `ElementsMatch` for unordered sets. Needs the `go` binary but no network.
- Root `structparser_test.go` reads `./examples/structparser/` by relative path; keep fixtures stable.
- `cmd/code-surgeon/main_test.go` has benchmarks only. No coverage threshold; add tests for new behaviour, especially in `codegraph/`.
