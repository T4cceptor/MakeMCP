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

package file

import (
	"fmt"
	"log"

	"github.com/T4cceptor/MakeMCP/pkg/config"
	"github.com/T4cceptor/MakeMCP/pkg/server"
)

// HandleFile handles the file command
func HandleFile(configPath, transportOverride, portOverride string) error {
	log.Printf("Loading MakeMCP configuration from: %s", configPath)
	
	// Load configuration
	app, err := config.LoadFromFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	
	// Apply overrides
	transport := config.TransportType(app.Transport)
	if transportOverride != "" {
		transport = config.TransportType(transportOverride)
		log.Printf("Overriding transport from config (%s) with: %s", app.Transport, transport)
	}
	
	port := "8080"
	if app.Port != nil && *app.Port != "" {
		port = *app.Port
	}
	if portOverride != "" {
		port = portOverride
		log.Printf("Overriding port with: %s", port)
	}
	
	// Start server
	mcpServer := server.NewMCPServer(app)
	return mcpServer.Start(transport, port)
}