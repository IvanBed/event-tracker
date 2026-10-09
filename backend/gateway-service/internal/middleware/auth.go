package middleware

import (
	"context"
	"internal/auth"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

type contextKey string

const (
	ServiceIDKey contextKey = "serviceID"
)

// AuthMiddleware checks JWT tokens and adds service info to the request context
func AuthMiddleware(authService *auth.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Authorization header required", http.StatusUnauthorized)
				return
			}
			// Check Bearer token format
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, "Invalid authorization format", http.StatusUnauthorized)
				return
			}
			tokenString := parts[1]
			// Validate the token
			claims, err := authService.ValidateToken(tokenString)
			if err != nil {
				http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
				return
			}
			// Extract service ID from claims
			serviceIDStr, ok := claims["sub"].(string)
			if !ok {
				http.Error(w, "Invalid token claims", http.StatusUnauthorized)
				return
			}
			serviceID, err := uuid.Parse(serviceIDStr)
			if err != nil {
				http.Error(w, "Invalid service ID in token", http.StatusUnauthorized)
				return
			}
			// Add service ID to request context
			ctx := context.WithValue(r.Context(), ServiceIDKey, serviceID)
			// Call the next handler with the enhanced context
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetserviceID retrieves the service ID from the request context
func GetserviceID(r *http.Request) (uuid.UUID, bool) {
	serviceID, ok := r.Context().Value(ServiceIDKey).(uuid.UUID)
	return serviceID, ok
}
