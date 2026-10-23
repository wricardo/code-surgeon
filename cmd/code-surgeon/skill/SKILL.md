---
name: code-surgeon
description: >
  User manual for code-surgeon — a Go code analysis CLI that parses Go source and builds a
  queryable call/type graph. Use when asked to understand or navigate Go code structure:
  who calls a function, what implements an interface, where a type is used, what a change
  affects, which entry points reach a function, dead code, call trees, a feature walkthrough,
  listing or inspecting tests, or listing a package's types/functions/methods.
  Triggers on mentions of code-surgeon, "who calls", "what implements", "what does this change
  affect", "unused functions", "call graph", "list the tests", "explain this Go feature".
---

# code-surgeon

Structural questions about Go code (callers, implementers, usages, impact, dead code) are
answered faster and more reliably by the graph than by `grep` + reading files. The graph is
type-checked (go/packages + go/types), so it follows interfaces, embedded types and generics
instead of matching text. Use plain `grep` for strings, log messages and non-Go files.

## Parse and test commands (no setup)

```bash
code-surgeon parse -r --path ./pkg                         # structs/functions/interfaces/methods, compact (llm format)
code-surgeon parse --format json --path ./pkg | jq '.modules[].packages[].functions[].name'
code-surgeon list-tests -r --path ./pkg                    # test names + descriptions, cheap on tokens
code-surgeon describe-test -r --name TestFoo --path ./pkg  # docs, signature, body of one test
```
Formats for `parse`: `llm` (default), `text_short`, `text_long`, `json`. Use `-r` for whole trees.

## Build the graph

```bash
code-surgeon graph build --tests        # type-checks ./... and writes .codesurgeon/graph.db
code-surgeon graph build --if-stale     # skip when nothing changed
code-surgeon graph stats                # sizes and build time
```
- Run it in the module root. A large module takes about a minute; this repo takes seconds.
- `--tests` includes `_test.go` files: needed for `impact` to list affected tests, and it makes
  callers include tests. Leave it off for a production-only view.
- The database is `.codesurgeon/graph.db` in the current directory. To avoid writing into a
  project, pass `--db /tmp/project.db` to every command (build and queries).
- Rebuild after code changes. Query commands print a warning on stderr when sources are newer
  than the graph; heed it before trusting results.

## Which command for which question

| Question | Command |
|---|---|
| Find by name prefix | `graph find -k Method Subscribe` |
| Find by concept (name substring, doc, signature) | `graph search "welcome message"` (`--doc`, `--sig`, `-k`, `--no-ext`) |
| Everything about one node | `graph describe <name\|id>` |
| What does a package declare | `graph decls <pkg>` / `graph funcs --exported <pkg>` / `graph types <pkg>` (`-r` = sub-packages) |
| Who calls X / what does X call | `graph callers X` / `graph callees X` (`-d 0` = transitive, `--refs` = also uses as a value) |
| Call hierarchy as a tree | `graph tree -d 4 --no-ext X` (`--up` for callers) |
| Which entry points reach X (main, handlers, tests) | `graph roots X` |
| How does A reach B | `graph path A B` |
| Who implements an interface / what a type implements | `graph implementers I` / `graph implements T` |
| Where is a type used (params, results, fields, embedding) | `graph usages T` |
| Anything else, by relation | `graph neighbors --rel HAS_METHOD --dir out T` |
| Dead-code candidates | `graph unused` (`--exported` to include public API) |
| What do my uncommitted changes affect | `graph impact` (`--base main` for everything since main) |

Relations for `neighbors --rel`: `BELONGS_TO`, `HAS_METHOD`, `HAS_FIELD`, `EMBEDS`,
`FIELD_TYPE`, `PARAM_TYPE`, `RETURNS_TYPE`, `CALLS`, `REFERENCES`, `IMPLEMENTS`, `IMPORTS`.
Node kinds: `Package`, `Struct`, `Interface`, `Type`, `Func`, `Method`, `Field`.

## Names and ids

- Arguments are an exact name (`Subscribe`, `Service.Subscribe`, a package's import path or
  short name) or an exact node id: `func:<pkg>.<Name>`, `method:<pkg>.<Type>.<Name>`,
  `struct:` / `iface:` / `type:<pkg>.<Name>`, `pkg:<path>`.
- An ambiguous name queries the union of all matches and lists the ids on stderr: copy one id
  to narrow it.
- When a name exists both in the project and in a dependency (`ParseFile` and
  `go/parser.ParseFile`) the project's declaration wins. Pass the external id to query the other.
- External code (stdlib, dependencies) appears as stub nodes marked `(external)`; you see that
  a function calls them, not what happens inside.

## Recipes

**Understand a feature** (e.g. "subscribe a customer"):
```bash
code-surgeon graph search --no-ext --doc "opt in"       # find entry points by concept
code-surgeon graph find -k Method Subscribe             # ... or by name
code-surgeon graph describe 'method:<pkg>.<Type>.Subscribe'
code-surgeon graph tree -d 4 --no-ext 'method:<pkg>.<Type>.Subscribe'   # top-down; crosses interface calls
code-surgeon graph roots 'method:<pkg>.<Type>.sendWelcome'              # which endpoints/jobs run this?
code-surgeon graph path <entry> <interesting call>                      # shortest chain
code-surgeon graph usages Subscription                                  # data model and where it flows
```
Then open the 5-10 functions the graph pointed at and read them. Tree output lists callees
alphabetically, not in execution order; read the function for ordering and conditions.

**Before refactoring X:** `graph callers -d 0 X` (everything that depends on it),
`graph roots X` (which entry points are affected), `graph usages T` for a type.

**After editing:** `graph build --tests && graph impact` lists changed declarations, everything
affected, and the affected tests to run first. Untracked files are not in git's diff: use
`git add -N <file>`.

**Dead code:**
```bash
code-surgeon graph unused -f json | jq -r '.[]|select(.file|test("generated|_templ.go|_test.go")|not)|"\(.name)\t\(.file):\(.line)"'
```
Candidates, not proof: verify with `grep` before deleting.

**Scripting:** every query takes `-f json` (an array of nodes with `id`, `kind`, `name`, `pkg`,
`file`, `line`, `end_line`, `exported`, `ext`, `sig`, `doc`; `search` returns hits with
`node`, `match`, `rank`, `snippet`).

## Pitfalls

- **Flags go before the argument.** `graph callers -d 0 -f json Foo` works; in
  `graph callers Foo -f json` the flags are silently ignored and you get a table.
- **`-d`:** `callers`/`callees` default to depth 1; `-d 0` is unlimited. `tree` defaults to 3.
- **One writer.** A running `graph build` locks the file; queries fail after 2s until it ends.
- **Interface calls:** `CALLS` points at the interface method. `tree`, `path`, `roots` and
  `impact` continue into implementations (marked `[dispatch]`); `callers`/`callees` do not.
- **`--tests` loads some packages twice;** the graph merges them by id. Graphs built before
  that was handled miss some `IMPLEMENTS` edges: rebuild old graphs.
- **Type errors** in the code print as `warning:` lines during build. The graph is built from
  what type-checked; declarations in broken packages may be missing.

## What the graph cannot see

- Order and conditions inside a function.
- Hops outside Go's type system: HTTP/RPC between services, a GraphQL client calling its own
  server, queues, and workflow engines that start work by name.
- Dynamic calls: function values in variables (a function used as a value gets a `REFERENCES`
  edge, the call through the variable does not), reflection, `go:linkname`.
- Generated code is included like any code; filter it by file name.

## Install or update this skill

```bash
code-surgeon install-skill              # writes ~/.claude/skills/code-surgeon/SKILL.md
code-surgeon install-skill --dir DIR    # write somewhere else
```
