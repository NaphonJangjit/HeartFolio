package model

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type Badge struct {
	ID          bson.ObjectID `bson:"_id,omitempty" json:"id"`
	Name        string        `bson:"name" json:"name"`
	Description string        `bson:"description" json:"description"`
	Category    string        `bson:"category" json:"category"`
	MaxLevel    int32         `bson:"max_level" json:"max_level"`
	Criteria    string        `bson:"criteria" json:"criteria"`
	Icon        string        `bson:"icon,omitempty" json:"icon"`
	CreatedAt   time.Time     `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time     `bson:"updated_at" json:"updated_at"`
}

type UserBadge struct {
	ID          bson.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID      bson.ObjectID `bson:"user_id" json:"user_id"`
	BadgeID     bson.ObjectID `bson:"badge_id" json:"badge_id"`
	Level       int32         `bson:"level" json:"level"`
	EarnedAt    time.Time     `bson:"earned_at" json:"earned_at"`
	LastUpdated time.Time     `bson:"last_updated" json:"last_updated"`
}
