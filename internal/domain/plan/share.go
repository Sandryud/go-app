package plan

import (
	"time"

	"github.com/google/uuid"
)

// PlanCopyChannel — канал, через который создана копия плана.
const PlanCopyChannelShare = "share"

// PlanShare — активная или отозванная публичная ссылка на план.
type PlanShare struct {
	ID        uuid.UUID
	PlanID    uuid.UUID
	Token     uuid.UUID
	CreatedAt time.Time
	RevokedAt *time.Time
}

// IsActive возвращает true, если ссылка не отозвана.
func (s *PlanShare) IsActive() bool {
	return s != nil && s.RevokedAt == nil
}
