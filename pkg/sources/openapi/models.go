package openapi

// ParameterLocation defines where a parameter is located in the HTTP request.
type ParameterLocation string

const (
	// ParameterLocationPath represents path parameters in the URL.
	ParameterLocationPath ParameterLocation = "path"
	// ParameterLocationQuery represents query parameters in the URL.
	ParameterLocationQuery ParameterLocation = "query"
	// ParameterLocationHeader represents header parameters in HTTP requests.
	ParameterLocationHeader ParameterLocation = "header"
	// ParameterLocationCookie represents cookie parameters in HTTP requests.
	ParameterLocationCookie ParameterLocation = "cookie"
	// ParameterLocationBody represents body parameters in HTTP requests.
	ParameterLocationBody ParameterLocation = "body"
)

// ParameterLocations are all available parameter locations as a list
var ParameterLocations []ParameterLocation = []ParameterLocation{ParameterLocationPath, ParameterLocationQuery, ParameterLocationHeader, ParameterLocationCookie}

// IsValid returns true if the parameter location is valid.
func (p ParameterLocation) IsValid() bool {
	switch p {
	case ParameterLocationPath, ParameterLocationQuery, ParameterLocationHeader, ParameterLocationCookie, ParameterLocationBody:
		return true
	default:
		return false
	}
}

// ToolParams groups all parameter maps by location for handler logic.
type ToolParams struct {
	Path   map[string]any `json:"path"`
	Query  map[string]any `json:"query"`
	Header map[string]any `json:"header"`
	Cookie map[string]any `json:"cookie"`
	Body   map[string]any `json:"body"`
}

// NewSplitParams returns a SplitParams struct with all maps initialized.
func NewSplitParams() ToolParams {
	return ToolParams{
		Path:   map[string]any{},
		Query:  map[string]any{},
		Header: map[string]any{},
		Cookie: map[string]any{},
		Body:   map[string]any{},
	}
}

// AttachToolParams takes paramList and attaches values to the correct ToolParams fields.
func (params *ToolParams) AttachToolParams(paramList []map[string]any) {
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

// ToolInputProperty defines a property in the input schema for an MCP tool.
type ToolInputProperty struct {
	Type        string            `json:"type"`
	Description string            `json:"description,omitempty"`
	Location    ParameterLocation `json:"location"` // OpenAPI 'in' value: path, query, header, cookie, body, etc.
}
