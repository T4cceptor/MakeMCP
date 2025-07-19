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
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/urfave/cli/v3"
)

func TestGetInternalCommands(t *testing.T) {
	commands := GetInternalCommands()

	// Test that we get the expected number of commands
	if len(commands) != 1 {
		t.Errorf("Expected 1 internal command, got %d", len(commands))
	}

	// Test the load command specifically
	loadCommand := commands[0]

	// Test command structure
	if loadCommand.Name != "load" {
		t.Errorf("Expected command name 'load', got '%s'", loadCommand.Name)
	}

	if loadCommand.Usage == "" {
		t.Error("Expected non-empty usage string")
	}

	if loadCommand.Description == "" {
		t.Error("Expected non-empty description string")
	}

	if loadCommand.ArgsUsage == "" {
		t.Error("Expected non-empty args usage string")
	}

	// Test that action function is assigned
	if loadCommand.Action == nil {
		t.Error("Expected action function to be assigned")
	}

	// Test specific values
	expectedUsage := "Load and start MCP server from existing config file"
	if loadCommand.Usage != expectedUsage {
		t.Errorf("Expected usage '%s', got '%s'", expectedUsage, loadCommand.Usage)
	}

	expectedArgsUsage := "<config-file-path>"
	if loadCommand.ArgsUsage != expectedArgsUsage {
		t.Errorf("Expected args usage '%s', got '%s'", expectedArgsUsage, loadCommand.ArgsUsage)
	}
}

func TestHandleLoadCommand(t *testing.T) {
	// Initialize registries (required for LoadFromFile to work)
	InitializeRegistries()

	// Create a temporary config file for testing
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test_config.json")

	// Create a basic config file with OpenAPI source type
	configContent := `{
		"name": "Test API",
		"version": "1.0.0",
		"sourceType": "openapi",
		"tools": [],
		"config": {
			"transport": "stdio",
			"configOnly": false,
			"port": "8080",
			"devMode": false,
			"type": "openapi",
			"file": "test",
			"flags": {},
			"args": []
		}
	}`

	if err := os.WriteFile(configFile, []byte(configContent), 0o644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	t.Run("Missing arguments", func(t *testing.T) {
		// Create a minimal CLI app to simulate the command execution
		app := &cli.Command{
			Name: "makemcp",
			Commands: []*cli.Command{
				{
					Name:   "load",
					Action: handleLoadCommand,
				},
			},
		}

		ctx := context.Background()

		// Run the app with no arguments (should trigger missing arguments error)
		err := app.Run(ctx, []string{"makemcp", "load"})

		if err == nil {
			t.Error("Expected error for missing arguments")
		}

		expectedError := "load command requires exactly one argument: the path to the config file"
		if err.Error() != expectedError {
			t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
		}
	})

	t.Run("Too many arguments", func(t *testing.T) {
		app := &cli.Command{
			Name: "makemcp",
			Commands: []*cli.Command{
				{
					Name:   "load",
					Action: handleLoadCommand,
				},
			},
		}

		ctx := context.Background()

		// Run with too many arguments
		err := app.Run(ctx, []string{"makemcp", "load", "file1.json", "file2.json"})

		if err == nil {
			t.Error("Expected error for too many arguments")
		}

		expectedError := "load command requires exactly one argument: the path to the config file"
		if err.Error() != expectedError {
			t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
		}
	})

	t.Run("Nonexistent config file", func(t *testing.T) {
		nonexistentFile := filepath.Join(tempDir, "nonexistent.json")

		app := &cli.Command{
			Name: "makemcp",
			Commands: []*cli.Command{
				{
					Name:   "load",
					Action: handleLoadCommand,
				},
			},
		}

		ctx := context.Background()

		// Run with nonexistent file
		err := app.Run(ctx, []string{"makemcp", "load", nonexistentFile})

		if err == nil {
			t.Error("Expected error for nonexistent file")
		}

		// Check that the error mentions the file loading failure
		if !strings.Contains(err.Error(), "failed to load configuration") {
			t.Errorf("Expected error to mention configuration loading failure, got: %s", err.Error())
		}
	})

	t.Run("Invalid config file", func(t *testing.T) {
		invalidConfigFile := filepath.Join(tempDir, "invalid_config.json")
		invalidContent := `{"invalid": json}`

		if err := os.WriteFile(invalidConfigFile, []byte(invalidContent), 0o644); err != nil {
			t.Fatalf("Failed to create invalid config file: %v", err)
		}

		app := &cli.Command{
			Name: "makemcp",
			Commands: []*cli.Command{
				{
					Name:   "load",
					Action: handleLoadCommand,
				},
			},
		}

		ctx := context.Background()

		// Run with invalid config file
		err := app.Run(ctx, []string{"makemcp", "load", invalidConfigFile})

		if err == nil {
			t.Error("Expected error for invalid config file")
		}

		// Check that the error mentions the file loading failure
		if !strings.Contains(err.Error(), "failed to load configuration") {
			t.Errorf("Expected error to mention configuration loading failure, got: %s", err.Error())
		}
	})

	// Note: We skip testing the valid config file case because it would try to start
	// an actual MCP server, which is not suitable for unit tests. In a real implementation,
	// we would need to inject dependencies or make StartServer mockable.
}
