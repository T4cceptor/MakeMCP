// Copyright 2025 MakeMCP Contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/urfave/cli/v3"
)

// version is set by build flags during release
var version = "dev"

func main() {
	app := &cli.Command{
		Name:    "makemcp",
		Usage:   "Create an MCP server out of anything.",
		Version: version,
		Commands: []*cli.Command{
			{
				Name:  "openapi",
				Usage: "Use OpenAPI specifications to launch an MCP server locally.",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "specs",
						Aliases: []string{"s"},
						Value:   "",
						Usage:   "Where to find the OpenAPI specification - can be either a properly formed URL, including protocol, or a file path to a JSON file.",
					},
					&cli.StringFlag{
						Name:    "base-url",
						Aliases: []string{"b"},
						Value:   "",
						Usage:   "Base URL of the OpenAPI specified API. This will be called when invoking the tools.",
					},
					&cli.StringFlag{
						Name:    "transport",
						Aliases: []string{"t"},
						Value:   string(TransportTypeStdio),
						Usage:   "Used transport protocol for this MCP server - can be either stdio or http.",
					},
					&cli.BoolFlag{
						Name:    "config-only",
						Aliases: []string{"oc"},
						Value:   false,
						Usage:   "If set to true only creates a config file and exits, no server will be started.",
					},
					&cli.StringFlag{
						Name:  "port",
						Value: "8080",
						Usage: "Defines the port on which the HTTP server is started, ignored if transport is set to stdio.",
					},
					&cli.BoolFlag{
						Name:  "dev-mode",
						Value: false,
						Usage: "Enable development mode - suppresses security warnings for local/private URLs. Use only for local development.",
					},
				},
				Action: func(context context.Context, cmd *cli.Command) error {
					log.Println("Creating config from flags and args (openapi subcommand)")
					params := CLIParams{
						Specs:      cmd.String("specs"),
						BaseURL:    cmd.String("base-url"),
						Transport:  TransportType(cmd.String("transport")),
						ConfigOnly: cmd.Bool("config-only"),
						Port:       cmd.String("port"),
						DevMode:    cmd.Bool("dev-mode"),
					}
					HandleOpenAPI(params)
					return nil
				},
			},
			{
				Name:  "file",
				Usage: "Load a MakeMCP configuration file and start the server.",
				Args:  true,
				ArgsUsage: "<config-file>",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "transport",
						Aliases: []string{"t"},
						Value:   "",
						Usage:   "Override transport protocol for this MCP server - can be either stdio or http. If not specified, uses the transport from the config file.",
					},
					&cli.StringFlag{
						Name:  "port",
						Value: "",
						Usage: "Override the port on which the HTTP server is started. If not specified, uses the port from the config file or defaults to 8080.",
					},
				},
				Action: func(context context.Context, cmd *cli.Command) error {
					args := cmd.Args()
					if args.Len() == 0 {
						return fmt.Errorf("config file path is required")
					}
					
					configPath := args.First()
					transportOverride := cmd.String("transport")
					portOverride := cmd.String("port")
					
					log.Printf("Loading MakeMCP configuration from: %s", configPath)
					HandleFile(configPath, transportOverride, portOverride)
					return nil
				},
			},
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
