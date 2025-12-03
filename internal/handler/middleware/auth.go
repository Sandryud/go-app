package middleware

import (
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"workout-app/internal/handler/response"
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
			response.Error(c, http.StatusUnauthorized, "missing_authorization_header", "Отсутствует заголовок Authorization", nil)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			log.Printf("invalid Authorization header format: value=%q", authHeader)
			response.Error(c, http.StatusUnauthorized, "invalid_authorization_header", "Некорректный формат заголовка Authorization", nil)
			return
		}

		tokenString := strings.TrimSpace(parts[1])
		if tokenString == "" {
			log.Printf("empty bearer token in Authorization header")
			response.Error(c, http.StatusUnauthorized, "invalid_authorization_header", "Некорректный формат заголовка Authorization", nil)
			return
		}

		claims, err := jwtService.ParseAccessToken(tokenString)
		if err != nil {
			log.Printf("invalid access token: err=%v", err)
			response.Error(c, http.StatusUnauthorized, "invalid_token", "Недействительный access-токен", nil)
			return
		}

		// Сохраняем данные пользователя в контексте Gin
		c.Set(ContextUserIDKey, claims.UserID)
		c.Set(ContextUserEmailKey, claims.Email)
		c.Set(ContextUserRoleKey, claims.Role)

		c.Next()
	}
}

// RequireRole возвращает middleware, которое проверяет, что роль пользователя входит
// в список разрешённых ролей. Используется поверх Auth или в группах с Auth.
func RequireRole(allowedRoles ...string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(allowedRoles))
	for _, r := range allowedRoles {
		if r == "" {
			continue
		}
		allowed[strings.ToLower(r)] = struct{}{}
	}

	return func(c *gin.Context) {
		role := strings.ToLower(c.GetString(ContextUserRoleKey))
		if role == "" {
			log.Printf("missing role in context for path=%s", c.Request.URL.Path)
			response.Error(c, http.StatusForbidden, "forbidden", "Недостаточно прав для доступа к ресурсу", nil)
			c.Abort()
			return
		}

		if len(allowed) == 0 {
			// Если роли не заданы, пропускаем без дополнительной проверки
			c.Next()
			return
		}

		if _, ok := allowed[role]; !ok {
			log.Printf("access denied for role=%s path=%s", role, c.Request.URL.Path)
			response.Error(c, http.StatusForbidden, "forbidden", "Недостаточно прав для доступа к ресурсу", nil)
			c.Abort()
			return
		}

		c.Next()
	}
}

