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
	"github.com/T4cceptor/MakeMCP/pkg/sources/openapi"
)

// SourcesRegistry is the global source registry.
var SourcesRegistry *SourceRegistry = &SourceRegistry{}

// InitializeSources registers all available source types and returns the registry.
func InitializeSources() *SourceRegistry {
	SourcesRegistry.Register(&openapi.OpenAPISource{})
	// <---------- New source goes here ------------>
	return SourcesRegistry
}

// SourceRegistry manages available sources.
type SourceRegistry map[string]MakeMCPSource

// Register adds a source to the registry.
func (r SourceRegistry) Register(source MakeMCPSource) {
	r[source.Name()] = source
}

// GetAll returns all registered sources.
func (r SourceRegistry) GetAll() map[string]MakeMCPSource {
	return r
}

// Get retrieves a source by name from the registry.
func (r SourceRegistry) Get(name string) MakeMCPSource {
	return r[name]
}
