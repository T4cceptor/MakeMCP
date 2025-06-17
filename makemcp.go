package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/urfave/cli/v3"
)

type MakeConfig struct {
	source_type string
	// what sources for MakeMCP do we have?
	// how can we abstract them sufficiently?

	// OpenAPI URL
	// OpenAPI spec file - file path to a json file
	// CLI command -> using "-h" or "--help" should provide us with enough understanding to parse it into an MCP server

	source_uri string
}

func NewMakeConfig() MakeConfig {
	return MakeConfig{
		source_type: "openapi_json",
	}
}

func main() {
	// options for MCP server creation:
	// 1. OpenAPI specs - via json -> file path
	// https://cloud.ibm.com/apidocs/cos/cos-compatibility.json

	cmd := &cli.Command{
		Name:  "makemcp",
		Usage: "Create an MCP server out of anything. \n The command defaults to using OpenAPI specifications to launch an MCP server locally.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "source-uri",
				Aliases: []string{"u"},
				Value:   "",
				Usage:   "Where to find the OpenAPI specification",
			},
			&cli.StringFlag{
				Name:    "source-type",
				Aliases: []string{"t"},
				Value:   "url",
				Usage:   "How the OpenAPI specs are provided, file or url.",
			},
		},
		Action: func(context context.Context, cmd *cli.Command) error {
			fmt.Println("boom! I say! 123")

			config := NewMakeConfig()
			config.source_uri = cmd.String("source-uri")
			Setup(config)
			return nil
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
