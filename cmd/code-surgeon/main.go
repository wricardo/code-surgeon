// Package main implements the code-surgeon CLI tool.
// Code-surgeon is a multi-purpose Go development tool that combines code analysis,
// AST parsing, and an embedded code graph for code structure exploration.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/urfave/cli/v2"
	codesurgeon "github.com/wricardo/code-surgeon"
	"github.com/wricardo/code-surgeon/log2"
)

func main() {
	// Initialize logger
	log2.Configure()

	app := &cli.App{
		Name:  "code-surgeon",
		Usage: "Go codebase analysis tool — parse, inspect, and query Go source code structure",
		Description: `code-surgeon parses Go source code into structured data (AST-based) and exposes it via CLI commands. Designed for use by LLMs and developer tooling to efficiently extract code structure without reading raw files.

CORE USE CASES:
  1. Understand what functions, structs, interfaces exist in a package
  2. List and inspect test functions efficiently (low token cost)
  3. Get full details (signature, body, docs) for specific functions or tests
  4. Build an embedded code graph and query callers, callees and implementers

KEY COMMANDS:

  parse            Parse Go source, output structs/functions/interfaces/methods.
                   Formats: llm (default), text_short, text_long, json.
                   Use --recursive (-r) for entire trees.
                   Example: code-surgeon parse -r --path ./mypackage
                   Example: code-surgeon parse --format json --path ./mypackage | jq '.modules[].packages[].functions[].name'

  list-tests       List all test functions with name and description. Compact output — low token cost.
                   Use this first to discover what tests exist before digging in.
                   Example: code-surgeon list-tests -r --path ./mypackage

  describe-test    Get full details (docs, signature, body) for one or more specific tests by name.
                   Use after list-tests to inspect a test without reading the whole file.
                   Example: code-surgeon describe-test -r --name TestFoo
                   Example: code-surgeon describe-test -r --name TestFoo --name TestBar

  graph            Build and query an embedded code graph (call graph, implementers, structure).
                   Example: code-surgeon graph build && code-surgeon graph callers -d 0 ParseDirectory

WORKFLOW FOR LLMs:
  Step 1 — discover structure:    code-surgeon parse -r --path ./pkg
  Step 2 — discover tests:        code-surgeon list-tests -r --path ./pkg
  Step 3 — inspect a test:        code-surgeon describe-test -r --name TestFoo --path ./pkg
  Step 4 — inspect a function:    code-surgeon parse --format json --path ./pkg | jq '...'`,
		Action: func(*cli.Context) error {
			fmt.Println("Run --help for more information.")
			return nil
		},
		Commands: []*cli.Command{
			{
				Name:    "parse",
				Aliases: []string{"p"},
				Usage:   "parse a file or directory",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "path",
						Aliases:  []string{"f"},
						Usage:    "path to the file or directory to parse",
						Value:    ".",
						Required: false,
					},
					&cli.BoolFlag{
						Name:    "recursive",
						Aliases: []string{"r"},
						Usage:   "recursively parse directories",
					},
					&cli.StringFlag{
						Name:  "format",
						Value: "llm",
						Usage: "format to print the parsed information: llm, text_short, test_long, json",
					},
					&cli.BoolFlag{
						Name:  "omit-nulls",
						Usage: "omit null and empty values from JSON output",
						Value: false,
					},
					&cli.BoolFlag{
						Name:  "plain-structs",
						Usage: "print plain structs",
						Value: true,
					},
					&cli.BoolFlag{
						Name:  "fields-plain-structs",
						Usage: "print fields of plain structs",
						Value: true,
					},
					&cli.BoolFlag{
						Name:  "structs-with-method",
						Usage: "print structs with methods",
						Value: true,
					},
					&cli.BoolFlag{
						Name:  "fields-structs-with-method",
						Usage: "print fields of structs with methods",
						Value: true,
					},
					&cli.BoolFlag{
						Name:  "methods",
						Usage: "print methods",
						Value: true,
					},
					&cli.BoolFlag{
						Name:  "functions",
						Usage: "print functions",
						Value: true,
					},
					&cli.BoolFlag{
						Name:  "comments",
						Usage: "print comments",
						Value: true,
					},
					&cli.BoolFlag{
						Name:  "tags",
						Usage: "print tags",
						Value: true,
					},
					&cli.StringSliceFlag{
						Name:  "ignore-rule",
						Usage: "ignore files or directories that match the rule. ",
					},
				},
				Action: func(cCtx *cli.Context) error {
					path := cCtx.String("path") // Get the 'path' argument

					ignores := cCtx.StringSlice("ignore-rule")

					if cCtx.Bool("recursive") {
						// ParseDirectoryRecursive
						parsed, err := codesurgeon.ParseDirectoryRecursive(path)
						if err != nil {
							log.Fatal().Err(err).Msg("Failed to parse directory")
						}

						fmt.Println(codesurgeon.PrettyPrint(parsed, cCtx.String("format"), ignores, cCtx.Bool("plain-structs"), cCtx.Bool("fields-plain-structs"), cCtx.Bool("structs-with-method"), cCtx.Bool("fields-structs-with-method"), cCtx.Bool("methods"), cCtx.Bool("functions"), cCtx.Bool("tags"), cCtx.Bool("comments"), cCtx.Bool("omit-nulls")))

					} else {
						parsed, err := codesurgeon.ParseDirectoryWithFilter(path, nil)
						if err != nil {
							log.Fatal().Err(err).Msg("Failed to parse directory")
						}
						fmt.Println(codesurgeon.PrettyPrint([]*codesurgeon.ParsedInfo{parsed}, cCtx.String("format"), ignores, cCtx.Bool("plain-structs"), cCtx.Bool("fields-plain-structs"), cCtx.Bool("structs-with-method"), cCtx.Bool("fields-structs-with-method"), cCtx.Bool("methods"), cCtx.Bool("functions"), cCtx.Bool("tags"), cCtx.Bool("comments"), cCtx.Bool("omit-nulls")))
					}
					return nil
				},
			},
			{
				Name:  "list-tests",
				Usage: "list all test functions in a file or directory with name and description",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "path",
						Aliases: []string{"f"},
						Usage:   "path to the file or directory to parse",
						Value:   ".",
					},
					&cli.BoolFlag{
						Name:    "recursive",
						Aliases: []string{"r"},
						Usage:   "recursively parse directories",
					},
					&cli.BoolFlag{
						Name:  "benchmarks",
						Usage: "include benchmark functions",
						Value: false,
					},
					&cli.StringFlag{
						Name:  "format",
						Usage: "output format: text, json",
						Value: "text",
					},
					&cli.BoolFlag{
						Name:  "omit-nulls",
						Usage: "omit null and empty values from JSON output (json format only)",
						Value: true,
					},
				},
				Action: func(cCtx *cli.Context) error {
					path := cCtx.String("path")
					includeBenchmarks := cCtx.Bool("benchmarks")

					var parsed []*codesurgeon.ParsedInfo
					var err error
					if cCtx.Bool("recursive") {
						parsed, err = codesurgeon.ParseDirectoryRecursive(path)
					} else {
						var p *codesurgeon.ParsedInfo
						p, err = codesurgeon.ParseDirectoryWithFilter(path, nil)
						if p != nil {
							parsed = []*codesurgeon.ParsedInfo{p}
						}
					}
					if err != nil {
						log.Fatal().Err(err).Msg("Failed to parse directory")
					}

					// Filter to only test/benchmark functions
					for _, info := range parsed {
						for mi := range info.Modules {
							for pi := range info.Modules[mi].Packages {
								pkg := &info.Modules[mi].Packages[pi]
								filtered := pkg.Functions[:0]
								for _, fn := range pkg.Functions {
									if fn.IsTest || (includeBenchmarks && fn.IsBenchmark) {
										filtered = append(filtered, fn)
									}
								}
								pkg.Functions = filtered
								pkg.Structs = nil
								pkg.Interfaces = nil
								pkg.Variables = nil
								pkg.Constants = nil
								pkg.Imports = nil
							}
						}
					}

					if cCtx.String("format") == "json" {
						fmt.Println(codesurgeon.PrettyPrint(parsed, "json", nil, false, false, false, false, false, true, false, true, cCtx.Bool("omit-nulls")))
						return nil
					}

					// text format: grouped by file/directory
					for _, info := range parsed {
						pos := info.Directory
						if info.File != "" {
							pos = info.File
						}
						for _, mod := range info.Modules {
							for _, pkg := range mod.Packages {
								if len(pkg.Functions) == 0 {
									continue
								}
								fmt.Printf("# %s (package %s)\n", pos, pkg.Package)
								for _, fn := range pkg.Functions {
									fmt.Printf("TNname:%s\n", fn.Name)
									if len(fn.Docs) > 0 {
										desc := strings.ReplaceAll(strings.Join(fn.Docs, "\n"), "\n", `\n`)
										fmt.Printf("TDesc:%s\n", desc)
									}
								}
								fmt.Println()
							}
						}
					}
					return nil
				},
			},
			{
				Name:  "describe-test",
				Usage: "show full details (docs, signature, body) for one or more test functions by name",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "path",
						Aliases: []string{"f"},
						Usage:   "path to the file or directory to parse",
						Value:   ".",
					},
					&cli.BoolFlag{
						Name:    "recursive",
						Aliases: []string{"r"},
						Usage:   "recursively parse directories",
					},
					&cli.StringSliceFlag{
						Name:     "name",
						Aliases:  []string{"n"},
						Usage:    "test function name(s) to describe (can be specified multiple times)",
						Required: true,
					},
				},
				Action: func(cCtx *cli.Context) error {
					path := cCtx.String("path")
					names := map[string]bool{}
					for _, n := range cCtx.StringSlice("name") {
						names[n] = true
					}

					var parsed []*codesurgeon.ParsedInfo
					var err error
					if cCtx.Bool("recursive") {
						parsed, err = codesurgeon.ParseDirectoryRecursive(path)
					} else {
						var p *codesurgeon.ParsedInfo
						p, err = codesurgeon.ParseDirectoryWithFilter(path, nil)
						if p != nil {
							parsed = []*codesurgeon.ParsedInfo{p}
						}
					}
					if err != nil {
						log.Fatal().Err(err).Msg("Failed to parse directory")
					}

					found := false
					for _, info := range parsed {
						pos := info.Directory
						if info.File != "" {
							pos = info.File
						}
						for _, mod := range info.Modules {
							for _, pkg := range mod.Packages {
								for _, fn := range pkg.Functions {
									if !names[fn.Name] {
										continue
									}
									found = true
									fmt.Printf("# %s (package %s)\n", pos, pkg.Package)
									fmt.Printf("TNname:%s\n", fn.Name)
									fmt.Printf("TSignature:%s\n", fn.Signature)
									if len(fn.Docs) > 0 {
										desc := strings.ReplaceAll(strings.Join(fn.Docs, "\n"), "\n", `\n`)
										fmt.Printf("TDesc:%s\n", desc)
									}
									if fn.Body != "" {
										fmt.Printf("TBody:\n%s\n", fn.Body)
									}
									fmt.Println()
								}
							}
						}
					}
					if !found {
						fmt.Fprintf(os.Stderr, "no tests found matching: %v\n", cCtx.StringSlice("name"))
					}
					return nil
				},
			},
			graphCommand(),
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal().Err(err).Msg("Application failed to run")
	}
}
