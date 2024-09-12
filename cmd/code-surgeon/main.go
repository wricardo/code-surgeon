package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"connectrpc.com/connect"
	"github.com/urfave/cli/v2"
	codesurgeon "github.com/wricardo/code-surgeon"
	"github.com/wricardo/code-surgeon/ai"
	"github.com/wricardo/code-surgeon/api"
	"github.com/wricardo/code-surgeon/api/apiconnect"
	"github.com/wricardo/code-surgeon/grpc/server"
)

const DEFAULT_PORT = 8002

func main() {
	app := &cli.App{
		Name:  "code-surgeon",
		Usage: "A CLI tool to help you manage your codebase",
		Action: func(*cli.Context) error {
			fmt.Println("boom! I say!")
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
				},
				Action: func(cCtx *cli.Context) error {
					path := cCtx.String("path") // Get the 'path' argument

					parsed, err := codesurgeon.ParseDirectoryWithFilter(path, nil)
					if err != nil {
						log.Fatal(err)
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
					req := ai.GenerateDocumentationRequest{
						Path:              cCtx.String("path"),
						OverwriteExisting: cCtx.Bool("overwrite"),
						ReceiverName:      cCtx.String("receiver"),
						FunctionName:      cCtx.String("function"),
					}
					ok, err := ai.GenerateDocumentation(req)
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
			{
				Name:  "gpt-service-server",
				Usage: "Run the gpt service server",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:     "port",
						Aliases:  []string{"p"},
						Usage:    "port number",
						Required: false,
						Value:    DEFAULT_PORT,
					},
					&cli.BoolFlag{
						Name:     "use-ngrok",
						Aliases:  []string{"n"},
						Usage:    "use ngrok to expose the server",
						Required: false,
						Value:    false,
					},
				},
				Action: func(cCtx *cli.Context) error {
					return server.Start(cCtx.Int("port"), cCtx.Bool("use-ngrok"))

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
					client := apiconnect.NewGptServiceClient(http.DefaultClient, cCtx.String("url"))
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
				Name:  "instructions",
				Usage: "get instructions to be used in custom chatgpt",
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
					rendered, err := ai.GetGPTInstructions(openAPI.Msg.Openapi)
					if err != nil {
						log.Println("Error getting prompt", err)
						return err
					}
					fmt.Println(rendered)
					return nil

				},
			},
			{
				Name:  "introduction",
				Usage: "introductions that are displayed to the user when he asks for it, this is used to give context to the llm.",
				Flags: []cli.Flag{
					// &cli.StringFlag{
					// 	Name:     "proto-filepath",
					// 	Aliases:  []string{"f"},
					// 	Usage:    "path to the api proto file",
					// 	Required: false,
					// 	Value:    "api/codesurgeon.proto",
					// },
					&cli.StringFlag{
						Name:     "url",
						Aliases:  []string{"u"},
						Usage:    "ngrok https url. e.g. https://xxxxx.ngrok-free.app",
						Required: false,
						Value:    fmt.Sprintf("http://localhost:%d", DEFAULT_PORT),
					},
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
						log.Println("Error getting prompt", err)
						return err
					}
					fmt.Println(rendered)
					return nil

				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
