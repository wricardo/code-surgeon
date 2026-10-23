package main

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"unicode/utf8"

	"github.com/urfave/cli/v2"

	"github.com/wricardo/code-surgeon/codegraph"
)

// graphSearchCommand returns `search`: find nodes by concept (name substring, doc comment,
// signature) when the exact name is not known.
func graphSearchCommand(dbFlag, formatFlag cli.Flag) *cli.Command {
	return &cli.Command{
		Name:      "search",
		Usage:     "find nodes by concept: substring of the name, doc comment or signature",
		ArgsUsage: "<text>",
		Description: `Case-insensitive substring search. By default it looks in names, doc comments and
signatures; --name, --doc and --sig restrict it. Doc comments are matched with line breaks
collapsed, so a phrase can span lines. Each node appears once, for its best match, ordered:
exact name, name prefix, name substring, doc, signature. Unlike 'find' (indexed name prefix),
search scans every node, which takes well under a second on graphs of tens of thousands of nodes.`,
		Flags: []cli.Flag{dbFlag, formatFlag,
			&cli.StringFlag{Name: "kind", Aliases: []string{"k"}, Usage: "Package, Struct, Interface, Type, Func, Method, Field"},
			&cli.BoolFlag{Name: "name", Usage: "search names"},
			&cli.BoolFlag{Name: "doc", Usage: "search doc comments"},
			&cli.BoolFlag{Name: "sig", Usage: "search signatures and field types"},
			&cli.BoolFlag{Name: "no-ext", Usage: "hide external (stdlib, dependency) nodes"},
			&cli.IntFlag{Name: "limit", Aliases: []string{"n"}, Value: 100, Usage: "maximum results; 0 = all"},
		},
		Action: func(c *cli.Context) error {
			query := strings.Join(c.Args().Slice(), " ")
			return withGraph(c, func(s *codegraph.Store) error {
				hits, err := s.Search(query, codegraph.SearchOptions{
					Kind: c.String("kind"), Name: c.Bool("name"), Doc: c.Bool("doc"), Sig: c.Bool("sig"), HideExternal: c.Bool("no-ext"),
				})
				if err != nil {
					return err
				}
				total := len(hits)
				if limit := c.Int("limit"); limit > 0 && total > limit {
					hits = hits[:limit]
					fmt.Fprintf(os.Stderr, "showing %d of %d matches (--limit 0 shows all, --kind/--no-ext narrow)\n", limit, total)
				}
				return printHits(hits, c.String("format"))
			})
		},
	}
}

func printHits(hits []codegraph.SearchHit, format string) error {
	switch format {
	case "json":
		if hits == nil {
			hits = []codegraph.SearchHit{}
		}
		return encodeJSON(hits)
	case "table":
		w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
		fmt.Fprintln(w, "KIND\tNAME\tPACKAGE\tLOCATION\tMATCH")
		for _, h := range hits {
			match := h.Match
			if h.Snippet != "" {
				match += ": " + truncateRunes(h.Snippet, 110)
			}
			n := h.Node
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", n.Kind, n.Name, n.Pkg, nodeLocation(n), match)
		}
		return w.Flush()
	}
	return fmt.Errorf("unknown --format %q (table, json)", format)
}

// truncateRunes shortens s to at most max runes, marking the cut with an ellipsis.
func truncateRunes(s string, max int) string {
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	r := []rune(s)
	return string(r[:max-1]) + "…"
}
