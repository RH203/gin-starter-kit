package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// APIResponse represents standard JSON API envelope
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   interface{} `json:"error,omitempty"`
}

// PaginationMeta holds pagination metrics
type PaginationMeta struct {
	CurrentPage int   `json:"current_page"`
	PageSize    int   `json:"page_size"`
	TotalItems  int64 `json:"total_items"`
	TotalPages  int   `json:"total_pages"`
}

// PaginatedData wraps paginated items and metadata
type PaginatedData struct {
	Items interface{}    `json:"items"`
	Meta  PaginationMeta `json:"meta"`
}

// Success writes a structured JSON response with specified status code
func Success(c *gin.Context, statusCode int, message string, data interface{}) {
	c.JSON(statusCode, APIResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// OK writes an HTTP 200 OK JSON response
func OK(c *gin.Context, message string, data interface{}) {
	Success(c, http.StatusOK, message, data)
}

// Created writes an HTTP 201 Created JSON response
func Created(c *gin.Context, message string, data interface{}) {
	Success(c, http.StatusCreated, message, data)
}

// Paginated writes an HTTP 200 OK JSON response with structured pagination metadata
func Paginated(c *gin.Context, message string, items interface{}, totalItems int64, page, pageSize int) {
	totalPages := 0
	if pageSize > 0 {
		totalPages = int((totalItems + int64(pageSize) - 1) / int64(pageSize))
	}
	Success(c, http.StatusOK, message, PaginatedData{
		Items: items,
		Meta: PaginationMeta{
			CurrentPage: page,
			PageSize:    pageSize,
			TotalItems:  totalItems,
			TotalPages:  totalPages,
		},
	})
}

// Error writes a structured JSON error response
func Error(c *gin.Context, statusCode int, message string, errDetail interface{}) {
	c.JSON(statusCode, APIResponse{
		Success: false,
		Message: message,
		Error:   errDetail,
	})
}
