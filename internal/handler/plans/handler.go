package plans

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	plandomain "workout-app/internal/domain/plan"
	userdomain "workout-app/internal/domain/user"
	"workout-app/internal/handler/middleware"
	"workout-app/internal/handler/response"
	plansuc "workout-app/internal/usecase/plans"
	"workout-app/pkg/logger"
)

// Handler обрабатывает HTTP-запросы для планов тренировок.
type Handler struct {
	plans plansuc.Service
	logger logger.Logger
}

// NewHandler создаёт handler планов.
func NewHandler(plans plansuc.Service, logger logger.Logger) *Handler {
	return &Handler{plans: plans, logger: logger}
}

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

// getCallerRoleFromContext возвращает роль вызывающего. При отсутствии ключа в контексте возвращает RoleUser (не admin).
func getCallerRoleFromContext(c *gin.Context) userdomain.Role {
	r := c.GetString(middleware.ContextUserRoleKey)
	if r == "" {
		return userdomain.RoleUser
	}
	return userdomain.Role(r)
}

func toPlanListItemResponse(p *plandomain.Plan) PlanListItemResponse {
	r := PlanListItemResponse{
		ID:        p.ID.String(),
		Name:      p.Name,
		IsActive:  p.IsActive,
		CreatedAt: formatTime(p.CreatedAt),
	}
	if p.Category != nil {
		s := string(*p.Category)
		r.Category = &s
	}
	if p.Level != nil {
		s := string(*p.Level)
		r.Level = &s
	}
	return r
}

func toPlanDetailResponse(p *plandomain.Plan) PlanDetailResponse {
	r := PlanDetailResponse{
		ID:       p.ID.String(),
		Name:     p.Name,
		IsActive: p.IsActive,
		Days:     make([]PlanDayDetailResponse, 0, len(p.Days)),
	}
	if p.Category != nil {
		s := string(*p.Category)
		r.Category = &s
	}
	if p.Level != nil {
		s := string(*p.Level)
		r.Level = &s
	}
	for _, d := range p.Days {
		day := PlanDayDetailResponse{
			ID:        d.ID.String(),
			Name:      d.Name,
			SortOrder: d.SortOrder,
			Exercises: make([]PlanDayExerciseItemResponse, 0, len(d.Exercises)),
		}
		for _, ex := range d.Exercises {
			day.Exercises = append(day.Exercises, PlanDayExerciseItemResponse{
				ID:              ex.ID.String(),
				ExerciseID:      ex.ExerciseID,
				Sets:            ex.Sets,
				Reps:            ex.Reps,
				WeightKg:        ex.WeightKg,
				DurationSeconds: ex.DurationSeconds,
				DistanceMeters:  ex.DistanceMeters,
				RestSeconds:     ex.RestSeconds,
				IsSuperset:      ex.IsSuperset,
				SupersetGroup:   ex.SupersetGroup,
				SortOrder:       ex.SortOrder,
			})
		}
		r.Days = append(r.Days, day)
	}
	return r
}

func toPlanCreatedResponse(p *plandomain.Plan) PlanCreatedResponse {
	r := PlanCreatedResponse{
		ID:        p.ID.String(),
		UserID:    p.UserID.String(),
		Name:      p.Name,
		IsActive:  p.IsActive,
		IsPublic:  p.IsPublic,
		CreatedAt: formatTime(p.CreatedAt),
		UpdatedAt: formatTime(p.UpdatedAt),
	}
	if p.Category != nil {
		s := string(*p.Category)
		r.Category = &s
	}
	if p.Level != nil {
		s := string(*p.Level)
		r.Level = &s
	}
	return r
}

func toPlanUpdatedResponse(p *plandomain.Plan) PlanUpdatedResponse {
	r := PlanUpdatedResponse{
		ID:        p.ID.String(),
		UserID:    p.UserID.String(),
		Name:      p.Name,
		IsActive:  p.IsActive,
		IsPublic:  p.IsPublic,
		CreatedAt: formatTime(p.CreatedAt),
		UpdatedAt: formatTime(p.UpdatedAt),
	}
	if p.Category != nil {
		s := string(*p.Category)
		r.Category = &s
	}
	if p.Level != nil {
		s := string(*p.Level)
		r.Level = &s
	}
	return r
}

func allowedCategoriesDetails() interface{} {
	s := make([]string, 0, len(plandomain.AllCategories))
	for _, c := range plandomain.AllCategories {
		s = append(s, string(c))
	}
	return map[string]interface{}{"allowed_category": s}
}

func allowedLevelsDetails() interface{} {
	s := make([]string, 0, len(plandomain.AllLevels))
	for _, l := range plandomain.AllLevels {
		s = append(s, string(l))
	}
	return map[string]interface{}{"allowed_level": s}
}

func isPlanOrDayNotFound(err error) bool {
	return errors.Is(err, plansuc.ErrPlanNotFound) || errors.Is(err, plansuc.ErrDayNotFound)
}

func responsePlanOrDayNotFound(c *gin.Context) {
	response.Error(c, http.StatusNotFound, "not_found", "План или день не найден", nil)
}

func isPlanDayOrExerciseNotFound(err error) bool {
	return errors.Is(err, plansuc.ErrPlanNotFound) || errors.Is(err, plansuc.ErrDayNotFound) || errors.Is(err, plansuc.ErrExerciseEntryNotFound)
}

func responsePlanDayOrExerciseNotFound(c *gin.Context) {
	response.Error(c, http.StatusNotFound, "not_found", "План, день или запись упражнения не найдены", nil)
}

// List godoc
// @Summary      Список планов текущего пользователя
// @Description  Возвращает планы текущего пользователя (id, name, is_active, category, level, created_at) без вложенных дней.
// @Tags         plans
// @Security     BearerAuth
// @Produce      json
// @Success      200  {array}   PlanListItemResponse
// @Failure      401  {object}  response.ErrorBody
// @Failure      500  {object}  response.ErrorBody
// @Router       /api/v1/plans [get]
func (h *Handler) List(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "unauthorized", "Требуется аутентификация", nil)
		return
	}
	list, err := h.plans.ListByUser(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("plans_list_error", map[string]any{
			"user_id": userID.String(),
			"path":    c.Request.URL.Path,
			"error":   err.Error(),
		})
		response.Error(c, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера", nil)
		return
	}
	resp := make([]PlanListItemResponse, 0, len(list))
	for _, p := range list {
		resp = append(resp, toPlanListItemResponse(p))
	}
	c.JSON(http.StatusOK, resp)
}

// GetByID godoc
// @Summary      Получить план по ID
// @Description  Возвращает один план со вложенными днями и упражнениями в днях. Доступ: владелец или план публичный (is_public = true).
// @Tags         plans
// @Security     BearerAuth
// @Produce      json
// @Param        id   path      string  true  "ID плана (UUID)"
// @Success      200  {object}  PlanDetailResponse
// @Failure      401  {object}  response.ErrorBody
// @Failure      403  {object}  response.ErrorBody
// @Failure      404  {object}  response.ErrorBody
// @Failure      500  {object}  response.ErrorBody
// @Router       /api/v1/plans/{id} [get]
func (h *Handler) GetByID(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "unauthorized", "Требуется аутентификация", nil)
		return
	}
	idStr := c.Param("id")
	if idStr == "" {
		response.Error(c, http.StatusBadRequest, "invalid_request", "ID плана обязателен", nil)
		return
	}
	planID, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid_request", "Некорректный формат ID плана", nil)
		return
	}
	plan, err := h.plans.GetByID(c.Request.Context(), planID, userID)
	if err != nil {
		if errors.Is(err, plansuc.ErrPlanNotFound) {
			response.Error(c, http.StatusNotFound, "plan_not_found", "План не найден", nil)
			return
		}
		if errors.Is(err, plansuc.ErrForbidden) {
			response.Error(c, http.StatusForbidden, "forbidden", "Нет доступа к этому плану", nil)
			return
		}
		h.logger.Error("plans_get_by_id_error", map[string]any{
			"plan_id": planID.String(),
			"user_id": userID.String(),
			"error":   err.Error(),
		})
		response.Error(c, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера", nil)
		return
	}
	c.JSON(http.StatusOK, toPlanDetailResponse(plan))
}

// Create godoc
// @Summary      Создать план
// @Description  Создаёт новый план. Публичный план (is_public: true) может создавать только admin — иначе 403. category и level при указании должны быть из допустимых значений.
// @Tags         plans
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        payload  body      CreatePlanRequest  true  "Данные плана"
// @Success      201      {object}  PlanCreatedResponse
// @Failure      400      {object}  response.ErrorBody  "Некорректное тело запроса, недопустимая category или level"
// @Failure      401      {object}  response.ErrorBody
// @Failure      403      {object}  response.ErrorBody
// @Failure      500      {object}  response.ErrorBody
// @Router       /api/v1/plans [post]
func (h *Handler) Create(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "unauthorized", "Требуется аутентификация", nil)
		return
	}
	var req CreatePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid_request", "Некорректное тело запроса", nil)
		return
	}
	input := plansuc.CreatePlanInput{
		Name:     req.Name,
		IsActive: req.IsActive,
		IsPublic: req.IsPublic,
		Category: req.Category,
		Level:    req.Level,
	}
	role := getCallerRoleFromContext(c)
	plan, err := h.plans.Create(c.Request.Context(), userID, role, input)
	if err != nil {
		if errors.Is(err, plansuc.ErrOnlyAdminCanPublishPlan) {
			response.Error(c, http.StatusForbidden, "forbidden", "Только администратор может создавать публичный план", nil)
			return
		}
		if errors.Is(err, plansuc.ErrInvalidCategory) {
			response.Error(c, http.StatusBadRequest, "invalid_category", "Недопустимая категория плана", allowedCategoriesDetails())
			return
		}
		if errors.Is(err, plansuc.ErrInvalidLevel) {
			response.Error(c, http.StatusBadRequest, "invalid_level", "Недопустимый уровень плана", allowedLevelsDetails())
			return
		}
		h.logger.Error("plans_create_error", map[string]any{
			"user_id": userID.String(),
			"error":   err.Error(),
		})
		response.Error(c, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера", nil)
		return
	}
	c.JSON(http.StatusCreated, toPlanCreatedResponse(plan))
}

// Update godoc
// @Summary      Обновить план
// @Description  Обновляет план. При is_public: true — только admin. При is_active: true снимает флаг с предыдущего активного плана. category и level при указании должны быть из допустимых значений.
// @Tags         plans
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id       path      string  true  "ID плана (UUID)"
// @Param        payload  body      UpdatePlanRequest  true  "Данные для обновления"
// @Success      200      {object}  PlanUpdatedResponse
// @Failure      400      {object}  response.ErrorBody  "Некорректное тело запроса, недопустимая category или level"
// @Failure      401      {object}  response.ErrorBody
// @Failure      403      {object}  response.ErrorBody
// @Failure      404      {object}  response.ErrorBody
// @Failure      500      {object}  response.ErrorBody
// @Router       /api/v1/plans/{id} [put]
func (h *Handler) Update(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "unauthorized", "Требуется аутентификация", nil)
		return
	}
	idStr := c.Param("id")
	if idStr == "" {
		response.Error(c, http.StatusBadRequest, "invalid_request", "ID плана обязателен", nil)
		return
	}
	planID, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid_request", "Некорректный формат ID плана", nil)
		return
	}
	var req UpdatePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid_request", "Некорректное тело запроса", nil)
		return
	}
	input := plansuc.UpdatePlanInput{
		Name:     req.Name,
		IsActive: req.IsActive,
		IsPublic: req.IsPublic,
		Category: req.Category,
		Level:    req.Level,
	}
	role := getCallerRoleFromContext(c)
	plan, err := h.plans.Update(c.Request.Context(), planID, userID, role, input)
	if err != nil {
		if errors.Is(err, plansuc.ErrPlanNotFound) {
			response.Error(c, http.StatusNotFound, "plan_not_found", "План не найден", nil)
			return
		}
		if errors.Is(err, plansuc.ErrForbidden) {
			response.Error(c, http.StatusForbidden, "forbidden", "Нет доступа к этому плану", nil)
			return
		}
		if errors.Is(err, plansuc.ErrOnlyAdminCanPublishPlan) {
			response.Error(c, http.StatusForbidden, "forbidden", "Только администратор может делать план публичным", nil)
			return
		}
		if errors.Is(err, plansuc.ErrInvalidCategory) {
			response.Error(c, http.StatusBadRequest, "invalid_category", "Недопустимая категория плана", allowedCategoriesDetails())
			return
		}
		if errors.Is(err, plansuc.ErrInvalidLevel) {
			response.Error(c, http.StatusBadRequest, "invalid_level", "Недопустимый уровень плана", allowedLevelsDetails())
			return
		}
		h.logger.Error("plans_update_error", map[string]any{
			"plan_id": planID.String(),
			"user_id": userID.String(),
			"error":   err.Error(),
		})
		response.Error(c, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера", nil)
		return
	}
	c.JSON(http.StatusOK, toPlanUpdatedResponse(plan))
}

// Delete godoc
// @Summary      Удалить план
// @Description  Удаляет план (CASCADE удалит дни и упражнения в днях). Только владелец.
// @Tags         plans
// @Security     BearerAuth
// @Param        id   path      string  true  "ID плана (UUID)"
// @Success      204  "No Content"
// @Failure      401  {object}  response.ErrorBody
// @Failure      403  {object}  response.ErrorBody
// @Failure      404  {object}  response.ErrorBody
// @Failure      500  {object}  response.ErrorBody
// @Router       /api/v1/plans/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "unauthorized", "Требуется аутентификация", nil)
		return
	}
	idStr := c.Param("id")
	if idStr == "" {
		response.Error(c, http.StatusBadRequest, "invalid_request", "ID плана обязателен", nil)
		return
	}
	planID, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid_request", "Некорректный формат ID плана", nil)
		return
	}
	err = h.plans.Delete(c.Request.Context(), planID, userID)
	if err != nil {
		if errors.Is(err, plansuc.ErrPlanNotFound) {
			response.Error(c, http.StatusNotFound, "plan_not_found", "План не найден", nil)
			return
		}
		if errors.Is(err, plansuc.ErrForbidden) {
			response.Error(c, http.StatusForbidden, "forbidden", "Нет доступа к этому плану", nil)
			return
		}
		h.logger.Error("plans_delete_error", map[string]any{
			"plan_id": planID.String(),
			"user_id": userID.String(),
			"error":   err.Error(),
		})
		response.Error(c, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера", nil)
		return
	}
	c.Status(http.StatusNoContent)
}

func toDayResponse(d *plandomain.PlanDay) DayResponse {
	return DayResponse{
		ID:        d.ID.String(),
		PlanID:    d.PlanID.String(),
		Name:      d.Name,
		SortOrder: d.SortOrder,
		CreatedAt: formatTime(d.CreatedAt),
		UpdatedAt: formatTime(d.UpdatedAt),
	}
}

func toDayExerciseResponse(ex *plandomain.PlanDayExercise) DayExerciseResponse {
	return DayExerciseResponse{
		ID:              ex.ID.String(),
		DayID:           ex.DayID.String(),
		ExerciseID:      ex.ExerciseID,
		Sets:            ex.Sets,
		Reps:            ex.Reps,
		WeightKg:        ex.WeightKg,
		DurationSeconds: ex.DurationSeconds,
		DistanceMeters:  ex.DistanceMeters,
		RestSeconds:     ex.RestSeconds,
		IsSuperset:      ex.IsSuperset,
		SupersetGroup:   ex.SupersetGroup,
		SortOrder:       ex.SortOrder,
		CreatedAt:       formatTime(ex.CreatedAt),
		UpdatedAt:       formatTime(ex.UpdatedAt),
	}
}

func toDayExerciseResponses(list []*plandomain.PlanDayExercise) []DayExerciseResponse {
	out := make([]DayExerciseResponse, 0, len(list))
	for _, ex := range list {
		out = append(out, toDayExerciseResponse(ex))
	}
	return out
}

func parsePlanDayIDs(c *gin.Context) (planID, dayID uuid.UUID, ok bool) {
	planID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid_request", "Некорректный формат ID плана", nil)
		return uuid.Nil, uuid.Nil, false
	}
	dayID, err = uuid.Parse(c.Param("dayId"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid_request", "Некорректный формат ID дня", nil)
		return uuid.Nil, uuid.Nil, false
	}
	return planID, dayID, true
}

// AddDay godoc
// @Summary      Добавить день в план
// @Description  Добавляет день в план (name, sort_order?). Только владелец плана.
// @Tags         plans
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id       path      string  true  "ID плана (UUID)"
// @Param        payload  body      CreateDayRequest  true  "Данные дня"
// @Success      201      {object}  DayResponse
// @Failure      400      {object}  response.ErrorBody
// @Failure      401      {object}  response.ErrorBody
// @Failure      403      {object}  response.ErrorBody
// @Failure      404      {object}  response.ErrorBody
// @Failure      500      {object}  response.ErrorBody
// @Router       /api/v1/plans/{id}/days [post]
func (h *Handler) AddDay(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "unauthorized", "Требуется аутентификация", nil)
		return
	}
	planID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid_request", "Некорректный формат ID плана", nil)
		return
	}
	var req CreateDayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid_request", "Некорректное тело запроса", nil)
		return
	}
	input := plansuc.CreateDayInput{Name: req.Name, SortOrder: req.SortOrder}
	day, err := h.plans.AddDay(c.Request.Context(), planID, userID, input)
	if err != nil {
		if errors.Is(err, plansuc.ErrPlanNotFound) {
			response.Error(c, http.StatusNotFound, "plan_not_found", "План не найден", nil)
			return
		}
		if errors.Is(err, plansuc.ErrForbidden) {
			response.Error(c, http.StatusForbidden, "forbidden", "Нет доступа к этому плану", nil)
			return
		}
		h.logger.Error("plans_add_day_error", map[string]any{
			"plan_id": planID.String(),
			"user_id": userID.String(),
			"error":   err.Error(),
		})
		response.Error(c, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера", nil)
		return
	}
	c.JSON(http.StatusCreated, toDayResponse(day))
}

// UpdateDay godoc
// @Summary      Обновить день плана
// @Description  Обновляет день (name, sort_order). Только владелец плана.
// @Tags         plans
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id       path      string  true  "ID плана (UUID)"
// @Param        dayId    path      string  true  "ID дня (UUID)"
// @Param        payload  body      UpdateDayRequest  true  "Данные для обновления"
// @Success      200      {object}  DayResponse
// @Failure      400      {object}  response.ErrorBody
// @Failure      401      {object}  response.ErrorBody
// @Failure      403      {object}  response.ErrorBody
// @Failure      404      {object}  response.ErrorBody
// @Failure      500      {object}  response.ErrorBody
// @Router       /api/v1/plans/{id}/days/{dayId} [put]
func (h *Handler) UpdateDay(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "unauthorized", "Требуется аутентификация", nil)
		return
	}
	planID, dayID, ok := parsePlanDayIDs(c)
	if !ok {
		return
	}
	var req UpdateDayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid_request", "Некорректное тело запроса", nil)
		return
	}
	input := plansuc.UpdateDayInput{Name: req.Name, SortOrder: req.SortOrder}
	day, err := h.plans.UpdateDay(c.Request.Context(), planID, dayID, userID, input)
	if err != nil {
		if isPlanOrDayNotFound(err) {
			responsePlanOrDayNotFound(c)
			return
		}
		if errors.Is(err, plansuc.ErrForbidden) {
			response.Error(c, http.StatusForbidden, "forbidden", "Нет доступа к этому плану", nil)
			return
		}
		h.logger.Error("plans_update_day_error", map[string]any{
			"plan_id": planID.String(),
			"user_id": userID.String(),
			"error":   err.Error(),
		})
		response.Error(c, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера", nil)
		return
	}
	c.JSON(http.StatusOK, toDayResponse(day))
}

// DeleteDay godoc
// @Summary      Удалить день из плана
// @Description  Удаляет день и все упражнения в нём. Только владелец плана.
// @Tags         plans
// @Security     BearerAuth
// @Param        id     path  string  true  "ID плана (UUID)"
// @Param        dayId  path  string  true  "ID дня (UUID)"
// @Success      204    "No Content"
// @Failure      401    {object}  response.ErrorBody
// @Failure      403    {object}  response.ErrorBody
// @Failure      404    {object}  response.ErrorBody
// @Failure      500    {object}  response.ErrorBody
// @Router       /api/v1/plans/{id}/days/{dayId} [delete]
func (h *Handler) DeleteDay(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "unauthorized", "Требуется аутентификация", nil)
		return
	}
	planID, dayID, ok := parsePlanDayIDs(c)
	if !ok {
		return
	}
	err = h.plans.DeleteDay(c.Request.Context(), planID, dayID, userID)
	if err != nil {
		if isPlanOrDayNotFound(err) {
			responsePlanOrDayNotFound(c)
			return
		}
		if errors.Is(err, plansuc.ErrForbidden) {
			response.Error(c, http.StatusForbidden, "forbidden", "Нет доступа к этому плану", nil)
			return
		}
		h.logger.Error("plans_delete_day_error", map[string]any{
			"plan_id": planID.String(),
			"user_id": userID.String(),
			"error":   err.Error(),
		})
		response.Error(c, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера", nil)
		return
	}
	c.Status(http.StatusNoContent)
}

// AddExerciseToDay godoc
// @Summary      Добавить упражнения в день
// @Description  Добавляет массив упражнений в день. Тело — JSON-массив объектов. Валидация по каждому элементу; при любой ошибке — 400 с details (index, code, message). exercise_id из каталога, sets: 1–20.
// @Tags         plans
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id       path      string  true  "ID плана (UUID)"
// @Param        dayId    path      string  true  "ID дня (UUID)"
// @Param        payload  body      []CreateDayExerciseRequest  true  "Массив упражнений"
// @Success      201      {array}   DayExerciseResponse
// @Failure      400      {object}  response.ErrorBody  "invalid_request или validation_error (details — массив ошибок по индексу)"
// @Failure      401      {object}  response.ErrorBody
// @Failure      403      {object}  response.ErrorBody
// @Failure      404      {object}  response.ErrorBody
// @Failure      500      {object}  response.ErrorBody
// @Router       /api/v1/plans/{id}/days/{dayId}/exercises [post]
func (h *Handler) AddExerciseToDay(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "unauthorized", "Требуется аутентификация", nil)
		return
	}
	planID, dayID, ok := parsePlanDayIDs(c)
	if !ok {
		return
	}
	var req []CreateDayExerciseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid_request", "Некорректное тело запроса", nil)
		return
	}
	if len(req) == 0 {
		response.Error(c, http.StatusBadRequest, "invalid_request", "Список упражнений не может быть пустым", nil)
		return
	}
	inputs := make([]plansuc.CreateDayExerciseInput, 0, len(req))
	for _, r := range req {
		inputs = append(inputs, plansuc.CreateDayExerciseInput{
			ExerciseID:      r.ExerciseID,
			Sets:            r.Sets,
			Reps:            r.Reps,
			WeightKg:        r.WeightKg,
			DurationSeconds: r.DurationSeconds,
			DistanceMeters:  r.DistanceMeters,
			RestSeconds:     r.RestSeconds,
			IsSuperset:      r.IsSuperset,
			SortOrder:       r.SortOrder,
		})
	}
	list, err := h.plans.AddExercisesToDay(c.Request.Context(), planID, dayID, userID, inputs)
	if err != nil {
		if isPlanOrDayNotFound(err) {
			responsePlanOrDayNotFound(c)
			return
		}
		if errors.Is(err, plansuc.ErrForbidden) {
			response.Error(c, http.StatusForbidden, "forbidden", "Нет доступа к этому плану", nil)
			return
		}
		if errors.Is(err, plansuc.ErrNoExercisesProvided) {
			response.Error(c, http.StatusBadRequest, "invalid_request", "Список упражнений не может быть пустым", nil)
			return
		}
		var valErr *plansuc.ErrValidation
		if errors.As(err, &valErr) {
			response.Error(c, http.StatusBadRequest, "validation_error", "Ошибки валидации упражнений", valErr.Errors)
			return
		}
		h.logger.Error("plans_add_exercises_error", map[string]any{
			"plan_id": planID.String(),
			"user_id": userID.String(),
			"error":   err.Error(),
		})
		response.Error(c, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера", nil)
		return
	}
	c.JSON(http.StatusCreated, toDayExerciseResponses(list))
}

// UpdateExerciseInDay godoc
// @Summary      Обновить упражнение в дне
// @Description  Обновляет параметры записи упражнения в дне (sets, reps, weight_kg и т.д.). sets при указании: 1–20.
// @Tags         plans
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id                path  string  true  "ID плана (UUID)"
// @Param        dayId             path  string  true  "ID дня (UUID)"
// @Param        exerciseEntryId  path  string  true  "ID записи упражнения в дне (UUID)"
// @Param        payload           body  UpdateDayExerciseRequest  true  "Данные для обновления"
// @Success      200               {object}  DayExerciseResponse
// @Failure      400               {object}  response.ErrorBody  "invalid_sets (sets не 1–20)"
// @Failure      401               {object}  response.ErrorBody
// @Failure      403               {object}  response.ErrorBody
// @Failure      404               {object}  response.ErrorBody
// @Failure      500               {object}  response.ErrorBody
// @Router       /api/v1/plans/{id}/days/{dayId}/exercises/{exerciseEntryId} [put]
func (h *Handler) UpdateExerciseInDay(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "unauthorized", "Требуется аутентификация", nil)
		return
	}
	planID, dayID, ok := parsePlanDayIDs(c)
	if !ok {
		return
	}
	exerciseEntryID, err := uuid.Parse(c.Param("exerciseEntryId"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid_request", "Некорректный формат ID записи упражнения", nil)
		return
	}
	var req UpdateDayExerciseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid_request", "Некорректное тело запроса", nil)
		return
	}
	input := plansuc.UpdateDayExerciseInput{
		Sets:            req.Sets,
		Reps:            req.Reps,
		WeightKg:        req.WeightKg,
		DurationSeconds: req.DurationSeconds,
		DistanceMeters:  req.DistanceMeters,
		RestSeconds:     req.RestSeconds,
		IsSuperset:      req.IsSuperset,
		SortOrder:       req.SortOrder,
	}
	ex, err := h.plans.UpdateExerciseInDay(c.Request.Context(), planID, dayID, exerciseEntryID, userID, input)
	if err != nil {
		if isPlanDayOrExerciseNotFound(err) {
			responsePlanDayOrExerciseNotFound(c)
			return
		}
		if errors.Is(err, plansuc.ErrForbidden) {
			response.Error(c, http.StatusForbidden, "forbidden", "Нет доступа к этому плану", nil)
			return
		}
		if errors.Is(err, plansuc.ErrInvalidSetsRange) {
			response.Error(c, http.StatusBadRequest, "invalid_sets", "Количество подходов должно быть от 1 до 20", nil)
			return
		}
		h.logger.Error("plans_update_exercise_error", map[string]any{
			"plan_id": planID.String(),
			"user_id": userID.String(),
			"error":   err.Error(),
		})
		response.Error(c, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера", nil)
		return
	}
	c.JSON(http.StatusOK, toDayExerciseResponse(ex))
}

// DeleteExerciseFromDay godoc
// @Summary      Удалить упражнение из дня
// @Description  Удаляет запись упражнения из дня плана.
// @Tags         plans
// @Security     BearerAuth
// @Param        id               path  string  true  "ID плана (UUID)"
// @Param        dayId            path  string  true  "ID дня (UUID)"
// @Param        exerciseEntryId  path  string  true  "ID записи упражнения в дне (UUID)"
// @Success      204  "No Content"
// @Failure      401  {object}  response.ErrorBody
// @Failure      403  {object}  response.ErrorBody
// @Failure      404  {object}  response.ErrorBody
// @Failure      500  {object}  response.ErrorBody
// @Router       /api/v1/plans/{id}/days/{dayId}/exercises/{exerciseEntryId} [delete]
func (h *Handler) DeleteExerciseFromDay(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "unauthorized", "Требуется аутентификация", nil)
		return
	}
	planID, dayID, ok := parsePlanDayIDs(c)
	if !ok {
		return
	}
	exerciseEntryID, err := uuid.Parse(c.Param("exerciseEntryId"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid_request", "Некорректный формат ID записи упражнения", nil)
		return
	}
	err = h.plans.DeleteExerciseFromDay(c.Request.Context(), planID, dayID, exerciseEntryID, userID)
	if err != nil {
		if isPlanDayOrExerciseNotFound(err) {
			responsePlanDayOrExerciseNotFound(c)
			return
		}
		if errors.Is(err, plansuc.ErrForbidden) {
			response.Error(c, http.StatusForbidden, "forbidden", "Нет доступа к этому плану", nil)
			return
		}
		h.logger.Error("plans_delete_exercise_error", map[string]any{
			"plan_id": planID.String(),
			"user_id": userID.String(),
			"error":   err.Error(),
		})
		response.Error(c, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера", nil)
		return
	}
	c.Status(http.StatusNoContent)
}
