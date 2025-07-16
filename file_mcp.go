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

package main

import (
	"fmt"
	"log"

	"github.com/mark3labs/mcp-go/server"
)

// HandleFile loads a MakeMCP configuration file and starts the server
func HandleFile(configPath, transportOverride, portOverride string) {
	// Load the MakeMCP app from file
	app, err := MakeMCPAppFromFile(configPath)
	if err != nil {
		log.Fatalf("Failed to load config file: %v", err)
	}

	// Add handler functions based on the loaded configuration
	if app.OpenAPIConfig != nil && app.OpenAPIConfig.BaseUrl != "" {
		log.Printf("Found OpenAPI configuration with base URL: %s", app.OpenAPIConfig.BaseUrl)
		AddOpenAPIHandlerFunctions(&app, NewAPIClient(app.OpenAPIConfig.BaseUrl))
	} else {
		log.Println("No OpenAPI configuration found, tools will not have handler functions")
	}

	// Apply overrides if provided
	transport := app.Transport
	if transportOverride != "" {
		transport = transportOverride
		log.Printf("Overriding transport from config (%s) with: %s", app.Transport, transport)
	}

	port := "8080" // default port
	if app.Port != nil && *app.Port != "" {
		port = *app.Port
	}
	if portOverride != "" {
		port = portOverride
		log.Printf("Overriding port with: %s", port)
	}

	// Create and start the MCP server
	mcp_server := GetMCPServer(app)

	switch TransportType(transport) {
	case TransportTypeHTTP:
		log.Printf("Starting HTTP MCP server on port %s...", port)
		streamable_server := server.NewStreamableHTTPServer(mcp_server)
		streamable_server.Start(fmt.Sprintf(":%s", port))
	case TransportTypeStdio:
		log.Println("Starting stdio MCP server...")
		if err := server.ServeStdio(mcp_server); err != nil {
			log.Printf("Server error: %v\n", err)
		}
	default:
		log.Fatalf("Unsupported transport type: %s", transport)
	}
}
