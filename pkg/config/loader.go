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

package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

// SaveToFile serializes the given MakeMCPApp as JSON and writes it to a file
// The filename is derived from the MCP server name (e.g., "myserver.makemcp.json")
func SaveToFile(app *MakeMCPApp) error {
	filename := fmt.Sprintf("%s.makemcp.json", app.Name)
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(app); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}
	log.Printf("MakeMCPApp saved to %s\n", filename)
	return nil
}

// LoadFromFile loads a MakeMCPApp from a JSON file
func LoadFromFile(filename string) (*MakeMCPApp, error) {
	var app MakeMCPApp

	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&app); err != nil {
		return nil, fmt.Errorf("failed to decode JSON: %w", err)
	}

	log.Printf("MakeMCPApp loaded from %s\n", filename)
	return &app, nil
}
