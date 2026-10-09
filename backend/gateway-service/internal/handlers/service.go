package auth

import (
	"encoding/json"
	"errors"
	"internal/auth"
	"net/http"
)

type AuthHandler struct {
	authService *auth.AuthService
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(authService *auth.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

type RegisterRequest struct {
	ServiceName string `json:"ServiceName"`
	Password    string `json:"Password"`
}

type RegisterResponse struct {
	Id          string `json:"id"`
	ServiceName string `json:"ServiceName"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	// Parse the request body
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	// Validate input
	if req.ServiceName == "" || req.Password == "" {
		http.Error(w, "Service name and password are required", http.StatusBadRequest)
		return
	}

	service, err := h.authService.Register(req.ServiceName, req.Password)
	if err != nil {
		if errors.Is(err, auth.ErrServiceInUse) {
			http.Error(w, "Service name  already in use", http.StatusConflict)
			return
		}
		http.Error(w, "Error creating service", http.StatusInternalServerError)
		return
	}

	response := RegisterResponse{
		Id:          service.ID.String(),
		ServiceName: service.ServiceName,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

type LoginRequest struct {
	ServiceName string `json:"ServiceName"`
	Password    string `json:"Password"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	// Parse the request body
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	// Attempt to login
	token, err := h.authService.Login(req.ServiceName, req.Password)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		} else {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}
	// Return the token
	response := LoginResponse{Token: token}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
