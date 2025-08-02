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
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// contextKey is a type for context keys to avoid collisions.
type contextKey string

const (
	// UserContextKey is the context key for authenticated user information.
	UserContextKey contextKey = "auth.user"
)

// TokenClaims represents validated JWT claims with standard and custom fields.
type TokenClaims struct {
	jwt.RegisteredClaims
	Scopes   []string `json:"scope,omitempty"`
	UserID   string   `json:"sub"`
	Username string   `json:"preferred_username,omitempty"`
	Email    string   `json:"email,omitempty"`
}

// UserContext holds authenticated user information extracted from a valid JWT token.
type UserContext struct {
	// User identification
	UserID   string `json:"userId"`
	Username string `json:"username,omitempty"`
	Email    string `json:"email,omitempty"`

	// Authentication context
	Token  string       `json:"-"`     // Original JWT token (not serialized)
	Scopes []string     `json:"scopes"` // User's authorized scopes
	Claims *TokenClaims `json:"-"`     // Full token claims (not serialized)
}

// HasScope checks if the user has a specific scope.
func (u *UserContext) HasScope(scope string) bool {
	return slices.Contains(u.Scopes, scope)
}

// HasAnyScope checks if the user has any of the provided scopes.
func (u *UserContext) HasAnyScope(scopes []string) bool {
	return slices.ContainsFunc(scopes, u.HasScope)
}

// HasAllScopes checks if the user has all of the provided scopes.
func (u *UserContext) HasAllScopes(scopes []string) bool {
	for _, scope := range scopes {
		if !u.HasScope(scope) {
			return false
		}
	}
	return true
}

// GetDisplayName returns the best available display name for the user.
func (u *UserContext) GetDisplayName() string {
	if u.Username != "" {
		return u.Username
	}
	if u.Email != "" {
		return u.Email
	}
	return u.UserID
}

// WithUserContext adds user context to a Go context.
func WithUserContext(ctx context.Context, userCtx *UserContext) context.Context {
	return context.WithValue(ctx, UserContextKey, userCtx)
}

// GetUserContext extracts user context from a Go context.
// Returns nil if no authenticated user context is present.
func GetUserContext(ctx context.Context) *UserContext {
	if userCtx, ok := ctx.Value(UserContextKey).(*UserContext); ok {
		return userCtx
	}
	return nil
}

// IsAuthenticated checks if the context contains an authenticated user.
func IsAuthenticated(ctx context.Context) bool {
	return GetUserContext(ctx) != nil
}

// RequireAuthentication returns an error if the context is not authenticated.
func RequireAuthentication(ctx context.Context) (*UserContext, error) {
	userCtx := GetUserContext(ctx)
	if userCtx == nil {
		return nil, fmt.Errorf("authentication required")
	}
	return userCtx, nil
}

// RequireScope returns an error if the user doesn't have the required scope.
func RequireScope(ctx context.Context, scope string) (*UserContext, error) {
	userCtx, err := RequireAuthentication(ctx)
	if err != nil {
		return nil, err
	}

	if !userCtx.HasScope(scope) {
		return nil, fmt.Errorf("insufficient scopes: requires %s", scope)
	}

	return userCtx, nil
}

// RequireAnyScope returns an error if the user doesn't have any of the required scopes.
func RequireAnyScope(ctx context.Context, scopes []string) (*UserContext, error) {
	userCtx, err := RequireAuthentication(ctx)
	if err != nil {
		return nil, err
	}

	if !userCtx.HasAnyScope(scopes) {
		return nil, fmt.Errorf("insufficient scopes: requires one of %s", strings.Join(scopes, ", "))
	}

	return userCtx, nil
}

// RequireAllScopes returns an error if the user doesn't have all of the required scopes.
func RequireAllScopes(ctx context.Context, scopes []string) (*UserContext, error) {
	userCtx, err := RequireAuthentication(ctx)
	if err != nil {
		return nil, err
	}

	if !userCtx.HasAllScopes(scopes) {
		return nil, fmt.Errorf("insufficient scopes: requires %s", strings.Join(scopes, ", "))
	}

	return userCtx, nil
}