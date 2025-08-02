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
	"log"
	"net/http"
	"strings"
)

// BearerAuthMiddleware provides HTTP middleware for JWT Bearer token authentication.
type BearerAuthMiddleware struct {
	validator *BearerTokenValidator
	config    *BearerAuthConfig
}

// NewBearerAuthMiddleware creates HTTP middleware for Bearer token authentication.
func NewBearerAuthMiddleware(config *BearerAuthConfig) (*BearerAuthMiddleware, error) {
	validator, err := NewBearerTokenValidator(config)
	if err != nil {
		return nil, err
	}

	return &BearerAuthMiddleware{
		validator: validator,
		config:    config,
	}, nil
}

// ExtractAndValidateToken extracts and validates Bearer token from HTTP request.
func (m *BearerAuthMiddleware) ExtractAndValidateToken(req *http.Request) (*UserContext, error) {
	// Extract Authorization header  
	authHeader := req.Header.Get("Authorization")
	if authHeader == "" {
		if m.config.Required {
			return nil, &AuthError{Type: "missing_token", Message: "Authorization header required"}
		}
		return nil, nil // Anonymous access allowed
	}

	// Validate Bearer token format
	const bearerPrefix = "Bearer "
	if !strings.HasPrefix(authHeader, bearerPrefix) {
		return nil, &AuthError{Type: "invalid_format", Message: "Authorization header must use Bearer format"}
	}

	// Extract token and validate
	tokenString := strings.TrimPrefix(authHeader, bearerPrefix)
	if tokenString == "" {
		return nil, &AuthError{Type: "empty_token", Message: "Bearer token cannot be empty"}
	}

	// Use our validator to validate the token (leveraging golang-jwt/jwt)
	return m.validator.ValidateToken(tokenString)
}

// Middleware returns an HTTP middleware function that validates Bearer tokens.
func (m *BearerAuthMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract and validate token
		userCtx, err := m.ExtractAndValidateToken(r)
		if err != nil {
			m.handleAuthError(w, err)
			return
		}

		// Add user context to request context if authentication succeeded
		if userCtx != nil {
			r = r.WithContext(WithUserContext(r.Context(), userCtx))
			log.Printf("Authenticated request from user: %s", userCtx.GetDisplayName())
		}

		// Continue to next handler
		next.ServeHTTP(w, r)
	})
}

// handleAuthError sends appropriate HTTP error response for authentication failures.
func (m *BearerAuthMiddleware) handleAuthError(w http.ResponseWriter, err error) {
	// Log the error
	log.Printf("Authentication error: %v", err)

	// Check if it's our custom AuthError type first
	if authErr, ok := err.(*AuthError); ok {
		switch authErr.Type {
		case "missing_token":
			http.Error(w, "Authorization required", http.StatusUnauthorized)
		case "invalid_format":
			http.Error(w, "Invalid authorization format", http.StatusBadRequest)
		case "empty_token":
			http.Error(w, "Empty bearer token", http.StatusBadRequest)
		default:
			http.Error(w, "Authentication failed", http.StatusUnauthorized)
		}
		return
	}

	// For other error types, check error message content
	switch {
	case strings.Contains(err.Error(), "insufficient scopes"):
		http.Error(w, "Insufficient permissions", http.StatusForbidden)
	case strings.Contains(err.Error(), "expired"):
		http.Error(w, "Token expired", http.StatusUnauthorized)
	case strings.Contains(err.Error(), "signature"):
		http.Error(w, "Invalid token signature", http.StatusUnauthorized)
	default:
		http.Error(w, "Authentication failed", http.StatusUnauthorized)
	}
}

// Close cleans up middleware resources.
func (m *BearerAuthMiddleware) Close() error {
	return m.validator.Close()
}

// AuthError represents authentication middleware errors.
type AuthError struct {
	Type    string
	Message string
}

// Error implements the error interface.
func (e *AuthError) Error() string {
	return e.Message
}