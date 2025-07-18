package openapi

import "github.com/T4cceptor/MakeMCP/pkg/config"

// OpenAPIConfig holds OpenAPI-specific parameters
type OpenAPIConfig struct {
	Specs          string `json:"specs"`          // URL to OpenAPI specs
	BaseURL        string `json:"baseURL"`        // Base URL of the API
	StrictValidate bool   `json:"strictValidate"` // Enable strict OpenAPI validation

	config.Config
}

// FromCLIParams extracts OpenAPI-specific parameters from generic CLIParams
func (p *OpenAPIConfig) FromCLIParams(cliParams *config.Config) error {
	p.Config = *cliParams
	if specs, ok := cliParams.CliFlags["specs"].(string); ok {
		p.Specs = specs
	}
	if baseURL, ok := cliParams.CliFlags["base-url"].(string); ok {
		p.BaseURL = baseURL
	}
	if strictValidate, ok := cliParams.CliFlags["strict"].(bool); ok {
		p.StrictValidate = strictValidate
	}
	return nil
}

// SplitParams groups all parameter maps by location for handler logic
type SplitParams struct {
	Path   map[string]any `json:"path"`
	Query  map[string]any `json:"query"`
	Header map[string]any `json:"header"`
	Cookie map[string]any `json:"cookie"`
	Body   map[string]any `json:"body"`
}

// NewSplitParams returns a SplitParams struct with all maps initialized
func NewSplitParams() SplitParams {
	return SplitParams{
		Path:   map[string]any{},
		Query:  map[string]any{},
		Header: map[string]any{},
		Cookie: map[string]any{},
		Body:   map[string]any{},
	}
}

// AttachParams takes paramList and attaches values to the correct SplitParams fields
func (params *SplitParams) AttachParams(paramList []map[string]any) {
	for _, param := range paramList {
		name, _ := param["parameter_name"].(string)
		value := param["parameter_value"]
		location, _ := param["location"].(string)
		switch location {
		case "path":
			params.Path[name] = value
		case "query":
			params.Query[name] = value
		case "header":
			params.Header[name] = value
		case "cookie":
			params.Cookie[name] = value
		case "body":
			params.Body[name] = value
		default:
			params.Query[name] = value // fallback
		}
	}
}

// ToolInputProperty defines a property in the input schema for an MCP tool
type ToolInputProperty struct {
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
	Location    string `json:"location"` // OpenAPI 'in' value: path, query, header, cookie, body, etc.
}
