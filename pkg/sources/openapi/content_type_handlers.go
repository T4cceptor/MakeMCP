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
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/url"
	"strings"

	"github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
)

// ContentTypeHandler defines the interface for handling different content types
type ContentTypeHandler interface {
	// GetContentTypes returns the content types this handler supports
	GetContentTypes() []string
	
	// ExtractParameters extracts tool input properties from an OpenAPI media type schema
	// Returns: properties map, required fields list, and error
	ExtractParameters(media *v3.MediaType) (map[string]ToolInputProperty, []string, error)
	
	// BuildRequestBody builds the HTTP request body from parsed parameters
	// Returns: request body reader and error
	BuildRequestBody(bodyParams map[string]any) (io.Reader, error)
	
	// GetParameterPrefix returns the prefix to use for parameters of this content type
	// This helps distinguish between different content types in the final tool schema
	GetParameterPrefix() string
}

// ContentTypeRegistry manages content type handlers
type ContentTypeRegistry struct {
	handlers map[string]ContentTypeHandler
	fallback ContentTypeHandler
}

// NewContentTypeRegistry creates a new registry with default handlers
func NewContentTypeRegistry() *ContentTypeRegistry {
	registry := &ContentTypeRegistry{
		handlers: make(map[string]ContentTypeHandler),
	}
	
	// Register default handlers
	registry.RegisterHandler(&JSONContentTypeHandler{})
	registry.RegisterHandler(&XMLContentTypeHandler{})
	registry.RegisterHandler(&FormURLEncodedHandler{})
	registry.RegisterHandler(&MultipartFormDataHandler{})
	registry.RegisterHandler(&PlainTextHandler{})
	
	// Set JSON as fallback for unknown content types
	registry.fallback = &JSONContentTypeHandler{}
	
	return registry
}

// RegisterHandler registers a content type handler
func (r *ContentTypeRegistry) RegisterHandler(handler ContentTypeHandler) {
	for _, contentType := range handler.GetContentTypes() {
		r.handlers[contentType] = handler
	}
}

// GetHandler returns the appropriate handler for a content type
func (r *ContentTypeRegistry) GetHandler(contentType string) ContentTypeHandler {
	if handler, exists := r.handlers[contentType]; exists {
		return handler
	}
	
	// Try wildcard patterns
	parts := strings.Split(contentType, "/")
	if len(parts) == 2 {
		wildcard := parts[0] + "/*"
		if handler, exists := r.handlers[wildcard]; exists {
			return handler
		}
	}
	
	// Return fallback handler
	return r.fallback
}

// JSONContentTypeHandler handles JSON content types
type JSONContentTypeHandler struct{}

func (h *JSONContentTypeHandler) GetContentTypes() []string {
	return []string{"application/json", "*/*", "application/hal+json", "application/vnd.api+json"}
}

func (h *JSONContentTypeHandler) ExtractParameters(media *v3.MediaType) (map[string]ToolInputProperty, []string, error) {
	properties := make(map[string]ToolInputProperty)
	var required []string

	if media.Schema == nil {
		return properties, required, nil
	}

	schema := media.Schema.Schema()
	if schema == nil || schema.Properties == nil || schema.Properties.Len() == 0 {
		return properties, required, nil
	}

	// Extract individual JSON properties
	for propPairs := schema.Properties.First(); propPairs != nil; propPairs = propPairs.Next() {
		propName := propPairs.Key()
		propSchemaProxy := propPairs.Value()
		propSchema := propSchemaProxy.Schema()
		
		properties[propName] = ToolInputProperty{
			Type:        getSchemaTypeString(propSchemaProxy),
			Description: propSchema.Description,
			Location:    "body",
		}
	}
	
	if schema.Required != nil {
		required = append(required, schema.Required...)
	}
	
	return properties, required, nil
}

func (h *JSONContentTypeHandler) BuildRequestBody(bodyParams map[string]any) (io.Reader, error) {
	if len(bodyParams) == 0 {
		return nil, nil
	}
	
	jsonBody, err := json.Marshal(bodyParams)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON body: %w", err)
	}
	return bytes.NewReader(jsonBody), nil
}

func (h *JSONContentTypeHandler) GetParameterPrefix() string {
	return "" // No prefix for JSON parameters
}

// XMLContentTypeHandler handles XML content types (both structured and raw)
type XMLContentTypeHandler struct{}

func (h *XMLContentTypeHandler) GetContentTypes() []string {
	return []string{"application/xml", "text/xml"}
}

func (h *XMLContentTypeHandler) ExtractParameters(media *v3.MediaType) (map[string]ToolInputProperty, []string, error) {
	properties := make(map[string]ToolInputProperty)
	var required []string

	if media.Schema == nil {
		return properties, required, nil
	}

	schema := media.Schema.Schema()
	if schema == nil {
		return properties, required, nil
	}

	// Check if this is structured XML (has properties)
	if schema.Properties != nil && schema.Properties.Len() > 0 {
		// Structured XML - extract individual properties
		for propPairs := schema.Properties.First(); propPairs != nil; propPairs = propPairs.Next() {
			propName := propPairs.Key()
			propSchemaProxy := propPairs.Value()
			propSchema := propSchemaProxy.Schema()
			
			properties[propName] = ToolInputProperty{
				Type:        getSchemaTypeString(propSchemaProxy),
				Description: propSchema.Description,
				Location:    "body",
			}
		}
		
		if schema.Required != nil {
			required = append(required, schema.Required...)
		}
	} else {
		// Raw XML - single body parameter
		properties["body"] = ToolInputProperty{
			Type:        "string",
			Description: "XML request body content",
			Location:    "body",
		}
		required = append(required, "body")
	}
	
	return properties, required, nil
}

func (h *XMLContentTypeHandler) BuildRequestBody(bodyParams map[string]any) (io.Reader, error) {
	if len(bodyParams) == 0 {
		return nil, nil
	}
	
	// Check if this is raw XML (single "body" parameter)  
	if bodyContent, exists := bodyParams["body"]; exists && len(bodyParams) == 1 {
		if bodyStr, ok := bodyContent.(string); ok {
			return strings.NewReader(bodyStr), nil
		}
		return nil, fmt.Errorf("XML body parameter must be a string containing valid XML")
	}
	
	// TODO: For structured XML, we should convert the object to actual XML
	// For now, fall back to JSON serialization (same as current behavior)
	// In the future, we could use xml.Marshal or a dedicated XML library
	jsonBody, err := json.Marshal(bodyParams)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal XML body (using JSON fallback): %w", err)
	}
	return bytes.NewReader(jsonBody), nil
}

func (h *XMLContentTypeHandler) GetParameterPrefix() string {
	return "" // No prefix for XML parameters
}

// FormURLEncodedHandler handles application/x-www-form-urlencoded
type FormURLEncodedHandler struct{}

func (h *FormURLEncodedHandler) GetContentTypes() []string {
	return []string{"application/x-www-form-urlencoded"}
}

func (h *FormURLEncodedHandler) ExtractParameters(media *v3.MediaType) (map[string]ToolInputProperty, []string, error) {
	properties := make(map[string]ToolInputProperty)
	var required []string

	if media.Schema == nil {
		return properties, required, nil
	}

	schema := media.Schema.Schema()
	if schema == nil || schema.Properties == nil || schema.Properties.Len() == 0 {
		// No structured schema - fall back to single body parameter
		properties["body"] = ToolInputProperty{
			Type:        "string",
			Description: "Form URL-encoded request body",
			Location:    "body",
		}
		required = append(required, "body")
		return properties, required, nil
	}

	// Extract form fields with prefix
	for propPairs := schema.Properties.First(); propPairs != nil; propPairs = propPairs.Next() {
		propName := propPairs.Key()
		propSchemaProxy := propPairs.Value()
		propSchema := propSchemaProxy.Schema()
		
		prefixedName := fmt.Sprintf("form__%s", propName)
		properties[prefixedName] = ToolInputProperty{
			Type:        getSchemaTypeString(propSchemaProxy),
			Description: propSchema.Description,
			Location:    "body",
		}
	}
	
	if schema.Required != nil {
		for _, req := range schema.Required {
			required = append(required, fmt.Sprintf("form__%s", req))
		}
	}
	
	return properties, required, nil
}

func (h *FormURLEncodedHandler) BuildRequestBody(bodyParams map[string]any) (io.Reader, error) {
	if len(bodyParams) == 0 {
		return nil, nil
	}
	
	// Check for single body parameter (raw form data)
	if bodyContent, exists := bodyParams["body"]; exists && len(bodyParams) == 1 {
		if bodyStr, ok := bodyContent.(string); ok {
			return strings.NewReader(bodyStr), nil
		}
		return nil, fmt.Errorf("form body parameter must be a string")
	}
	
	// Build URL encoded form data from form__ prefixed parameters
	formData := url.Values{}
	for paramName, value := range bodyParams {
		if strings.HasPrefix(paramName, "form__") {
			fieldName := strings.TrimPrefix(paramName, "form__")
			formData.Set(fieldName, fmt.Sprintf("%v", value))
		}
	}
	
	if len(formData) == 0 {
		return nil, fmt.Errorf("no form__ prefixed parameters found for form URL encoding")
	}
	
	encodedData := formData.Encode()
	return strings.NewReader(encodedData), nil
}

func (h *FormURLEncodedHandler) GetParameterPrefix() string {
	return "form__"
}

// MultipartFormDataHandler handles multipart/form-data
type MultipartFormDataHandler struct{}

func (h *MultipartFormDataHandler) GetContentTypes() []string {
	return []string{"multipart/form-data"}
}

func (h *MultipartFormDataHandler) ExtractParameters(media *v3.MediaType) (map[string]ToolInputProperty, []string, error) {
	properties := make(map[string]ToolInputProperty)
	var required []string

	if media.Schema == nil {
		return properties, required, nil
	}

	schema := media.Schema.Schema()
	if schema == nil || schema.Properties == nil || schema.Properties.Len() == 0 {
		// No structured schema - fall back to single body parameter
		properties["body"] = ToolInputProperty{
			Type:        "string",
			Description: "Multipart form data request body",
			Location:    "body",
		}
		required = append(required, "body")
		return properties, required, nil
	}

	// Extract multipart fields with prefix and file detection
	for propPairs := schema.Properties.First(); propPairs != nil; propPairs = propPairs.Next() {
		propName := propPairs.Key()
		propSchemaProxy := propPairs.Value()
		propSchema := propSchemaProxy.Schema()
		
		prefixedName := fmt.Sprintf("multipart__%s", propName)
		
		// Detect file uploads (binary format)
		propType := getSchemaTypeString(propSchemaProxy)
		if propSchema != nil && propSchema.Format == "binary" {
			propType = "file"
		}
		
		properties[prefixedName] = ToolInputProperty{
			Type:        propType,
			Description: propSchema.Description,
			Location:    "body",
		}
	}
	
	if schema.Required != nil {
		for _, req := range schema.Required {
			required = append(required, fmt.Sprintf("multipart__%s", req))
		}
	}
	
	return properties, required, nil
}

func (h *MultipartFormDataHandler) BuildRequestBody(bodyParams map[string]any) (io.Reader, error) {
	if len(bodyParams) == 0 {
		return nil, nil
	}
	
	// Check for single body parameter (raw multipart data)
	if bodyContent, exists := bodyParams["body"]; exists && len(bodyParams) == 1 {
		if bodyStr, ok := bodyContent.(string); ok {
			return strings.NewReader(bodyStr), nil
		}
		return nil, fmt.Errorf("multipart body parameter must be a string")
	}
	
	// Build multipart form data from multipart__ prefixed parameters
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	
	hasMultipartParams := false
	for paramName, value := range bodyParams {
		if strings.HasPrefix(paramName, "multipart__") {
			hasMultipartParams = true
			fieldName := strings.TrimPrefix(paramName, "multipart__")
			
			// TODO: Handle file uploads properly
			// For now, treat everything as form fields
			if err := writer.WriteField(fieldName, fmt.Sprintf("%v", value)); err != nil {
				return nil, fmt.Errorf("failed to write multipart field %s: %w", fieldName, err)
			}
		}
	}
	
	if !hasMultipartParams {
		return nil, fmt.Errorf("no multipart__ prefixed parameters found for multipart form data")
	}
	
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}
	
	// Note: The boundary should be set in the Content-Type header by the caller
	// It can be retrieved via writer.Boundary()
	
	return &body, nil
}

func (h *MultipartFormDataHandler) GetParameterPrefix() string {
	return "multipart__"
}

// PlainTextHandler handles text/plain and text/* content types
type PlainTextHandler struct{}

func (h *PlainTextHandler) GetContentTypes() []string {
	return []string{"text/plain", "text/*"}
}

func (h *PlainTextHandler) ExtractParameters(media *v3.MediaType) (map[string]ToolInputProperty, []string, error) {
	properties := make(map[string]ToolInputProperty)
	var required []string

	// Plain text always uses single body parameter
	properties["body"] = ToolInputProperty{
		Type:        "string",
		Description: "Plain text request body content",
		Location:    "body",
	}
	required = append(required, "body")
	
	return properties, required, nil
}

func (h *PlainTextHandler) BuildRequestBody(bodyParams map[string]any) (io.Reader, error) {
	if len(bodyParams) == 0 {
		return nil, nil
	}
	
	if bodyContent, exists := bodyParams["body"]; exists {
		if bodyStr, ok := bodyContent.(string); ok {
			return strings.NewReader(bodyStr), nil
		}
		return nil, fmt.Errorf("plain text body parameter must be a string")
	}
	
	return nil, fmt.Errorf("plain text content type requires a 'body' parameter")
}

func (h *PlainTextHandler) GetParameterPrefix() string {
	return "" // No prefix for plain text
}

// Helper function to get schema type string (this should use the existing implementation)
func getSchemaTypeString(schemaProxy *base.SchemaProxy) string {
	// TODO: This should use the existing getSchemaTypeString method from LibopenAPIAdapter
	// For now, provide a basic implementation
	if schemaProxy == nil {
		return "string"
	}
	
	schema := schemaProxy.Schema()
	if schema == nil {
		return "string"
	}
	
	if len(schema.Type) > 0 {
		return schema.Type[0]
	}
	
	return "string"
}