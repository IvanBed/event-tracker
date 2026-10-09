package auth

import (
	"context"
	"database/sql"
	"errors"
	_ "log"
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

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid token")
	ErrExpiredToken       = errors.New("token has expired")
	ErrServiceInUse       = errors.New("service already in db")
)

type AuthService struct {
	ServiceRepo    *ServiceRepo
	JwtSecret      []byte
	AccessTokenTTL time.Duration
}

func NewAuthService(serviceRepo *ServiceRepo, jwtSecret string, accessTokenTTL time.Duration) *AuthService {
	return &AuthService{
		ServiceRepo:    serviceRepo,
		JwtSecret:      []byte(jwtSecret),
		AccessTokenTTL: accessTokenTTL,
	}
}

//, jwtSecret: []byte("test-secret"), accessTokenTTL: 25

func (authService *AuthService) generateAccessToken(service *ServiceDesc) (string, error) {
	// Set the expiration time
	expirationTime := time.Now().Add(time.Duration(authService.AccessTokenTTL))
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
	tokenString, err := token.SignedString(authService.JwtSecret)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

// ValidateToken verifies a JWT token and returns the claims
func (authService *AuthService) ValidateToken(tokenString string) (jwt.MapClaims, error) {
	// Parse the token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate the signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return authService.JwtSecret, nil
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

func (authService *AuthService) Login(serviceName, password string) (string, error) {

	service, err := authService.ServiceRepo.GetServiceByName(serviceName)
	if err != nil {
		return "", ErrInvalidCredentials
	}
	// Verify the password
	if err := crypto.CheckPasswordHash(password, service.Password); err != nil {
		return "", ErrInvalidCredentials
	}

	token, err := authService.generateAccessToken(service)
	if err != nil {
		return "", err
	}
	return token, nil
}

func (authService *AuthService) Register(serviceName, password string) (*ServiceDesc, error) {
	_, err := authService.ServiceRepo.GetServiceByName(serviceName)
	if err == nil {
		return nil, ErrServiceInUse
	}

	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	hashedPassword, err := crypto.HashPassword(password)
	if err != nil {
		return nil, err
	}

	service, err := authService.ServiceRepo.CreateService(context.Background(), serviceName, hashedPassword)
	if err != nil {
		return nil, err
	}
	return service, nil
}
