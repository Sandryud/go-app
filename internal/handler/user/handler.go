package user

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	domain "workout-app/internal/domain/user"
	"workout-app/internal/handler/middleware"
	"workout-app/internal/handler/response"
	repo "workout-app/internal/repository/interfaces"
	useruc "workout-app/internal/usecase/user"
)

// Handler обрабатывает HTTP-запросы, связанные с профилем пользователя.
type Handler struct {
	users useruc.Service
}

// NewHandler создаёт новый UserHandler.
func NewHandler(users useruc.Service) *Handler {
	return &Handler{users: users}
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

// GetMe возвращает профиль текущего пользователя.
func (h *Handler) GetMe(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "unauthorized", "Требуется аутентификация", nil)
		return
	}

	user, err := h.users.GetProfile(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			log.Printf("user not found in GetMe: user_id=%s", userID)
			response.Error(c, http.StatusNotFound, "user_not_found", "Пользователь не найден", nil)
			return
		}
		log.Printf("internal error in GetMe: user_id=%s err=%v", userID, err)
		response.Error(c, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера", nil)
		return
	}

	c.JSON(http.StatusOK, toProfileResponse(user))
}

// UpdateMe обновляет профиль текущего пользователя.
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
			log.Printf("email conflict in UpdateMe: user_id=%s err=%v", userID, err)
			response.Error(c, http.StatusConflict, "email_already_exists", "Указанный email уже используется", nil)
			return
		case errors.Is(err, repo.ErrUsernameExists):
			log.Printf("username conflict in UpdateMe: user_id=%s err=%v", userID, err)
			response.Error(c, http.StatusConflict, "username_already_exists", "Указанный никнейм уже используется", nil)
			return
		case errors.Is(err, repo.ErrNotFound):
			log.Printf("user not found in UpdateMe: user_id=%s", userID)
			response.Error(c, http.StatusNotFound, "user_not_found", "Пользователь не найден", nil)
			return
		default:
			log.Printf("internal error in UpdateMe: user_id=%s err=%v", userID, err)
			response.Error(c, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера", nil)
			return
		}
	}

	c.JSON(http.StatusOK, toProfileResponse(user))
}

// DeleteMe мягко удаляет аккаунт текущего пользователя.
func (h *Handler) DeleteMe(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "unauthorized", "Требуется аутентификация", nil)
		return
	}

	if err := h.users.DeleteAccount(c.Request.Context(), userID); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			log.Printf("user not found in DeleteMe: user_id=%s", userID)
			response.Error(c, http.StatusNotFound, "user_not_found", "Пользователь не найден", nil)
			return
		}
		log.Printf("internal error in DeleteMe: user_id=%s err=%v", userID, err)
		response.Error(c, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера", nil)
		return
	}

	c.Status(http.StatusNoContent)
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


