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
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	core "github.com/T4cceptor/MakeMCP/pkg/core"
)

// OpenAPIParams holds OpenAPI-specific parameters with type safety.
type OpenAPIParams struct {
	*core.SharedParams

	// OpenAPI-specific fields
	Specs          string `json:"specs"`          // URL or file path to OpenAPI specification
	BaseURL        string `json:"baseURL"`        // Base URL of the API for tool calls
	Timeout        int    `json:"timeout"`        // HTTP timeout in seconds
	StrictValidate bool   `json:"strictValidate"` // Enable strict OpenAPI validation
}

// NewOpenAPIParams creates a new OpenAPIParams with default values.
func NewOpenAPIParams(sharedParams *core.SharedParams) *OpenAPIParams {
	return &OpenAPIParams{
		SharedParams:   sharedParams,
		Specs:          "",
		BaseURL:        "",
		Timeout:        30, // default 30 seconds
		StrictValidate: false,
	}
}

// GetSharedParams returns the shared parameters.
func (p *OpenAPIParams) GetSharedParams() *core.SharedParams {
	return p.SharedParams
}

// GetSourceType returns the source type identifier.
func (p *OpenAPIParams) GetSourceType() string {
	return "openapi"
}

// Validate performs OpenAPI-specific parameter validation.
func (p *OpenAPIParams) Validate() error {
	// Validate required fields
	if p.Specs == "" {
		return errors.New("specs parameter is required - must specify OpenAPI specification URL or file path")
	}

	if p.BaseURL == "" {
		return errors.New("base-url parameter is required - must specify the API base URL for tool execution")
	}

	// Validate URLs if they look like URLs (contain protocol)
	if strings.Contains(p.Specs, "://") {
		if _, err := url.Parse(p.Specs); err != nil {
			return fmt.Errorf("invalid specs URL: %w", err)
		}
	}

	if strings.Contains(p.BaseURL, "://") {
		if _, err := url.Parse(p.BaseURL); err != nil {
			return fmt.Errorf("invalid base URL: %w", err)
		}
	}

	// Validate timeout
	if p.Timeout <= 0 {
		return errors.New("timeout must be greater than 0")
	}

	return nil
}

// ToJSON returns a JSON representation for logging and debugging.
func (p *OpenAPIParams) ToJSON() string {
	jsonBytes, err := json.Marshal(p)
	if err != nil {
		return `{"error": "failed to marshal OpenAPIParams to JSON"}`
	}
	return string(jsonBytes)
}

// ParseFromCLIInput creates OpenAPIParams from raw CLI input with type safety.
func ParseFromCLIInput(input *core.CLIParamsInput) (*OpenAPIParams, error) {
	params := NewOpenAPIParams(input.SharedParams)

	// Extract and validate specs parameter
	if specs, ok := input.CliFlags["specs"].(string); ok && specs != "" {
		params.Specs = specs
	} else {
		return nil, errors.New("specs flag is required")
	}

	// Extract and validate base-url parameter
	if baseURL, ok := input.CliFlags["base-url"].(string); ok && baseURL != "" {
		params.BaseURL = baseURL
	} else {
		return nil, errors.New("base-url flag is required")
	}

	// Extract optional timeout parameter
	if timeout, ok := input.CliFlags["timeout"]; ok {
		switch t := timeout.(type) {
		case int:
			if t > 0 {
				params.Timeout = t
			}
		case string:
			// Handle string to int conversion if needed
			if t != "" {
				return nil, fmt.Errorf("timeout must be an integer, got string: %s", t)
			}
		}
	}

	// Extract optional strict parameter
	if strict, ok := input.CliFlags["strict"].(bool); ok {
		params.StrictValidate = strict
	}

	// Validate the constructed parameters
	if err := params.Validate(); err != nil {
		return nil, fmt.Errorf("parameter validation failed: %w", err)
	}

	return params, nil
}
