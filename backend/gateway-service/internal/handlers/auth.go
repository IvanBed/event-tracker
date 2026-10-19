package handlers

import (
	"encoding/json"
	"errors"
	"internal/auth"
	"net/http"
	"time"
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
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	// Parse the request body
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	// Define refresh token TTL (e.g., 7 days)
	refreshTokenTTL := 7 * 24 * time.Hour
	// Attempt to login with refresh token generation
	accessToken, refreshToken, err := h.authService.LoginWithRefresh(req.ServiceName, req.Password, refreshTokenTTL)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		} else {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}
	// Return the tokens
	response := LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type RefreshResponse struct {
	Token string `json:"access_token"`
}

func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	// Attempt to refresh the token
	token, err := h.authService.RefreshAccessToken(req.RefreshToken)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidToken) || errors.Is(err, auth.ErrExpiredToken) {
			http.Error(w, "Invalid or expired refresh token", http.StatusUnauthorized)
		} else {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}
	// Return the new access token
	response := RefreshResponse{Token: token}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
