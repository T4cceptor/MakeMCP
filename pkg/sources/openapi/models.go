package openapi

import "github.com/T4cceptor/MakeMCP/pkg/config"

// ParameterLocation defines where a parameter is located in the HTTP request
type ParameterLocation string

const (
	ParameterLocationPath   ParameterLocation = "path"
	ParameterLocationQuery  ParameterLocation = "query"
	ParameterLocationHeader ParameterLocation = "header"
	ParameterLocationCookie ParameterLocation = "cookie"
	ParameterLocationBody   ParameterLocation = "body"
)

// IsValid returns true if the parameter location is valid
func (p ParameterLocation) IsValid() bool {
	switch p {
	case ParameterLocationPath, ParameterLocationQuery, ParameterLocationHeader, ParameterLocationCookie, ParameterLocationBody:
		return true
	default:
		return false
	}
}

// OpenAPIConfig holds OpenAPI-specific parameters
type OpenAPIConfig struct {
	Specs          string `json:"specs"`          // URL to OpenAPI specs
	BaseURL        string `json:"baseURL"`        // Base URL of the API
	StrictValidate bool   `json:"strictValidate"` // Enable strict OpenAPI validation
	Timeout        int    `json:"timeout"`        // HTTP timeout in seconds

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
	if timeout, ok := cliParams.CliFlags["timeout"].(int); ok {
		p.Timeout = timeout
	} else {
		p.Timeout = 30 // default timeout
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
		paramLocation := ParameterLocation(location)
		if !paramLocation.IsValid() {
			// Invalid parameter location, default to query
			paramLocation = ParameterLocationQuery
		}
		
		switch paramLocation {
		case ParameterLocationPath:
			params.Path[name] = value
		case ParameterLocationQuery:
			params.Query[name] = value
		case ParameterLocationHeader:
			params.Header[name] = value
		case ParameterLocationCookie:
			params.Cookie[name] = value
		case ParameterLocationBody:
			params.Body[name] = value
		}
	}
}

// ToolInputProperty defines a property in the input schema for an MCP tool
type ToolInputProperty struct {
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
	Location    ParameterLocation `json:"location"` // OpenAPI 'in' value: path, query, header, cookie, body, etc.
}
