package main

import (
	"fmt"
	"os"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/rs/zerolog/log"

	"github.com/joho/godotenv"
	"github.com/urfave/cli/v2"
	codesurgeon "github.com/wricardo/code-surgeon"
	"github.com/wricardo/code-surgeon/ai"
	"github.com/wricardo/code-surgeon/log2"
	"github.com/wricardo/code-surgeon/neo4j2"
)

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
		Usage: "A CLI tool to help you manage your codebase",
		Action: func(*cli.Context) error {
			fmt.Println("--help for more information.")
			return nil
		},
		Commands: []*cli.Command{
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

						fmt.Println(codesurgeon.PrettyPrint(parsed, cCtx.String("format"), ignores, cCtx.Bool("plain-structs"), cCtx.Bool("fields-plain-structs"), cCtx.Bool("structs-with-method"), cCtx.Bool("fields-structs-with-method"), cCtx.Bool("methods"), cCtx.Bool("functions"), cCtx.Bool("tags"), cCtx.Bool("comments")))

					} else {
						parsed, err := codesurgeon.ParseDirectoryWithFilter(path, nil)
						if err != nil {
							log.Fatal().Err(err).Msg("Failed to parse directory")
						}
						fmt.Println(codesurgeon.PrettyPrint([]*codesurgeon.ParsedInfo{parsed}, cCtx.String("format"), ignores, cCtx.Bool("plain-structs"), cCtx.Bool("fields-plain-structs"), cCtx.Bool("structs-with-method"), cCtx.Bool("fields-structs-with-method"), cCtx.Bool("methods"), cCtx.Bool("functions"), cCtx.Bool("tags"), cCtx.Bool("comments")))
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
					neo4jDbUri, _ := myEnv["NEO4J_DB_URI"]
					neo4jDbUser, _ := myEnv["NEO4J_DB_USER"]
					neo4jDbPassword, _ := myEnv["NEO4J_DB_PASSWORD"]
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
					neo4jDbUri, _ := myEnv["NEO4J_DB_URI"]
					neo4jDbUser, _ := myEnv["NEO4J_DB_USER"]
					neo4jDbPassword, _ := myEnv["NEO4J_DB_PASSWORD"]
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
					neo4jDbUri, _ := myEnv["NEO4J_DB_URI"]
					neo4jDbUser, _ := myEnv["NEO4J_DB_USER"]
					neo4jDbPassword, _ := myEnv["NEO4J_DB_PASSWORD"]
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
