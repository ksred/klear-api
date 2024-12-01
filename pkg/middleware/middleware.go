package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/time/rate"
)

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

var (
	visitors = make(map[string]*visitor)
	mu       sync.RWMutex

	// Configure limits per endpoint type
	authLimit    = rate.Limit(10.0 / 60.0)   // 10 requests per minute
	tradingLimit = rate.Limit(100.0 / 60.0)  // 100 requests per minute
	statusLimit  = rate.Limit(1000.0 / 60.0) // 1000 requests per minute
)

// Cleanup old visitors periodically
func init() {
	go cleanupVisitors()
}

func getLimiter(path, clientIP string) *rate.Limiter {
	mu.Lock()
	defer mu.Unlock()

	key := clientIP + ":" + path
	v, exists := visitors[key]

	if !exists {
		var limit rate.Limit
		switch {
		case strings.HasPrefix(path, "/api/v1/auth"):
			limit = authLimit
		case strings.HasPrefix(path, "/api/v1/orders"):
			limit = tradingLimit
		case strings.HasPrefix(path, "/api/v1/status"):
			limit = statusLimit
		default:
			limit = rate.Inf // No limit for other paths
		}

		v = &visitor{
			limiter:  rate.NewLimiter(limit, 1), // burst of 1
			lastSeen: time.Now(),
		}
		visitors[key] = v
	}

	v.lastSeen = time.Now()
	return v.limiter
}

func cleanupVisitors() {
	for {
		time.Sleep(time.Minute)

		mu.Lock()
		for ip, v := range visitors {
			if time.Since(v.lastSeen) > 3*time.Minute {
				delete(visitors, ip)
			}
		}
		mu.Unlock()
	}
}

func RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientID := c.GetString("clientID")
		if clientID == "" {
			clientID = c.ClientIP()
		}

		limiter := getLimiter(c.FullPath(), clientID)
		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "RATE_LIMIT_EXCEEDED",
					"message": "Rate limit exceeded. Please try again later.",
				},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientID, err := validateAndExtractToken(c)
		if err != nil {
			return
		}

		c.Set("clientID", clientID)
		c.Next()
	}
}

func validateAndExtractToken(c *gin.Context) (string, error) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization header required"})
		c.Abort()
		return "", fmt.Errorf("authorization header required")
	}

	bearerToken := strings.Split(authHeader, " ")
	if len(bearerToken) != 2 || strings.ToLower(bearerToken[0]) != "bearer" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header format"})
		c.Abort()
		return "", fmt.Errorf("invalid authorization header format")
	}

	tokenString := bearerToken[1]
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// TODO: Move this to configuration
		return []byte("klear-secret-key"), nil
	})

	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		c.Abort()
		return "", fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
		c.Abort()
		return "", fmt.Errorf("invalid token claims")
	}

	// Store claims in context for later use
	c.Set("clientID", claims["client_id"])
	c.Set("claims", claims)

	return claims["client_id"].(string), nil
}

func InternalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// For internal requests, we could use several possibilities depending on the implementation:
		// - IP whitelisting
		// - API key
		// - JWT token
		// For now, we will use a simple API key, the same as for the public API
		clientID, err := validateAndExtractToken(c)
		if err != nil {
			return
		}

		c.Set("clientID", clientID)
		c.Next()
	}
}
