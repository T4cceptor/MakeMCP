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

// MakeMCPApp holds all information about the MCP server.
// Main data structure representing a complete MCP application configuration
type MakeMCPApp struct {
	Name       string        `json:"name"`       // Name of the App
	Version    string        `json:"version"`    // Version of the app
	SourceType string        `json:"sourceType"` // Type of source (openapi, cli, etc.)
	Tools      []MakeMCPTool `json:"tools"`      // Tools the MCP server will provide
	AppParams  AppParams     `json:"config"`     // Source-specific parameters
}

// NewMakeMCPApp creates a new MakeMCPApp with provided parameters.
func NewMakeMCPApp(name, version string, appParams AppParams) MakeMCPApp {
	return MakeMCPApp{
		Name:       name,
		Version:    version,
		SourceType: appParams.GetSourceType(),
		Tools:      []MakeMCPTool{},
		AppParams:  appParams,
	}
}
