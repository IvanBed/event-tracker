package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	_ "log"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"internal/crypto"
)

type ServiceDesc struct {
	Id          string
	ServiceName string
	Password    string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type RegisterRequest struct {
	ServiceName string `json:"ServiceName"`
	Password    string `json:"Password"`
}

type RegisterResponse struct {
	Id          string `json:"id"`
	ServiceName string `json:"ServiceName"`
}

type LoginRequest struct {
	ServiceName string `json:"ServiceName"`
	Password    string `json:"Password"`
}

type LoginResponse struct {
	Id          string `json:"id"`
	ServiceName string `json:"ServiceName"`
}

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid token")
	ErrExpiredToken       = errors.New("token has expired")
	ErrServiceInUse       = errors.New("service already in db")
)

type JwtConfig struct {
	jwtSecret      []byte
	accessTokenTTL time.Duration
}

var jwtConfig JwtConfig

func generateAccessToken(service *ServiceDesc) (string, error) {
	// Set the expiration time
	expirationTime := time.Now().Add(time.Duration(jwtConfig.accessTokenTTL))
	// Create the JWT claims
	claims := jwt.MapClaims{
		"sub":      service.Id,
		"username": service.ServiceName,
		"exp":      expirationTime.Unix(),
		"iat":      time.Now().Unix(),
	}
	// Create the token with claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	// Sign the token with our secret key
	tokenString, err := token.SignedString(jwtConfig.jwtSecret)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

// ValidateToken verifies a JWT token and returns the claims
func ValidateToken(tokenString string) (jwt.MapClaims, error) {
	// Parse the token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate the signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return authStorage.jwtSecret, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}
	// Extract and validate claims
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, ErrInvalidToken
}
func serviceCheck(serviceName string, password string) bool {
	hashedPassword, err := crypto.HashPassword(password)
	ctx := context.Background()

	res, err := CheckAuthentication(ctx, serviceName, hashedPassword)

	if err != nil {
		return false
	}
	return res
}

func registerService(req RegisterRequest) (*ServiceDesc, error) {
	_, err := GetServiceByName(req.ServiceName)
	if err == nil {
		return nil, ErrServiceInUse
	}

	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	// Hash the password
	hashedPassword, err := crypto.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	service, err := CreateService(context.Background(), req.ServiceName, hashedPassword)
	if err != nil {
		return nil, err
	}
	return service, nil
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	// Validate input
	if req.ServiceName == "" || req.Password == "" {
		http.Error(w, "ServiceName and password are required", http.StatusBadRequest)
		return
	}
	// Call the auth service to register the user
	serviceDesc, err := registerService(req)
	if err != nil {
		http.Error(w, "Error creating service account", http.StatusInternalServerError)
		return
	}
	// Return the created user (without sensitive data)
	response := RegisterResponse{
		Id:          serviceDesc.Id,
		ServiceName: serviceDesc.ServiceName,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)

}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var req LoginRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if req.ServiceName == "" || req.Password == "" {
		http.Error(w, "ServiceName and password are required", http.StatusBadRequest)
		return
	}

	if serviceCheck(req.ServiceName, req.Password) {
		tokenString, err := createToken(req.ServiceName)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Errorf("No username found")
			//log.Printf("No username found")
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, tokenString)
		return
	} else {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, "Invalid credentials")
		//log.Println("Invalid credentials")
	}
}

func ProtectedHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tokenString := r.Header.Get("Authorization")
	if tokenString == "" {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, "Missing authorization header")
		//log.Println("Missing authorization header")
		return
	}
	tokenString = tokenString[len("Bearer "):]

	err := verifyToken(tokenString)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, "Invalid token")
		//log.Println("Invalid token")
		return
	}

	fmt.Fprint(w, "Welcome to the the protected area")
}
