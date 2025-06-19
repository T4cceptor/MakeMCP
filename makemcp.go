package main

import (
	"context"
	"log"
	"os"

	"github.com/urfave/cli/v3"
)

func main() {
	app := &cli.Command{
		Name:  "makemcp",
		Usage: "Create an MCP server out of anything.",
		Commands: []*cli.Command{
			{
				Name:  "openapi",
				Usage: "Use OpenAPI specifications to launch an MCP server locally.",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "source",
						Aliases: []string{"s"},
						Value:   "",
						Usage:   "Where to find the OpenAPI specification - can be either a properly formed URL, including protocol, or a file path to a JSON file.",
					},
					&cli.StringFlag{
						Name:    "base-url",
						Aliases: []string{"b"},
						Value:   "",
						Usage:   "Which URL to call when invoking the different tools",
					},
				},
				Action: func(context context.Context, cmd *cli.Command) error {
					log.Println("Creating config from flags and args (openapi subcommand)")
					source := cmd.String("source")
					baseURL := cmd.String("base-url")
					McpFromOpenAPISpec(source, baseURL)
					return nil
				},
			},
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
