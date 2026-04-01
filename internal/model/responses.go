package model

import "time"

// UserBadgeDetail is a denormalized view of a user's badge with full badge info.
type UserBadgeDetail struct {
	UserBadgeID string    `json:"user_badge_id"`
	Badge       *Badge    `json:"badge"`
	Level       int32     `json:"level"`
	EarnedAt    time.Time `json:"earned_at"`
	LastUpdated time.Time `json:"last_updated"`
}

// UserQuestDetail is a denormalized view of a user's quest with full quest info.
type UserQuestDetail struct {
	UserQuestID string      `json:"user_quest_id"`
	Quest       *Quest      `json:"quest"`
	Status      QuestStatus `json:"status"`
	StartedAt   time.Time   `json:"started_at"`
	CompletedAt *time.Time  `json:"completed_at,omitempty"`
	EXPEarned   int64       `json:"exp_earned"`
}
