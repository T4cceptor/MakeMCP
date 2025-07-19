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

package core

import "encoding/json"

// SourceParams defines the interface for source-specific parameters
// Each source type implements this interface with their own typed parameters
type SourceParams interface {
	// GetSharedParams returns the shared parameters that all sources need
	GetSharedParams() *SharedParams

	// Validate performs source-specific parameter validation
	Validate() error

	// ToJSON returns a JSON representation for logging and debugging
	ToJSON() string

	// GetSourceType returns the source type identifier
	GetSourceType() string
}

// SharedParams holds parameters that are common across all source types
type SharedParams struct {
	Transport  TransportType `json:"transport"`  // stdio or http
	ConfigOnly bool          `json:"configOnly"` // if true, only creates config file
	Port       string        `json:"port"`       // only valid with transport=http
	DevMode    bool          `json:"devMode"`    // true if running in development mode
	SourceType string        `json:"sourceType"` // type of source (openapi, cli, etc.)
	File       string        `json:"file"`       // filename (without extension) for config file
}

// NewSharedParams creates a new SharedParams with default values
func NewSharedParams(sourceType string, transport TransportType) *SharedParams {
	// Validate transport type - if invalid, default to stdio
	if !transport.IsValid() {
		transport = TransportTypeStdio
	}

	return &SharedParams{
		Transport:  transport,
		ConfigOnly: false,
		Port:       "8080",
		DevMode:    false,
		SourceType: sourceType,
		File:       "makemcp",
	}
}

// ToJSON returns a JSON representation of the SharedParams for logging and debugging
func (s SharedParams) ToJSON() string {
	jsonBytes, err := json.Marshal(s)
	if err != nil {
		return `{"error": "failed to marshal SharedParams to JSON"}`
	}
	return string(jsonBytes)
}

// CLIParamsInput holds raw CLI input that needs to be parsed into typed parameters
type CLIParamsInput struct {
	SharedParams *SharedParams  `json:"shared"`
	CliFlags     map[string]any `json:"flags"` // raw CLI flags to be parsed by sources
	CliArgs      []string       `json:"args"`  // raw CLI arguments
}

// ToJSON returns a JSON representation of the CLIParamsInput for logging and debugging
func (c CLIParamsInput) ToJSON() string {
	jsonBytes, err := json.Marshal(c)
	if err != nil {
		return `{"error": "failed to marshal CLIParamsInput to JSON"}`
	}
	return string(jsonBytes)
}
