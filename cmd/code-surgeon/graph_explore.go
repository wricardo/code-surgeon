package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/urfave/cli/v2"

	"github.com/wricardo/code-surgeon/codegraph"
)

// graphExploreCommands returns the commands that explore around one node: usages, describe,
// tree, roots and path.
func graphExploreCommands(dbFlag, formatFlag, refsFlag cli.Flag) []*cli.Command {
	callable := []string{codegraph.KindFunc, codegraph.KindMethod}
	typeKinds := []string{codegraph.KindStruct, codegraph.KindInterface, codegraph.KindType}

	return []*cli.Command{
		{
			Name:      "usages",
			Usage:     "where a type is used: parameters, results, fields, embedding",
			ArgsUsage: "<type name|id>",
			Flags:     []cli.Flag{dbFlag, formatFlag},
			Action: func(c *cli.Context) error {
				return withGraph(c, func(s *codegraph.Store) error {
					nodes, err := resolveRef(s, c.Args().First(), typeKinds...)
					if err != nil {
						return err
					}
					var all codegraph.Usages
					for _, n := range nodes {
						u, err := s.Usages(n.ID)
						if err != nil {
							return err
						}
						all.Params = append(all.Params, u.Params...)
						all.Returns = append(all.Returns, u.Returns...)
						all.Fields = append(all.Fields, u.Fields...)
						all.EmbeddedBy = append(all.EmbeddedBy, u.EmbeddedBy...)
					}
					return printSections(c.String("format"), []section{
						{"takes it as a parameter", "params", all.Params},
						{"returns it", "returns", all.Returns},
						{"holds it in a field", "fields", all.Fields},
						{"embeds it", "embedded_by", all.EmbeddedBy},
					})
				})
			},
		},
		{
			Name:      "describe",
			Usage:     "everything the graph knows about one node: location, signature, doc and neighbours by relation",
			ArgsUsage: "<name|id>",
			Flags:     []cli.Flag{dbFlag, formatFlag, &cli.BoolFlag{Name: "all", Usage: "do not truncate long neighbour lists"}},
			Action: func(c *cli.Context) error {
				return withGraph(c, func(s *codegraph.Store) error {
					nodes, err := resolveRef(s, c.Args().First())
					if err != nil {
						return err
					}
					var descs []*codegraph.Description
					for _, n := range nodes {
						d, err := s.Describe(n.ID)
						if err != nil {
							return err
						}
						descs = append(descs, d)
					}
					if c.String("format") == "json" {
						return encodeJSON(descs)
					}
					for i, d := range descs {
						if i > 0 {
							fmt.Println()
						}
						printDescription(d, c.Bool("all"))
					}
					return nil
				})
			},
		},
		{
			Name:      "tree",
			Usage:     "call tree of a function or method (callees, or callers with --up)",
			ArgsUsage: "<name|id>",
			Description: `Expands the target into a tree. Each node is expanded where it first appears;
later occurrences are marked (*), so cycles and shared helpers do not repeat. Calls through an
interface continue into the implementations, marked [dispatch]. Use --depth 0 for the full tree.`,
			Flags: []cli.Flag{dbFlag, formatFlag, refsFlag,
				&cli.IntFlag{Name: "depth", Aliases: []string{"d"}, Value: 3, Usage: "levels to expand; 0 = unlimited"},
				&cli.BoolFlag{Name: "up", Usage: "show callers instead of callees"},
				&cli.BoolFlag{Name: "no-ext", Usage: "hide external (stdlib, dependency) nodes"},
				&cli.BoolFlag{Name: "no-dispatch", Usage: "do not continue through interface calls into implementations"},
			},
			Action: func(c *cli.Context) error {
				return withGraph(c, func(s *codegraph.Store) error {
					nodes, err := resolveRef(s, c.Args().First(), callable...)
					if err != nil {
						return err
					}
					opts := codegraph.TreeOptions{
						Dir: codegraph.DirOut, MaxDepth: c.Int("depth"), Rels: callRels(c),
						HideExternal: c.Bool("no-ext"), NoDispatch: c.Bool("no-dispatch"),
					}
					if c.Bool("up") {
						opts.Dir = codegraph.DirIn
					}
					var trees []*codegraph.TreeNode
					for _, n := range nodes {
						t, err := s.Tree(n.ID, opts)
						if err != nil {
							return err
						}
						trees = append(trees, t)
					}
					if c.String("format") == "json" {
						return encodeJSON(trees)
					}
					for i, t := range trees {
						if i > 0 {
							fmt.Println()
						}
						printTree(t)
					}
					return nil
				})
			},
		},
		{
			Name:      "roots",
			Usage:     "entry points that can reach a function: callers all the way up to code nothing calls",
			ArgsUsage: "<name|id>",
			Description: `Walks callers transitively (calls and references, through interface dispatch) and
reports the nodes nothing uses: main, handlers wired through generated code, tests, unused
code, and package initialisers. Answers "which endpoints or jobs end up running this?".`,
			Flags: []cli.Flag{dbFlag, formatFlag},
			Action: func(c *cli.Context) error {
				return withGraph(c, func(s *codegraph.Store) error {
					nodes, err := resolveRef(s, c.Args().First(), callable...)
					if err != nil {
						return err
					}
					ids := make([]string, len(nodes))
					for i, n := range nodes {
						ids[i] = n.ID
					}
					roots, err := s.Roots(ids)
					if err != nil {
						return err
					}
					if len(roots) == 0 {
						fmt.Fprintln(os.Stderr, "no entry point found: the callers only form a cycle")
					}
					return printNodes(roots, c.String("format"))
				})
			},
		},
		{
			Name:      "path",
			Usage:     "shortest call chain from one function to another",
			ArgsUsage: "<from name|id> <to name|id>",
			Flags: []cli.Flag{dbFlag, formatFlag, refsFlag,
				&cli.BoolFlag{Name: "no-dispatch", Usage: "do not continue through interface calls into implementations"},
			},
			Action: func(c *cli.Context) error {
				if c.NArg() != 2 {
					return errors.New("usage: graph path <from> <to>")
				}
				return withGraph(c, func(s *codegraph.Store) error {
					from, err := resolveRef(s, c.Args().Get(0), callable...)
					if err != nil {
						return err
					}
					to, err := resolveRef(s, c.Args().Get(1), callable...)
					if err != nil {
						return err
					}
					steps, err := s.Path(nodeIDs(from), nodeIDs(to), callRels(c), !c.Bool("no-dispatch"))
					if errors.Is(err, codegraph.ErrNoPath) {
						return fmt.Errorf("no call path from %q to %q (calls through function values and across HTTP/RPC are not visible; try --refs or drop --no-dispatch)", c.Args().Get(0), c.Args().Get(1))
					}
					if err != nil {
						return err
					}
					return printPath(steps, c.String("format"))
				})
			},
		},
	}
}

// callRels returns CALLS, plus REFERENCES when --refs is set.
func callRels(c *cli.Context) []string {
	rels := []string{codegraph.RelCalls}
	if c.Bool("refs") {
		rels = append(rels, codegraph.RelReferences)
	}
	return rels
}

func nodeIDs(ns []codegraph.Node) []string {
	ids := make([]string, len(ns))
	for i, n := range ns {
		ids[i] = n.ID
	}
	return ids
}

// preferLocal drops external stub nodes when the match also contains declared code, so that
// "ParseFile" means the project's ParseFile and not go/parser.ParseFile. Pass the stub's id
// to query it.
func preferLocal(ns []codegraph.Node) []codegraph.Node {
	if !slices.ContainsFunc(ns, func(n codegraph.Node) bool { return !n.Ext }) {
		return ns
	}
	return slices.DeleteFunc(slices.Clone(ns), func(n codegraph.Node) bool { return n.Ext })
}

// resolveRef resolves a name or id to nodes, keeping only the given kinds (any when none).
// Several matches are all returned and listed on stderr so the user can narrow with an id.
func resolveRef(s *codegraph.Store, ref string, kinds ...string) ([]codegraph.Node, error) {
	if ref == "" {
		return nil, errors.New("missing <name|id> argument")
	}
	nodes, err := s.Resolve(ref)
	if err != nil {
		return nil, err
	}
	if len(kinds) > 0 {
		nodes = slices.DeleteFunc(nodes, func(n codegraph.Node) bool { return !slices.Contains(kinds, n.Kind) })
	}
	nodes = preferLocal(nodes)
	if len(nodes) == 0 {
		want := "node"
		if len(kinds) > 0 {
			want = strings.ToLower(strings.Join(kinds, "/"))
		}
		return nil, fmt.Errorf("no %s matches %q (try `graph find`)", want, ref)
	}
	if len(nodes) > 1 {
		fmt.Fprintf(os.Stderr, "%q matched %d nodes (pass an id to narrow):\n", ref, len(nodes))
		for _, n := range nodes {
			fmt.Fprintf(os.Stderr, "  %s\n", n.ID)
		}
	}
	return nodes, nil
}

func encodeJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

type section struct {
	title string
	key   string
	nodes []codegraph.Node
}

// printSections prints node lists under headings, or as one JSON object keyed by section.
func printSections(format string, secs []section) error {
	switch format {
	case "json":
		obj := map[string][]codegraph.Node{}
		for _, s := range secs {
			obj[s.key] = nonNil(s.nodes)
		}
		return encodeJSON(obj)
	case "table":
		for _, s := range secs {
			fmt.Printf("== %s (%d)\n", strings.ToUpper(s.title), len(s.nodes))
			if len(s.nodes) > 0 {
				if err := printNodes(s.nodes, "table"); err != nil {
					return err
				}
			}
			fmt.Println()
		}
		return nil
	}
	return fmt.Errorf("unknown --format %q (table, json)", format)
}

func nodeLocation(n codegraph.Node) string {
	switch {
	case n.Ext && n.Pkg == "":
		return "(external)"
	case n.Ext:
		return "(external: " + n.Pkg + ")"
	case n.File != "":
		return fmt.Sprintf("%s:%d", n.File, n.Line)
	}
	return ""
}

func printTree(t *codegraph.TreeNode) {
	fmt.Printf("%s  %s\n", t.Node.Name, nodeLocation(t.Node))
	seen := printTreeChildren(t, "")
	if seen {
		fmt.Println("\n(*) already expanded above")
	}
}

func printTreeChildren(t *codegraph.TreeNode, prefix string) (anySeen bool) {
	for i, c := range t.Children {
		branch, next := "├── ", "│   "
		if i == len(t.Children)-1 {
			branch, next = "└── ", "    "
		}
		label := c.Node.Name + "  " + nodeLocation(c.Node)
		if c.Via == codegraph.ViaDispatch {
			label += "  [dispatch]"
		}
		if c.Seen {
			label += "  (*)"
			anySeen = true
		}
		if c.More > 0 {
			label += fmt.Sprintf("  [+%d more, raise --depth]", c.More)
		}
		fmt.Println(prefix + branch + strings.TrimRight(label, " "))
		if printTreeChildren(c, prefix+next) {
			anySeen = true
		}
	}
	return anySeen
}

func printPath(steps []codegraph.PathStep, format string) error {
	switch format {
	case "json":
		return encodeJSON(steps)
	case "table":
		w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
		fmt.Fprintln(w, "STEP\tKIND\tNAME\tLOCATION\tVIA")
		for i, st := range steps {
			fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n", i+1, st.Node.Kind, st.Node.Name, nodeLocation(st.Node), st.Via)
		}
		return w.Flush()
	}
	return fmt.Errorf("unknown --format %q (table, json)", format)
}

// relOrder lists relations in reading order; others follow alphabetically.
var relOrder = []string{
	codegraph.RelBelongsTo, codegraph.RelHasField, codegraph.RelHasMethod, codegraph.RelEmbeds,
	codegraph.RelImplements, codegraph.RelFieldType, codegraph.RelParamType, codegraph.RelReturnsType,
	codegraph.RelCalls, codegraph.RelReferences, codegraph.RelImports,
}

const describeLimit = 10

func printDescription(d *codegraph.Description, all bool) {
	n := d.Node
	fmt.Printf("%s  %s\n", n.Kind, n.Name)
	fmt.Printf("  id:   %s\n", n.ID)
	if n.Pkg != "" {
		fmt.Printf("  pkg:  %s\n", n.Pkg)
	}
	if loc := nodeLocation(n); loc != "" {
		if n.EndLine > n.Line {
			loc += fmt.Sprintf("-%d", n.EndLine)
		}
		fmt.Printf("  at:   %s\n", loc)
	}
	if n.Sig != "" {
		fmt.Printf("  sig:  %s\n", n.Sig)
	}
	if n.Doc != "" {
		fmt.Printf("  doc:  %s\n", strings.ReplaceAll(strings.TrimSpace(n.Doc), "\n", "\n        "))
	}
	for _, g := range []struct {
		arrow string
		rels  map[string][]codegraph.Node
	}{{"→", d.Out}, {"←", d.In}} {
		for _, rel := range orderedRels(g.rels) {
			ns := g.rels[rel]
			fmt.Printf("\n%s %s (%d)\n", g.arrow, rel, len(ns))
			shown := ns
			if !all && len(ns) > describeLimit {
				shown = ns[:describeLimit]
			}
			for _, m := range shown {
				fmt.Printf("    %-9s %s  %s\n", m.Kind, m.Name, nodeLocation(m))
			}
			if len(shown) < len(ns) {
				fmt.Printf("    ... %d more (--all to list)\n", len(ns)-len(shown))
			}
		}
	}
}

func orderedRels(m map[string][]codegraph.Node) []string {
	var out []string
	for _, r := range relOrder {
		if _, ok := m[r]; ok {
			out = append(out, r)
		}
	}
	var rest []string
	for r := range m {
		if !slices.Contains(relOrder, r) {
			rest = append(rest, r)
		}
	}
	sort.Strings(rest)
	return append(out, rest...)
}
