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
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/MicahParks/keyfunc/v2"
	"github.com/golang-jwt/jwt/v5"
)

// BearerTokenValidator is a lightweight wrapper around golang-jwt/jwt for MakeMCP-specific needs.
type BearerTokenValidator struct {
	config  *BearerAuthConfig
	keyFunc jwt.Keyfunc
	jwks    *keyfunc.JWKS
	parser  *jwt.Parser
}

// NewBearerTokenValidator creates a JWT validator that leverages golang-jwt/jwt's built-in validation.
func NewBearerTokenValidator(config *BearerAuthConfig) (*BearerTokenValidator, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid bearer auth configuration: %w", err)
	}

	validator := &BearerTokenValidator{
		config: config,
	}

	// Initialize key function (JWKS or static key)
	if err := validator.initializeKeyFunc(); err != nil {
		return nil, fmt.Errorf("failed to initialize key function: %w", err)
	}

	// Create parser with validation options - let golang-jwt/jwt handle the heavy lifting
	parserOptions := []jwt.ParserOption{
		jwt.WithValidMethods([]string{config.Algorithm}),
	}
	
	// Add optional issuer validation
	if config.Issuer != "" {
		parserOptions = append(parserOptions, jwt.WithIssuer(config.Issuer))
	}
	
	// Add optional audience validation
	if config.Audience != "" {
		parserOptions = append(parserOptions, jwt.WithAudience(config.Audience))
	}
	
	validator.parser = jwt.NewParser(parserOptions...)

	return validator, nil
}

// initializeKeyFunc sets up the key function (JWKS or static key).
func (v *BearerTokenValidator) initializeKeyFunc() error {
	if v.config.JWKSUri != "" {
		return v.initializeJWKS()
	}
	return v.initializeStaticKey()
}

// initializeJWKS sets up JWKS-based key validation using keyfunc library.
func (v *BearerTokenValidator) initializeJWKS() error {
	jwks, err := keyfunc.Get(v.config.JWKSUri, keyfunc.Options{
		RefreshInterval: time.Duration(v.config.CacheTTL) * time.Second,
	})
	if err != nil {
		return fmt.Errorf("failed to get JWKS from %s: %w", v.config.JWKSUri, err)
	}

	v.jwks = jwks
	v.keyFunc = jwks.Keyfunc
	return nil
}

// initializeStaticKey sets up static public key validation.
func (v *BearerTokenValidator) initializeStaticKey() error {
	publicKey, err := jwt.ParseRSAPublicKeyFromPEM([]byte(v.config.PublicKey))
	if err != nil {
		return fmt.Errorf("failed to parse RSA public key: %w", err)
	}

	v.keyFunc = func(token *jwt.Token) (any, error) {
		return publicKey, nil
	}

	return nil
}

// ValidateToken validates a JWT token and returns user context.
// This is a thin wrapper - golang-jwt/jwt does all the heavy lifting.
func (v *BearerTokenValidator) ValidateToken(tokenString string) (*UserContext, error) {
	// Let golang-jwt/jwt handle parsing, signature validation, and standard claims validation
	token, err := v.parser.ParseWithClaims(tokenString, &TokenClaims{}, v.keyFunc)
	if err != nil {
		return nil, fmt.Errorf("token validation failed: %w", err)
	}

	// Extract validated claims
	claims, ok := token.Claims.(*TokenClaims)
	if !ok {
		return nil, jwt.ErrTokenInvalidClaims
	}

	// Only validate our custom requirements (scopes)
	if err := v.validateScopes(claims.Scopes); err != nil {
		return nil, err
	}

	// Create user context from validated claims
	return &UserContext{
		UserID:   claims.UserID,
		Username: claims.Username,
		Email:    claims.Email,
		Token:    tokenString,
		Scopes:   claims.Scopes,
		Claims:   claims,
	}, nil
}

// validateScopes checks if the token has required scopes (our only custom validation).
func (v *BearerTokenValidator) validateScopes(tokenScopes []string) error {
	if len(v.config.RequiredScopes) == 0 {
		return nil // No scope validation required
	}

	for _, required := range v.config.RequiredScopes {
		if !slices.Contains(tokenScopes, required) {
			return fmt.Errorf("insufficient scopes: requires %s, has %s",
				strings.Join(v.config.RequiredScopes, ", "),
				strings.Join(tokenScopes, ", "))
		}
	}

	return nil
}

// Close cleans up resources.
func (v *BearerTokenValidator) Close() error {
	if v.jwks != nil {
		v.jwks.EndBackground()
	}
	return nil
}