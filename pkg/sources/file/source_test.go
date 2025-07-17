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
	"testing"

	"github.com/T4cceptor/MakeMCP/pkg/config"
)

func TestFileSource_Name(t *testing.T) {
	source := &FileSource{}
	
	if source.Name() != "file" {
		t.Errorf("Expected name 'file', got %s", source.Name())
	}
}

func TestFileSource_GetDefaultConfig(t *testing.T) {
	source := &FileSource{}
	
	config := source.GetDefaultConfig()
	
	if config["path"] != "" {
		t.Errorf("Expected empty path, got %v", config["path"])
	}
}

func TestFileSource_Validate(t *testing.T) {
	source := &FileSource{}
	
	t.Run("Invalid file path", func(t *testing.T) {
		err := source.Validate("nonexistent.json")
		if err == nil {
			t.Error("Expected error for nonexistent file")
		}
	})
	
	// Note: We can't easily test valid file path without creating a test file
	// This would require setting up test fixtures
}

func TestFileSource_Parse(t *testing.T) {
	source := &FileSource{}
	baseConfig := config.MakeMCPApp{
		Name:      "Test App",
		Version:   "1.0.0",
		Transport: "stdio",
	}
	
	t.Run("Invalid file path", func(t *testing.T) {
		_, err := source.Parse("nonexistent.json", baseConfig)
		if err == nil {
			t.Error("Expected error for nonexistent file")
		}
	})
	
	// Note: We can't easily test valid file path without creating a test file
	// This would require setting up test fixtures
}

func TestFileSource_GetCommand(t *testing.T) {
	source := &FileSource{}
	
	command := source.GetCommand()
	
	if command.Name != "file" {
		t.Errorf("Expected command name 'file', got %s", command.Name)
	}
	
	if command.Usage == "" {
		t.Error("Expected non-empty usage")
	}
	
	if command.ArgsUsage != "<config-file>" {
		t.Errorf("Expected ArgsUsage '<config-file>', got %s", command.ArgsUsage)
	}
	
	// Check that required flags are present
	transportFlag := false
	portFlag := false
	
	for _, flag := range command.Flags {
		switch flag.Names()[0] {
		case "transport":
			transportFlag = true
		case "port":
			portFlag = true
		}
	}
	
	if !transportFlag {
		t.Error("Expected transport flag")
	}
	
	if !portFlag {
		t.Error("Expected port flag")
	}
}