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

package openapi

import (
	"fmt"
	"log"

	"github.com/T4cceptor/MakeMCP/pkg/config"
	"github.com/T4cceptor/MakeMCP/pkg/server"
	"github.com/T4cceptor/MakeMCP/pkg/sources"
)

// HandleOpenAPI handles the OpenAPI command
func HandleOpenAPI(params config.CLIParams) error {
	log.Println("Creating config from OpenAPI specification")

	// Security checks
	if !params.DevMode {
		WarnURLSecurity(params.Specs, "OpenAPI spec", false)
		WarnURLSecurity(params.BaseURL, "Base URL", false)
	}

	// Create base configuration
	baseConfig := config.NewMakeMCPApp("", "", params.Transport)
	if params.Port != "" {
		baseConfig.Port = &params.Port
	}

	// Parse with OpenAPI source
	app, err := sources.ParseWithSource("openapi", params.Specs, baseConfig)
	if err != nil {
		return fmt.Errorf("failed to parse OpenAPI specification: %w", err)
	}

	// Set OpenAPI config for backward compatibility
	app.OpenAPIConfig = &config.OpenAPIConfig{
		BaseURL: params.BaseURL,
	}

	// Save configuration
	if err := app.SaveToFile(); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	// Exit if config-only mode
	if params.ConfigOnly {
		log.Println("Configuration file created. Exiting.")
		return nil
	}

	// Start server
	mcpServer := server.NewMCPServer(app)
	return mcpServer.Start(params.Transport, params.Port)
}
