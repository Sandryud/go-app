package response

import "github.com/gin-gonic/gin"

// ErrorBody описывает стандартный формат ошибки API.
type ErrorBody struct {
	Code    string      `json:"code"`
	Message string      `json:"message,omitempty"`
	Details interface{} `json:"details,omitempty"`
}

// TokenPair описывает пару access/refresh токенов.
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// Error отправляет JSON-ответ с ошибкой в едином формате.
func Error(c *gin.Context, status int, code, message string, details interface{}) {
	c.JSON(status, gin.H{
		"error": ErrorBody{
			Code:    code,
			Message: message,
			Details: details,
		},
	})
}
