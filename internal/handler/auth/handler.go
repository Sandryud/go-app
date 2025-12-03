package auth

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"workout-app/internal/handler/response"
	repo "workout-app/internal/repository/interfaces"
	useruc "workout-app/internal/usecase/user"
	jwtsvc "workout-app/pkg/jwt"
	"workout-app/pkg/password"
)

// Handler обрабатывает HTTP-запросы, связанные с аутентификацией.
type Handler struct {
	users  useruc.Service
	repo   repo.UserRepository
	jwt    jwtsvc.Service
}

// NewHandler создаёт новый AuthHandler.
func NewHandler(users useruc.Service, repo repo.UserRepository, jwt jwtsvc.Service) *Handler {
	return &Handler{
		users: users,
		repo:  repo,
		jwt:   jwt,
	}
}

// Register godoc
// @Summary      Регистрация пользователя
// @Description  Регистрация по email/паролю/username. Возвращает пару access/refresh токенов.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        payload  body      RegisterRequest      true  "Данные для регистрации"
// @Success      201      {object}  LoginResponse
// @Failure      400      {object}  response.ErrorBody
// @Failure      409      {object}  response.ErrorBody
// @Failure      500      {object}  response.ErrorBody
// @Router       /api/v1/auth/register [post]
func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid_request", "Некорректное тело запроса", err.Error())
		return
	}

	// Хешируем пароль
	hash, err := password.Hash(req.Password)
	if err != nil {
		log.Printf("error hashing password in Register: email=%s err=%v", req.Email, err)
		response.Error(c, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера", nil)
		return
	}

	user, err := h.users.Register(c.Request.Context(), req.Email, hash, req.Username)
	if err != nil {
		switch {
		case errors.Is(err, repo.ErrEmailExists):
			log.Printf("email conflict in Register: email=%s err=%v", req.Email, err)
			response.Error(c, http.StatusConflict, "email_already_exists", "Указанный email уже используется", nil)
		case errors.Is(err, repo.ErrUsernameExists):
			log.Printf("username conflict in Register: username=%s err=%v", req.Username, err)
			response.Error(c, http.StatusConflict, "username_already_exists", "Указанный никнейм уже используется", nil)
		default:
			log.Printf("internal error in Register: email=%s username=%s err=%v", req.Email, req.Username, err)
			response.Error(c, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера", nil)
		}
		return
	}

	access, err := h.jwt.GenerateAccessToken(user)
	if err != nil {
		log.Printf("error generating access token in Register: user_id=%s err=%v", user.ID, err)
		response.Error(c, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера", nil)
		return
	}
	refresh, _, err := h.jwt.GenerateRefreshToken(user)
	if err != nil {
		log.Printf("error generating refresh token in Register: user_id=%s err=%v", user.ID, err)
		response.Error(c, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера", nil)
		return
	}

	resp := LoginResponse{
		UserID:   user.ID.String(),
		Email:    user.Email,
		Username: user.Username,
		Tokens: TokenPair{
			AccessToken:  access,
			RefreshToken: refresh,
		},
	}

	c.JSON(http.StatusCreated, resp)
}

// Login godoc
// @Summary      Вход по email и паролю
// @Description  Аутентификация пользователя. Возвращает пару access/refresh токенов.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        payload  body      LoginRequest         true  "Данные для входа"
// @Success      200      {object}  LoginResponse
// @Failure      400      {object}  response.ErrorBody
// @Failure      401      {object}  response.ErrorBody
// @Failure      500      {object}  response.ErrorBody
// @Router       /api/v1/auth/login [post]
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid_request", "Некорректное тело запроса", err.Error())
		return
	}

	// Ищем пользователя по email
	user, err := h.repo.GetByEmail(c.Request.Context(), req.Email)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			// Не раскрываем, что именно неверно
			response.Error(c, http.StatusUnauthorized, "invalid_credentials", "Неверный email или пароль", nil)
			return
		}
		log.Printf("internal error in Login (GetByEmail): email=%s err=%v", req.Email, err)
		response.Error(c, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера", nil)
		return
	}

	// Проверяем пароль
	if err := password.Compare(user.PasswordHash, req.Password); err != nil {
		response.Error(c, http.StatusUnauthorized, "invalid_credentials", "Неверный email или пароль", nil)
		return
	}

	access, err := h.jwt.GenerateAccessToken(user)
	if err != nil {
		log.Printf("error generating access token in Login: user_id=%s err=%v", user.ID, err)
		response.Error(c, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера", nil)
		return
	}
	refresh, _, err := h.jwt.GenerateRefreshToken(user)
	if err != nil {
		log.Printf("error generating refresh token in Login: user_id=%s err=%v", user.ID, err)
		response.Error(c, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера", nil)
		return
	}

	resp := LoginResponse{
		UserID:   user.ID.String(),
		Email:    user.Email,
		Username: user.Username,
		Tokens: TokenPair{
			AccessToken:  access,
			RefreshToken: refresh,
		},
	}

	c.JSON(http.StatusOK, resp)
}

// Refresh godoc
// @Summary      Обновление токенов
// @Description  Обновление пары access/refresh токенов по действительному refresh-токену.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        payload  body      RefreshRequest       true  "Refresh токен"
// @Success      200      {object}  LoginResponse
// @Failure      400      {object}  response.ErrorBody
// @Failure      401      {object}  response.ErrorBody
// @Failure      500      {object}  response.ErrorBody
// @Router       /api/v1/auth/refresh [post]
func (h *Handler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid_request", "Некорректное тело запроса", err.Error())
		return
	}

	claims, err := h.jwt.ParseRefreshToken(req.RefreshToken)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "invalid_refresh_token", "Недействительный refresh-токен", nil)
		return
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "invalid_refresh_token", "Недействительный refresh-токен", nil)
		return
	}

	user, err := h.repo.GetByID(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			response.Error(c, http.StatusUnauthorized, "invalid_refresh_token", "Недействительный refresh-токен", nil)
			return
		}
		log.Printf("internal error in Refresh (GetByID): user_id=%s err=%v", userID, err)
		response.Error(c, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера", nil)
		return
	}

	access, err := h.jwt.GenerateAccessToken(user)
	if err != nil {
		log.Printf("error generating access token in Refresh: user_id=%s err=%v", user.ID, err)
		response.Error(c, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера", nil)
		return
	}
	refresh, _, err := h.jwt.GenerateRefreshToken(user)
	if err != nil {
		log.Printf("error generating refresh token in Refresh: user_id=%s err=%v", user.ID, err)
		response.Error(c, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера", nil)
		return
	}

	resp := LoginResponse{
		UserID:   user.ID.String(),
		Email:    user.Email,
		Username: user.Username,
		Tokens: TokenPair{
			AccessToken:  access,
			RefreshToken: refresh,
		},
	}

	c.JSON(http.StatusOK, resp)
}


