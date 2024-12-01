package auth

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidCredentials = errors.New("invalid API credentials")
	ErrTokenGeneration   = errors.New("failed to generate token")
)

// Test credentials
var (
	TestAPIKey = "test-api-key"
	TestAPISecret = "test-api-secret"
)

// Credentials represents the API authentication credentials
type Credentials struct {
	APIKey    string `json:"api_key"`
	APISecret string `json:"api_secret"`
}

// TokenResponse represents the JWT token response
type TokenResponse struct {
	Token      string    `json:"jwt_token"`
	Expiration time.Time `json:"expiration"`
}

// Claims represents the JWT claims structure
type Claims struct {
	jwt.RegisteredClaims
	ClientID    string   `json:"client_id"`
	Permissions []string `json:"permissions"`
}

// Service handles authentication operations
type Service struct {
	jwtSecret []byte
	// In a real implementation, this would be replaced with a database
	apiCredentials map[string]string // map[APIKey]APISecret
}

// NewService creates a new authentication service
func NewService(jwtSecret string) *Service {
	return &Service{
		jwtSecret: []byte(jwtSecret),
		// This is just for demonstration - in production, use a proper database
		apiCredentials: make(map[string]string),
	}
}

// GenerateToken generates a JWT token for valid API credentials
func (s *Service) GenerateToken(creds Credentials) (*TokenResponse, error) {
	// Verify API credentials
	if !s.validateCredentials(creds) {
		return nil, ErrInvalidCredentials
	}

	// Create token expiration time (24 hours from now)
	expiration := time.Now().Add(24 * time.Hour)

	// Create the claims
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiration),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
		ClientID:    creds.APIKey, // Using API key as client ID for simplicity
		Permissions: []string{"trade"}, // Default permission
	}

	// Create the token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign the token
	tokenString, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return nil, ErrTokenGeneration
	}

	return &TokenResponse{
		Token:      tokenString,
		Expiration: expiration,
	}, nil
}

// ValidateToken validates a JWT token and returns the claims
func (s *Service) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// validateCredentials checks if the API credentials are valid
func (s *Service) validateCredentials(creds Credentials) bool {
	secret, exists := s.apiCredentials[creds.APIKey]
	return exists && secret == creds.APISecret
}

// RegisterAPICredentials registers new API credentials (for testing/demo purposes)
func (s *Service) RegisterAPICredentials(apiKey, apiSecret string) {
	s.apiCredentials[apiKey] = apiSecret
}

// GinHandlers contains HTTP handlers for auth endpoints
type GinHandlers struct {
	service *Service
}

// NewGinHandlers creates new HTTP handlers for auth endpoints
func NewGinHandlers(service *Service) *GinHandlers {
	return &GinHandlers{
		service: service,
	}
}

// GenerateTokenHandler handles the token generation HTTP request
func (h *GinHandlers) GenerateTokenHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var creds Credentials
		if err := c.ShouldBindJSON(&creds); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}

		token, err := h.service.GenerateToken(creds)
		if err != nil {
			status := http.StatusInternalServerError
			if err == ErrInvalidCredentials {
				status = http.StatusUnauthorized
			}
			c.JSON(status, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, token)
	}
}
