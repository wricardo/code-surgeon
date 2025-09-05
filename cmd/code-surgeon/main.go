package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/rs/zerolog/log"

	"connectrpc.com/connect"
	"github.com/joho/godotenv"
	"github.com/urfave/cli/v2"
	codesurgeon "github.com/wricardo/code-surgeon"
	"github.com/wricardo/code-surgeon/ai"
	"github.com/wricardo/code-surgeon/api"
	"github.com/wricardo/code-surgeon/api/apiconnect"
	"github.com/wricardo/code-surgeon/grpc"
	"github.com/wricardo/code-surgeon/log2"
	"github.com/wricardo/code-surgeon/neo4j2"
)

const DEFAULT_PORT = 8010

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
				Name:  "server",
				Usage: "Run the gpt service server",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:     "port",
						Aliases:  []string{"p"},
						Usage:    "port number",
						Required: false,
						Value:    DEFAULT_PORT,
					},
				},
				Action: func(cCtx *cli.Context) error {
					opts := grpc.NewServerOptions()
					serv := grpc.NewServer(opts)

					return serv.Start()
				},
			},
			{
				Name:  "openapi-json",
				Usage: "Generate open api json",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "url",
						Aliases:  []string{"u"},
						Usage:    "ngrok https url. e.g. https://xxxxx.ngrok-free.app",
						Required: false,
						Value:    fmt.Sprintf("http://localhost:%d", DEFAULT_PORT),
					},
				},
				Action: func(cCtx *cli.Context) error {
					client := apiconnect.NewGptServiceClient(http.DefaultClient, cCtx.String("orl"))
					ctx := cCtx.Context
					openAPI, err := client.GetOpenAPI(ctx, connect.NewRequest(&api.GetOpenAPIRequest{}))
					if err != nil {
						return err
					}
					fmt.Println(openAPI.Msg.Openapi)
					return nil
				},
			},
			{
				Name: "mcp-server",
				Action: func(cCtx *cli.Context) error {
					wally := NewMCPServer()
					if err := wally.ServeStdio(); err != nil {
						log.Fatal().Err(err).Msg("Error serving")
					}
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
				Name:  "document-functions",
				Usage: "generate AI documentation for golang code on a  path",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "path",
						Aliases:  []string{"p"},
						Usage:    "path to golang file or folder to generate documentation for go all files",
						Required: false,
						Value:    ".",
					},
					&cli.BoolFlag{
						Name:     "overwrite",
						Usage:    "overwrite existing documentation",
						Required: false,
						Value:    false,
					},
					&cli.StringFlag{
						Name:     "receiver",
						Aliases:  []string{"r"},
						Usage:    "receiver name for the method",
						Value:    "",
						Required: false,
					},
					&cli.StringFlag{
						Name:     "function",
						Aliases:  []string{"f"},
						Usage:    "function name",
						Required: false,
					},
				},
				Action: func(cCtx *cli.Context) error {
					instructorClient := ai.GetInstructor()

					req := ai.GenerateDocumentationRequest{
						Path:              cCtx.String("path"),
						OverwriteExisting: cCtx.Bool("overwrite"),
						ReceiverName:      cCtx.String("receiver"),
						FunctionName:      cCtx.String("function"),
					}
					ok, err := ai.GenerateDocumentation(instructorClient, req)
					if err != nil {
						return err
					}
					if ok {
						fmt.Println("Documentation generated successfully")
					} else {
						fmt.Println("Nothing to do")
					}
					return nil
				},
			},
			// {
			// 	Name:  "instructions",
			// 	Usage: "get instructions to be used in custom chatgpt",
			// 	Flags: []cli.Flag{
			// 		&cli.StringFlag{
			// 			Name:     "url",
			// 			Aliases:  []string{"u"},
			// 			Usage:    "ngrok https url. e.g. https://xxxxx.ngrok-free.app",
			// 			Required: false,
			// 			Value:    fmt.Sprintf("http://localhost:%d", DEFAULT_PORT),
			// 		},
			// 	},
			// 	Action: func(cCtx *cli.Context) error {
			// 		client := apiconnect.NewGptServiceClient(http.DefaultClient, cCtx.String("url"))
			// 		ctx := cCtx.Context
			// 		openAPI, err := client.GetOpenAPI(ctx, connect.NewRequest(&api.GetOpenAPIRequest{}))
			// 		if err != nil {
			// 			return err
			// 		}
			// 		rendered, err := ai.GetGPTInstructions(openAPI.Msg.Openapi)
			// 		if err != nil {
			// 			log.Println("Error getting prompt", err)
			// 			return err
			// 		}
			// 		fmt.Println(rendered)
			// 		return nil

			// 	},
			// },
			{
				Name:  "introduction",
				Usage: "introductions that are displayed to the user when he asks for it, this is used to give context to the llm.",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "url",
						Aliases:  []string{"u"},
						Usage:    "ngrok https url. e.g. https://xxxxx.ngrok-free.app",
						Required: false,
						Value:    fmt.Sprintf("http://localhost:%d", DEFAULT_PORT),
					},
				},
				Action: func(cCtx *cli.Context) error {
					client := apiconnect.NewGptServiceClient(http.DefaultClient, cCtx.String("url"))
					ctx := cCtx.Context
					openAPI, err := client.GetOpenAPI(ctx, connect.NewRequest(&api.GetOpenAPIRequest{}))
					if err != nil {
						return err
					}
					rendered, err := ai.GetGPTIntroduction(openAPI.Msg.Openapi)
					if err != nil {
						log.Info().Err(err).Msg("Error getting prompt")
						return err
					}
					fmt.Println(rendered)
					return nil
				},
			},
			{
				Name:    "ingest-knowledgebase",
				Aliases: []string{},
				Usage:   "Ingest knowledge base folder",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "path",
						Aliases:  []string{"p"},
						Usage:    "path to the knowledge base folder",
						Value:    "./knowledgebase",
						Required: false,
					},
				},
				Action: func(cCtx *cli.Context) error {
					ngrokDomain, useNgrok := myEnv["NGROK_DOMAIN"]
					if !useNgrok {
						ngrokDomain = "http://localhost:8010"
					}

					// Get the path from flags
					path := cCtx.String("path")

					// Read all .txt files in the directory
					entries, err := os.ReadDir(path)
					if err != nil {
						return fmt.Errorf("failed to read directory: %w", err)
					}

					// Connect to the gRPC server
					client := apiconnect.NewGptServiceClient(http.DefaultClient, "https://"+ngrokDomain)

					var questionAnswers []*api.QuestionAnswer

					// Process each .txt file
					for _, entry := range entries {
						if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".txt") {
							filePath := filepath.Join(path, entry.Name())
							content, err := os.ReadFile(filePath)
							if err != nil {
								log.Error().Err(err).Str("file", filePath).Msg("Failed to read file")
								continue
							}

							// Split content into lines
							lines := strings.Split(string(content), "\n")

							// Find first non-empty line for question
							var question string
							var answerLines []string
							foundQuestion := false

							for _, line := range lines {
								trimmedLine := strings.TrimSpace(line)
								if !foundQuestion && trimmedLine != "" {
									question = trimmedLine
									foundQuestion = true
									continue
								}
								if foundQuestion {
									answerLines = append(answerLines, line)
								}
							}

							if question != "" {
								qa := &api.QuestionAnswer{
									Question: question,
									Answer:   strings.Join(answerLines, "\n"),
								}
								questionAnswers = append(questionAnswers, qa)
							}
						}
					}

					// Send all question-answers to the server
					if len(questionAnswers) > 0 {
						req := &api.AddKnowledgeRequest{
							QuestionAnswer: questionAnswers,
						}
						_, err := client.AddKnowledge(cCtx.Context, connect.NewRequest(req))
						if err != nil {
							return fmt.Errorf("failed to add knowledge: %w", err)
						}
						fmt.Printf("Successfully ingested %d knowledge base entries\n", len(questionAnswers))
					} else {
						fmt.Println("No knowledge base entries found")
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
