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
	"log"
	"mime/multipart"
	"net/url"
	"strings"

	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
)

// Common helper functions for parameter extraction

// extractSchemaProperties extracts properties from an OpenAPI schema
// Returns properties map, required fields list, and error
func extractSchemaProperties(media *v3.MediaType, prefix string) (map[string]ToolInputProperty, []string, error) {
	properties := make(map[string]ToolInputProperty)
	var required []string

	if !hasSchemaProps(media) {
		return properties, required, nil
	}
	schema := media.Schema.Schema()
	// Extract individual properties
	for propName, propSchemaProxy := range schema.Properties.FromNewest() {
		// Apply prefix if specified
		paramName := propName
		if prefix != "" {
			paramName = fmt.Sprintf("%s__%s", prefix, propName)
		}

		propSchema := propSchemaProxy.Schema()
		properties[paramName] = ToolInputProperty{
			Type:        GetSchemaTypeString(propSchemaProxy),
			Description: propSchema.Description,
			Location:    "body",
		}
	}

	// Handle required fields with prefix
	// TODO: double check if this makes sense
	prefixAddition := ""
	if prefix != "" {
		prefixAddition = fmt.Sprintf("%s__", prefix)
	}
	if schema.Required != nil {
		for _, req := range schema.Required {
			required = append(required, prefixAddition+req)
		}
	}

	return properties, required, nil
}

// createFallbackBodyParameter creates a single body parameter when no schema is available
func createFallbackBodyParameter(description string) (map[string]ToolInputProperty, []string, error) {
	properties := map[string]ToolInputProperty{
		"body": {
			Type:        "string",
			Description: description,
			Location:    "body",
		},
	}
	required := []string{"body"}
	return properties, required, nil
}

// getSingleBodyParam returns params["body"], true if "body" is the only parameter
// to exist in bodyParams, otherwise nil, false
func getSingleBodyParam(bodyParams map[string]any) (any, bool) {
	val, exists := bodyParams["body"]
	if exists && len(bodyParams) == 1 {
		return val, true
	}
	return nil, false
}

// buildRawBodyFromParams handles single "body" parameter for raw content
func buildRawBodyFromParams(bodyParams map[string]any, contentType string) (io.Reader, error) {
	if len(bodyParams) == 0 {
		return nil, nil
	}

	if bodyContent, exists := getSingleBodyParam(bodyParams); exists {
		// TODO: double check if this is correct, seems wrong to be honest
		if bodyStr, ok := bodyContent.(string); ok {
			return strings.NewReader(bodyStr), nil
		}
		return nil, fmt.Errorf("%s body parameter must be a string", contentType)
	}

	return nil, fmt.Errorf("%s content type requires a 'body' parameter", contentType)
}

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
	// Note: the order determines the order in which content type assignment takes place!
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

// GetAllContentTypes returns all content types of all handlers in this registry in order of registration (FIFO)
func (r *ContentTypeRegistry) GetAllContentTypes() []string {
	var result []string
	for _, handler := range r.handlers {
		result = append(result, handler.GetContentTypes()...)
	}
	return result
}

// JSONContentTypeHandler handles JSON content types
type JSONContentTypeHandler struct{}

// GetContentTypes returns the content types supported by this handler.
func (h *JSONContentTypeHandler) GetContentTypes() []string {
	return []string{"application/json", "*/*", "application/hal+json", "application/vnd.api+json"}
}

// ExtractParameters extracts parameters from JSON schema for tool input.
func (h *JSONContentTypeHandler) ExtractParameters(media *v3.MediaType) (map[string]ToolInputProperty, []string, error) {
	return extractSchemaProperties(media, "")
}

// BuildRequestBody builds a JSON request body from the provided parameters.
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

// XMLContentTypeHandler handles XML content types (both structured and raw)
type XMLContentTypeHandler struct{}

// GetContentTypes returns the content types supported by this handler.
func (h *XMLContentTypeHandler) GetContentTypes() []string {
	return []string{"application/xml", "text/xml"}
}

// ExtractParameters extracts parameters from XML schema for tool input.
func (h *XMLContentTypeHandler) ExtractParameters(media *v3.MediaType) (map[string]ToolInputProperty, []string, error) {
	if media.Schema == nil { // TODO: check if Schema is the only viable property here
		return createFallbackBodyParameter("XML request body content")
	}

	schema := media.Schema.Schema()
	if schema == nil {
		return createFallbackBodyParameter("XML request body content")
	}

	// Check if this is structured XML (has properties)
	// TODO: refactor this into getSchemaProperties - then replace above code with a single if condition
	if schema.Properties != nil && schema.Properties.Len() > 0 {
		// Structured XML - extract individual properties
		return extractSchemaProperties(media, "")
	} else {
		// Raw XML - single body parameter
		return createFallbackBodyParameter("XML request body content")
	}
}

// BuildRequestBody builds an XML request body from the provided parameters.
func (h *XMLContentTypeHandler) BuildRequestBody(bodyParams map[string]any) (io.Reader, error) {
	if len(bodyParams) == 0 {
		return nil, nil
	}

	// Check if this is raw XML (single "body" parameter)
	if bodyContent, exists := getSingleBodyParam(bodyParams); exists {
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

// FormURLEncodedHandler handles application/x-www-form-urlencoded
type FormURLEncodedHandler struct{}

// GetContentTypes returns the content types supported by this handler.
func (h *FormURLEncodedHandler) GetContentTypes() []string {
	return []string{"application/x-www-form-urlencoded"}
}

// hasSchemaProps returns true of the provided media has a schema with valid properties (len > 0)
func hasSchemaProps(media *v3.MediaType) bool {
	if media.Schema == nil {
		return false
	}
	schema := media.Schema.Schema()
	// here it is slightly better I feel like, but could still be simplified
	if schema == nil || schema.Properties == nil || schema.Properties.Len() == 0 {
		return false
	}
	return true
}

// getExamples returns either all examples attached to the media
// or an empty string if no examples are available
func getExamples(media *v3.MediaType) string {
	var result string = ""
	if media.Example != nil {
		exampleJson, err := json.Marshal(media.Example)
		if err != nil {
			log.Print("Unable to marshal example. Continuing.")
		} else {
			result += string(exampleJson)
		}
	}
	if media.Examples != nil && media.Examples.Len() > 0 {
		for key, exampleVal := range media.Examples.FromNewest() {
			exampleValJson, err := json.Marshal(exampleVal)
			if err != nil {
				log.Printf("Unable to marshal example for key %s. Continuing.", key)
			} else {
				result = fmt.Sprintf("%s\n- %s: %s", result, key, exampleValJson)
			}
		}
	}
	return result
}

// ExtractParameters extracts parameters from form schema for tool input.
func (h *FormURLEncodedHandler) ExtractParameters(media *v3.MediaType) (map[string]ToolInputProperty, []string, error) {
	if !hasSchemaProps(media) {
		examples := getExamples(media)
		return createFallbackBodyParameter("Form URL-encoded request body. " + examples)
	}

	return extractSchemaProperties(media, "form")
}

// BuildRequestBody builds a form-encoded request body from the provided parameters.
func (h *FormURLEncodedHandler) BuildRequestBody(bodyParams map[string]any) (io.Reader, error) {
	if len(bodyParams) == 0 {
		return nil, nil
	}

	// Check for single body parameter (raw form data)
	if bodyContent, exists := getSingleBodyParam(bodyParams); exists {
		if bodyStr, ok := bodyContent.(string); ok {
			return strings.NewReader(bodyStr), nil
		}
		return nil, fmt.Errorf("form body parameter must be a string")
	}

	// Build URL encoded form data from form__ prefixed parameters
	formData := url.Values{}
	for paramName, value := range bodyParams {
		if fieldName, found := strings.CutPrefix(paramName, "form__"); found {
			formData.Set(fieldName, fmt.Sprintf("%v", value))
		}
	}

	if len(formData) == 0 {
		return nil, fmt.Errorf("no form__ prefixed parameters found for form URL encoding")
	}

	encodedData := formData.Encode()
	return strings.NewReader(encodedData), nil
}

// MultipartFormDataHandler handles multipart/form-data
type MultipartFormDataHandler struct{}

// GetContentTypes returns the content types supported by this handler.
func (h *MultipartFormDataHandler) GetContentTypes() []string {
	return []string{"multipart/form-data"}
}

// ExtractParameters extracts parameters from multipart form schema for tool input.
func (h *MultipartFormDataHandler) ExtractParameters(media *v3.MediaType) (map[string]ToolInputProperty, []string, error) {
	if media.Schema == nil {
		return createFallbackBodyParameter("Multipart form data request body")
	}

	schema := media.Schema.Schema()
	if schema == nil || schema.Properties == nil || schema.Properties.Len() == 0 {
		return createFallbackBodyParameter("Multipart form data request body")
	}

	// Use the common extraction function with prefix, but we need to handle the special file detection
	properties, required, err := extractSchemaProperties(media, "multipart")
	if err != nil {
		return nil, nil, err
	}

	// Post-process to handle file detection for multipart
	for propPairs := schema.Properties.First(); propPairs != nil; propPairs = propPairs.Next() {
		propName := propPairs.Key()
		propSchemaProxy := propPairs.Value()
		propSchema := propSchemaProxy.Schema()

		prefixedName := fmt.Sprintf("multipart__%s", propName)

		// Detect file uploads (binary format) and update the type
		if propSchema != nil && propSchema.Format == "binary" {
			if prop, exists := properties[prefixedName]; exists {
				prop.Type = "file"
				properties[prefixedName] = prop
			}
		}
	}

	return properties, required, nil
}

// BuildRequestBody builds a multipart form request body from the provided parameters.
func (h *MultipartFormDataHandler) BuildRequestBody(bodyParams map[string]any) (io.Reader, error) {
	if len(bodyParams) == 0 {
		return nil, nil
	}

	// Check for single body parameter (raw multipart data)
	if bodyContent, exists := getSingleBodyParam(bodyParams); exists {
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
		if fieldName, found := strings.CutPrefix(paramName, "multipart__"); found {
			hasMultipartParams = true

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

// PlainTextHandler handles text/plain and text/* content types
type PlainTextHandler struct{}

// GetContentTypes returns the content types supported by this handler.
func (h *PlainTextHandler) GetContentTypes() []string {
	return []string{"text/plain", "text/*"}
}

// ExtractParameters extracts parameters from plain text schema for tool input.
func (h *PlainTextHandler) ExtractParameters(media *v3.MediaType) (map[string]ToolInputProperty, []string, error) {
	return createFallbackBodyParameter("Plain text request body content")
}

// BuildRequestBody builds a plain text request body from the provided parameters.
func (h *PlainTextHandler) BuildRequestBody(bodyParams map[string]any) (io.Reader, error) {
	return buildRawBodyFromParams(bodyParams, "plain text")
}
