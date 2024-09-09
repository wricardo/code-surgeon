package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/urfave/cli/v2"
	codesurgeon "github.com/wricardo/code-surgeon"
	"github.com/wricardo/code-surgeon/ai"
)

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
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
