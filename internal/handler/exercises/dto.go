package exercises

// VersionResponse — ответ эндпоинта GET /api/v1/exercises/version.
type VersionResponse struct {
	Version string `json:"version"`
}
