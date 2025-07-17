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

package openapi

import (
	"context"
	"fmt"
	"log"

	config "github.com/T4cceptor/MakeMCP/pkg/config"
	"github.com/T4cceptor/MakeMCP/pkg/server"
	sources "github.com/T4cceptor/MakeMCP/pkg/sources"
	"github.com/urfave/cli/v3"
)

// GetCommand returns the CLI command for this source
func (s *OpenAPISource) GetCommand() *cli.Command {
	return &cli.Command{
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
				Value:   string(config.TransportTypeStdio),
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
		Action: func(ctx context.Context, cmd *cli.Command) error {
			params := config.CLIParams{
				Specs:      cmd.String("specs"),
				BaseURL:    cmd.String("base-url"),
				Transport:  config.TransportType(cmd.String("transport")),
				ConfigOnly: cmd.Bool("config-only"),
				Port:       cmd.String("port"),
				DevMode:    cmd.Bool("dev-mode"),
			}
			return sources.HandleInput(
				GetOpenAPISource(),
				params,
			)
		},
	}
}

func GetOpenAPISource() OpenAPISource {
	return OpenAPISource{}
}

// HandleOpenAPI handles the OpenAPI command
func HandleOpenAPI(params config.CLIParams) error {
	log.Println("Creating config from OpenAPI specification")

	// Security checks
	if !params.DevMode {
		WarnURLSecurity(params.Specs, "OpenAPI spec", false)
		WarnURLSecurity(params.BaseURL, "Base URL", false)
	}

	// Create base configuration
	baseConfig := config.NewMakeMCPApp("", "", params.Transport)
	if params.Port != "" {
		baseConfig.Port = &params.Port
	}

	// Parse with OpenAPI source
	source := &OpenAPISource{}
	app, err := source.Parse(params.Specs, baseConfig)
	if err != nil {
		return fmt.Errorf("failed to parse OpenAPI specification: %w", err)
	}

	// Set OpenAPI config for backward compatibility
	app.OpenAPIConfig = &config.OpenAPIConfig{
		BaseURL: params.BaseURL,
	}

	// Save configuration
	if err := config.SaveToFile(app); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	// Exit if config-only mode
	if params.ConfigOnly {
		log.Println("Configuration file created. Exiting.")
		return nil
	}

	// Start server
	mcpServer := server.NewMCPServer(app)
	return mcpServer.Start(params.Transport, params.Port)
}
