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
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	core "github.com/T4cceptor/MakeMCP/pkg/core"
	"github.com/T4cceptor/MakeMCP/pkg/sources"
)

// SaveToFile serializes the given MakeMCPApp as JSON and writes it to a file.
// The filename is derived from the file parameter or defaults to app name (e.g., "makemcp.json")
func SaveToFile(app *core.MakeMCPApp) error {
	var filename string
	sharedParams := app.SourceParams.GetSharedParams()
	if sharedParams.File != "" {
		filename = fmt.Sprintf("%s.json", sharedParams.File)
	} else {
		filename = fmt.Sprintf("%s_makemcp.json", app.Name)
	}

	// Ensure the directory exists
	dir := filepath.Dir(filename)
	if dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("Failed to close file: %v", err)
		}
	}()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(app); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	// Get absolute path for logging
	absPath, err := filepath.Abs(filename)
	if err != nil {
		log.Printf("MakeMCPApp saved to %s\n", filename)
	} else {
		log.Printf("MakeMCPApp saved to %s\n", absPath)
	}
	return nil
}

// LoadFromFile loads a MakeMCPApp from a JSON file.
func LoadFromFile(filename string) (*core.MakeMCPApp, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("Failed to close file: %v", err)
		}
	}()

	// Read all data first
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Parse just the metadata to get source type
	var metadata struct {
		SourceType string `json:"sourceType"`
	}
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to decode metadata: %w", err)
	}

	// Get source from registry
	source := sources.SourcesRegistry.Get(metadata.SourceType)
	if source == nil {
		return nil, fmt.Errorf("unknown source type: %s", metadata.SourceType)
	}

	// Use source's UnmarshalConfig method directly
	// Note: Sources need to implement UnmarshalConfig method
	app, err := source.UnmarshalConfig(data)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if err := source.AttachToolHandlers(app); err != nil {
		return nil, fmt.Errorf("failed to attach tool handlers: %w", err)
	}

	log.Printf("MakeMCPApp loaded from %s\n", filename)
	return app, nil
}
