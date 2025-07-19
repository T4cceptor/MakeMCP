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
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// APIClient struct to encapsulate baseURL and http.Client.
type APIClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewAPIClient creates a new APIClient with the given baseURL and timeout.
func NewAPIClient(baseURL string, timeoutSeconds int) *APIClient {
	return &APIClient{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: time.Duration(timeoutSeconds) * time.Second,
		},
	}
}

// boolPtr returns a pointer to the given bool value.
func boolPtr(val bool) *bool {
	return &val
}

// URLSecurityIssue represents a potential security concern with a URL.
type URLSecurityIssue struct {
	Type        string
	Description string
	URL         string
}

// CheckURLSecurity analyzes a URL for potential security issues.
func CheckURLSecurity(rawURL string) []URLSecurityIssue {
	var issues []URLSecurityIssue

	// Skip file paths
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		return issues
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return issues
	}

	hostname := parsedURL.Hostname()

	// Check for localhost and loopback addresses
	if hostname == "localhost" || hostname == "127.0.0.1" || hostname == "::1" {
		issues = append(issues, URLSecurityIssue{
			Type:        "localhost",
			Description: "URL points to localhost/loopback address",
			URL:         rawURL,
		})
	}

	// Check for private IP ranges
	if ip := net.ParseIP(hostname); ip != nil {
		if isPrivateIP(ip) {
			issues = append(issues, URLSecurityIssue{
				Type:        "private_ip",
				Description: "URL points to private IP address",
				URL:         rawURL,
			})
		}
	}

	// Check for cloud metadata endpoints
	cloudMetadataHosts := []string{
		"169.254.169.254",          // AWS/Azure metadata
		"metadata.google.internal", // GCP metadata
		"100.100.100.200",          // Alibaba Cloud metadata
	}

	for _, metadataHost := range cloudMetadataHosts {
		if hostname == metadataHost {
			issues = append(issues, URLSecurityIssue{
				Type:        "cloud_metadata",
				Description: "URL points to cloud metadata endpoint",
				URL:         rawURL,
			})
			break
		}
	}

	// Check for link-local addresses (169.254.x.x)
	if ip := net.ParseIP(hostname); ip != nil {
		if ip.IsLinkLocalUnicast() {
			issues = append(issues, URLSecurityIssue{
				Type:        "link_local",
				Description: "URL points to link-local address",
				URL:         rawURL,
			})
		}
	}

	return issues
}

// isPrivateIP checks if an IP address is in a private range.
func isPrivateIP(ip net.IP) bool {
	// Private IPv4 ranges
	private4Ranges := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
	}

	for _, cidr := range private4Ranges {
		_, network, _ := net.ParseCIDR(cidr)
		if network.Contains(ip) {
			return true
		}
	}

	// Private IPv6 ranges
	if ip.To4() == nil { // IPv6
		// fc00::/7 (unique local addresses)
		_, network, _ := net.ParseCIDR("fc00::/7")
		if network.Contains(ip) {
			return true
		}
	}

	return false
}

// WarnURLSecurity logs security warnings for suspicious URLs.
func WarnURLSecurity(rawURL string, urlType string, devMode bool) {
	if devMode {
		return
	}

	issues := CheckURLSecurity(rawURL)
	if len(issues) == 0 {
		return
	}

	log.Printf("⚠️  SECURITY WARNING: %s URL has potential security concerns:", urlType)
	for _, issue := range issues {
		log.Printf("   - %s: %s", issue.Type, issue.Description)
	}
	log.Printf("   URL: %s", rawURL)
	log.Printf("   To suppress these warnings for local development, use the --dev-mode flag")
	log.Println()
}

// substitutePathParams replaces path parameters in URL template.
func substitutePathParams(path string, pathParams map[string]any) string {
	for k, v := range pathParams {
		placeholder := fmt.Sprintf("{%s}", k)
		path = strings.ReplaceAll(path, placeholder, fmt.Sprintf("%v", v))
	}
	return path
}

// encodeQueryParams encodes query parameters for URL.
func encodeQueryParams(queryParams map[string]any) string {
	values := url.Values{}
	for k, v := range queryParams {
		values.Set(k, fmt.Sprintf("%v", v))
	}
	return values.Encode()
}

// parsePrefixedParameters parses parameters using prefix approach and returns SplitParams.
func parsePrefixedParameters(argsRaw map[string]any) ToolParams {
	params := NewSplitParams()
	for prefixedName, value := range argsRaw {
		parts := strings.SplitN(prefixedName, "__", 2)
		if len(parts) != 2 {
			continue
		}
		location, paramName := parts[0], parts[1]

		paramLocation := ParameterLocation(location)
		if !paramLocation.IsValid() {
			// Invalid parameter location, skip this parameter
			continue
		}

		switch paramLocation {
		case ParameterLocationPath:
			params.Path[paramName] = value
		case ParameterLocationQuery:
			params.Query[paramName] = value
		case ParameterLocationHeader:
			params.Header[paramName] = value
		case ParameterLocationCookie:
			params.Cookie[paramName] = value
		case ParameterLocationBody:
			params.Body[paramName] = value
		}
	}
	return params
}
