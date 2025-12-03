package user

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	domain "workout-app/internal/domain/user"
	"workout-app/internal/handler/middleware"
	"workout-app/internal/handler/response"
	repo "workout-app/internal/repository/interfaces"
	useruc "workout-app/internal/usecase/user"
	"workout-app/pkg/logger"
)

// Handler обрабатывает HTTP-запросы, связанные с профилем пользователя.
type Handler struct {
	users  useruc.Service
	logger logger.Logger
}

// NewHandler создаёт новый UserHandler.
func NewHandler(users useruc.Service, logger logger.Logger) *Handler {
	return &Handler{
		users:  users,
		logger: logger,
	}
}

// getUserIDFromContext извлекает идентификатор пользователя из контекста запроса.
// Возвращает ошибку unauthorized в случае отсутствия или некорректного значения.
func getUserIDFromContext(c *gin.Context) (uuid.UUID, error) {
	idStr := c.GetString(middleware.ContextUserIDKey)
	if idStr == "" {
		return uuid.Nil, errors.New("missing_user_id_in_context")
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return uuid.Nil, errors.New("invalid_user_id_in_context")
	}

	return id, nil
}

// GetMe godoc
// @Summary      Получить профиль текущего пользователя
// @Description  Возвращает профиль пользователя, извлечённого из access-токена.
// @Tags         user
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  ProfileResponse
// @Failure      401  {object}  response.ErrorBody
// @Failure      404  {object}  response.ErrorBody
// @Failure      500  {object}  response.ErrorBody
// @Router       /api/v1/users/me [get]
func (h *Handler) GetMe(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "unauthorized", "Требуется аутентификация", nil)
		return
	}

	user, err := h.users.GetProfile(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			h.logger.Info("user_not_found_in_get_me", map[string]any{
				"user_id": userID.String(),
				"path":    c.Request.URL.Path,
				"method":  c.Request.Method,
			})
			response.Error(c, http.StatusNotFound, "user_not_found", "Пользователь не найден", nil)
			return
		}
		h.logger.Error("internal_error_in_get_me", map[string]any{
			"user_id": userID.String(),
			"path":    c.Request.URL.Path,
			"method":  c.Request.Method,
			"error":   err.Error(),
		})
		response.Error(c, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера", nil)
		return
	}

	c.JSON(http.StatusOK, toProfileResponse(user))
}

// UpdateMe godoc
// @Summary      Обновить профиль текущего пользователя
// @Description  Частичное обновление профиля (username, имя, уровень подготовки и т.п.).
// @Tags         user
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        payload  body      ProfileUpdateRequest  true  "Данные профиля"
// @Success      200      {object}  ProfileResponse
// @Failure      400      {object}  response.ErrorBody
// @Failure      401      {object}  response.ErrorBody
// @Failure      404      {object}  response.ErrorBody
// @Failure      409      {object}  response.ErrorBody
// @Failure      500      {object}  response.ErrorBody
// @Router       /api/v1/users/me [put]
func (h *Handler) UpdateMe(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "unauthorized", "Требуется аутентификация", nil)
		return
	}

	var req ProfileUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid_request", "Некорректное тело запроса", err.Error())
		return
	}

	input := useruc.ProfileUpdateInput{}

	if req.Username != nil {
		input.Username = req.Username
	}
	if req.FirstName != nil {
		input.FirstName = req.FirstName
	}
	if req.LastName != nil {
		input.LastName = req.LastName
	}
	if req.BirthDate != nil {
		input.BirthDate = req.BirthDate
	}
	if req.Gender != nil {
		input.Gender = req.Gender
	}
	if req.AvatarURL != nil {
		input.AvatarURL = req.AvatarURL
	}
	if req.TrainingLevel != nil {
		level := domain.TrainingLevel(*req.TrainingLevel)
		input.TrainingLevel = &level
	}

	user, err := h.users.UpdateProfile(c.Request.Context(), userID, input)
	if err != nil {
		switch {
		case errors.Is(err, repo.ErrEmailExists):
			h.logger.Info("email_conflict_in_update_me", map[string]any{
				"user_id": userID.String(),
				"path":    c.Request.URL.Path,
				"method":  c.Request.Method,
				"error":   err.Error(),
			})
			response.Error(c, http.StatusConflict, "email_already_exists", "Указанный email уже используется", nil)
			return
		case errors.Is(err, repo.ErrUsernameExists):
			h.logger.Info("username_conflict_in_update_me", map[string]any{
				"user_id": userID.String(),
				"path":    c.Request.URL.Path,
				"method":  c.Request.Method,
				"error":   err.Error(),
			})
			response.Error(c, http.StatusConflict, "username_already_exists", "Указанный никнейм уже используется", nil)
			return
		case errors.Is(err, repo.ErrNotFound):
			h.logger.Info("user_not_found_in_update_me", map[string]any{
				"user_id": userID.String(),
				"path":    c.Request.URL.Path,
				"method":  c.Request.Method,
			})
			response.Error(c, http.StatusNotFound, "user_not_found", "Пользователь не найден", nil)
			return
		default:
			h.logger.Error("internal_error_in_update_me", map[string]any{
				"user_id": userID.String(),
				"path":    c.Request.URL.Path,
				"method":  c.Request.Method,
				"error":   err.Error(),
			})
			response.Error(c, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера", nil)
			return
		}
	}

	c.JSON(http.StatusOK, toProfileResponse(user))
}

// DeleteMe godoc
// @Summary      Удалить текущий аккаунт
// @Description  Soft-delete (устанавливает deleted_at, не удаляя физически).
// @Tags         user
// @Security     BearerAuth
// @Produce      json
// @Success      204  "Аккаунт удалён"
// @Failure      401  {object}  response.ErrorBody
// @Failure      404  {object}  response.ErrorBody
// @Failure      500  {object}  response.ErrorBody
// @Router       /api/v1/users/me [delete]
func (h *Handler) DeleteMe(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "unauthorized", "Требуется аутентификация", nil)
		return
	}

	if err := h.users.DeleteAccount(c.Request.Context(), userID); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			h.logger.Info("user_not_found_in_delete_me", map[string]any{
				"user_id": userID.String(),
				"path":    c.Request.URL.Path,
				"method":  c.Request.Method,
			})
			response.Error(c, http.StatusNotFound, "user_not_found", "Пользователь не найден", nil)
			return
		}
		h.logger.Error("internal_error_in_delete_me", map[string]any{
			"user_id": userID.String(),
			"path":    c.Request.URL.Path,
			"method":  c.Request.Method,
			"error":   err.Error(),
		})
		response.Error(c, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера", nil)
		return
	}

	c.Status(http.StatusNoContent)
}

// GetByID godoc
// @Summary      Получить публичный профиль пользователя по ID
// @Description  Возвращает публичный профиль пользователя по идентификатору. Доступно любому аутентифицированному пользователю.
// @Tags         user
// @Security     BearerAuth
// @Produce      json
// @Param        id   path      string  true  "ID пользователя (UUID)"
// @Success      200  {object}  PublicProfileResponse
// @Failure      400  {object}  response.ErrorBody
// @Failure      401  {object}  response.ErrorBody
// @Failure      404  {object}  response.ErrorBody
// @Failure      500  {object}  response.ErrorBody
// @Router       /api/v1/users/{id} [get]
func (h *Handler) GetByID(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		response.Error(c, http.StatusBadRequest, "invalid_request", "ID пользователя обязателен", nil)
		return
	}

	userID, err := uuid.Parse(idStr)
	if err != nil {
		h.logger.Info("invalid_user_id_format", map[string]any{
			"id":     idStr,
			"path":   c.Request.URL.Path,
			"method": c.Request.Method,
		})
		response.Error(c, http.StatusBadRequest, "invalid_request", "Некорректный формат ID пользователя", nil)
		return
	}

	user, err := h.users.GetByID(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			h.logger.Info("user_not_found_in_get_by_id", map[string]any{
				"user_id": userID.String(),
				"path":    c.Request.URL.Path,
				"method":  c.Request.Method,
			})
			response.Error(c, http.StatusNotFound, "user_not_found", "Пользователь не найден", nil)
			return
		}
		h.logger.Error("internal_error_in_get_by_id", map[string]any{
			"user_id": userID.String(),
			"path":    c.Request.URL.Path,
			"method":  c.Request.Method,
			"error":   err.Error(),
		})
		response.Error(c, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера", nil)
		return
	}

	c.JSON(http.StatusOK, toPublicProfileResponse(user))
}

// ListUsers godoc
// @Summary      Получить список всех пользователей (админ)
// @Description  Возвращает список всех активных пользователей. Доступно только для роли admin.
// @Tags         user
// @Security     BearerAuth
// @Produce      json
// @Success      200  {array}   ProfileResponse
// @Failure      401  {object}  response.ErrorBody
// @Failure      403  {object}  response.ErrorBody
// @Failure      500  {object}  response.ErrorBody
// @Router       /api/v1/admin/users [get]
func (h *Handler) ListUsers(c *gin.Context) {
	users, err := h.users.ListUsers(c.Request.Context())
	if err != nil {
		h.logger.Error("internal_error_in_list_users", map[string]any{
			"path":   c.Request.URL.Path,
			"method": c.Request.Method,
			"error":  err.Error(),
		})
		response.Error(c, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера", nil)
		return
	}

	resp := make([]ProfileResponse, 0, len(users))
	for _, u := range users {
		resp = append(resp, toProfileResponse(u))
	}

	c.JSON(http.StatusOK, resp)
}

// toProfileResponse маппит доменную модель в DTO.
func toProfileResponse(u *domain.User) ProfileResponse {
	return ProfileResponse{
		ID:            u.ID.String(),
		Email:         u.Email,
		Username:      u.Username,
		FirstName:     u.FirstName,
		LastName:      u.LastName,
		BirthDate:     u.BirthDate,
		Gender:        u.Gender,
		AvatarURL:     u.AvatarURL,
		Role:          string(u.Role),
		TrainingLevel: string(u.TrainingLevel),
		CreatedAt:     u.CreatedAt,
		UpdatedAt:     u.UpdatedAt,
	}
}

// toPublicProfileResponse маппит доменную модель в публичный DTO (без email).
func toPublicProfileResponse(u *domain.User) PublicProfileResponse {
	return PublicProfileResponse{
		ID:            u.ID.String(),
		Username:      u.Username,
		FirstName:     u.FirstName,
		LastName:      u.LastName,
		BirthDate:     u.BirthDate,
		Gender:        u.Gender,
		AvatarURL:     u.AvatarURL,
		Role:          string(u.Role),
		TrainingLevel: string(u.TrainingLevel),
		CreatedAt:     u.CreatedAt,
		UpdatedAt:     u.UpdatedAt,
	}
}


