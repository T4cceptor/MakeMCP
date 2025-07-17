package openapi

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
