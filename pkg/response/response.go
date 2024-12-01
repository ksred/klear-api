package response

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Response represents a standardized API response
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *Error      `json:"error,omitempty"`
}

// Error represents an error response
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Common error codes
const (
	ErrCodeNotFound          = "NOT_FOUND"
	ErrCodeBadRequest        = "BAD_REQUEST"
	ErrCodeUnauthorized      = "UNAUTHORIZED"
	ErrCodeForbidden         = "FORBIDDEN"
	ErrCodeInternalError     = "INTERNAL_ERROR"
	ErrCodeValidationFailed  = "VALIDATION_FAILED"
	ErrCodeDuplicateResource = "DUPLICATE_RESOURCE"
)

// Handle processes the error and returns appropriate response
func Handle(c *gin.Context, data interface{}, err error) {
	if err == nil {
		Success(c, data)
		return
	}

	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		NotFound(c, "Resource not found")
	case errors.Is(err, gorm.ErrDuplicatedKey):
		Conflict(c, "Resource already exists")
	default:
		handleError(c, err)
	}
}

// Success sends a successful response
func Success(c *gin.Context, data interface{}) {
	status := http.StatusOK
	if c.Request.Method == "POST" {
		status = http.StatusCreated
	}

	c.JSON(status, Response{
		Success: true,
		Data:    data,
	})
}

// NotFound sends a 404 response
func NotFound(c *gin.Context, message string) {
	c.JSON(http.StatusNotFound, Response{
		Success: false,
		Error: &Error{
			Code:    ErrCodeNotFound,
			Message: message,
		},
	})
}

// BadRequest sends a 400 response
func BadRequest(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, Response{
		Success: false,
		Error: &Error{
			Code:    ErrCodeBadRequest,
			Message: message,
		},
	})
}

// Unauthorized sends a 401 response
func Unauthorized(c *gin.Context, message string) {
	c.JSON(http.StatusUnauthorized, Response{
		Success: false,
		Error: &Error{
			Code:    ErrCodeUnauthorized,
			Message: message,
		},
	})
}

// Forbidden sends a 403 response
func Forbidden(c *gin.Context, message string) {
	c.JSON(http.StatusForbidden, Response{
		Success: false,
		Error: &Error{
			Code:    ErrCodeForbidden,
			Message: message,
		},
	})
}

// InternalError sends a 500 response
func InternalError(c *gin.Context, message string) {
	c.JSON(http.StatusInternalServerError, Response{
		Success: false,
		Error: &Error{
			Code:    ErrCodeInternalError,
			Message: message,
		},
	})
}

// Conflict sends a 409 response
func Conflict(c *gin.Context, message string) {
	c.JSON(http.StatusConflict, Response{
		Success: false,
		Error: &Error{
			Code:    ErrCodeDuplicateResource,
			Message: message,
		},
	})
}

// handleError determines the appropriate error response
func handleError(c *gin.Context, err error) {
	// Add custom error type checks here
	// For example:
	// if validationErr, ok := err.(*ValidationError); ok {
	//     BadRequest(c, validationErr.Error())
	//     return
	// }

	// Default to internal server error
	InternalError(c, "An unexpected error occurred")
} 