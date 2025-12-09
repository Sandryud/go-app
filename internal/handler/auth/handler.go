package auth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"workout-app/internal/handler/response"
	repo "workout-app/internal/repository/interfaces"
	authuc "workout-app/internal/usecase/auth"
	"workout-app/pkg/logger"
)

// Handler обрабатывает HTTP-запросы, связанные с аутентификацией.
type Handler struct {
	auth   authuc.Service
	logger logger.Logger
}

// NewHandler создаёт новый AuthHandler.
func NewHandler(authSvc authuc.Service, logger logger.Logger) *Handler {
	return &Handler{
		auth:   authSvc,
		logger: logger,
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
		case errors.Is(err, authuc.ErrAccountDeleted):
			h.logger.Info("account_deleted_in_register", map[string]any{
				"email":  req.Email,
				"path":   c.Request.URL.Path,
				"method": c.Request.Method,
			})
			response.Error(c, http.StatusConflict, "account_deleted", "Account with this email was deleted. Use the account restoration endpoint.", nil)
		case errors.Is(err, authuc.ErrEmailUnverifiedExists):
			h.logger.Info("unverified_email_conflict_in_register", map[string]any{
				"email":  req.Email,
				"path":   c.Request.URL.Path,
				"method": c.Request.Method,
			})
			response.Error(c, http.StatusConflict, "email_unverified", "Account with this email already exists but is not verified. Please request a new verification code.", nil)
		case errors.Is(err, repo.ErrEmailExists):
			h.logger.Info("email_conflict_in_register", map[string]any{
				"email":  req.Email,
				"path":   c.Request.URL.Path,
				"method": c.Request.Method,
			})
			response.Error(c, http.StatusConflict, "email_already_exists", "Email is already in use", nil)
		case errors.Is(err, repo.ErrUsernameExists):
			h.logger.Info("username_conflict_in_register", map[string]any{
				"username": req.Username,
				"path":     c.Request.URL.Path,
				"method":   c.Request.Method,
			})
			response.Error(c, http.StatusConflict, "username_already_exists", "Username is already in use", nil)
		default:
			h.logger.Error("internal_error_in_register", map[string]any{
				"email":    req.Email,
				"username": req.Username,
				"path":     c.Request.URL.Path,
				"method":   c.Request.Method,
				"error":    err.Error(),
			})
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
		case errors.Is(err, authuc.ErrAccountDeleted):
			h.logger.Info("account_deleted_in_login", map[string]any{
				"email":  req.Email,
				"path":   c.Request.URL.Path,
				"method": c.Request.Method,
			})
			response.Error(c, http.StatusForbidden, "account_deleted", "Your account was deleted. Would you like to restore it?", nil)
		case errors.Is(err, authuc.ErrInvalidCredentials):
			response.Error(c, http.StatusUnauthorized, "invalid_credentials", "Invalid email or password", nil)
		case errors.Is(err, authuc.ErrEmailNotVerified):
			response.Error(c, http.StatusForbidden, "email_not_verified", "Email is not verified", nil)
		default:
			h.logger.Error("internal_error_in_login", map[string]any{
				"email":  req.Email,
				"path":   c.Request.URL.Path,
				"method": c.Request.Method,
				"error":  err.Error(),
			})
			response.Error(c, http.StatusInternalServerError, "internal_error", "Internal server error", nil)
		}
		return
	}

	resp := LoginResponse{
		UserID:   user.ID.String(),
		Email:    user.Email,
		Username: user.Username,
		Tokens: response.TokenPair{
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
			h.logger.Error("internal_error_in_refresh", map[string]any{
				"path":   c.Request.URL.Path,
				"method": c.Request.Method,
				"error":  err.Error(),
			})
			response.Error(c, http.StatusInternalServerError, "internal_error", "Internal server error", nil)
		}
		return
	}

	resp := LoginResponse{
		UserID:   user.ID.String(),
		Email:    user.Email,
		Username: user.Username,
		Tokens: response.TokenPair{
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
			h.logger.Error("internal_error_in_resend_verification", map[string]any{
				"email":  req.Email,
				"path":   c.Request.URL.Path,
				"method": c.Request.Method,
				"error":  err.Error(),
			})
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
			h.logger.Error("internal_error_in_verify_email", map[string]any{
				"email":  req.Email,
				"path":   c.Request.URL.Path,
				"method": c.Request.Method,
				"error":  err.Error(),
			})
			response.Error(c, http.StatusInternalServerError, "internal_error", "Internal server error", nil)
		}
		return
	}

	resp := LoginResponse{
		UserID:   user.ID.String(),
		Email:    user.Email,
		Username: user.Username,
		Tokens: response.TokenPair{
			AccessToken:  access,
			RefreshToken: refresh,
		},
	}

	c.JSON(http.StatusOK, resp)
}

// RestoreAccount godoc
// @Summary      Восстановление удалённого аккаунта
// @Description  Восстанавливает мягко удалённый аккаунт по email и паролю. Возвращает профиль пользователя и пару access/refresh токенов.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        payload  body      RestoreAccountRequest  true  "Данные для восстановления"
// @Success      200      {object}  RestoreAccountResponse
// @Failure      400      {object}  response.ErrorBody
// @Failure      401      {object}  response.ErrorBody
// @Failure      403      {object}  response.ErrorBody
// @Failure      500      {object}  response.ErrorBody
// @Router       /api/v1/auth/restore [post]
func (h *Handler) RestoreAccount(c *gin.Context) {
	var req RestoreAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid_request", "Invalid request body", err.Error())
		return
	}

	user, access, refresh, err := h.auth.RestoreAccount(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, authuc.ErrInvalidCredentials):
			h.logger.Info("invalid_password_in_restore_account", map[string]any{
				"email":  req.Email,
				"path":   c.Request.URL.Path,
				"method": c.Request.Method,
			})
			response.Error(c, http.StatusUnauthorized, "invalid_credentials", "Invalid email or password", nil)
		case errors.Is(err, authuc.ErrAccountNotDeleted):
			h.logger.Info("account_not_deleted_in_restore", map[string]any{
				"email":  req.Email,
				"path":   c.Request.URL.Path,
				"method": c.Request.Method,
			})
			response.Error(c, http.StatusBadRequest, "account_not_deleted", "Account is not deleted", nil)
		case errors.Is(err, authuc.ErrEmailNotVerified):
			h.logger.Info("email_not_verified_in_restore", map[string]any{
				"email":  req.Email,
				"path":   c.Request.URL.Path,
				"method": c.Request.Method,
			})
			response.Error(c, http.StatusForbidden, "email_not_verified", "Email is not verified. Please verify your email first.", nil)
		case errors.Is(err, repo.ErrNotFound):
			// For security, return the same error as invalid password
			h.logger.Info("user_not_found_in_restore", map[string]any{
				"email":  req.Email,
				"path":   c.Request.URL.Path,
				"method": c.Request.Method,
			})
			response.Error(c, http.StatusUnauthorized, "invalid_credentials", "Invalid email or password", nil)
		default:
			ctx := map[string]any{
				"email":  req.Email,
				"path":   c.Request.URL.Path,
				"method": c.Request.Method,
				"error":  err.Error(),
			}
			h.logger.Error("internal_error_in_restore_account", ctx)
			response.Error(c, http.StatusInternalServerError, "internal_error", "Internal server error", nil)
		}
		return
	}

	resp := RestoreAccountResponse{
		UserID:   user.ID.String(),
		Email:    user.Email,
		Username: user.Username,
		Tokens: response.TokenPair{
			AccessToken:  access,
			RefreshToken: refresh,
		},
	}

	c.JSON(http.StatusOK, resp)
}
