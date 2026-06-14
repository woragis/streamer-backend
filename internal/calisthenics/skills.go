package calisthenics

type ProficiencyLevel string

const (
	ProficiencyUnknown     ProficiencyLevel = "unknown"
	ProficiencyLearning    ProficiencyLevel = "learning"
	ProficiencyAttempting  ProficiencyLevel = "attempting"
	ProficiencyConsistent  ProficiencyLevel = "consistent"
	ProficiencyMastered    ProficiencyLevel = "mastered"
)

func ValidProficiencyLevel(l ProficiencyLevel) bool {
	switch l {
	case ProficiencyUnknown, ProficiencyLearning, ProficiencyAttempting, ProficiencyConsistent, ProficiencyMastered:
		return true
	default:
		return false
	}
}

type MovementCategory struct {
	ID     string `json:"id"`
	RoomID string `json:"roomId"`
	Name   string `json:"name"`
}

type Movement struct {
	ID             string   `json:"id"`
	RoomID         string   `json:"roomId"`
	Slug           string   `json:"slug"`
	Name           string   `json:"name"`
	CategoryID     string   `json:"categoryId"`
	Description    string   `json:"description"`
	Prerequisites  []string `json:"prerequisites"`
	Proficiency    *MovementProficiency `json:"proficiency,omitempty"`
}

type MovementProficiency struct {
	MovementID         string           `json:"movementId"`
	Level              ProficiencyLevel `json:"level"`
	Notes              string           `json:"notes"`
	BestHoldSeconds    *int             `json:"bestHoldSeconds,omitempty"`
	BestReps           *int             `json:"bestReps,omitempty"`
	ProgressionVariant *string          `json:"progressionVariant,omitempty"`
	UpdatedAt          string           `json:"updatedAt"`
}

type SkillAcquisition struct {
	ID                string           `json:"id"`
	RoomID            string           `json:"roomId"`
	MovementID        string           `json:"movementId"`
	MovementName      string           `json:"movementName,omitempty"`
	LiveSessionID     *string          `json:"liveSessionId,omitempty"`
	AcquiredAt        string           `json:"acquiredAt"`
	ProficiencyBefore ProficiencyLevel `json:"proficiencyBefore"`
	ProficiencyAfter  ProficiencyLevel `json:"proficiencyAfter"`
	Notes             string           `json:"notes"`
	EvidenceURL       *string          `json:"evidenceUrl,omitempty"`
	Acknowledged      bool             `json:"acknowledged"`
}

type SkillAlert struct {
	AcquisitionID    string           `json:"acquisitionId"`
	MovementID       string           `json:"movementId"`
	MovementName     string           `json:"movementName"`
	AcquiredAt       string           `json:"acquiredAt"`
	Notes            string           `json:"notes"`
	ProficiencyAfter ProficiencyLevel `json:"proficiencyAfter"`
}

type SkillStats struct {
	AcquisitionCount int      `json:"acquisitionCount"`
	MovementIDs      []string `json:"movementIds,omitempty"`
	Month            string   `json:"month,omitempty"`
	LiveSessionID    string   `json:"liveSessionId,omitempty"`
}

type CreateMovementInput struct {
	Slug          string   `json:"slug"`
	Name          string   `json:"name"`
	CategoryID    string   `json:"categoryId"`
	Description   string   `json:"description,omitempty"`
	Prerequisites []string `json:"prerequisites,omitempty"`
}

type UpdateMovementInput struct {
	Name          *string  `json:"name,omitempty"`
	CategoryID    *string  `json:"categoryId,omitempty"`
	Description   *string  `json:"description,omitempty"`
	Prerequisites []string `json:"prerequisites,omitempty"`
}

type UpdateProficiencyInput struct {
	Level              ProficiencyLevel `json:"level"`
	Notes              *string          `json:"notes,omitempty"`
	BestHoldSeconds    *int             `json:"bestHoldSeconds,omitempty"`
	BestReps           *int             `json:"bestReps,omitempty"`
	ProgressionVariant *string          `json:"progressionVariant,omitempty"`
}

type CreateAcquisitionInput struct {
	MovementID        string            `json:"movementId"`
	LiveSessionID     *string           `json:"liveSessionId,omitempty"`
	ProficiencyBefore *ProficiencyLevel `json:"proficiencyBefore,omitempty"`
	ProficiencyAfter  ProficiencyLevel  `json:"proficiencyAfter"`
	Notes             string            `json:"notes,omitempty"`
	EvidenceURL       *string           `json:"evidenceUrl,omitempty"`
	UpdateProficiency *bool             `json:"updateProficiency,omitempty"`
}

type CreatePracticeLogInput struct {
	MovementID       string  `json:"movementId"`
	LiveSessionID    *string `json:"liveSessionId,omitempty"`
	DurationSeconds  int     `json:"durationSeconds,omitempty"`
	Notes            string  `json:"notes,omitempty"`
}
