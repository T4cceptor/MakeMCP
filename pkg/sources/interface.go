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
	"github.com/urfave/cli/v3"
	"github.com/T4cceptor/MakeMCP/pkg/config"
)

// Source defines the interface for all MCP source implementations
type Source interface {
	// Name returns the name of the source type
	Name() string
	
	// Parse converts the input into a MakeMCPApp configuration
	Parse(input string, baseConfig config.MakeMCPApp) (*config.MakeMCPApp, error)
	
	// Validate checks if the input is valid for this source type
	Validate(input string) error
	
	// GetDefaultConfig returns default configuration for this source
	GetDefaultConfig() map[string]interface{}
	
	// GetCommand returns the CLI command for this source
	GetCommand() *cli.Command
}

// Registry manages available sources
type Registry struct {
	sources map[string]Source
}

// NewRegistry creates a new source registry
func NewRegistry() *Registry {
	return &Registry{
		sources: make(map[string]Source),
	}
}

// Register adds a source to the registry
func (r *Registry) Register(source Source) {
	r.sources[source.Name()] = source
}

// Get retrieves a source by name
func (r *Registry) Get(name string) (Source, bool) {
	source, exists := r.sources[name]
	return source, exists
}

// List returns all available source names
func (r *Registry) List() []string {
	names := make([]string, 0, len(r.sources))
	for name := range r.sources {
		names = append(names, name)
	}
	return names
}

// GetAll returns all registered sources
func (r *Registry) GetAll() map[string]Source {
	result := make(map[string]Source)
	for name, source := range r.sources {
		result[name] = source
	}
	return result
}