package handler

import (
	"encoding/json"
	"net/http"
)

type UserHandler struct {
	repo userRepository
}

func NewUserHandler(repo userRepository) *UserHandler {
	return &UserHandler{repo: repo}
}

type createUserRequest struct {
	Email    string `json:"email"`
	FullName string `json:"full_name"`
	Pin      string `json:"pin"`
}

func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.Email == "" || req.FullName == "" || req.Pin == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "email, full_name, and pin are required"})
		return
	}
	if len(req.Pin) < 4 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "pin must be at least 4 characters"})
		return
	}

	// In production, hash the PIN with bcrypt. Using plain text here for the lab.
	// TODO: Add bcrypt hashing
	user, err := h.repo.Create(r.Context(), req.Email, req.FullName, req.Pin)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, user)
}

func (h *UserHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id is required"})
		return
	}

	user, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if user == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
		return
	}

	writeJSON(w, http.StatusOK, user)
}
