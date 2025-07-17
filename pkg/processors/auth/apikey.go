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

package auth

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/T4cceptor/MakeMCP/pkg/config"
	"github.com/T4cceptor/MakeMCP/pkg/processors"
)

// APIKeyProcessor adds API key authentication to requests
type APIKeyProcessor struct{}

// Name returns the processor name
func (p *APIKeyProcessor) Name() string {
	return "apikey"
}

// Stage returns the processing stage
func (p *APIKeyProcessor) Stage() config.ProcessorStage {
	return config.StagePreRequest
}

// Process adds API key to the request
func (p *APIKeyProcessor) Process(ctx context.Context, data *processors.ProcessorData) error {
	if data.HTTPRequest == nil {
		return fmt.Errorf("HTTP request is required for API key processor")
	}

	// Get processor configuration
	cfg, ok := data.Metadata["processor_config"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid processor configuration")
	}

	// Get API key value
	apiKey, err := p.getAPIKey(cfg)
	if err != nil {
		return fmt.Errorf("failed to get API key: %w", err)
	}

	// Get location (header, query, or cookie)
	location, ok := cfg["location"].(string)
	if !ok {
		location = "header" // default to header
	}

	// Get parameter name
	paramName, ok := cfg["param_name"].(string)
	if !ok {
		paramName = "X-API-Key" // default header name
	}

	// Add API key to request
	switch location {
	case "header":
		data.HTTPRequest.Header.Set(paramName, apiKey)
	case "query":
		q := data.HTTPRequest.URL.Query()
		q.Set(paramName, apiKey)
		data.HTTPRequest.URL.RawQuery = q.Encode()
	case "cookie":
		data.HTTPRequest.AddCookie(&http.Cookie{
			Name:  paramName,
			Value: apiKey,
		})
	default:
		return fmt.Errorf("unsupported location: %s", location)
	}

	return nil
}

// GetDefaultConfig returns default configuration
func (p *APIKeyProcessor) GetDefaultConfig() map[string]interface{} {
	return map[string]interface{}{
		"location":   "header",
		"param_name": "X-API-Key",
		"value":      "${API_KEY}",
	}
}

// ValidateConfig validates processor configuration
func (p *APIKeyProcessor) ValidateConfig(cfg map[string]interface{}) error {
	// Check required fields
	if _, ok := cfg["value"]; !ok {
		return fmt.Errorf("'value' is required")
	}

	// Validate location if provided
	if location, ok := cfg["location"].(string); ok {
		if location != "header" && location != "query" && location != "cookie" {
			return fmt.Errorf("location must be 'header', 'query', or 'cookie'")
		}
	}

	return nil
}

// getAPIKey retrieves the API key from configuration
func (p *APIKeyProcessor) getAPIKey(cfg map[string]interface{}) (string, error) {
	value, ok := cfg["value"].(string)
	if !ok {
		return "", fmt.Errorf("API key value must be a string")
	}

	// Handle environment variable substitution
	if strings.HasPrefix(value, "${") && strings.HasSuffix(value, "}") {
		envVar := strings.TrimPrefix(strings.TrimSuffix(value, "}"), "${")
		envValue := os.Getenv(envVar)
		if envValue == "" {
			return "", fmt.Errorf("environment variable %s is not set", envVar)
		}
		return envValue, nil
	}

	return value, nil
}