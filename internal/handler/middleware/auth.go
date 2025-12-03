package middleware

import (
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	jwtsvc "workout-app/pkg/jwt"
)

const (
	ContextUserIDKey    = "userID"
	ContextUserEmailKey = "userEmail"
	ContextUserRoleKey  = "userRole"
)

// Auth возвращает middleware для аутентификации по JWT access-токену.
// Ожидает заголовок Authorization: Bearer <token>.
func Auth(jwtService jwtsvc.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			log.Printf("missing Authorization header: path=%s", c.Request.URL.Path)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing_authorization_header"})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			log.Printf("invalid Authorization header format: value=%q", authHeader)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid_authorization_header"})
			return
		}

		tokenString := strings.TrimSpace(parts[1])
		if tokenString == "" {
			log.Printf("empty bearer token in Authorization header")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid_authorization_header"})
			return
		}

		claims, err := jwtService.ParseAccessToken(tokenString)
		if err != nil {
			log.Printf("invalid access token: err=%v", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid_token"})
			return
		}

		// Сохраняем данные пользователя в контексте Gin
		c.Set(ContextUserIDKey, claims.UserID)
		c.Set(ContextUserEmailKey, claims.Email)
		c.Set(ContextUserRoleKey, claims.Role)

		c.Next()
	}
}

