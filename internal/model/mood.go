package model

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type MoodEntry struct {
	ID        bson.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID    bson.ObjectID `bson:"user_id" json:"user_id"`
	Mood      string        `bson:"mood" json:"mood"`
	Note      string        `bson:"note,omitempty" json:"note"`
	CreatedAt time.Time     `bson:"created_at" json:"created_at"`
}
