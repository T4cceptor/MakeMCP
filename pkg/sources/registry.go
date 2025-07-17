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

	"github.com/T4cceptor/MakeMCP/pkg/config"
)

// SourcesRegistry is the global source registry
var SourcesRegistry *SourceRegistry = NewSourceRegistry()

func InitializeSources() *SourceRegistry {
	// TODO: implement code to register all available sources
	return SourcesRegistry
}

// ParseWithSource parses input using a specific source type
func ParseWithSource(sourceType, input string, baseConfig config.MakeMCPApp) (*config.MakeMCPApp, error) {
	source, exists := SourcesRegistry.Get(sourceType)
	if !exists {
		return nil, fmt.Errorf("unknown source type: %s", sourceType)
	}

	return source.Parse(input, baseConfig)
}

// ValidateWithSource validates input using a specific source type
func ValidateWithSource(sourceType, input string) error {
	source, exists := SourcesRegistry.Get(sourceType)
	if !exists {
		return fmt.Errorf("unknown source type: %s", sourceType)
	}

	return source.Validate(input)
}

// GetAvailableSources returns a list of all available source types
func GetAvailableSources() []string {
	return SourcesRegistry.List()
}

// GetSourceConfig gets the default configuration for a source type
func GetSourceConfig(sourceType string) (map[string]interface{}, error) {
	source, exists := SourcesRegistry.Get(sourceType)
	if !exists {
		return nil, fmt.Errorf("unknown source type: %s", sourceType)
	}

	return source.GetDefaultConfig(), nil
}
