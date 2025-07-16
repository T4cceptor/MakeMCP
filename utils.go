// Contains generic utils
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// boolPtr returns a pointer to the given bool value.
// Useful for assigning *bool fields in structs.
func boolPtr(val bool) *bool { return &val }

// =============================================================================
// URL SECURITY VALIDATION
// =============================================================================

// URLSecurityIssue represents a potential security concern with a URL
type URLSecurityIssue struct {
	Type        string
	Description string
	URL         string
}

// checkURLSecurity analyzes a URL for potential security issues
func checkURLSecurity(rawURL string) []URLSecurityIssue {
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

// isPrivateIP checks if an IP address is in a private range
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

// warnURLSecurity logs security warnings for suspicious URLs
func warnURLSecurity(rawURL string, urlType string, devMode bool) {
	if devMode {
		return
	}

	issues := checkURLSecurity(rawURL)
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

// MakeMCPAppFromFile loads a MakeMCPApp from a JSON file.
func MakeMCPAppFromFile(filename string) (MakeMCPApp, error) {
	var app MakeMCPApp
	file, err := os.Open(filename)
	if err != nil {
		return app, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&app); err != nil {
		return app, fmt.Errorf("failed to decode JSON: %w", err)
	}

	log.Printf("MakeMCPApp loaded from %s\n", filename)
	return app, nil
}
