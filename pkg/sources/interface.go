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

package sources

import (
	"fmt"
	"log"

	"github.com/T4cceptor/MakeMCP/pkg/config"
	"github.com/T4cceptor/MakeMCP/pkg/server"
	"github.com/urfave/cli/v3"
)

// Source defines the interface for all MCP source implementations
type Source interface {
	// Name returns the name of the source type
	Name() string

	// Creates a new MakeMCPApp using the provided CLIParams
	CreateApp(params config.CLIParams) (*config.MakeMCPApp, error)

	// Adds tool handlers to MakeMCPApp,
	// making it ready to be hosted as an MCP server
	AttachHandlers(*config.MakeMCPApp) *config.MakeMCPApp

	// Validate checks if the input is valid for this source type
	Validate(input string) error

	// GetCommand returns the CLI command for this source
	GetCommand() *cli.Command
}

// TODO: implement code to use Parse, validate, and "get handlers"
// to create a Ready to host app
// then pass it to a CreateServer method which should handle the rest

func HandleInput(source Source, params config.CLIParams) error {
	app, err := source.CreateApp(params)
	if err != nil {
		return fmt.Errorf("failed to create configuration: %w", err)
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

	// Attach tool handlers
	app = source.AttachHandlers(app)
	mcpServer := server.NewMCPServer(app)
	return mcpServer.Start(params.Transport, params.Port)
}

// SourceRegistry manages available sources
type SourceRegistry struct {
	sources map[string]Source
}

// NewSourceRegistry creates a new source registry
func NewSourceRegistry() *SourceRegistry {
	return &SourceRegistry{
		sources: make(map[string]Source),
	}
}

// Register adds a source to the registry
func (r *SourceRegistry) Register(source Source) {
	r.sources[source.Name()] = source
}

// Get retrieves a source by name
func (r *SourceRegistry) Get(name string) (Source, bool) {
	source, exists := r.sources[name]
	return source, exists
}

// List returns all available source names
func (r *SourceRegistry) List() []string {
	names := make([]string, 0, len(r.sources))
	for name := range r.sources {
		names = append(names, name)
	}
	return names
}

// GetAll returns all registered sources
func (r *SourceRegistry) GetAll() map[string]Source {
	result := make(map[string]Source)
	for name, source := range r.sources {
		result[name] = source
	}
	return result
}
