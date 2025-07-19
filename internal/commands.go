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
	"path/filepath"

	"github.com/urfave/cli/v3"
)

// GetInternalCommands returns all CLI commands related to MakeMCP config file management
func GetInternalCommands() []*cli.Command {
	return []*cli.Command{
		{
			Name:        "load",
			Usage:       "Load and start MCP server from existing config file",
			Description: "Loads a MakeMCP configuration file and starts the MCP server with the saved configuration.",
			ArgsUsage:   "<config-file-path>",
			Action:      handleLoadCommand,
		},
	}
}

// handleLoadCommand handles the load command to start server from config file
func handleLoadCommand(ctx context.Context, cmd *cli.Command) error {
	// Validate arguments
	args := cmd.Args().Slice()
	if len(args) != 1 {
		return fmt.Errorf("load command requires exactly one argument: the path to the config file")
	}

	configPath := args[0]
	
	// Get absolute path for logging
	absPath, err := filepath.Abs(configPath)
	if err != nil {
		log.Printf("Loading MakeMCP configuration from: %s", configPath)
	} else {
		log.Printf("Loading MakeMCP configuration from: %s", absPath)
	}

	// Load configuration from file
	app, err := LoadFromFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration from %s: %w", configPath, err)
	}

	log.Printf("Loaded configuration for MCP server: %s v%s", app.Name, app.Version)

	// TODO: Reconstruct tool handlers based on source type
	// For now, this will work for basic config loading but handlers won't be attached
	// This will need to be implemented once tool marshaling/unmarshaling is ready

	// Start server using existing logic
	return StartServer(app)
}