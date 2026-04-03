package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/NaphonJangjit/HeartFolio/internal/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateAndValidateToken(t *testing.T) {
	secret := []byte("test-secret-key-for-testing-only")
	userID := "abc123"
	email := "user@test.com"
	role := "user"

	token, err := middleware.GenerateToken(secret, userID, email, role)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := middleware.ValidateToken(secret, token)
	require.NoError(t, err)

	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, email, claims.Email)
	assert.Equal(t, role, claims.Role)
}

func TestValidateToken_InvalidSecret(t *testing.T) {
	secret1 := []byte("secret-one")
	secret2 := []byte("secret-two")

	token, err := middleware.GenerateToken(secret1, "id", "email", "user")
	require.NoError(t, err)

	_, err = middleware.ValidateToken(secret2, token)
	assert.Error(t, err, "should fail with wrong secret")
}

func TestValidateToken_Garbage(t *testing.T) {
	_, err := middleware.ValidateToken([]byte("secret"), "not.a.jwt")
	assert.Error(t, err, "should fail with garbage token")
}

func TestContextHelpers(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, middleware.ContextKeyUserID, "user-123")
	ctx = context.WithValue(ctx, middleware.ContextKeyRole, "admin")

	assert.Equal(t, "user-123", middleware.UserIDFromContext(ctx))
	assert.Equal(t, "admin", middleware.RoleFromContext(ctx))
}

func TestAuth_MissingHeader(t *testing.T) {
	secret := []byte("test-secret")
	handler := middleware.Auth(secret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called without auth header")
	}))

	r := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuth_ValidToken(t *testing.T) {
	secret := []byte("test-secret")
	token, _ := middleware.GenerateToken(secret, "user-id", "user@test.com", "user")

	called := false
	handler := middleware.Auth(secret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		assert.Equal(t, "user-id", middleware.UserIDFromContext(r.Context()))
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	assert.True(t, called, "handler was not called with valid token")
}

func TestRequireAdmin_NonAdmin(t *testing.T) {
	handler := middleware.RequireAdmin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called for non-admin")
	}))

	ctx := context.WithValue(context.Background(), middleware.ContextKeyRole, "user")
	r := httptest.NewRequest(http.MethodGet, "/test", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestRequireAdmin_Admin(t *testing.T) {
	called := false
	handler := middleware.RequireAdmin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	ctx := context.WithValue(context.Background(), middleware.ContextKeyRole, "admin")
	r := httptest.NewRequest(http.MethodGet, "/test", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	assert.True(t, called, "handler was not called for admin")
}
