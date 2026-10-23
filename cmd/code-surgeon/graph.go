package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/urfave/cli/v2"

	"github.com/wricardo/code-surgeon/codegraph"
)

const defaultGraphDB = ".codesurgeon/graph.db"

// graphCommand returns the `graph` command family: build an embedded (bbolt) code graph
// and query it from the CLI.
func graphCommand() *cli.Command {
	dbFlag := &cli.StringFlag{Name: "db", Value: defaultGraphDB, Usage: "path of the graph database file"}
	formatFlag := &cli.StringFlag{Name: "format", Aliases: []string{"f"}, Value: "table", Usage: "output format: table, json"}
	depthFlag := &cli.IntFlag{Name: "depth", Aliases: []string{"d"}, Value: 1, Usage: "max hops to follow; 0 = unlimited (transitive)"}
	refsFlag := &cli.BoolFlag{Name: "refs", Usage: "also follow REFERENCES (functions used as values, not just called)"}

	cmd := &cli.Command{
		Name:  "graph",
		Usage: "build and query an embedded code graph",
		Subcommands: []*cli.Command{
			{
				Name:      "build",
				Usage:     "load Go packages with type info and (re)build the graph database",
				ArgsUsage: "[package patterns...] (default ./...)",
				Flags: []cli.Flag{
					dbFlag,
					&cli.StringFlag{Name: "dir", Value: ".", Usage: "module directory to load packages from"},
					&cli.BoolFlag{Name: "tests", Usage: "include _test.go files (needed for `impact` to report affected tests)"},
					&cli.BoolFlag{Name: "if-stale", Usage: "skip the build when the database exists, was built with the same options and no source file changed since"},
				},
				Action: func(c *cli.Context) error {
					opts := codegraph.BuildOptions{Dir: c.String("dir"), Patterns: c.Args().Slice(), Tests: c.Bool("tests")}
					if c.Bool("if-stale") && codegraph.UpToDate(c.String("db"), opts) {
						fmt.Printf("graph %s is up to date\n", c.String("db"))
						return nil
					}
					res, err := codegraph.Build(opts)
					if err != nil {
						return err
					}
					for i, w := range res.Warnings {
						if i == 10 {
							fmt.Fprintf(os.Stderr, "... %d more warnings\n", len(res.Warnings)-i)
							break
						}
						fmt.Fprintln(os.Stderr, "warning:", w)
					}
					s, err := codegraph.Open(c.String("db"), false)
					if err != nil {
						return err
					}
					defer s.Close()
					if err := s.Replace(res.Nodes, res.Edges, res.Meta); err != nil {
						return err
					}
					fmt.Printf("graph written to %s: %d nodes, %d edges\n", c.String("db"), len(res.Nodes), len(res.Edges))
					return nil
				},
			},
			{
				Name:      "find",
				Usage:     "find nodes whose name starts with a prefix",
				ArgsUsage: "<name-prefix>",
				Flags:     []cli.Flag{dbFlag, formatFlag, &cli.StringFlag{Name: "kind", Aliases: []string{"k"}, Usage: "Package, Struct, Interface, Type, Func, Method, Field"}},
				Action: func(c *cli.Context) error {
					return withGraph(c, func(s *codegraph.Store) error {
						ns, err := s.Find(c.String("kind"), c.Args().First())
						if err != nil {
							return err
						}
						return printNodes(ns, c.String("format"))
					})
				},
			},
			{
				Name:      "types",
				Usage:     "list the structs, interfaces and named types declared in a package",
				ArgsUsage: "<package import path | short name>",
				Flags: []cli.Flag{dbFlag, formatFlag,
					&cli.StringSliceFlag{Name: "kind", Aliases: []string{"k"}, Usage: "restrict to struct, interface or type (repeatable; default: all three)"},
					&cli.BoolFlag{Name: "exported", Usage: "only exported types"},
					&cli.BoolFlag{Name: "recursive", Aliases: []string{"r"}, Usage: "include sub-packages"},
				},
				Action: func(c *cli.Context) error {
					ref := c.Args().First()
					if ref == "" {
						return fmt.Errorf("missing <package> argument")
					}
					var kinds []string
					for _, k := range c.StringSlice("kind") {
						switch strings.ToLower(k) {
						case "struct":
							kinds = append(kinds, codegraph.KindStruct)
						case "interface":
							kinds = append(kinds, codegraph.KindInterface)
						case "type":
							kinds = append(kinds, codegraph.KindType)
						default:
							return fmt.Errorf("unknown --kind %q (struct, interface, type)", k)
						}
					}
					return withGraph(c, func(s *codegraph.Store) error {
						ns, err := s.Types(ref, codegraph.TypesOptions{Recursive: c.Bool("recursive"), Kinds: kinds, ExportedOnly: c.Bool("exported")})
						if err != nil {
							return err
						}
						return printNodes(ns, c.String("format"))
					})
				},
			},
			{
				Name:      "callers",
				Usage:     "functions/methods that call the target (--depth 0 for transitive callers)",
				ArgsUsage: "<name|id>",
				Flags:     []cli.Flag{dbFlag, formatFlag, depthFlag, refsFlag},
				Action:    traverse(codegraph.DirIn),
			},
			{
				Name:      "callees",
				Usage:     "functions/methods the target calls (--depth 0 for transitive callees)",
				ArgsUsage: "<name|id>",
				Flags:     []cli.Flag{dbFlag, formatFlag, depthFlag, refsFlag},
				Action:    traverse(codegraph.DirOut),
			},
			{
				Name:      "implementers",
				Usage:     "types that implement the interface",
				ArgsUsage: "<interface name|id>",
				Flags:     []cli.Flag{dbFlag, formatFlag},
				Action:    traverseFixedDepth(codegraph.DirIn, codegraph.RelImplements),
			},
			{
				Name:      "implements",
				Usage:     "interfaces the type implements",
				ArgsUsage: "<type name|id>",
				Flags:     []cli.Flag{dbFlag, formatFlag},
				Action:    traverseFixedDepth(codegraph.DirOut, codegraph.RelImplements),
			},
			{
				Name:      "neighbors",
				Usage:     "follow edges from the target, optionally by relation and direction",
				ArgsUsage: "<name|id>",
				Flags: []cli.Flag{dbFlag, formatFlag, depthFlag,
					&cli.StringFlag{Name: "rel", Usage: "relation, e.g. HAS_METHOD, HAS_FIELD, CALLS, REFERENCES, IMPORTS, RETURNS_TYPE (default: any)"},
					&cli.StringFlag{Name: "dir", Value: "out", Usage: "out or in"},
				},
				Action: func(c *cli.Context) error {
					dir := codegraph.DirOut
					switch c.String("dir") {
					case "out":
					case "in":
						dir = codegraph.DirIn
					default:
						return fmt.Errorf("--dir must be out or in")
					}
					var rels []string
					if r := c.String("rel"); r != "" {
						rels = []string{r}
					}
					return runTraversal(c, dir, rels, c.Int("depth"))
				},
			},
			{
				Name:  "unused",
				Usage: "functions/methods nothing calls or references (candidates for dead code)",
				Description: `Reports declarations with no static caller and no reference as a value.
Never reported: main, init, tests, interface method declarations, and methods reachable
through interface dispatch. Reflection, linkname and groups of functions that only use each
other are not detected, so treat the result as candidates to review.`,
				Flags: []cli.Flag{dbFlag, formatFlag, &cli.BoolFlag{Name: "exported", Usage: "also report exported functions/methods"}},
				Action: func(c *cli.Context) error {
					return withGraph(c, func(s *codegraph.Store) error {
						ns, err := s.Unused(codegraph.UnusedOptions{Exported: c.Bool("exported")})
						if err != nil {
							return err
						}
						return printNodes(ns, c.String("format"))
					})
				},
			},
			{
				Name:  "impact",
				Usage: "declarations and tests affected by uncommitted changes (or changes since --base)",
				Description: `Maps the lines changed according to git to declarations, then follows callers
(CALLS and REFERENCES, plus interface dispatch) transitively. Build the graph with --tests to
get the list of affected tests. New untracked files are not in git's diff; use 'git add -N <file>'.`,
				Flags: []cli.Flag{dbFlag, formatFlag, &cli.StringFlag{Name: "base", Value: "HEAD", Usage: "git revision to diff against"}},
				Action: func(c *cli.Context) error {
					return withGraph(c, func(s *codegraph.Store) error {
						meta, err := s.Meta()
						if err != nil {
							return err
						}
						diff, err := exec.Command("git", "-C", meta.Dir, "diff", "-U0", "--no-prefix", "--relative", c.String("base")).Output()
						if err != nil {
							return fmt.Errorf("git diff failed in %s: %w", meta.Dir, err)
						}
						res, err := s.Impact(codegraph.ParseDiffRanges(string(diff)))
						if err != nil {
							return err
						}
						return printImpact(res, c.String("format"))
					})
				},
			},
			{
				Name:   "stats",
				Usage:  "node and edge counts per kind/relation, and when the graph was built",
				Flags:  []cli.Flag{dbFlag},
				Action: graphStats,
			},
		},
	}
	cmd.Subcommands = append(cmd.Subcommands, graphExploreCommands(dbFlag, formatFlag, refsFlag)...)
	cmd.Subcommands = append(cmd.Subcommands, graphDeclCommands(dbFlag, formatFlag)...)
	cmd.Subcommands = append(cmd.Subcommands, graphSearchCommand(dbFlag, formatFlag))
	return cmd
}

// traverse follows CALLS (and REFERENCES with --refs) in the given direction.
func traverse(dir codegraph.Dir) cli.ActionFunc {
	return func(c *cli.Context) error {
		rels := []string{codegraph.RelCalls}
		if c.Bool("refs") {
			rels = append(rels, codegraph.RelReferences)
		}
		return runTraversal(c, dir, rels, c.Int("depth"))
	}
}

func traverseFixedDepth(dir codegraph.Dir, rel string) cli.ActionFunc {
	return func(c *cli.Context) error { return runTraversal(c, dir, []string{rel}, 1) }
}

// withGraph opens the database read-only for the duration of fn and warns on stderr
// when source files changed after the graph was built.
func withGraph(c *cli.Context, fn func(*codegraph.Store) error) error {
	s, err := codegraph.Open(c.String("db"), true)
	if err != nil {
		return err
	}
	defer s.Close()
	if newest, stale, err := s.Stale(); err == nil && stale {
		fmt.Fprintf(os.Stderr, "warning: %s changed after the graph was built; results may be outdated (run `code-surgeon graph build`)\n", newest)
	}
	return fn(s)
}

// runTraversal resolves the first argument and prints everything reachable from it.
func runTraversal(c *cli.Context, dir codegraph.Dir, rels []string, depth int) error {
	ref := c.Args().First()
	if ref == "" {
		return fmt.Errorf("missing <name|id> argument")
	}
	return withGraph(c, func(s *codegraph.Store) error {
		start, err := s.Resolve(ref)
		if err != nil {
			return err
		}
		if slices.Contains(rels, codegraph.RelCalls) { // a package named like a function (e.g. main) must not be a call-graph start
			start = slices.DeleteFunc(start, func(n codegraph.Node) bool {
				return n.Kind != codegraph.KindFunc && n.Kind != codegraph.KindMethod
			})
		}
		start = preferLocal(start)
		if len(start) == 0 {
			return fmt.Errorf("no node matches %q (try `graph find`)", ref)
		}
		if len(start) > 1 {
			fmt.Fprintf(os.Stderr, "%q matched %d nodes; showing the union (pass an id to narrow):\n", ref, len(start))
			for _, n := range start {
				fmt.Fprintf(os.Stderr, "  %s\n", n.ID)
			}
		}
		ns, err := s.FromNodes(start).Transitive(dir, depth, rels...).Nodes()
		if err != nil {
			return err
		}
		return printNodes(ns, c.String("format"))
	})
}

func graphStats(c *cli.Context) error {
	return withGraph(c, func(s *codegraph.Store) error {
		st, err := s.Stats()
		if err != nil {
			return err
		}
		fmt.Printf("built %s  dir=%s  patterns=%v  tests=%v\n", st.Meta.BuiltAt.Format("2006-01-02 15:04:05Z"), st.Meta.Dir, st.Meta.Patterns, st.Meta.Tests)
		for _, m := range []map[string]int{st.Nodes, st.Edges} {
			keys := make([]string, 0, len(m))
			for k := range m {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				fmt.Printf("  %-14s %d\n", k, m[k])
			}
			fmt.Println()
		}
		return nil
	})
}

func printImpact(res *codegraph.ImpactResult, format string) error {
	switch format {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(struct {
			Changed  []codegraph.Node `json:"changed"`
			Affected []codegraph.Node `json:"affected"`
			Tests    []codegraph.Node `json:"tests"`
		}{nonNil(res.Changed), nonNil(res.Affected), nonNil(res.Tests)})
	case "table":
		for _, sec := range []struct {
			title string
			nodes []codegraph.Node
		}{{"CHANGED", res.Changed}, {"AFFECTED", res.Affected}, {"AFFECTED TESTS", res.Tests}} {
			fmt.Printf("== %s (%d)\n", sec.title, len(sec.nodes))
			if len(sec.nodes) > 0 {
				if err := printNodes(sec.nodes, "table"); err != nil {
					return err
				}
			}
			fmt.Println()
		}
		return nil
	}
	return fmt.Errorf("unknown --format %q (table, json)", format)
}

func nonNil(ns []codegraph.Node) []codegraph.Node {
	if ns == nil {
		return []codegraph.Node{}
	}
	return ns
}

func printNodes(ns []codegraph.Node, format string) error {
	switch format {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(nonNil(ns))
	case "table":
		w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
		fmt.Fprintln(w, "KIND\tNAME\tPACKAGE\tLOCATION")
		for _, n := range ns {
			loc := ""
			switch {
			case n.Ext:
				loc = "(external)"
			case n.File != "":
				loc = fmt.Sprintf("%s:%d", n.File, n.Line)
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", n.Kind, n.Name, n.Pkg, loc)
		}
		return w.Flush()
	}
	return fmt.Errorf("unknown --format %q (table, json)", format)
}
