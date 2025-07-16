package main

import (
	"bytes"
	"log"
	"net"
	"os"
	"strings"
	"testing"
)

func TestCheckURLSecurity(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		expectedIssues []string
	}{
		{
			name:           "Safe HTTPS URL",
			url:            "https://api.example.com/openapi.json",
			expectedIssues: []string{},
		},
		{
			name:           "Safe HTTP URL",
			url:            "http://api.example.com/openapi.json",
			expectedIssues: []string{},
		},
		{
			name:           "Localhost URL",
			url:            "http://localhost:8080/openapi.json",
			expectedIssues: []string{"localhost"},
		},
		{
			name:           "127.0.0.1 URL",
			url:            "http://127.0.0.1:8080/openapi.json",
			expectedIssues: []string{"localhost"},
		},
		{
			name:           "IPv6 localhost",
			url:            "http://[::1]:8080/openapi.json",
			expectedIssues: []string{"localhost"},
		},
		{
			name:           "Private IP 10.x.x.x",
			url:            "http://10.0.0.1:8080/openapi.json",
			expectedIssues: []string{"private_ip"},
		},
		{
			name:           "Private IP 192.168.x.x",
			url:            "http://192.168.1.100:8080/openapi.json",
			expectedIssues: []string{"private_ip"},
		},
		{
			name:           "Private IP 172.16.x.x",
			url:            "http://172.16.0.1:8080/openapi.json",
			expectedIssues: []string{"private_ip"},
		},
		{
			name:           "AWS metadata endpoint",
			url:            "http://169.254.169.254/latest/meta-data/",
			expectedIssues: []string{"cloud_metadata", "link_local"},
		},
		{
			name:           "GCP metadata endpoint",
			url:            "http://metadata.google.internal/computeMetadata/v1/",
			expectedIssues: []string{"cloud_metadata"},
		},
		{
			name:           "Alibaba Cloud metadata endpoint",
			url:            "http://100.100.100.200/latest/meta-data/",
			expectedIssues: []string{"cloud_metadata"},
		},
		{
			name:           "Link-local address",
			url:            "http://169.254.1.1:8080/openapi.json",
			expectedIssues: []string{"link_local"},
		},
		{
			name:           "File path (should be ignored)",
			url:            "/path/to/openapi.json",
			expectedIssues: []string{},
		},
		{
			name:           "Invalid URL format",
			url:            "not-a-url",
			expectedIssues: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := checkURLSecurity(tt.url)
			
			if len(issues) != len(tt.expectedIssues) {
				t.Errorf("Expected %d issues, got %d", len(tt.expectedIssues), len(issues))
			}
			
			for _, expectedType := range tt.expectedIssues {
				found := false
				for _, issue := range issues {
					if issue.Type == expectedType {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected issue type '%s' not found", expectedType)
				}
			}
		})
	}
}

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		{"Public IP", "8.8.8.8", false},
		{"Private 10.x.x.x", "10.0.0.1", true},
		{"Private 192.168.x.x", "192.168.1.1", true},
		{"Private 172.16.x.x", "172.16.0.1", true},
		{"Private 172.31.x.x", "172.31.255.255", true},
		{"Non-private 172.15.x.x", "172.15.0.1", false},
		{"Non-private 172.32.x.x", "172.32.0.1", false},
		{"IPv6 private", "fc00::1", true},
		{"IPv6 public", "2001:db8::1", false},
		{"Localhost IPv4", "127.0.0.1", false}, // localhost is handled separately
		{"IPv6 localhost", "::1", false},       // localhost is handled separately
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := parseIP(tt.ip)
			if ip == nil {
				t.Fatalf("Failed to parse IP: %s", tt.ip)
			}
			
			result := isPrivateIP(ip)
			if result != tt.expected {
				t.Errorf("Expected %v for IP %s, got %v", tt.expected, tt.ip, result)
			}
		})
	}
}

func TestWarnURLSecurity(t *testing.T) {
	tests := []struct {
		name            string
		url             string
		urlType         string
		devMode         bool
		expectWarning   bool
		expectedContent []string
	}{
		{
			name:            "Safe URL - no warning",
			url:             "https://api.example.com/openapi.json",
			urlType:         "OpenAPI spec",
			devMode:         false,
			expectWarning:   false,
			expectedContent: []string{},
		},
		{
			name:            "Localhost URL - warning in normal mode",
			url:             "http://localhost:8080/openapi.json",
			urlType:         "OpenAPI spec",
			devMode:         false,
			expectWarning:   true,
			expectedContent: []string{"SECURITY WARNING", "OpenAPI spec", "localhost", "dev-mode"},
		},
		{
			name:            "Localhost URL - no warning in dev mode",
			url:             "http://localhost:8080/openapi.json",
			urlType:         "OpenAPI spec",
			devMode:         true,
			expectWarning:   false,
			expectedContent: []string{},
		},
		{
			name:            "Private IP - warning in normal mode",
			url:             "http://192.168.1.100/openapi.json",
			urlType:         "Base URL",
			devMode:         false,
			expectWarning:   true,
			expectedContent: []string{"SECURITY WARNING", "Base URL", "private_ip", "dev-mode"},
		},
		{
			name:            "Cloud metadata - warning in normal mode",
			url:             "http://169.254.169.254/latest/meta-data/",
			urlType:         "OpenAPI spec",
			devMode:         false,
			expectWarning:   true,
			expectedContent: []string{"SECURITY WARNING", "cloud_metadata", "dev-mode"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture log output
			var buf bytes.Buffer
			log.SetOutput(&buf)
			defer log.SetOutput(os.Stderr)

			warnURLSecurity(tt.url, tt.urlType, tt.devMode)

			output := buf.String()
			
			if tt.expectWarning && output == "" {
				t.Error("Expected warning but got none")
			}
			
			if !tt.expectWarning && output != "" {
				t.Errorf("Expected no warning but got: %s", output)
			}
			
			for _, expectedContent := range tt.expectedContent {
				if !strings.Contains(output, expectedContent) {
					t.Errorf("Expected output to contain '%s', got: %s", expectedContent, output)
				}
			}
		})
	}
}

func TestURLSecurityIntegration(t *testing.T) {
	tests := []struct {
		name        string
		specs       string
		baseURL     string
		devMode     bool
		expectLogs  bool
		logContains []string
	}{
		{
			name:        "Safe URLs - no warnings",
			specs:       "https://api.example.com/openapi.json",
			baseURL:     "https://api.example.com",
			devMode:     false,
			expectLogs:  false,
			logContains: []string{},
		},
		{
			name:        "Localhost URLs - warnings in normal mode",
			specs:       "http://localhost:8081/openapi.json",
			baseURL:     "http://localhost:8081",
			devMode:     false,
			expectLogs:  true,
			logContains: []string{"OpenAPI spec", "Base URL", "localhost", "dev-mode"},
		},
		{
			name:        "Localhost URLs - no warnings in dev mode",
			specs:       "http://localhost:8081/openapi.json",
			baseURL:     "http://localhost:8081",
			devMode:     true,
			expectLogs:  false,
			logContains: []string{},
		},
		{
			name:        "Mixed URLs - only private IP warned",
			specs:       "https://api.example.com/openapi.json",
			baseURL:     "http://192.168.1.100",
			devMode:     false,
			expectLogs:  true,
			logContains: []string{"Base URL", "private_ip"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture log output
			var buf bytes.Buffer
			log.SetOutput(&buf)
			defer log.SetOutput(os.Stderr)

			// Simulate the security check logic from HandleOpenAPI
			if !tt.devMode {
				warnURLSecurity(tt.specs, "OpenAPI spec", false)
				warnURLSecurity(tt.baseURL, "Base URL", false)
			}

			output := buf.String()
			
			if tt.expectLogs && output == "" {
				t.Error("Expected log output but got none")
			}
			
			if !tt.expectLogs && output != "" {
				t.Errorf("Expected no log output but got: %s", output)
			}
			
			for _, expectedContent := range tt.logContains {
				if !strings.Contains(output, expectedContent) {
					t.Errorf("Expected log to contain '%s', got: %s", expectedContent, output)
				}
			}
		})
	}
}

// Helper function to parse IP address for testing
func parseIP(ip string) net.IP {
	return net.ParseIP(ip)
}