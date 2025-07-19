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

package internal

import (
	"context"
	"fmt"
	"log"
	"slices"

	"github.com/T4cceptor/MakeMCP/pkg/config"
	"github.com/T4cceptor/MakeMCP/pkg/sources"
	"github.com/urfave/cli/v3"
)

// InitializeRegistries initializes the global registries with default implementations
func InitializeRegistries() {
	// Initialize sources registry (sources register themselves)
	sources.InitializeSources()
}

var defaultFlags []cli.Flag = []cli.Flag{
	&cli.StringFlag{
		Name:    "transport",
		Aliases: []string{"t"},
		Value:   string(config.TransportTypeStdio),
		Usage:   "Used transport protocol for this MCP server - can be either stdio or http.",
	},
	&cli.BoolFlag{
		Name:    "config-only",
		Aliases: []string{"co"},
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
}

// GetCommands returns all CLI commands by combining source and internal commands
func GetCommands() []*cli.Command {
	var commands []*cli.Command

	// Add internal commands (config file management)
	commands = append(commands, GetInternalCommands()...)

	// Auto-discover commands from registered sources
	for _, source := range sources.SourcesRegistry.GetAll() {
		sourceCommand := source.GetCommand()
		sourceCommand.Flags = append(sourceCommand.Flags, defaultFlags...)
		sourceCommand.Action = func(ctx context.Context, cmd *cli.Command) error {
			inputConfig := GetInputConfig(source, cmd)
			return HandleInput(
				source,
				inputConfig,
			)
		}
		commands = append(commands, sourceCommand)
	}
	return commands
}

func GetInputConfig(source sources.MakeMCPSource, cmd *cli.Command) *config.CLIParams {
	cliFlags := map[string]any{}
	for _, flag := range cmd.Flags {
		// We only forward flags that are not already in the default flags
		if !slices.Contains(defaultFlags, flag) {
			cliFlags[flag.Names()[0]] = flag.Get()
		}
	}
	cliArgs := cmd.Args().Slice()
	return &config.CLIParams{
		BaseCLIParams: config.BaseCLIParams{
			Transport:  config.TransportType(cmd.String("transport")),
			ConfigOnly: cmd.Bool("config-only"),
			Port:       cmd.String("port"),
			DevMode:    cmd.Bool("dev-mode"),
			SourceType: source.Name(),
		},
		CliFlags: cliFlags,
		CliArgs:  cliArgs,
	}
}

// HandleInput is the main orchestration function that processes CLI params with a source and manages server lifecycle
func HandleInput(source sources.MakeMCPSource, params *config.CLIParams) error {
	log.Printf("Creating config from %s source with params: %s", source.Name(), params.ToJSON())

	// Parse with the source to create MakeMCPApp
	app, err := source.Parse(params)
	if err != nil {
		return fmt.Errorf("failed to parse with %s source: %w", source.Name(), err)
	}

	// Save configuration
	if err := SaveToFile(app); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	// Exit if config-only mode
	if app.CliParams.ConfigOnly {
		log.Println("Configuration file created. Exiting.")
		return nil
	}

	// Create and attach handler functions to the app
	source.AttachToolHandlers(app)

	// Start server
	return StartServer(app)
}
