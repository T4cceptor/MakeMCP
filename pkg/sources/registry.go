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

	core "github.com/T4cceptor/MakeMCP/pkg/core"
	"github.com/T4cceptor/MakeMCP/pkg/sources/openapi"
)

// SourcesRegistry is the global source registry
var SourcesRegistry *SourceRegistry = &SourceRegistry{}

func InitializeSources() *SourceRegistry {
	// Register all available sources
	SourcesRegistry.Register(&openapi.OpenAPISource{})
	return SourcesRegistry
}

// SourceRegistry manages available sources
type SourceRegistry map[string]MakeMCPSource

// Register adds a source to the registry
func (r SourceRegistry) Register(source MakeMCPSource) {
	r[source.Name()] = source
}

// GetAll returns all registered sources
func (r SourceRegistry) GetAll() map[string]MakeMCPSource {
	result := make(map[string]MakeMCPSource)
	for name, source := range r {
		result[name] = source
	}
	return result
}

func (r SourceRegistry) Get(name string) MakeMCPSource {
	return r[name]
}

// CreateHandlers attaches handler functions to all tools in the MakeMCPApp
func CreateHandlers(app *core.MakeMCPApp) error {
	sourceType := app.SourceParams.GetSharedParams().SourceType
	source, exists := (*SourcesRegistry)[sourceType]
	if !exists {
		return fmt.Errorf("unknown source type: %s", sourceType)
	}
	return source.AttachToolHandlers(app)
}
