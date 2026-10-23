package main

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/urfave/cli/v2"

	"github.com/wricardo/code-surgeon/codegraph"
)

// graphDeclCommands returns `decls` (all declarations of a package) and `funcs` (functions
// and methods only), which share one implementation.
func graphDeclCommands(dbFlag, formatFlag cli.Flag) []*cli.Command {
	exportedFlag := &cli.BoolFlag{Name: "exported", Usage: "only exported declarations"}
	recursiveFlag := &cli.BoolFlag{Name: "recursive", Aliases: []string{"r"}, Usage: "include sub-packages"}

	return []*cli.Command{
		{
			Name:      "decls",
			Usage:     "list the declarations of a package: types, functions and methods with signatures",
			ArgsUsage: "<package import path | short name>",
			Description: `Struct fields are not listed (use 'graph describe <struct>'). Methods include
interface method declarations. Output columns: kind, name, location, signature.`,
			Flags: []cli.Flag{dbFlag, formatFlag, exportedFlag, recursiveFlag,
				&cli.StringSliceFlag{Name: "kind", Aliases: []string{"k"}, Usage: "restrict to struct, interface, type, func or method (repeatable; default: all)"},
			},
			Action: declsAction(nil),
		},
		{
			Name:      "funcs",
			Usage:     "list the functions and methods of a package with signatures (shorthand for decls -k func -k method)",
			ArgsUsage: "<package import path | short name>",
			Flags:     []cli.Flag{dbFlag, formatFlag, exportedFlag, recursiveFlag},
			Action:    declsAction([]string{codegraph.KindFunc, codegraph.KindMethod}),
		},
	}
}

// declsAction lists declarations; fixedKinds overrides --kind when non-nil.
func declsAction(fixedKinds []string) cli.ActionFunc {
	return func(c *cli.Context) error {
		ref := c.Args().First()
		if ref == "" {
			return fmt.Errorf("missing <package> argument")
		}
		kinds := fixedKinds
		if kinds == nil {
			for _, k := range c.StringSlice("kind") {
				kind, ok := map[string]string{
					"struct": codegraph.KindStruct, "interface": codegraph.KindInterface, "type": codegraph.KindType,
					"func": codegraph.KindFunc, "method": codegraph.KindMethod,
				}[strings.ToLower(k)]
				if !ok {
					return fmt.Errorf("unknown --kind %q (struct, interface, type, func, method)", k)
				}
				kinds = append(kinds, kind)
			}
		}
		return withGraph(c, func(s *codegraph.Store) error {
			ns, err := s.Decls(ref, codegraph.DeclsOptions{Recursive: c.Bool("recursive"), Kinds: kinds, ExportedOnly: c.Bool("exported")})
			if err != nil {
				return err
			}
			return printDecls(ns, c.String("format"), c.Bool("recursive"))
		})
	}
}

// printDecls prints declarations with their signatures; the package column is shown only
// when several packages can appear.
func printDecls(ns []codegraph.Node, format string, withPkg bool) error {
	switch format {
	case "json":
		return encodeJSON(nonNil(ns))
	case "table":
		w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
		header := "KIND\tNAME\tLOCATION\tSIGNATURE"
		if withPkg {
			header = "KIND\tNAME\tPACKAGE\tLOCATION\tSIGNATURE"
		}
		fmt.Fprintln(w, header)
		for _, n := range ns {
			if withPkg {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", n.Kind, n.Name, n.Pkg, nodeLocation(n), n.Sig)
			} else {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", n.Kind, n.Name, nodeLocation(n), n.Sig)
			}
		}
		return w.Flush()
	}
	return fmt.Errorf("unknown --format %q (table, json)", format)
}
