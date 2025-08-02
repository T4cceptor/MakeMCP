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
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestUserContext_HasScope(t *testing.T) {
	userCtx := &UserContext{
		Scopes: []string{"read", "write", "admin"},
	}

	tests := []struct {
		scope    string
		expected bool
	}{
		{"read", true},
		{"write", true},
		{"admin", true},
		{"delete", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run("scope_"+tt.scope, func(t *testing.T) {
			result := userCtx.HasScope(tt.scope)
			if result != tt.expected {
				t.Errorf("HasScope(%q) = %t, expected %t", tt.scope, result, tt.expected)
			}
		})
	}
}

func TestUserContext_HasAnyScope(t *testing.T) {
	userCtx := &UserContext{
		Scopes: []string{"read", "write"},
	}

	tests := []struct {
		name     string
		scopes   []string
		expected bool
	}{
		{
			name:     "has one of multiple scopes",
			scopes:   []string{"read", "admin"},
			expected: true,
		},
		{
			name:     "has all scopes",
			scopes:   []string{"read", "write"},
			expected: true,
		},
		{
			name:     "has none of the scopes",
			scopes:   []string{"admin", "delete"},
			expected: false,
		},
		{
			name:     "empty scope list",
			scopes:   []string{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := userCtx.HasAnyScope(tt.scopes)
			if result != tt.expected {
				t.Errorf("HasAnyScope(%v) = %t, expected %t", tt.scopes, result, tt.expected)
			}
		})
	}
}

func TestUserContext_HasAllScopes(t *testing.T) {
	userCtx := &UserContext{
		Scopes: []string{"read", "write", "admin"},
	}

	tests := []struct {
		name     string
		scopes   []string
		expected bool
	}{
		{
			name:     "has all scopes",
			scopes:   []string{"read", "write"},
			expected: true,
		},
		{
			name:     "missing one scope",
			scopes:   []string{"read", "delete"},
			expected: false,
		},
		{
			name:     "has no scopes from list",
			scopes:   []string{"delete", "create"},
			expected: false,
		},
		{
			name:     "empty scope list",
			scopes:   []string{},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := userCtx.HasAllScopes(tt.scopes)
			if result != tt.expected {
				t.Errorf("HasAllScopes(%v) = %t, expected %t", tt.scopes, result, tt.expected)
			}
		})
	}
}

func TestUserContext_GetDisplayName(t *testing.T) {
	tests := []struct {
		name     string
		userCtx  *UserContext
		expected string
	}{
		{
			name: "with username",
			userCtx: &UserContext{
				UserID:   "user123",
				Username: "john.doe",
				Email:    "john@example.com",
			},
			expected: "john.doe",
		},
		{
			name: "with email only",
			userCtx: &UserContext{
				UserID: "user123",
				Email:  "john@example.com",
			},
			expected: "john@example.com",
		},
		{
			name: "with user ID only",
			userCtx: &UserContext{
				UserID: "user123",
			},
			expected: "user123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.userCtx.GetDisplayName()
			if result != tt.expected {
				t.Errorf("GetDisplayName() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestContextManagement(t *testing.T) {
	userCtx := &UserContext{
		UserID:   "user123",
		Username: "testuser",
		Email:    "test@example.com",
		Scopes:   []string{"read", "write"},
	}

	t.Run("context with user", func(t *testing.T) {
		ctx := context.Background()
		ctx = WithUserContext(ctx, userCtx)

		// Test retrieval
		retrieved := GetUserContext(ctx)
		if retrieved == nil {
			t.Fatal("expected user context to be retrieved")
		}

		if retrieved.UserID != userCtx.UserID {
			t.Errorf("expected UserID %q, got %q", userCtx.UserID, retrieved.UserID)
		}

		// Test IsAuthenticated
		if !IsAuthenticated(ctx) {
			t.Error("expected context to be authenticated")
		}
	})

	t.Run("context without user", func(t *testing.T) {
		ctx := context.Background()

		// Test retrieval
		retrieved := GetUserContext(ctx)
		if retrieved != nil {
			t.Error("expected no user context")
		}

		// Test IsAuthenticated
		if IsAuthenticated(ctx) {
			t.Error("expected context not to be authenticated")
		}
	})
}

func TestRequireAuthentication(t *testing.T) {
	userCtx := &UserContext{
		UserID: "user123",
		Scopes: []string{"read"},
	}

	t.Run("authenticated context", func(t *testing.T) {
		ctx := WithUserContext(context.Background(), userCtx)

		result, err := RequireAuthentication(ctx)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if result == nil {
			t.Error("expected user context")
		} else if result.UserID != userCtx.UserID {
			t.Errorf("expected UserID %q, got %q", userCtx.UserID, result.UserID)
		}
	})

	t.Run("unauthenticated context", func(t *testing.T) {
		ctx := context.Background()

		result, err := RequireAuthentication(ctx)
		if err == nil {
			t.Error("expected authentication error")
		}
		if result != nil {
			t.Error("expected no user context")
		}
		if !strings.Contains(err.Error(), "authentication required") {
			t.Errorf("expected authentication required error, got: %v", err)
		}
	})
}

func TestRequireScope(t *testing.T) {
	userCtx := &UserContext{
		UserID: "user123",
		Scopes: []string{"read", "write"},
	}

	t.Run("user has required scope", func(t *testing.T) {
		ctx := WithUserContext(context.Background(), userCtx)

		result, err := RequireScope(ctx, "read")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if result == nil {
			t.Error("expected user context")
		}
	})

	t.Run("user missing required scope", func(t *testing.T) {
		ctx := WithUserContext(context.Background(), userCtx)

		result, err := RequireScope(ctx, "admin")
		if err == nil {
			t.Error("expected authorization error")
		}
		if result != nil {
			t.Error("expected no user context")
		}
		if !strings.Contains(err.Error(), "insufficient scopes") {
			t.Errorf("expected insufficient scopes error, got: %v", err)
		}
	})

	t.Run("unauthenticated context", func(t *testing.T) {
		ctx := context.Background()

		result, err := RequireScope(ctx, "read")
		if err == nil {
			t.Error("expected authentication error")
		}
		if result != nil {
			t.Error("expected no user context")
		}
	})
}

func TestRequireAnyScope(t *testing.T) {
	userCtx := &UserContext{
		UserID: "user123",
		Scopes: []string{"read", "write"},
	}

	t.Run("user has one of required scopes", func(t *testing.T) {
		ctx := WithUserContext(context.Background(), userCtx)

		result, err := RequireAnyScope(ctx, []string{"read", "admin"})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if result == nil {
			t.Error("expected user context")
		}
	})

	t.Run("user has none of required scopes", func(t *testing.T) {
		ctx := WithUserContext(context.Background(), userCtx)

		result, err := RequireAnyScope(ctx, []string{"admin", "delete"})
		if err == nil {
			t.Error("expected authorization error")
		}
		if result != nil {
			t.Error("expected no user context")
		}
		if !strings.Contains(err.Error(), "insufficient scopes") {
			t.Errorf("expected insufficient scopes error, got: %v", err)
		}
	})
}

func TestRequireAllScopes(t *testing.T) {
	userCtx := &UserContext{
		UserID: "user123",
		Scopes: []string{"read", "write", "admin"},
	}

	t.Run("user has all required scopes", func(t *testing.T) {
		ctx := WithUserContext(context.Background(), userCtx)

		result, err := RequireAllScopes(ctx, []string{"read", "write"})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if result == nil {
			t.Error("expected user context")
		}
	})

	t.Run("user missing one of required scopes", func(t *testing.T) {
		ctx := WithUserContext(context.Background(), userCtx)

		result, err := RequireAllScopes(ctx, []string{"read", "delete"})
		if err == nil {
			t.Error("expected authorization error")
		}
		if result != nil {
			t.Error("expected no user context")
		}
		if !strings.Contains(err.Error(), "insufficient scopes") {
			t.Errorf("expected insufficient scopes error, got: %v", err)
		}
	})
}

func TestTokenClaims(t *testing.T) {
	// Test that TokenClaims properly embeds jwt.RegisteredClaims
	claims := &TokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "https://auth.example.com",
			Subject:   "user123",
			Audience:  []string{"test-audience"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		UserID:   "user123",
		Username: "testuser",
		Email:    "test@example.com",
		Scopes:   []string{"read", "write"},
	}

	// Test that we can access standard claims
	if claims.Issuer != "https://auth.example.com" {
		t.Errorf("expected issuer %q, got %q", "https://auth.example.com", claims.Issuer)
	}

	if claims.Subject != "user123" {
		t.Errorf("expected subject %q, got %q", "user123", claims.Subject)
	}

	// Test that we can access custom claims
	if claims.UserID != "user123" {
		t.Errorf("expected UserID %q, got %q", "user123", claims.UserID)
	}

	if len(claims.Scopes) != 2 {
		t.Errorf("expected 2 scopes, got %d", len(claims.Scopes))
	}
}