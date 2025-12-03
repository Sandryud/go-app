package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	domain "workout-app/internal/domain/user"
	"workout-app/internal/handler/response"
	"workout-app/pkg/logger"
	jwtsvc "workout-app/pkg/jwt"
)

const (
	ContextUserIDKey    = "userID"
	ContextUserEmailKey = "userEmail"
	ContextUserRoleKey  = "userRole"
)

// Auth возвращает middleware для аутентификации по JWT access-токену.
// Ожидает заголовок Authorization: Bearer <token>.
func Auth(jwtService jwtsvc.Service, log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			log.Info("missing_authorization_header", map[string]any{
				"path":   c.Request.URL.Path,
				"method": c.Request.Method,
			})
			response.Error(c, http.StatusUnauthorized, "missing_authorization_header", "Отсутствует заголовок Authorization", nil)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			log.Info("invalid_authorization_header", map[string]any{
				"path":   c.Request.URL.Path,
				"method": c.Request.Method,
				"value":  authHeader,
			})
			response.Error(c, http.StatusUnauthorized, "invalid_authorization_header", "Некорректный формат заголовка Authorization", nil)
			return
		}

		tokenString := strings.TrimSpace(parts[1])
		if tokenString == "" {
			log.Info("empty_bearer_token", map[string]any{
				"path":   c.Request.URL.Path,
				"method": c.Request.Method,
			})
			response.Error(c, http.StatusUnauthorized, "invalid_authorization_header", "Некорректный формат заголовка Authorization", nil)
			return
		}

		claims, err := jwtService.ParseAccessToken(tokenString)
		if err != nil {
			log.Info("invalid_access_token", map[string]any{
				"path":   c.Request.URL.Path,
				"method": c.Request.Method,
				"error":  err.Error(),
			})
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
func RequireRole(log logger.Logger, allowedRoles ...domain.Role) gin.HandlerFunc {
	allowed := make(map[domain.Role]struct{}, len(allowedRoles))
	for _, r := range allowedRoles {
		if r == "" {
			continue
		}
		allowed[r] = struct{}{}
	}

	return func(c *gin.Context) {
		rawRole := c.GetString(ContextUserRoleKey)
		role := domain.Role(rawRole)
		if role == "" {
			log.Info("missing_role_in_context", map[string]any{
				"path":   c.Request.URL.Path,
				"method": c.Request.Method,
			})
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
			log.Info("access_denied_by_role", map[string]any{
				"path":   c.Request.URL.Path,
				"method": c.Request.Method,
				"role":   role,
			})
			response.Error(c, http.StatusForbidden, "forbidden", "Недостаточно прав для доступа к ресурсу", nil)
			c.Abort()
			return
		}

		c.Next()
	}
}

