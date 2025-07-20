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

package openapi_integration

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	//"github.com/T4cceptor/MakeMCP/pkg/core"
	//"github.com/T4cceptor/MakeMCP/pkg/sources/openapi"
)

// Test data structure for OpenAPI integration tests
type testCase struct {
	name               string
	specFile           string
	baseURL            string
	expectedResultFile string // Path to expected JSON result file
	expectedSource     string
}

// getTestCases returns the test cases for OpenAPI integration tests
func getTestCases() []testCase {
	return []testCase{
		{
			name:               "FastAPI",
			specFile:           "sample_specifications/fastapi.json",
			baseURL:            "http://localhost:8081",
			expectedResultFile: "expected_result/fastapi_makemcp.json",
			expectedSource:     "openapi",
		},
		{
			name:               "GoFuego",
			specFile:           "sample_specifications/gofuego.json",
			baseURL:            "http://localhost:8120",
			expectedResultFile: "expected_result/gofuego_makemcp.json",
			expectedSource:     "openapi",
		},
		{
			name:               "Salesforce",
			specFile:           "sample_specifications/salesforce_1.json",
			baseURL:            "https://api.salesforce.com/einstein/platform/v1",
			expectedResultFile: "expected_result/salesforce_makemcp.json",
			expectedSource:     "openapi",
		},
		{
			name:               "AdobeAEM",
			specFile:           "sample_specifications/adobe_aem_3_7_1.json",
			baseURL:            "http://adobe.local",
			expectedResultFile: "expected_result/adobe_aem_makemcp.json",
			expectedSource:     "openapi",
		},
	}
}

// buildMakeMCPBinary builds the makemcp binary if it doesn't exist
func buildMakeMCPBinary(t *testing.T) string {
	t.Helper()

	// Get the project root (two levels up from testbed/openapi)
	projectRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("Failed to get project root: %v", err)
	}

	binaryPath := filepath.Join(projectRoot, "build", "makemcp")

	// Check if binary already exists
	if _, err := os.Stat(binaryPath); err == nil {
		return binaryPath
	}

	// Build the binary
	t.Logf("Building makemcp binary...")
	cmd := exec.Command("make", "build")
	cmd.Dir = projectRoot
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build makemcp binary: %v\nOutput: %s", err, output)
	}

	// Verify the binary was created
	if _, err := os.Stat(binaryPath); err != nil {
		t.Fatalf("Binary not found after build: %v", err)
	}

	return binaryPath
}

// runMakeMCP executes the makemcp CLI with the given arguments and returns the output
func runMakeMCP(t *testing.T, binaryPath string, args ...string) ([]byte, error) {
	t.Helper()

	cmd := exec.Command(binaryPath, args...)
	cmd.Dir = filepath.Dir(binaryPath) // Run from build directory

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("makemcp command failed: %v\nArgs: %v\nOutput: %s", err, args, output)
	}

	return output, err
}

// TestOpenAPIIntegration tests the full integration of OpenAPI specs to makemcp configs
func TestOpenAPIIntegration(t *testing.T) {
	// Build the binary once for all tests
	binaryPath := buildMakeMCPBinary(t)

	testCases := getTestCases()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a temporary directory for this test
			tempDir, err := os.MkdirTemp("", fmt.Sprintf("makemcp_test_%s_", strings.ToLower(tc.name)))
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			// Prepare paths
			specPath, err := filepath.Abs(tc.specFile)
			if err != nil {
				t.Fatalf("Failed to get absolute path for spec file: %v", err)
			}

			outputPath := filepath.Join(tempDir, "makemcp.json")

			// Run makemcp command to generate config
			// Use the full path for the --file flag (without .json extension)
			outputFileBase := strings.TrimSuffix(outputPath, ".json")

			args := []string{
				"openapi",
				"--specs", specPath,
				"--base-url", tc.baseURL,
				"--config-only", "true",
				"--file", outputFileBase,
			}

			output, err := runMakeMCP(t, binaryPath, args...)
			if err != nil {
				t.Fatalf("makemcp command failed: %v\nOutput: %s", err, output)
			}

			// Verify the config file was created
			if _, err := os.Stat(outputPath); err != nil {
				t.Fatalf("Config file not created: %v", err)
			}

			// Compare generated config with expected result
			actualData, err := os.ReadFile(outputPath)
			if err != nil {
				t.Fatalf("Failed to read generated config: %v", err)
			}

			expectedData, err := os.ReadFile(tc.expectedResultFile)
			if err != nil {
				t.Fatalf("Failed to read expected result: %v", err)
			}

			var actual, expected map[string]any
			if err := json.Unmarshal(actualData, &actual); err != nil {
				t.Fatalf("Failed to parse actual JSON: %v", err)
			}
			if err := json.Unmarshal(expectedData, &expected); err != nil {
				t.Fatalf("Failed to parse expected JSON: %v", err)
			}

			// Compare key fields
			compareField := func(field string) {
				if actual[field] != expected[field] {
					t.Errorf("%s mismatch: expected %v, got %v", field, expected[field], actual[field])
				}
			}
			
			compareField("name")
			compareField("version")
			compareField("sourceType")

			// Compare tools by name (order may vary)
			actualTools, expectedTools := actual["tools"].([]any), expected["tools"].([]any)
			if len(actualTools) != len(expectedTools) {
				t.Errorf("Tools count mismatch: expected %d, got %d", len(expectedTools), len(actualTools))
			}

			getToolNames := func(tools []any) map[string]bool {
				names := make(map[string]bool)
				for _, tool := range tools {
					if name, ok := tool.(map[string]any)["name"].(string); ok {
						names[name] = true
					}
				}
				return names
			}

			actualNames, expectedNames := getToolNames(actualTools), getToolNames(expectedTools)
			for name := range expectedNames {
				if !actualNames[name] {
					t.Errorf("Missing expected tool: %s", name)
				}
			}
			for name := range actualNames {
				if !expectedNames[name] {
					t.Errorf("Unexpected tool found: %s", name)
				}
			}

			t.Logf("✅ %s: %d tools match expected results", tc.name, len(actualTools))
		})
	}
}

// TestOpenAPISpecValidation tests that the sample OpenAPI specs are valid
func TestOpenAPISpecValidation(t *testing.T) {
	testCases := getTestCases()

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("ValidateSpec_%s", tc.name), func(t *testing.T) {
			specPath, err := filepath.Abs(tc.specFile)
			if err != nil {
				t.Fatalf("Failed to get absolute path for spec file: %v", err)
			}

			// Check if spec file exists
			if _, err := os.Stat(specPath); err != nil {
				t.Fatalf("Spec file does not exist: %v", err)
			}

			// Read and parse the spec file
			specData, err := os.ReadFile(specPath)
			if err != nil {
				t.Fatalf("Failed to read spec file: %v", err)
			}

			var spec map[string]any
			if err := json.Unmarshal(specData, &spec); err != nil {
				t.Fatalf("Failed to parse spec JSON: %v", err)
			}

			// Validate basic OpenAPI structure
			if openapi, exists := spec["openapi"]; !exists {
				t.Error("OpenAPI version field missing")
			} else if openapiStr, ok := openapi.(string); ok {
				if !strings.HasPrefix(openapiStr, "3.") {
					t.Errorf("Expected OpenAPI 3.x, got %s", openapiStr)
				}
			}

			if _, exists := spec["info"]; !exists {
				t.Error("Info section missing")
			}

			if _, exists := spec["paths"]; !exists {
				t.Error("Paths section missing")
			}

			t.Logf("Successfully validated OpenAPI spec: %s", tc.name)
		})
	}
}

// TestConfigOutputDirectory tests that configs are properly written to the results directory
func TestConfigOutputDirectory(t *testing.T) {
	binaryPath := buildMakeMCPBinary(t)

	// Test with the results directory
	resultsDir, err := filepath.Abs("results")
	if err != nil {
		t.Fatalf("Failed to get absolute path for results directory: %v", err)
	}

	// Ensure results directory exists
	if err := os.MkdirAll(resultsDir, 0o755); err != nil {
		t.Fatalf("Failed to create results directory: %v", err)
	}

	testCases := getTestCases()

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("ResultsDir_%s", tc.name), func(t *testing.T) {
			specPath, err := filepath.Abs(tc.specFile)
			if err != nil {
				t.Fatalf("Failed to get absolute path for spec file: %v", err)
			}

			outputFileName := fmt.Sprintf("%s_makemcp.json", strings.ToLower(tc.name))
			outputPath := filepath.Join(resultsDir, outputFileName)

			// Clean up any existing file
			os.Remove(outputPath)

			// Run makemcp command
			// Use the full path for the --file flag (without .json extension)
			outputFileBase := strings.TrimSuffix(outputPath, ".json")

			args := []string{
				"openapi",
				"--specs", specPath,
				"--base-url", tc.baseURL,
				"--config-only", "true",
				"--file", outputFileBase,
			}

			output, err := runMakeMCP(t, binaryPath, args...)
			if err != nil {
				t.Fatalf("makemcp command failed: %v\nOutput: %s", err, output)
			}

			// Verify the config file was created in results directory
			if _, err := os.Stat(outputPath); err != nil {
				t.Fatalf("Config file not created in results directory: %v", err)
			}

			t.Logf("Successfully created config in results directory: %s", outputPath)
		})
	}
}
