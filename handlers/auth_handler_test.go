package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fitreg/api/models"
	"github.com/fitreg/api/services"
)

type mockAuthService struct {
	googleLoginFn func(credential string) (*services.AuthResponse, error)
}

func (m *mockAuthService) GoogleLogin(credential string) (*services.AuthResponse, error) {
	return m.googleLoginFn(credential)
}

// --- GoogleLogin ---

func TestAuthHandler_GoogleLogin_ReturnsToken(t *testing.T) {
	mock := &mockAuthService{
		googleLoginFn: func(credential string) (*services.AuthResponse, error) {
			return &services.AuthResponse{
				Token: "jwt-token",
				User:  &models.UserProfile{ID: 1, Name: "Alice"},
			}, nil
		},
	}
	h := NewAuthHandler(mock)

	body, _ := json.Marshal(GoogleLoginRequest{Credential: "google-id-token"})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/auth/google", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")

	h.GoogleLogin(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp services.AuthResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Token != "jwt-token" {
		t.Errorf("expected token 'jwt-token', got %q", resp.Token)
	}
	if resp.User == nil || resp.User.Name != "Alice" {
		t.Errorf("unexpected user: %+v", resp.User)
	}
}

func TestAuthHandler_GoogleLogin_MissingCredential_Returns400(t *testing.T) {
	h := NewAuthHandler(&mockAuthService{})

	body, _ := json.Marshal(GoogleLoginRequest{Credential: ""})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/auth/google", bytes.NewReader(body))

	h.GoogleLogin(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAuthHandler_GoogleLogin_InvalidBody_Returns400(t *testing.T) {
	h := NewAuthHandler(&mockAuthService{})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/auth/google", bytes.NewReader([]byte("{bad json")))

	h.GoogleLogin(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAuthHandler_GoogleLogin_InvalidToken_Returns401(t *testing.T) {
	mock := &mockAuthService{
		googleLoginFn: func(credential string) (*services.AuthResponse, error) {
			return nil, errors.New("invalid google token")
		},
	}
	h := NewAuthHandler(mock)

	body, _ := json.Marshal(GoogleLoginRequest{Credential: "bad-token"})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/auth/google", bytes.NewReader(body))

	h.GoogleLogin(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}
