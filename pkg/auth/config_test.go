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

package auth

import (
	"strings"
	"testing"
)

func TestBearerAuthConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      *BearerAuthConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "disabled auth should pass validation",
			config: &BearerAuthConfig{
				Enabled: false,
			},
			expectError: false,
		},
		{
			name: "valid JWKS configuration",
			config: &BearerAuthConfig{
				Enabled:   true,
				JWKSUri:   "https://auth.example.com/.well-known/jwks.json",
				Algorithm: "RS256",
				Issuer:    "https://auth.example.com",
				Audience:  "test-audience",
			},
			expectError: false,
		},
		{
			name: "valid static key configuration",
			config: &BearerAuthConfig{
				Enabled:   true,
				PublicKey: "-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMI...",
				Algorithm: "RS256",
			},
			expectError: false,
		},
		{
			name: "missing key source should fail",
			config: &BearerAuthConfig{
				Enabled: true,
			},
			expectError: true,
			errorMsg:    "either jwksUri or publicKey must be provided",
		},
		{
			name: "both key sources should fail",
			config: &BearerAuthConfig{
				Enabled:   true,
				JWKSUri:   "https://auth.example.com/.well-known/jwks.json",
				PublicKey: "-----BEGIN PUBLIC KEY-----\ntest",
			},
			expectError: true,
			errorMsg:    "cannot specify both jwksUri and publicKey",
		},
		{
			name: "non-HTTPS JWKS URI should fail",
			config: &BearerAuthConfig{
				Enabled: true,
				JWKSUri: "http://auth.example.com/.well-known/jwks.json",
			},
			expectError: true,
			errorMsg:    "jwksUri must use HTTPS",
		},
		{
			name: "invalid algorithm should fail",
			config: &BearerAuthConfig{
				Enabled:   true,
				JWKSUri:   "https://auth.example.com/.well-known/jwks.json",
				Algorithm: "INVALID",
			},
			expectError: true,
			errorMsg:    "unsupported JWT algorithm",
		},
		{
			name: "excessive cache TTL should fail",
			config: &BearerAuthConfig{
				Enabled:  true,
				JWKSUri:  "https://auth.example.com/.well-known/jwks.json",
				CacheTTL: 4000, // > 3600
			},
			expectError: true,
			errorMsg:    "cacheTtl cannot exceed 3600 seconds",
		},
		{
			name: "invalid issuer URL should fail",
			config: &BearerAuthConfig{
				Enabled: true,
				JWKSUri: "https://auth.example.com/.well-known/jwks.json",
				Issuer:  "invalid-url",
			},
			expectError: true,
			errorMsg:    "issuer must be a valid URL",
		},
		{
			name: "defaults should be set",
			config: &BearerAuthConfig{
				Enabled: true,
				JWKSUri: "https://auth.example.com/.well-known/jwks.json",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				
				// Check that defaults were set
				if tt.config.Enabled {
					if tt.config.Algorithm == "" {
						t.Errorf("algorithm should be set to default")
					}
					if tt.config.CacheTTL == 0 {
						t.Errorf("cacheTTL should be set to default")
					}
				}
			}
		})
	}
}

func TestBearerAuthConfig_GetKeySource(t *testing.T) {
	tests := []struct {
		name     string
		config   *BearerAuthConfig
		expected string
	}{
		{
			name: "JWKS source",
			config: &BearerAuthConfig{
				JWKSUri: "https://auth.example.com/.well-known/jwks.json",
			},
			expected: "JWKS from https://auth.example.com/.well-known/jwks.json",
		},
		{
			name: "static key source",
			config: &BearerAuthConfig{
				PublicKey: "-----BEGIN PUBLIC KEY-----\ntest",
			},
			expected: "Static public key",
		},
		{
			name:     "no source",
			config:   &BearerAuthConfig{},
			expected: "No key source configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.GetKeySource()
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestBearerAuthConfig_HasScopeValidation(t *testing.T) {
	tests := []struct {
		name     string
		config   *BearerAuthConfig
		expected bool
	}{
		{
			name: "with required scopes",
			config: &BearerAuthConfig{
				RequiredScopes: []string{"read", "write"},
			},
			expected: true,
		},
		{
			name:     "without required scopes",
			config:   &BearerAuthConfig{},
			expected: false,
		},
		{
			name: "empty required scopes",
			config: &BearerAuthConfig{
				RequiredScopes: []string{},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.HasScopeValidation()
			if result != tt.expected {
				t.Errorf("expected %t, got %t", tt.expected, result)
			}
		})
	}
}

func TestIsValidAlgorithm(t *testing.T) {
	validAlgorithms := []string{
		"RS256", "RS384", "RS512",
		"ES256", "ES384", "ES512",
		"PS256", "PS384", "PS512",
	}

	invalidAlgorithms := []string{
		"HS256", "HS384", "HS512", // HMAC algorithms not supported
		"NONE", "none",
		"INVALID",
		"",
	}

	for _, alg := range validAlgorithms {
		t.Run("valid_"+alg, func(t *testing.T) {
			if !isValidAlgorithm(alg) {
				t.Errorf("algorithm %s should be valid", alg)
			}
		})
	}

	for _, alg := range invalidAlgorithms {
		t.Run("invalid_"+alg, func(t *testing.T) {
			if isValidAlgorithm(alg) {
				t.Errorf("algorithm %s should be invalid", alg)
			}
		})
	}
}