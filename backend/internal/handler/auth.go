package handler

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"
)

// AuthHandler wires the /auth/* endpoints used by the React frontend.
// This is a simplified implementation for the system-design lab — tokens are
// random hex strings (no JWT signing) and passwords are stored as plain text
// in the pin_hash column. Do NOT use this in production.
type AuthHandler struct {
	repo userRepository
}

func NewAuthHandler(repo userRepository) *AuthHandler {
	return &AuthHandler{repo: repo}
}

// generateToken produces a random 32-byte hex token.
func generateToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// Register handles POST /auth/register
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username    string `json:"username"`
		Email       string `json:"email"`
		Password    string `json:"password"`
		FullName    string `json:"fullName"`
		PhoneNumber string `json:"phoneNumber"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.Email == "" || req.FullName == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "email, fullName, and password are required"})
		return
	}
	if len(req.Password) < 8 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "password must be at least 8 characters"})
		return
	}

	user, err := h.repo.Create(r.Context(), req.Email, req.FullName, req.Password)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{
		"userId":    user.ID,
		"username":  req.Username,
		"email":     user.Email,
		"fullName":  user.FullName,
		"createdAt": user.CreatedAt.Format(time.RFC3339),
	})
}

// Login handles POST /auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.Username == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "username and password are required"})
		return
	}

	// Look up by email (the frontend uses the email address as the "username")
	user, err := h.repo.GetByEmail(r.Context(), req.Username)
	if err != nil || user == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Invalid credentials"})
		return
	}

	// Plain-text comparison (lab only — pin_hash stores the password as-is)
	if user.PinHash != req.Password {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Invalid credentials"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"accessToken":  generateToken(),
		"refreshToken": generateToken(),
		"tokenType":    "Bearer",
		"expiresIn":    3600,
		"userId":       user.ID,
		"username":     user.Email,
	})
}

// Logout handles POST /auth/logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "logged out"})
}
