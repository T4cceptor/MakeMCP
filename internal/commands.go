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

package cli

import (
	"github.com/urfave/cli/v3"

	"github.com/T4cceptor/MakeMCP/pkg/processors"
	"github.com/T4cceptor/MakeMCP/pkg/processors/auth"
	"github.com/T4cceptor/MakeMCP/pkg/sources"
	"github.com/T4cceptor/MakeMCP/pkg/sources/file"
	"github.com/T4cceptor/MakeMCP/pkg/sources/openapi"
)

// InitializeRegistries initializes the global registries with default implementations
func InitializeRegistries() {
	// Register sources
	sources.SourcesRegistry.Register(&openapi.OpenAPISource{})
	sources.SourcesRegistry.Register(&file.FileSource{})

	// Register processors
	processors.DefaultRegistry.Register(&auth.APIKeyProcessor{})
}

// GetCommands returns all CLI commands by auto-discovering from source registry
func GetCommands() []*cli.Command {
	var commands []*cli.Command

	// Auto-discover commands from registered sources
	for _, source := range sources.SourcesRegistry.GetAll() {
		commands = append(commands, source.GetCommand())
	}

	return commands
}
