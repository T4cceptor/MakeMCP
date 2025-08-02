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
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Test helper to generate RSA key pairs for middleware tests
func generateMiddlewareTestKeyPair(t *testing.T) (*rsa.PrivateKey, string) {
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

// Test helper to create a valid JWT token
func createTestJWT(t *testing.T, privateKey *rsa.PrivateKey, claims jwt.Claims) string {
	t.Helper()

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatalf("failed to sign test token: %v", err)
	}

	return tokenString
}

// Test helper to create test middleware with static key
func createTestMiddleware(t *testing.T, config *BearerAuthConfig) (*BearerAuthMiddleware, error) {
	t.Helper()
	return NewBearerAuthMiddleware(config)
}

// Test helper to create a test HTTP handler that captures context
func createTestHandler(t *testing.T, captureContext *bool, captureUser **UserContext) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*captureContext = IsAuthenticated(r.Context())
		*captureUser = GetUserContext(r.Context())
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})
}

func TestNewBearerAuthMiddleware(t *testing.T) {
	_, publicKeyPEM := generateMiddlewareTestKeyPair(t)

	tests := []struct {
		name        string
		config      *BearerAuthConfig
		expectError bool
	}{
		{
			name: "valid config with public key",
			config: &BearerAuthConfig{
				Enabled:   true,
				PublicKey: publicKeyPEM,
				Algorithm: "RS256",
				Required:  true,
			},
			expectError: false,
		},
		{
			name: "valid config with JWKS URI",
			config: &BearerAuthConfig{
				Enabled:   true,
				JWKSUri:   "https://example.com/.well-known/jwks.json",
				Algorithm: "RS256",
				Required:  true,
			},
			expectError: true, // Will fail because JWKS endpoint doesn't exist
		},
		{
			name: "invalid config - no key source",
			config: &BearerAuthConfig{
				Enabled:   true,
				Algorithm: "RS256",
				Required:  true,
			},
			expectError: true,
		},
		{
			name: "disabled config",
			config: &BearerAuthConfig{
				Enabled: false,
			},
			expectError: true, // Will fail validation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware, err := NewBearerAuthMiddleware(tt.config)
			
			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
				if middleware != nil {
					t.Error("expected nil middleware on error")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if middleware == nil {
					t.Error("expected middleware but got nil")
				}
			}
		})
	}
}

func TestExtractAndValidateToken(t *testing.T) {
	privateKey, publicKeyPEM := generateMiddlewareTestKeyPair(t)

	// Create middleware with static key
	config := &BearerAuthConfig{
		Enabled:   true,
		PublicKey: publicKeyPEM,
		Algorithm: "RS256",
		Required:  true,
	}
	middleware, err := createTestMiddleware(t, config)
	if err != nil {
		t.Fatalf("failed to create test middleware: %v", err)
	}
	defer middleware.Close()

	// Create valid token
	claims := &TokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "test-user",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		UserID:   "test-user",
		Username: "testuser",
		Email:    "test@example.com",
		Scopes:   []string{"read", "write"},
	}
	validToken := createTestJWT(t, privateKey, claims)

	// Create expired token
	expiredClaims := &TokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "test-user",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
		UserID: "test-user",
	}
	expiredToken := createTestJWT(t, privateKey, expiredClaims)

	tests := []struct {
		name           string
		authHeader     string
		expectedError  bool
		expectedUser   bool
		errorContains  string
	}{
		{
			name:          "valid bearer token",
			authHeader:    "Bearer " + validToken,
			expectedError: false,
			expectedUser:  true,
		},
		{
			name:          "missing authorization header (required)",
			authHeader:    "",
			expectedError: true,
			errorContains: "Authorization header required",
		},
		{
			name:          "invalid format - no Bearer prefix",
			authHeader:    validToken,
			expectedError: true,
			errorContains: "Authorization header must use Bearer format",
		},
		{
			name:          "invalid format - wrong prefix",
			authHeader:    "Basic " + validToken,
			expectedError: true,
			errorContains: "Authorization header must use Bearer format",
		},
		{
			name:          "empty token after Bearer",
			authHeader:    "Bearer ",
			expectedError: true,
			errorContains: "Bearer token cannot be empty",
		},
		{
			name:          "expired token",
			authHeader:    "Bearer " + expiredToken,
			expectedError: true,
			errorContains: "expired",
		},
		{
			name:          "invalid token signature",
			authHeader:    "Bearer invalid.token.signature",
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			userCtx, err := middleware.ExtractAndValidateToken(req)

			if tt.expectedError {
				if err == nil {
					t.Error("expected error but got none")
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("expected error to contain %q, got: %v", tt.errorContains, err)
				}
				if userCtx != nil {
					t.Error("expected nil user context on error")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if tt.expectedUser && userCtx == nil {
					t.Error("expected user context but got nil")
				}
				if !tt.expectedUser && userCtx != nil {
					t.Error("expected nil user context but got user")
				}
			}

			if userCtx != nil {
				// Validate user context fields
				if userCtx.UserID != "test-user" {
					t.Errorf("expected UserID 'test-user', got %q", userCtx.UserID)
				}
				if userCtx.Username != "testuser" {
					t.Errorf("expected Username 'testuser', got %q", userCtx.Username)
				}
				if userCtx.Email != "test@example.com" {
					t.Errorf("expected Email 'test@example.com', got %q", userCtx.Email)
				}
			}
		})
	}
}

func TestExtractAndValidateToken_OptionalAuth(t *testing.T) {
	privateKey, publicKeyPEM := generateMiddlewareTestKeyPair(t)

	// Create middleware with optional authentication
	config := &BearerAuthConfig{
		Enabled:   true,
		PublicKey: publicKeyPEM,
		Algorithm: "RS256",
		Required:  false, // Authentication is optional
	}
	middleware, err := createTestMiddleware(t, config)
	if err != nil {
		t.Fatalf("failed to create test middleware: %v", err)
	}
	defer middleware.Close()

	// Test missing header with optional auth
	req := httptest.NewRequest("GET", "/test", nil)
	userCtx, err := middleware.ExtractAndValidateToken(req)

	if err != nil {
		t.Errorf("unexpected error with optional auth: %v", err)
	}
	if userCtx != nil {
		t.Error("expected nil user context for anonymous request")
	}

	// Test valid token with optional auth
	claims := &TokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "test-user",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		UserID: "test-user",
	}
	validToken := createTestJWT(t, privateKey, claims)

	req.Header.Set("Authorization", "Bearer "+validToken)
	userCtx, err = middleware.ExtractAndValidateToken(req)

	if err != nil {
		t.Errorf("unexpected error with valid token: %v", err)
	}
	if userCtx == nil {
		t.Error("expected user context with valid token")
	}
}

func TestExtractAndValidateToken_ScopeValidation(t *testing.T) {
	privateKey, publicKeyPEM := generateMiddlewareTestKeyPair(t)

	// Create middleware with required scopes
	config := &BearerAuthConfig{
		Enabled:        true,
		PublicKey:      publicKeyPEM,
		Algorithm:      "RS256",
		Required:       true,
		RequiredScopes: []string{"read", "admin"},
	}
	middleware, err := createTestMiddleware(t, config)
	if err != nil {
		t.Fatalf("failed to create test middleware: %v", err)
	}
	defer middleware.Close()

	tests := []struct {
		name          string
		tokenScopes   []string
		expectedError bool
		errorContains string
	}{
		{
			name:          "sufficient scopes",
			tokenScopes:   []string{"read", "admin", "write"},
			expectedError: false,
		},
		{
			name:          "missing required scope",
			tokenScopes:   []string{"read", "write"},
			expectedError: true,
			errorContains: "insufficient scopes",
		},
		{
			name:          "no scopes",
			tokenScopes:   []string{},
			expectedError: true,
			errorContains: "insufficient scopes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims := &TokenClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Subject:   "test-user",
					ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
					IssuedAt:  jwt.NewNumericDate(time.Now()),
				},
				UserID: "test-user",
				Scopes: tt.tokenScopes,
			}
			token := createTestJWT(t, privateKey, claims)

			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Authorization", "Bearer "+token)

			userCtx, err := middleware.ExtractAndValidateToken(req)

			if tt.expectedError {
				if err == nil {
					t.Error("expected error but got none")
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("expected error to contain %q, got: %v", tt.errorContains, err)
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
				}
			}
		})
	}
}

func TestMiddleware(t *testing.T) {
	privateKey, publicKeyPEM := generateMiddlewareTestKeyPair(t)

	// Create middleware
	config := &BearerAuthConfig{
		Enabled:   true,
		PublicKey: publicKeyPEM,
		Algorithm: "RS256",
		Required:  true,
	}
	middleware, err := createTestMiddleware(t, config)
	if err != nil {
		t.Fatalf("failed to create test middleware: %v", err)
	}
	defer middleware.Close()

	// Create valid token
	claims := &TokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "test-user",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		UserID:   "test-user",
		Username: "testuser",
		Email:    "test@example.com",
		Scopes:   []string{"read", "write"},
	}
	validToken := createTestJWT(t, privateKey, claims)

	tests := []struct {
		name               string
		authHeader         string
		expectedStatus     int
		expectedAuth       bool
		expectedUserID     string
		responseContains   string
	}{
		{
			name:           "valid token - request succeeds",
			authHeader:     "Bearer " + validToken,
			expectedStatus: http.StatusOK,
			expectedAuth:   true,
			expectedUserID: "test-user",
			responseContains: "success",
		},
		{
			name:           "missing token - unauthorized",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
			expectedAuth:   false,
			responseContains: "Authorization required",
		},
		{
			name:           "invalid format - bad request",
			authHeader:     "Basic " + validToken,
			expectedStatus: http.StatusBadRequest,
			expectedAuth:   false,
			responseContains: "Invalid authorization format",
		},
		{
			name:           "empty token - bad request",
			authHeader:     "Bearer ",
			expectedStatus: http.StatusBadRequest,
			expectedAuth:   false,
			responseContains: "Empty bearer token",
		},
		{
			name:           "invalid token - unauthorized",
			authHeader:     "Bearer invalid.token.here",
			expectedStatus: http.StatusUnauthorized,
			expectedAuth:   false,
			responseContains: "Authentication failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test handler that captures context
			var contextAuth bool
			var contextUser *UserContext
			handler := createTestHandler(t, &contextAuth, &contextUser)

			// Wrap handler with middleware
			wrappedHandler := middleware.Middleware(handler)

			// Create request
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			// Record response
			recorder := httptest.NewRecorder()
			wrappedHandler.ServeHTTP(recorder, req)

			// Check status code
			if recorder.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, recorder.Code)
			}

			// Check response body
			if tt.responseContains != "" {
				body := recorder.Body.String()
				if !strings.Contains(body, tt.responseContains) {
					t.Errorf("expected response to contain %q, got: %s", tt.responseContains, body)
				}
			}

			// Check context for successful requests
			if tt.expectedStatus == http.StatusOK {
				if contextAuth != tt.expectedAuth {
					t.Errorf("expected context auth %v, got %v", tt.expectedAuth, contextAuth)
				}

				if tt.expectedAuth {
					if contextUser == nil {
						t.Error("expected user context but got nil")
					} else if contextUser.UserID != tt.expectedUserID {
						t.Errorf("expected UserID %q, got %q", tt.expectedUserID, contextUser.UserID)
					}
				} else if contextUser != nil {
					t.Error("expected nil user context but got user")
				}
			}
		})
	}
}

func TestMiddleware_OptionalAuth(t *testing.T) {
	privateKey, publicKeyPEM := generateMiddlewareTestKeyPair(t)

	// Create middleware with optional authentication
	config := &BearerAuthConfig{
		Enabled:   true,
		PublicKey: publicKeyPEM,
		Algorithm: "RS256",
		Required:  false,
	}
	middleware, err := createTestMiddleware(t, config)
	if err != nil {
		t.Fatalf("failed to create test middleware: %v", err)
	}
	defer middleware.Close()

	// Test anonymous request
	var contextAuth bool
	var contextUser *UserContext
	handler := createTestHandler(t, &contextAuth, &contextUser)
	wrappedHandler := middleware.Middleware(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	recorder := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Errorf("expected status %d for anonymous request, got %d", http.StatusOK, recorder.Code)
	}
	if contextAuth {
		t.Error("expected context auth false for anonymous request")
	}
	if contextUser != nil {
		t.Error("expected nil user context for anonymous request")
	}

	// Test authenticated request
	claims := &TokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "test-user",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		UserID: "test-user",
	}
	validToken := createTestJWT(t, privateKey, claims)

	contextAuth = false
	contextUser = nil
	req = httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+validToken)
	recorder = httptest.NewRecorder()
	wrappedHandler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Errorf("expected status %d for authenticated request, got %d", http.StatusOK, recorder.Code)
	}
	if !contextAuth {
		t.Error("expected context auth true for authenticated request")
	}
	if contextUser == nil {
		t.Error("expected user context for authenticated request")
	}
}

func TestHandleAuthError(t *testing.T) {
	_, publicKeyPEM := generateMiddlewareTestKeyPair(t)

	config := &BearerAuthConfig{
		Enabled:   true,
		PublicKey: publicKeyPEM,
		Algorithm: "RS256",
		Required:  true,
	}
	middleware, err := createTestMiddleware(t, config)
	if err != nil {
		t.Fatalf("failed to create test middleware: %v", err)
	}
	defer middleware.Close()

	tests := []struct {
		name               string
		error              error
		expectedStatus     int
		expectedMessage    string
	}{
		{
			name:            "missing token error",
			error:           &AuthError{Type: "missing_token", Message: "Authorization header required"},
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Authorization required",
		},
		{
			name:            "invalid format error",
			error:           &AuthError{Type: "invalid_format", Message: "Authorization header must use Bearer format"},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "Invalid authorization format",
		},
		{
			name:            "empty token error",
			error:           &AuthError{Type: "empty_token", Message: "Bearer token cannot be empty"},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "Empty bearer token",
		},
		{
			name:            "insufficient scopes error",
			error:           fmt.Errorf("insufficient scopes: requires admin"),
			expectedStatus:  http.StatusForbidden,
			expectedMessage: "Insufficient permissions",
		},
		{
			name:            "expired token error",
			error:           fmt.Errorf("token expired at 2024-01-01"),
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Token expired",
		},
		{
			name:            "signature error",
			error:           fmt.Errorf("invalid signature"),
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Invalid token signature",
		},
		{
			name:            "generic error",
			error:           fmt.Errorf("some other error"),
			expectedStatus:  http.StatusUnauthorized,
			expectedMessage: "Authentication failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			middleware.handleAuthError(recorder, tt.error)

			if recorder.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, recorder.Code)
			}

			body := strings.TrimSpace(recorder.Body.String())
			if !strings.Contains(body, tt.expectedMessage) {
				t.Errorf("expected response to contain %q, got: %s", tt.expectedMessage, body)
			}
		})
	}
}

func TestMiddleware_Close(t *testing.T) {
	_, publicKeyPEM := generateMiddlewareTestKeyPair(t)

	config := &BearerAuthConfig{
		Enabled:   true,
		PublicKey: publicKeyPEM,
		Algorithm: "RS256",
		Required:  true,
	}

	middleware, err := createTestMiddleware(t, config)
	if err != nil {
		t.Fatalf("failed to create test middleware: %v", err)
	}

	// Close should not return error for static key configuration
	err = middleware.Close()
	if err != nil {
		t.Errorf("unexpected error on close: %v", err)
	}
}

func TestAuthError(t *testing.T) {
	err := &AuthError{
		Type:    "test_error",
		Message: "This is a test error",
	}

	if err.Error() != "This is a test error" {
		t.Errorf("expected error message 'This is a test error', got %q", err.Error())
	}
}