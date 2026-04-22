// Package main implements the code-surgeon CLI tool.
// Code-surgeon is a multi-purpose Go development tool that combines code analysis,
// AST parsing, and Neo4j graph database integration for code structure exploration.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/rs/zerolog/log"

	"github.com/joho/godotenv"
	"github.com/urfave/cli/v2"
	codesurgeon "github.com/wricardo/code-surgeon"
	"github.com/wricardo/code-surgeon/ai"
	"github.com/wricardo/code-surgeon/log2"
	"github.com/wricardo/code-surgeon/neo4j2"
)

func getEnvOrDefault(env map[string]string, key, defaultValue string) string {
	if val, ok := env[key]; ok && val != "" {
		return val
	}
	return defaultValue
}

func main() {
	// Initialize logger
	log2.Configure()

	var myEnv map[string]string
	myEnv, err := godotenv.Read()
	if err != nil {
		log.Warn().Msg("Error loading .env file")
	}

	app := &cli.App{
		Name:  "code-surgeon",
		Usage: "Go codebase analysis tool — parse, inspect, and query Go source code structure",
		Description: `code-surgeon parses Go source code into structured data (AST-based) and exposes it via CLI commands. Designed for use by LLMs and developer tooling to efficiently extract code structure without reading raw files.

CORE USE CASES:
  1. Understand what functions, structs, interfaces exist in a package
  2. List and inspect test functions efficiently (low token cost)
  3. Get full details (signature, body, docs) for specific functions or tests
  4. Store code structure in Neo4j for graph queries

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

  to-neo4j         Store parsed code structure into Neo4j graph database.
                   Example: code-surgeon to-neo4j --path . --recursive

  query            Run a Cypher query against the Neo4j graph.
                   Example: code-surgeon query --cypher "MATCH (f:Function) RETURN f.name LIMIT 10"

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
				Name:  "up",
				Usage: "start Neo4j infrastructure and verify it's ready",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "detach",
						Usage: "run containers in background",
						Value: true,
					},
					&cli.IntFlag{
						Name:  "timeout",
						Usage: "timeout in seconds to wait for Neo4j to be ready",
						Value: 60,
					},
				},
				Action: func(cCtx *cli.Context) error {
					fmt.Println("🚀 Starting Neo4j infrastructure...")

					// Check if docker compose is available (try v2 first, then v1)
					dockerCmd := "docker"
					composeArgs := []string{"compose"}

					if _, err := exec.LookPath(dockerCmd); err != nil {
						return fmt.Errorf("docker not found in PATH. Please install Docker")
					}

					// Test if docker compose v2 works
					testCmd := exec.Command(dockerCmd, "compose", "version")
					if err := testCmd.Run(); err != nil {
						// Fall back to docker-compose v1
						if _, err := exec.LookPath("docker-compose"); err != nil {
							return fmt.Errorf("docker compose not found. Please install Docker Compose")
						}
						dockerCmd = "docker-compose"
						composeArgs = []string{}
					}

					// Read embedded docker-compose template
					templateData, err := codesurgeon.FS.ReadFile("templates/docker-compose.codesurgeon.yaml")
					if err != nil {
						return fmt.Errorf("failed to read embedded docker-compose template: %w", err)
					}

					// Write to temp file
					tmpFile, err := os.CreateTemp("", "code-surgeon-docker-compose-*.yaml")
					if err != nil {
						return fmt.Errorf("failed to create temp file: %w", err)
					}
					defer os.Remove(tmpFile.Name())

					if _, err := tmpFile.Write(templateData); err != nil {
						tmpFile.Close()
						return fmt.Errorf("failed to write temp file: %w", err)
					}
					tmpFile.Close()

					// Start docker compose
					args := append(composeArgs, "-f", tmpFile.Name(), "up")
					if cCtx.Bool("detach") {
						args = append(args, "-d")
					}

					cmd := exec.Command(dockerCmd, args...)
					cmd.Stdout = os.Stdout
					cmd.Stderr = os.Stderr

					if err := cmd.Run(); err != nil {
						return fmt.Errorf("failed to start docker compose: %w", err)
					}

					fmt.Println("\n⏳ Waiting for Neo4j to be ready...")

					// Hardcoded credentials matching docker-compose.yaml
					neo4jDbUri := "neo4j://localhost:7687"
					neo4jDbUser := "neo4j"
					neo4jDbPassword := "neo4jneo4j"

					timeout := time.Duration(cCtx.Int("timeout")) * time.Second
					deadline := time.Now().Add(timeout)

					for time.Now().Before(deadline) {
						driver, closeFn, err := neo4j2.Connect(cCtx.Context, neo4jDbUri, neo4jDbUser, neo4jDbPassword)
						if err == nil {
							// Try a simple query to verify it's fully ready
							sess := driver.NewSession(cCtx.Context, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
							_, err := sess.Run(cCtx.Context, "RETURN 1", nil)
							sess.Close(cCtx.Context)
							closeFn()

							if err == nil {
								fmt.Println("\n✅ Neo4j is ready!")
								fmt.Println("\n📊 Neo4j Browser: http://localhost:7474")
								fmt.Printf("🔐 Username: %s\n", neo4jDbUser)
								fmt.Printf("🔑 Password: %s\n", neo4jDbPassword)
								fmt.Println("\n💡 Next steps:")
								fmt.Println("   code-surgeon to-neo4j --path . --recursive")
								return nil
							}
						}
						time.Sleep(2 * time.Second)
						fmt.Print(".")
					}

					return fmt.Errorf("timeout waiting for Neo4j to be ready")
				},
			},
			{
				Name:  "init",
				Usage: "initialize code-surgeon configuration files",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "output",
						Value: "docker-compose.codesurgeon.yaml",
						Usage: "output filename for the docker-compose file",
					},
					&cli.BoolFlag{
						Name:  "force",
						Usage: "overwrite existing files",
					},
				},
				Action: func(cCtx *cli.Context) error {
					outputFile := cCtx.String("output")
					force := cCtx.Bool("force")

					// Check if file already exists
					if _, err := os.Stat(outputFile); err == nil && !force {
						return fmt.Errorf("file %s already exists. Use --force to overwrite", outputFile)
					}

					// Read the embedded template
					templateData, err := codesurgeon.FS.ReadFile("templates/docker-compose.codesurgeon.yaml")
					if err != nil {
						return fmt.Errorf("failed to read template: %w", err)
					}

					// Write the template to the output file
					err = os.WriteFile(outputFile, templateData, 0644)
					if err != nil {
						return fmt.Errorf("failed to write file: %w", err)
					}

					fmt.Printf("Created %s\n", outputFile)
					fmt.Println("\nNext steps:")
					fmt.Printf("1. Start Neo4j: docker-compose -f %s up -d\n", outputFile)
					fmt.Println("2. Access Neo4j Browser: http://localhost:7474")
					fmt.Println("3. Login with username 'neo4j' and password 'neo4jneo4j'")
					fmt.Println("4. Set environment variables in .env file:")
					fmt.Println("   NEO4J_DB_URI=neo4j://localhost:7687")
					fmt.Println("   NEO4J_DB_USER=neo4j")
					fmt.Println("   NEO4J_DB_PASSWORD=neo4jneo4j")

					return nil
				},
			},
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
			{
				Name:    "generate-embeddings",
				Aliases: []string{"ge"},
				Usage:   "Generate embeddings for the specified path",
				Flags:   []cli.Flag{},
				Action: func(cCtx *cli.Context) error {
					neo4jDbUri := getEnvOrDefault(myEnv, "NEO4J_DB_URI", "neo4j://localhost:7687")
					neo4jDbUser := getEnvOrDefault(myEnv, "NEO4J_DB_USER", "neo4j")
					neo4jDbPassword := getEnvOrDefault(myEnv, "NEO4J_DB_PASSWORD", "neo4jneo4j")
					driver, closeFn, err := neo4j2.Connect(cCtx.Context, neo4jDbUri, neo4jDbUser, neo4jDbPassword)
					if err != nil {
						log.Info().Err(err).Msg("Error connecting to Neo4j (proceeding anyway)")
						return err
					} else {
						defer closeFn()
					}
					sess := driver.NewSession(cCtx.Context, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
					defer sess.Close(cCtx.Context)
					return neo4j2.GenerateEmbeddings(driver, ai.GetInstructor().Client)
				},
			},
			{
				Name:    "to-neo4j",
				Aliases: []string{"tn"},
				Usage:   "List all Go modules under the current folder and pretty-print them to stdout",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "path",
						Aliases:  []string{"p"},
						Usage:    "path to the file or directory to parse",
						Value:    "./.",
						Required: false,
					},
					&cli.BoolFlag{
						Name:  "deep",
						Value: false,
					},
					&cli.BoolFlag{
						Name:  "recursive",
						Value: false,
					},
				},
				Action: func(cCtx *cli.Context) error {
					return neo4j2.ToNeo4j(cCtx.Context, cCtx.String("path"), cCtx.Bool("deep"), myEnv, cCtx.Bool("recursive"))
				},
			},
			{
				Name:  "clear-neo4j",
				Usage: "clear all nodes and relationships in neo4j",
				Action: func(cCtx *cli.Context) error {
					ctx := cCtx.Context
					neo4jDbUri := getEnvOrDefault(myEnv, "NEO4J_DB_URI", "neo4j://localhost:7687")
					neo4jDbUser := getEnvOrDefault(myEnv, "NEO4J_DB_USER", "neo4j")
					neo4jDbPassword := getEnvOrDefault(myEnv, "NEO4J_DB_PASSWORD", "neo4jneo4j")
					driver, closeFn, err := neo4j2.Connect(ctx, neo4jDbUri, neo4jDbUser, neo4jDbPassword)
					if err != nil {
						log.Info().Err(err).Msg("Error connecting to Neo4j (proceeding anyway)")
					} else {
						defer closeFn()
					}

					err = neo4j2.ClearAll(ctx, driver)
					if err != nil {
						return err
					}
					return nil
				},
			},

			{
				Name:  "get-schema",
				Usage: "get the schema of the neo4j database",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "format",
						Aliases:  []string{"f"},
						Usage:    "format to print the schema: json, llm",
						Required: false,
						Value:    "json",
					},
				},
				Action: func(cCtx *cli.Context) error {
					ctx := cCtx.Context
					neo4jDbUri := getEnvOrDefault(myEnv, "NEO4J_DB_URI", "neo4j://localhost:7687")
					neo4jDbUser := getEnvOrDefault(myEnv, "NEO4J_DB_USER", "neo4j")
					neo4jDbPassword := getEnvOrDefault(myEnv, "NEO4J_DB_PASSWORD", "neo4jneo4j")
					driver, closeFn, err := neo4j2.Connect(ctx, neo4jDbUri, neo4jDbUser, neo4jDbPassword)
					if err != nil {
						log.Info().Err(err).Msg("Error connecting to Neo4j (proceeding anyway)")
					} else {
						defer closeFn()
					}

					schema, err := neo4j2.GetSchema(ctx, driver)
					if err != nil {
						return err
					}
					fmt.Println(schema.Format(cCtx.String("format")))
					return nil
				},
			},

			{
				Name:    "query",
				Aliases: []string{"q"},
				Usage:   "execute a Cypher query against the Neo4j database",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "cypher",
						Aliases:  []string{"c"},
						Usage:    "Cypher query to execute",
						Required: true,
					},
					&cli.StringFlag{
						Name:    "format",
						Aliases: []string{"f"},
						Usage:   "output format: json, table, raw",
						Value:   "table",
					},
				},
				Action: func(cCtx *cli.Context) error {
					ctx := cCtx.Context
					neo4jDbUri := getEnvOrDefault(myEnv, "NEO4J_DB_URI", "neo4j://localhost:7687")
					neo4jDbUser := getEnvOrDefault(myEnv, "NEO4J_DB_USER", "neo4j")
					neo4jDbPassword := getEnvOrDefault(myEnv, "NEO4J_DB_PASSWORD", "neo4jneo4j")

					driver, closeFn, err := neo4j2.Connect(ctx, neo4jDbUri, neo4jDbUser, neo4jDbPassword)
					if err != nil {
						return fmt.Errorf("failed to connect to Neo4j: %w", err)
					}
					defer closeFn()

					cypherQuery := cCtx.String("cypher")
					results, err := neo4j2.QueryNeo4J(ctx, driver, cypherQuery, nil)
					if err != nil {
						return fmt.Errorf("query execution failed: %w", err)
					}

					format := cCtx.String("format")
					err = neo4j2.FormatQueryResults(results, format)
					if err != nil {
						return fmt.Errorf("failed to format results: %w", err)
					}

					return nil
				},
			},

			// {
			// 	Name:  "add-metadata-to-neo4j",
			// 	Usage: "",
			// 	Flags: []cli.Flag{
			// 		&cli.StringFlag{
			// 			Name:     "path",
			// 			Aliases:  []string{"p"},
			// 			Usage:    "path of folder to add metadata to neo4j. All other folders will be ignored.",
			// 			Required: false,
			// 			Value:    "",
			// 		},
			// 	},
			// 	Action: func(cCtx *cli.Context) error {
			// 		ctx := cCtx.Context
			// 		neo4jDbUri, _ := myEnv["NEO4J_DB_URI"]
			// 		neo4jDbUser, _ := myEnv["NEO4J_DB_USER"]
			// 		neo4jDbPassword, _ := myEnv["NEO4J_DB_PASSWORD"]

			// 		driver, closeFn, err := neo4j2.Connect(ctx, neo4jDbUri, neo4jDbUser, neo4jDbPassword)
			// 		if err != nil {
			// 			log.Println("Error connecting to Neo4j (proceeding anyway):", err)
			// 		} else {
			// 			defer closeFn()
			// 		}

			// 		files, err := neo4j2.ListFileFromFunctions(ctx, driver)
			// 		if err != nil {
			// 			return err
			// 		}
			// 		fmt.Printf("files\n%s", spew.Sdump(files)) // TODO: wallace debug

			// 		for _, file := range files {
			// 			if strings.Contains(file, "@") {
			// 				log.Printf("Skipping file %s\n", file)
			// 				continue
			// 			}

			// 			if cCtx.String("path") != "" && !strings.HasPrefix(file, cCtx.String("path")) {
			// 				log.Printf("Skipping file %s\n", file)
			// 				continue
			// 			}

			// 			info, err := codesurgeon.ParseDirectory(file)
			// 			if err != nil {
			// 				log.Println("Error parsing file", file, err)
			// 				continue
			// 			}

			// 			for _, function := range info.Packages[0].Functions {
			// 				// UpsertFunctionInfo
			// 				err := neo4j2.UpsertFunctionInfo(ctx, driver, file, "", function.Name, strings.Join(function.Docs, "\n"), info.Packages[0].Package)
			// 				if err != nil {
			// 					log.Println("Error upserting function", function, err)
			// 					// return err
			// 				}
			// 			}
			// 		}

			// 		return nil

			// 	},
			// },
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal().Err(err).Msg("Application failed to run")
	}
}
