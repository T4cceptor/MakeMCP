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
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Test helper to generate RSA key pairs
func generateTestKeyPair(t *testing.T) (*rsa.PrivateKey, string) {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate test key: %v", err)
	}

	publicKeyDER, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		t.Fatalf("failed to marshal public key: %v", err)
	}

	publicKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyDER,
	})

	return privateKey, string(publicKeyPEM)
}

// Test helper to create a test JWT token
func createTestToken(t *testing.T, privateKey *rsa.PrivateKey, claims *TokenClaims) string {
	t.Helper()

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatalf("failed to sign test token: %v", err)
	}

	return tokenString
}

func TestNewBearerTokenValidator(t *testing.T) {
	_, publicKeyPEM := generateTestKeyPair(t)

	tests := []struct {
		name        string
		config      *BearerAuthConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid static key config",
			config: &BearerAuthConfig{
				Enabled:   true,
				PublicKey: publicKeyPEM,
				Algorithm: "RS256",
			},
			expectError: false,
		},
		{
			name: "invalid config should fail",
			config: &BearerAuthConfig{
				Enabled: true,
				// Missing key source
			},
			expectError: true,
			errorMsg:    "invalid bearer auth configuration",
		},
		{
			name: "invalid public key should fail",
			config: &BearerAuthConfig{
				Enabled:   true,
				PublicKey: "invalid-key",
				Algorithm: "RS256",
			},
			expectError: true,
			errorMsg:    "failed to parse RSA public key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator, err := NewBearerTokenValidator(tt.config)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errorMsg, err.Error())
				}
				if validator != nil {
					t.Error("expected nil validator on error")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if validator == nil {
					t.Error("expected validator but got nil")
				} else {
					// Clean up
					_ = validator.Close()
				}
			}
		})
	}
}

func TestBearerTokenValidator_ValidateToken(t *testing.T) {
	privateKey, publicKeyPEM := generateTestKeyPair(t)

	config := &BearerAuthConfig{
		Enabled:        true,
		PublicKey:      publicKeyPEM,
		Algorithm:      "RS256",
		Issuer:         "https://test.example.com",
		Audience:       "test-audience",
		RequiredScopes: []string{"read"},
	}

	validator, err := NewBearerTokenValidator(config)
	if err != nil {
		t.Fatalf("failed to create validator: %v", err)
	}
	defer validator.Close()

	now := time.Now()

	tests := []struct {
		name        string
		claims      *TokenClaims
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid token",
			claims: &TokenClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Issuer:    "https://test.example.com",
					Subject:   "user123",
					Audience:  []string{"test-audience"},
					ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
					IssuedAt:  jwt.NewNumericDate(now),
				},
				UserID:   "user123",
				Username: "testuser",
				Email:    "test@example.com",
				Scopes:   []string{"read", "write"},
			},
			expectError: false,
		},
		{
			name: "wrong issuer",
			claims: &TokenClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Issuer:    "https://wrong.example.com",
					Subject:   "user123",
					Audience:  []string{"test-audience"},
					ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
					IssuedAt:  jwt.NewNumericDate(now),
				},
				UserID: "user123",
				Scopes: []string{"read"},
			},
			expectError: true,
			errorMsg:    "token validation failed",
		},
		{
			name: "wrong audience",
			claims: &TokenClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Issuer:    "https://test.example.com",
					Subject:   "user123",
					Audience:  []string{"wrong-audience"},
					ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
					IssuedAt:  jwt.NewNumericDate(now),
				},
				UserID: "user123",
				Scopes: []string{"read"},
			},
			expectError: true,
			errorMsg:    "token validation failed",
		},
		{
			name: "expired token",
			claims: &TokenClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Issuer:    "https://test.example.com",
					Subject:   "user123",
					Audience:  []string{"test-audience"},
					ExpiresAt: jwt.NewNumericDate(now.Add(-time.Hour)), // Expired
					IssuedAt:  jwt.NewNumericDate(now.Add(-2 * time.Hour)),
				},
				UserID: "user123",
				Scopes: []string{"read"},
			},
			expectError: true,
			errorMsg:    "token validation failed",
		},
		{
			name: "missing required scope",
			claims: &TokenClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Issuer:    "https://test.example.com",
					Subject:   "user123",
					Audience:  []string{"test-audience"},
					ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
					IssuedAt:  jwt.NewNumericDate(now),
				},
				UserID: "user123",
				Scopes: []string{"write"}, // Missing "read" scope
			},
			expectError: true,
			errorMsg:    "insufficient scopes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenString := createTestToken(t, privateKey, tt.claims)

			userCtx, err := validator.ValidateToken(tokenString)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errorMsg, err.Error())
				}
				if userCtx != nil {
					t.Error("expected nil user context on error")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if userCtx == nil {
					t.Error("expected user context but got nil")
				} else {
					// Validate user context fields
					if userCtx.UserID != tt.claims.UserID {
						t.Errorf("expected UserID %q, got %q", tt.claims.UserID, userCtx.UserID)
					}
					if userCtx.Username != tt.claims.Username {
						t.Errorf("expected Username %q, got %q", tt.claims.Username, userCtx.Username)
					}
					if userCtx.Email != tt.claims.Email {
						t.Errorf("expected Email %q, got %q", tt.claims.Email, userCtx.Email)
					}
					if userCtx.Token != tokenString {
						t.Error("expected original token string to be preserved")
					}
					if len(userCtx.Scopes) != len(tt.claims.Scopes) {
						t.Errorf("expected %d scopes, got %d", len(tt.claims.Scopes), len(userCtx.Scopes))
					}
				}
			}
		})
	}
}

func TestBearerTokenValidator_ValidateToken_OptionalValidation(t *testing.T) {
	privateKey, publicKeyPEM := generateTestKeyPair(t)

	// Config without optional validations
	config := &BearerAuthConfig{
		Enabled:   true,
		PublicKey: publicKeyPEM,
		Algorithm: "RS256",
		// No issuer, audience, or required scopes
	}

	validator, err := NewBearerTokenValidator(config)
	if err != nil {
		t.Fatalf("failed to create validator: %v", err)
	}
	defer validator.Close()

	now := time.Now()

	// Token without issuer/audience should be valid
	claims := &TokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user123",
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
		UserID: "user123",
		// No scopes
	}

	tokenString := createTestToken(t, privateKey, claims)
	userCtx, err := validator.ValidateToken(tokenString)

	if err != nil {
		t.Errorf("unexpected error with optional validation: %v", err)
	}
	if userCtx == nil {
		t.Error("expected user context")
	}
}

func TestBearerTokenValidator_InvalidTokens(t *testing.T) {
	_, publicKeyPEM := generateTestKeyPair(t)

	config := &BearerAuthConfig{
		Enabled:   true,
		PublicKey: publicKeyPEM,
		Algorithm: "RS256",
	}

	validator, err := NewBearerTokenValidator(config)
	if err != nil {
		t.Fatalf("failed to create validator: %v", err)
	}
	defer validator.Close()

	tests := []struct {
		name  string
		token string
	}{
		{
			name:  "empty token",
			token: "",
		},
		{
			name:  "malformed token",
			token: "not.a.jwt",
		},
		{
			name:  "invalid signature",
			token: "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiYWRtaW4iOnRydWV9.invalid-signature",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userCtx, err := validator.ValidateToken(tt.token)

			if err == nil {
				t.Error("expected error for invalid token")
			}
			if userCtx != nil {
				t.Error("expected nil user context for invalid token")
			}
		})
	}
}

func TestBearerTokenValidator_validateScopes(t *testing.T) {
	config := &BearerAuthConfig{
		RequiredScopes: []string{"read", "write"},
	}

	validator := &BearerTokenValidator{config: config}

	tests := []struct {
		name        string
		tokenScopes []string
		expectError bool
	}{
		{
			name:        "has all required scopes",
			tokenScopes: []string{"read", "write", "admin"},
			expectError: false,
		},
		{
			name:        "missing one required scope",
			tokenScopes: []string{"read"},
			expectError: true,
		},
		{
			name:        "no scopes",
			tokenScopes: []string{},
			expectError: true,
		},
		{
			name:        "different scopes",
			tokenScopes: []string{"admin", "delete"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateScopes(tt.tokenScopes)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestBearerTokenValidator_validateScopes_NoRequiredScopes(t *testing.T) {
	config := &BearerAuthConfig{
		// No required scopes
	}

	validator := &BearerTokenValidator{config: config}

	// Should pass validation regardless of token scopes
	err := validator.validateScopes([]string{})
	if err != nil {
		t.Errorf("unexpected error when no scopes required: %v", err)
	}

	err = validator.validateScopes([]string{"any", "scopes"})
	if err != nil {
		t.Errorf("unexpected error when no scopes required: %v", err)
	}
}