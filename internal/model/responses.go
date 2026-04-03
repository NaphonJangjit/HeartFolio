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

// SkillBadgeInfo holds a single badge's contribution to a skill category.
type SkillBadgeInfo struct {
	BadgeID  string `json:"badge_id"`
	Name     string `json:"name"`
	Level    int32  `json:"level"`
	MaxLevel int32  `json:"max_level"`
}

// SkillStat holds the detailed breakdown for one skill category.
type SkillStat struct {
	Category   string           `json:"category"`
	TotalLevel int32            `json:"total_level"`
	BadgeCount int32            `json:"badge_count"`
	Badges     []SkillBadgeInfo `json:"badges"`
}

// QuestSummary holds aggregated quest completion statistics.
type QuestSummary struct {
	TotalStarted   int32 `json:"total_started"`
	TotalCompleted int32 `json:"total_completed"`
	TotalAbandoned int32 `json:"total_abandoned"`
	TotalEXPEarned int64 `json:"total_exp_earned"`
}

// Portfolio is the full emotional portfolio aggregation for a user.
type Portfolio struct {
	UserID       string        `json:"user_id"`
	Email        string        `json:"email"`
	Level        int32         `json:"level"`
	EXP          int64         `json:"exp"`
	Tier         string        `json:"tier"`
	TotalBadges  int32         `json:"total_badges"`
	SkillStats   []SkillStat   `json:"skill_stats"`
	QuestSummary QuestSummary  `json:"quest_summary"`
	RecentMoods  []*MoodEntry  `json:"recent_moods"`
}
