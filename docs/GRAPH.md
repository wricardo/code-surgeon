# Code Graph Guide

`code-surgeon graph` builds a type-checked graph of a Go code base and stores it in one embedded
[bbolt](https://github.com/etcd-io/bbolt) file. No server, Docker or database is needed. Use it to
answer structural questions without reading every file: who calls this, what implements that,
what breaks if I change this, what is dead.

Command reference: [COMMANDS.md](COMMANDS.md#embedded-code-graph-commands). This guide covers
workflows and pitfalls.

## Contents
- [Quick start](#quick-start)
- [Mental model](#mental-model)
- [Recipes](#recipes)
  - [Understand a feature](#understand-a-feature)
  - [Check what a change affects](#check-what-a-change-affects)
  - [Find dead code](#find-dead-code)
  - [Find implementations and injection points](#find-implementations-and-injection-points)
  - [Get oriented in an unfamiliar package](#get-oriented-in-an-unfamiliar-package)
  - [Script it with JSON](#script-it-with-json)
  - [Query from Go](#query-from-go)
- [Pitfalls](#pitfalls)
- [What the graph cannot see](#what-the-graph-cannot-see)

## Quick start
```bash
make install                          # installs code-surgeon into $GOBIN
cd /path/to/your/go/module
code-surgeon graph build --tests      # type-checks ./... and writes .codesurgeon/graph.db
code-surgeon graph stats              # node/edge counts and when it was built
code-surgeon graph callers -d 0 ParseDirectory
```
- Build time depends on module size: a few seconds for this repo, about a minute for a
  400-package module with tests.
- Rebuild after code changes. `graph build --if-stale` skips the work when nothing changed.
- `--tests` includes `_test.go` files. It is required for `impact` to report affected tests, and
  it makes `callers` include test callers. Leave it off for a production-only view.
- The database defaults to `.codesurgeon/graph.db` in the current directory (git-ignored in this
  repo). To keep a project clean, pass `--db /tmp/project.db` to every command, or add
  `.codesurgeon/` to that project's `.gitignore`.

## Mental model
**Nodes** have a kind (`Package`, `Struct`, `Interface`, `Type`, `Func`, `Method`, `Field`) and a
stable id:

| Kind | Id format | Example |
|---|---|---|
| Package | `pkg:<import path>` | `pkg:example.com/app/billing` |
| Struct / Interface / Type | `struct:` / `iface:` / `type:` + `<path>.<Name>` | `struct:example.com/app/billing.Invoice` |
| Func | `func:<path>.<Name>` | `func:example.com/app/billing.NewInvoice` |
| Method | `method:<path>.<Type>.<Name>` | `method:example.com/app/billing.Invoice.Total` |
| Field | `field:<path>.<Type>.<Name>` | `field:example.com/app/billing.Invoice.ID` |

Methods and fields are named `Type.Member` and can also be found by the bare member name.
Declarations outside the loaded packages (stdlib, dependencies, `error`) appear as stub nodes
marked `(external)`: you can see that a function calls `gorm.DB.Find`, not what it does inside.

**Edges** are directed:

| Relation | Meaning |
|---|---|
| `CALLS` | function/method (or package, for `var` initialisers) calls a statically known callee |
| `REFERENCES` | function used as a value (callback, handler table, method value), not called |
| `IMPLEMENTS` | type satisfies an interface from the loaded packages |
| `HAS_METHOD`, `HAS_FIELD`, `EMBEDS` | structure of a type |
| `PARAM_TYPE`, `RETURNS_TYPE`, `FIELD_TYPE` | named types mentioned by a signature or field |
| `BELONGS_TO`, `IMPORTS` | declaration to package, package to package |

**Arguments** are an exact node id, or an exact name (`Subscribe` or `Service.Subscribe`).
An ambiguous name queries the union of every match and prints the matched ids on stderr: copy
one id to narrow it. When a name matches both project code and external code (`ParseFile` and
`go/parser.ParseFile`), the project's declarations win; pass the external node's id to query it.

An **interface call** points at the interface method, not at implementations. Join with
`IMPLEMENTS` (or use `impact`, which does it for you) to reach concrete code.

## Recipes

### Understand a feature
Goal: map a feature such as "subscribe a customer to a campaign" from entry point to storage.

```bash
# 1. Find entry points. By name when you know it, by concept when you don't.
code-surgeon graph find -k Method Subscribe
code-surgeon graph search --no-ext --doc "opt in"          # docs mentioning the idea, whatever the function is called
code-surgeon graph search -k Func "welcome"                # name substring: finds sendWelcomeMessage, not just Welcome*

# 2. Everything the graph knows about it: location, signature, doc, neighbours by relation.
code-surgeon graph describe 'method:example.com/app/internal/app.subscriptionService.Subscribe'

# 3. Read the feature top-down. The tree crosses interface calls into implementations ([dispatch]).
code-surgeon graph tree -d 4 --no-ext 'method:example.com/app/graph.mutationResolver.SubscribeToCampaign'

# 4. Which entry points end up running a piece of it? (endpoints, jobs, tests)
code-surgeon graph roots 'method:example.com/app/internal/app.subscriptionService.sendWelcomeMessage'

# 5. How does the entry point reach the interesting call? Shortest chain, with the relation per step.
code-surgeon graph path 'method:...mutationResolver.SubscribeToCampaign' 'method:...WorkflowStarter.ExecuteWorkflow'

# 6. Data model, and where it flows.
code-surgeon graph usages Subscription                     # parameters, results, fields, embedding
code-surgeon graph neighbors --rel HAS_FIELD 'struct:example.com/app/models.Subscription'
code-surgeon graph implementers 'iface:example.com/app/internal/app.SubscriptionRepository'
code-surgeon graph neighbors --dir in --rel FIELD_TYPE 'iface:example.com/app/internal/app.SubscriptionService'  # where it is injected
```
Read the output as a reading list: the 5 to 10 functions it points at are the code to open.
The tree lists callees alphabetically, not in execution order. To see how
far the feature reaches, count the packages in the transitive callees:
`graph callees -d 0 -f json <id> | jq -r '.[]|select(.ext != true)|.pkg' | sort | uniq -c`.

`REFERENCES` matters here: handlers registered in a table or passed as callbacks are not
called directly. Add `--refs` to `callers` to include them.

### Check what a change affects
```bash
code-surgeon graph build --tests      # tests are needed to list affected tests
# ... edit code ...
code-surgeon graph build --if-stale   # rebuild so line numbers match your edits
code-surgeon graph impact             # uncommitted changes vs HEAD
code-surgeon graph impact --base main # everything since main
```
`impact` maps the lines changed according to `git diff` to declarations, then walks callers
transitively (`CALLS` and `REFERENCES`, including interface dispatch). It prints three sections:
`CHANGED`, `AFFECTED` and `AFFECTED TESTS`. Run the listed tests first.

- Build the graph from the edited code: `impact` uses the line ranges stored in the graph.
  A stale graph prints a warning.
- Untracked files are not in git's diff. Use `git add -N <file>`.

For a single function, ask directly: `graph callers -d 0 <name>` lists everything that
transitively depends on it.

### Find dead code
```bash
code-surgeon graph unused             # unexported functions and methods nothing uses
code-surgeon graph unused --exported  # include exported ones (may be public API)
```
Reports declarations with no caller and no reference as a value. A function's own recursion does
not count. Never reported: `main`, `init`, tests, interface method declarations, and methods that
may be reached through interface dispatch.

Treat the output as candidates and verify before deleting:
- reflection, `//go:linkname`, cgo exports and name-based dispatch are invisible;
- groups of functions that only call each other are not detected;
- generated code (`*_templ.go`, `*.pb.go`, `generated.go`) adds noise. Filter by file:
  ```bash
  code-surgeon graph unused -f json | jq -r '.[]|select(.file|test("generated|_templ.go|_test.go")|not)|"\(.name)\t\(.file):\(.line)"'
  ```

### Find implementations and injection points
```bash
code-surgeon graph implementers Shape   # concrete types implementing an interface
code-surgeon graph implements Rect      # interfaces a type satisfies
code-surgeon graph neighbors --dir in --rel PARAM_TYPE Shape   # functions taking a Shape
code-surgeon graph neighbors --dir in --rel RETURNS_TYPE Shape # functions returning one
```
`IMPLEMENTS` covers interfaces declared in the loaded packages, with both value and pointer
receivers, and skips generic types.

### Get oriented in an unfamiliar package
```bash
code-surgeon graph types example.com/app/billing          # every struct, interface and named type declared in a package
code-surgeon graph types -k interface --exported billing  # filter by kind; short name works when unique
code-surgeon graph types -r example.com/app               # include sub-packages
code-surgeon graph funcs --exported example.com/app/billing   # its API surface: exported functions and methods with signatures
code-surgeon graph decls example.com/app/billing               # everything it declares (types, functions, methods)
code-surgeon graph neighbors --rel HAS_METHOD Server     # what can a Server do?
code-surgeon graph neighbors --dir in --rel IMPORTS example.com/app/billing   # who imports this package?
code-surgeon graph neighbors --rel IMPORTS example.com/app/billing            # what does it depend on?
```

### Script it with JSON
Every query takes `-f json`: an array of nodes with `id`, `kind`, `name`, `pkg`, `file`, `line`,
`end_line`, `exported`, `ext`, `sig` and `doc`.
```bash
# local (non-external) transitive callers, as file:line
code-surgeon graph callers -d 0 -f json Foo | jq -r '.[]|select(.ext != true)|"\(.file):\(.line)"'
# feed ids into the next command
for id in $(code-surgeon graph implementers -f json Shape | jq -r '.[].id'); do
  code-surgeon graph neighbors --rel HAS_METHOD "$id"
done
```

### Query from Go
```go
s, err := codegraph.Open(".codesurgeon/graph.db", true) // true = read-only
if err != nil { return err }
defer s.Close()

callers, _ := s.Match("helper").In(codegraph.RelCalls).Nodes()
impact, _ := s.Match("helper").Transitive(codegraph.DirIn, 0, codegraph.RelCalls, codegraph.RelReferences).Nodes()
exported, _ := s.Match("Shape").In(codegraph.RelImplements).
    Where(func(n codegraph.Node) bool { return n.Exported }).Nodes()
unused, _ := s.Unused(codegraph.UnusedOptions{})
```
Chain steps with `Out`, `In`, `Transitive` and `Where`; each step replaces the current set.
`Match` resolves names the same way the CLI does; `From(ids...)` starts from known ids.

## Pitfalls
- **Flags go before the argument.** `graph callers -d 0 -f json Foo` works. In
  `graph callers Foo -f json` the flags are silently ignored (urfave/cli v2 stops parsing at the
  first non-flag), so you get a table where you expected JSON.
- **The graph is a snapshot.** Query commands warn on stderr when a `.go` file, `go.mod` or
  `go.sum` is newer than the build. Deleted files are not detected. Rebuild with
  `graph build --if-stale`.
- **One writer.** `graph build` holds an exclusive lock on the file. Queries during a build fail
  after a 2 second timeout.
- **`--tests` loads some packages twice** (`p` and `p [p.test]`). The graph merges them by id and
  checks `IMPLEMENTS` across both variants. Graphs built before that fix miss implementations
  that live in packages compiled against the plain variant: rebuild old graphs.
- **Package named like a function** (for example `main`): `callers`/`callees` only start from
  functions and methods, so the package node does not pollute the result.
- **Type errors in your code** are printed as `warning:` lines. The graph is still built from
  what type-checked, but declarations in broken packages may be missing.

## What the graph cannot see
- **Order and conditions.** It shows what a function calls, not in what order or under which
  `if`. Open the function to see the branching.
- **Hops outside Go's type system:** HTTP or RPC between services, GraphQL clients calling their own
  server, message queues, and workflow engines that start work by name (Temporal, job queues).
  The graph shows that a function starts a workflow, not which workflow body runs.
- **Dynamic calls:** function values held in variables, reflection, `go:linkname`. A function
  used as a value gets a `REFERENCES` edge, but the call through the variable is not a `CALLS`.
- **Generated code** is included like any other code. Filter it out of results by file name.
- **External code** appears only as stubs.
