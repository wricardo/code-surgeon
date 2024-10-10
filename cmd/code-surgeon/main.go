package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"connectrpc.com/connect"
	"github.com/davecgh/go-spew/spew"
	"github.com/joho/godotenv"
	"github.com/urfave/cli/v2"
	codesurgeon "github.com/wricardo/code-surgeon"
	"github.com/wricardo/code-surgeon/ai"
	"github.com/wricardo/code-surgeon/api"
	"github.com/wricardo/code-surgeon/api/apiconnect"
	"github.com/wricardo/code-surgeon/chatcli"
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
				Name: "message",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name: "chat-id",
					},
				},
				Action: func(cCtx *cli.Context) error {
					ngrokDomain, useNgrok := myEnv["NGROK_DOMAIN"]
					if !useNgrok {
						ngrokDomain = "http://localhost:8010"
					}

					// Setup signal handling for graceful exit
					signalChan := make(chan os.Signal, 1)
					signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

					// connect to the grpc server
					client := apiconnect.NewGptServiceClient(http.DefaultClient, "https://"+ngrokDomain) // replace with actual server URL

					message := &api.Message{
						Text: "",
					}
					// populate text with all of stdin
					stdinBytes, err := ioutil.ReadAll(os.Stdin)
					if err != nil {
						fmt.Println("Error reading stdin:", err)
						return err
					}
					message.Text = string(stdinBytes)
					sendMsgReq := &api.SendMessageRequest{
						ChatId:  cCtx.String("chat-id"),
						Message: message,
					}

					ctx := context.Background()
					response, err := client.SendMessage(ctx, connect.NewRequest(sendMsgReq))
					if err != nil {
						fmt.Println("Error sending message:", err)
						return err
					}
					if response.Msg != nil {
						fmt.Println(response.Msg.Message.ChatString())
					}
					return nil
				},
			},
			{
				Name: "new-chat",
				Action: func(cCtx *cli.Context) error {
					ngrokDomain, useNgrok := myEnv["NGROK_DOMAIN"]
					if !useNgrok {
						ngrokDomain = "http://localhost:8010"
					}

					// Setup signal handling for graceful exit
					signalChan := make(chan os.Signal, 1)
					signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

					// connect to the grpc server
					client := apiconnect.NewGptServiceClient(http.DefaultClient, "https://"+ngrokDomain) // replace with actual server URL

					ctx := context.Background()
					response, err := client.NewChat(ctx, connect.NewRequest(&api.NewChatRequest{}))
					if err != nil {
						fmt.Println("Error sending message:", err)
						return err
					}
					if response.Msg != nil {
						fmt.Println(response.Msg.Chat.Id)
					}
					return nil
				},
			},
			{
				Name: "chat",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name: "chat-id",
						// Required: true,
					},
				},
				Action: func(cCtx *cli.Context) error {
					domain, useNgrok := myEnv["NGROK_DOMAIN"]
					if !useNgrok {
						domain = "http://localhost:8010"
					}

					// Setup signal handling for graceful exit
					signalChan := make(chan os.Signal, 1)
					signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
					shutdownChan := make(chan struct{})

					// connect to the grpc server
					chatId := cCtx.String("chat-id")
					if chatId == "" {
						chatId = uuid.New().String()
					}

					var wg sync.WaitGroup
					wg.Add(1)
					go func() {
						// Start CLI
						cliChat := chatcli.NewCliChat("http://" + domain)
						defer wg.Done()
						err := cliChat.Start(shutdownChan, chatId)
						if err != nil {
							log.Fatal().Err(err).Msg("Error starting chat")
						}
						return
					}()
					<-signalChan
					close(shutdownChan)
					fmt.Println("\nshutting down")
					wg.Wait()
					fmt.Println("\nBye")
					return nil
				},
			},
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
				},
				Action: func(cCtx *cli.Context) error {
					path := cCtx.String("path") // Get the 'path' argument

					parsed, err := codesurgeon.ParseDirectoryWithFilter(path, nil)
					if err != nil {
						log.Fatal().Err(err).Msg("Failed to parse directory")
					}
					encoded, _ := json.Marshal(parsed)
					fmt.Println(string(encoded))
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

					// Execute the command to list Go modules in JSON format
					cmd := exec.Command("go", "list", "-json", cCtx.String("path"))
					output, err := cmd.Output()
					if err != nil {
						return fmt.Errorf("failed to execute go list command: %w", err)
					}

					// fmt.Println(string(output))

					// Parse the JSON output and pretty-print
					decoder := json.NewDecoder(strings.NewReader(string(output)))

					for decoder.More() {
						fmt.Println("decoder.More()")
						var module codesurgeon.GoList
						if err := decoder.Decode(&module); err != nil {
							log.Printf("failed to decode module: %v", err)
							continue
						}

						info, err := codesurgeon.ParseDirectory(module.Dir)
						if err != nil {
							log.Info().Err(err).Msgf("Error parsing file %s", module.Dir)
							continue
						}

						fmt.Println("len functions", len(info.Packages[0].Functions))
						for k, function := range info.Packages[0].Functions {
							funcFilePath, err := codesurgeon.FindFunction(module.Dir, "", function.Name)
							if err != nil {
								log.Info().Err(err).Msgf("Error finding function file %s", function.Name)
							} else {
								log.Info().Msgf("funcFilePath: %s", funcFilePath)
							}
							err = neo4j2.UpsertFunctionInfo(ctx, driver, funcFilePath, "", function.Name, strings.Join(function.Docs, "\n"), info.Packages[0].Package, module.ImportPath)
							if err != nil {
								log.Info().Err(err).Msgf("Error upserting function %s", function.Name)
								return err
							}
							log.Info().Msgf("function %d: %s", k, function.Name)
						}
						for k, struct_ := range info.Packages[0].Structs {
							for k2, method := range struct_.Methods {
								fmt.Printf("method\n%s", spew.Sdump(method)) // TODO: wallace debug
								funcFilePath, err := codesurgeon.FindFunction(module.Dir, struct_.Name, method.Name)
								if err != nil {
									log.Info().Err(err).Msgf("Error finding function file %s", method.Name)
								} else {
									log.Info().Msgf("funcFilePath: %s", funcFilePath)
								}
								err = neo4j2.UpsertFunctionInfo(ctx, driver, funcFilePath, struct_.Name, method.Name, strings.Join(method.Docs, "\n"), info.Packages[0].Package, module.ImportPath)
								if err != nil {
									log.Info().Err(err).Msgf("Error upserting function %s", method.Name)
									return err
								}
								log.Info().Msgf("method %d %d: %s", k, k2, method.Name)
							}
						}

						log.Printf("Parsed %s\n", module.Dir)
					}

					return nil
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
