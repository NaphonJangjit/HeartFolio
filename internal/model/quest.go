package model

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type Quest struct {
	ID                bson.ObjectID `bson:"_id,omitempty" json:"id"`
	Name              string        `bson:"name" json:"name"`
	Description       string        `bson:"description" json:"description"`
	Category          string        `bson:"category" json:"category"`
	EXPReward         int64         `bson:"exp_reward" json:"exp_reward"`
	MoodTags          []string      `bson:"mood_tags" json:"mood_tags"`
	Difficulty        string        `bson:"difficulty" json:"difficulty"`
	EstimatedDuration int32         `bson:"estimated_duration" json:"estimated_duration"`
	BadgeID           bson.ObjectID `bson:"badge_id,omitempty" json:"badge_id,omitempty"`
	BadgeLevel        int32         `bson:"badge_level,omitempty" json:"badge_level,omitempty"`
	IsActive          bool          `bson:"is_active" json:"is_active"`
	CreatedAt         time.Time     `bson:"created_at" json:"created_at"`
	UpdatedAt         time.Time     `bson:"updated_at" json:"updated_at"`
}

type QuestStatus string

const (
	QuestStatusActive    QuestStatus = "active"
	QuestStatusCompleted QuestStatus = "completed"
	QuestStatusAbandoned QuestStatus = "abandoned"
)

type UserQuest struct {
	ID          bson.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID      bson.ObjectID `bson:"user_id" json:"user_id"`
	QuestID     bson.ObjectID `bson:"quest_id" json:"quest_id"`
	Status      QuestStatus   `bson:"status" json:"status"`
	StartedAt   time.Time     `bson:"started_at" json:"started_at"`
	CompletedAt *time.Time    `bson:"completed_at,omitempty" json:"completed_at,omitempty"`
	EXPEarned   int64         `bson:"exp_earned" json:"exp_earned"`
}
