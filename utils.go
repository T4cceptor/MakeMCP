// Contains generic utils
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

// boolPtr returns a pointer to the given bool value.
// Useful for assigning *bool fields in structs.
func boolPtr(val bool) *bool { return &val }

// NewAPIClient creates a new APIClient with the given baseURL.
func NewAPIClient(baseURL string) *APIClient {
	return &APIClient{
		BaseURL:    baseURL,
		HTTPClient: &http.Client{},
	}
}

// SaveMakeMCPAppToFile serializes the given MakeMCPApp as JSON and writes it to a file in the current directory.
// The filename is derived from the MCP server name (e.g., "myserver.makemcp.json").
func SaveMakeMCPAppToFile(app MakeMCPApp) error {
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
