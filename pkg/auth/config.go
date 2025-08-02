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
)

// BearerAuthConfig holds JWT Bearer token authentication configuration.
type BearerAuthConfig struct {
	// Enabled determines if Bearer token authentication is active
	Enabled bool `json:"enabled"`

	// Token validation options (mutually exclusive)
	JWKSUri   string `json:"jwksUri,omitempty"`   // JWKS endpoint for key discovery
	PublicKey string `json:"publicKey,omitempty"` // Direct RSA public key (PEM format)

	// JWT validation parameters
	Algorithm string `json:"algorithm"` // JWT signing algorithm (RS256, RS512, etc.)

	// Claims validation
	Issuer         string   `json:"issuer,omitempty"`         // Expected token issuer
	Audience       string   `json:"audience,omitempty"`       // Expected token audience
	RequiredScopes []string `json:"requiredScopes,omitempty"` // Required scopes in token

	// Behavior configuration
	Required bool `json:"required"` // Whether authentication is mandatory
	CacheTTL int  `json:"cacheTtl"` // JWKS cache TTL in seconds
}

// Validate checks the configuration for consistency and completeness.
func (c *BearerAuthConfig) Validate() error {
	if !c.Enabled {
		return nil // No validation needed for disabled auth
	}

	// Must have either JWKS URI or public key
	if c.JWKSUri == "" && c.PublicKey == "" {
		return fmt.Errorf("either jwksUri or publicKey must be provided when authentication is enabled")
	}

	// Cannot have both JWKS URI and public key
	if c.JWKSUri != "" && c.PublicKey != "" {
		return fmt.Errorf("cannot specify both jwksUri and publicKey, choose one")
	}

	// Validate JWKS URI format
	if c.JWKSUri != "" {
		if !strings.HasPrefix(c.JWKSUri, "https://") {
			return fmt.Errorf("jwksUri must use HTTPS")
		}
	}

	// Validate algorithm
	if c.Algorithm == "" {
		c.Algorithm = "RS256" // Set default
	}
	if !isValidAlgorithm(c.Algorithm) {
		return fmt.Errorf("unsupported JWT algorithm: %s", c.Algorithm)
	}

	// Validate cache TTL
	if c.CacheTTL <= 0 {
		c.CacheTTL = 300 // Default 5 minutes
	}
	if c.CacheTTL > 3600 {
		return fmt.Errorf("cacheTtl cannot exceed 3600 seconds (1 hour)")
	}

	// Validate issuer format if provided (issuer validation is optional)
	if c.Issuer != "" && !strings.HasPrefix(c.Issuer, "https://") && !strings.HasPrefix(c.Issuer, "http://") {
		return fmt.Errorf("issuer must be a valid URL (if provided)")
	}

	return nil
}

// isValidAlgorithm checks if the JWT algorithm is supported.
func isValidAlgorithm(alg string) bool {
	supportedAlgorithms := []string{
		"RS256", "RS384", "RS512",
		"ES256", "ES384", "ES512",
		"PS256", "PS384", "PS512",
	}

	return slices.Contains(supportedAlgorithms, alg)
}

// GetKeySource returns a description of the key source for logging.
func (c *BearerAuthConfig) GetKeySource() string {
	if c.JWKSUri != "" {
		return fmt.Sprintf("JWKS from %s", c.JWKSUri)
	}
	if c.PublicKey != "" {
		return "Static public key"
	}
	return "No key source configured"
}

// HasScopeValidation returns true if scope validation is configured.
func (c *BearerAuthConfig) HasScopeValidation() bool {
	return len(c.RequiredScopes) > 0
}