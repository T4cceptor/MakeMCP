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

package file

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/T4cceptor/MakeMCP/pkg/config"
)

// FileSource implements the sources.Source interface for MakeMCP configuration files
type FileSource struct{}

// Name returns the name of this source type
func (s *FileSource) Name() string {
	return "file"
}

// Parse loads a MakeMCPApp from a JSON file
func (s *FileSource) Parse(input string, baseConfig config.MakeMCPApp) (*config.MakeMCPApp, error) {
	// For file source, we load directly from the file
	return config.LoadFromFile(input)
}

// Validate checks if the input is a valid file path
func (s *FileSource) Validate(input string) error {
	// Try to load the file to validate it
	_, err := config.LoadFromFile(input)
	return err
}

// GetDefaultConfig returns the default configuration for file sources
func (s *FileSource) GetDefaultConfig() map[string]interface{} {
	return map[string]interface{}{
		"path": "",
	}
}

// GetCommand returns the CLI command for this source
func (s *FileSource) GetCommand() *cli.Command {
	return &cli.Command{
		Name:      "file",
		Usage:     "Load a MakeMCP configuration file and start the server.",
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
		Action: func(ctx context.Context, cmd *cli.Command) error {
			args := cmd.Args()
			if args.Len() == 0 {
				return fmt.Errorf("config file path is required")
			}
			
			configPath := args.First()
			transportOverride := cmd.String("transport")
			portOverride := cmd.String("port")
			
			return HandleFile(configPath, transportOverride, portOverride)
		},
	}
}