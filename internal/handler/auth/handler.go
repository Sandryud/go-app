package auth

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"workout-app/internal/handler/response"
	repo "workout-app/internal/repository/interfaces"
	authuc "workout-app/internal/usecase/auth"
)

// Handler обрабатывает HTTP-запросы, связанные с аутентификацией.
type Handler struct {
	auth authuc.Service
}

// NewHandler создаёт новый AuthHandler.
func NewHandler(authSvc authuc.Service) *Handler {
	return &Handler{
		auth: authSvc,
	}
}

// Register godoc
// @Summary      Регистрация пользователя
// @Description  Регистрация по email/паролю/username. Возвращает пару access/refresh токенов.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        payload  body      RegisterRequest      true  "Данные для регистрации"
// @Success      201      {object}  RegisterResponse
// @Failure      400      {object}  response.ErrorBody
// @Failure      409      {object}  response.ErrorBody
// @Failure      500      {object}  response.ErrorBody
// @Router       /api/v1/auth/register [post]
func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid_request", "Invalid request body", err.Error())
		return
	}

	user, err := h.auth.Register(c.Request.Context(), req.Email, req.Password, req.Username)
	if err != nil {
		switch {
		case errors.Is(err, authuc.ErrEmailUnverifiedExists):
			log.Printf("unverified email conflict in Register: email=%s err=%v", req.Email, err)
			response.Error(c, http.StatusConflict, "email_unverified", "Account with this email already exists but is not verified. Please request a new verification code.", nil)
		case errors.Is(err, repo.ErrEmailExists):
			log.Printf("email conflict in Register: email=%s err=%v", req.Email, err)
			response.Error(c, http.StatusConflict, "email_already_exists", "Email is already in use", nil)
		case errors.Is(err, repo.ErrUsernameExists):
			log.Printf("username conflict in Register: username=%s err=%v", req.Username, err)
			response.Error(c, http.StatusConflict, "username_already_exists", "Username is already in use", nil)
		default:
			log.Printf("internal error in Register: email=%s username=%s err=%v", req.Email, req.Username, err)
			response.Error(c, http.StatusInternalServerError, "internal_error", "Internal server error", nil)
		}
		return
	}

	resp := RegisterResponse{
		UserID:   user.ID.String(),
		Email:    user.Email,
		Username: user.Username,
		Message:  "Verification code has been sent to your email",
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
		response.Error(c, http.StatusBadRequest, "invalid_request", "Invalid request body", err.Error())
		return
	}

	user, access, refresh, err := h.auth.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, authuc.ErrInvalidCredentials):
			response.Error(c, http.StatusUnauthorized, "invalid_credentials", "Invalid email or password", nil)
		case errors.Is(err, authuc.ErrEmailNotVerified):
			response.Error(c, http.StatusForbidden, "email_not_verified", "Email is not verified", nil)
		default:
			log.Printf("internal error in Login: email=%s err=%v", req.Email, err)
			response.Error(c, http.StatusInternalServerError, "internal_error", "Internal server error", nil)
		}
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
		response.Error(c, http.StatusBadRequest, "invalid_request", "Invalid request body", err.Error())
		return
	}

	user, access, refresh, err := h.auth.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		switch {
		case errors.Is(err, authuc.ErrInvalidRefreshToken):
			response.Error(c, http.StatusUnauthorized, "invalid_refresh_token", "Invalid refresh token", nil)
		case errors.Is(err, authuc.ErrEmailNotVerified):
			response.Error(c, http.StatusForbidden, "email_not_verified", "Email is not verified", nil)
		default:
			log.Printf("internal error in Refresh: err=%v", err)
			response.Error(c, http.StatusInternalServerError, "internal_error", "Internal server error", nil)
		}
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

// ResendVerification godoc
// @Summary      Повторная отправка кода подтверждения email
// @Description  Отправляет новый код подтверждения на указанный email, если аккаунт ещё не подтверждён.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        payload  body      ResendVerificationRequest  true  "Email для повторной отправки кода"
// @Success      200      {object}  ResendVerificationResponse
// @Failure      400      {object}  response.ErrorBody
// @Failure      500      {object}  response.ErrorBody
// @Router       /api/v1/auth/resend-verification [post]
func (h *Handler) ResendVerification(c *gin.Context) {
	var req ResendVerificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid_request", "Invalid request body", err.Error())
		return
	}

	err := h.auth.ResendVerificationCode(c.Request.Context(), req.Email)
	if err != nil {
		switch {
		case errors.Is(err, authuc.ErrEmailAlreadyVerified):
			// Email уже подтверждён — мягкий ответ 200
			c.JSON(http.StatusOK, ResendVerificationResponse{
				Message: "Email is already verified",
			})
			return
		default:
			log.Printf("internal error in ResendVerification: email=%s err=%v", req.Email, err)
			response.Error(c, http.StatusInternalServerError, "internal_error", "Internal server error", nil)
			return
		}
	}

	c.JSON(http.StatusOK, ResendVerificationResponse{
		Message: "If an account with this email exists, a verification code has been sent",
	})
}

// VerifyEmail godoc
// @Summary      Подтверждение email кодом
// @Description  Подтверждает email пользователя по одноразовому коду и возвращает пару access/refresh токенов.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        payload  body      VerifyEmailRequest   true  "Данные для подтверждения email"
// @Success      200      {object}  LoginResponse
// @Failure      400      {object}  response.ErrorBody
// @Failure      401      {object}  response.ErrorBody
// @Failure      403      {object}  response.ErrorBody
// @Failure      500      {object}  response.ErrorBody
// @Router       /api/v1/auth/verify-email [post]
func (h *Handler) VerifyEmail(c *gin.Context) {
	var req VerifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid_request", "Invalid request body", err.Error())
		return
	}

	user, access, refresh, err := h.auth.VerifyEmail(c.Request.Context(), req.Email, req.Code)
	if err != nil {
		switch {
		case errors.Is(err, authuc.ErrEmailAlreadyVerified):
			response.Error(c, http.StatusConflict, "email_already_verified", "Email is already verified", nil)
		case errors.Is(err, authuc.ErrVerificationCodeNotFound):
			response.Error(c, http.StatusBadRequest, "verification_code_not_found", "Verification code not found or expired. Please request a new verification code.", nil)
		case errors.Is(err, authuc.ErrVerificationCodeInvalid):
			response.Error(c, http.StatusBadRequest, "verification_code_invalid", "Verification code is invalid", nil)
		case errors.Is(err, authuc.ErrVerificationAttemptsExceeded):
			response.Error(c, http.StatusBadRequest, "verification_attempts_exceeded", "Verification attempts limit exceeded. Please request a new code.", nil)
		default:
			log.Printf("internal error in VerifyEmail: email=%s err=%v", req.Email, err)
			response.Error(c, http.StatusInternalServerError, "internal_error", "Internal server error", nil)
		}
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
