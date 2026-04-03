package model

import "go.mongodb.org/mongo-driver/v2/bson"

type PrivacySettings struct {
	ShowBadges bool `bson:"show_badges" json:"show_badges"`
	ShowStats  bool `bson:"show_stats" json:"show_stats"`
	ShowQuests bool `bson:"show_quests" json:"show_quests"`
}

type User struct {
	ID       bson.ObjectID   `bson:"_id,omitempty" json:"id"`
	Email    string          `bson:"email" json:"email"`
	Password string          `bson:"password" json:"-"`
	Role     string          `bson:"role" json:"role"`
	EXP      int64           `bson:"exp" json:"exp"`
	Level    int32           `bson:"level" json:"level"`
	Privacy  PrivacySettings `bson:"privacy" json:"privacy"`
}

// DefaultPrivacy returns privacy settings with everything visible.
func DefaultPrivacy() PrivacySettings {
	return PrivacySettings{
		ShowBadges: true,
		ShowStats:  true,
		ShowQuests: true,
	}
}

// Tier returns the user's tier based on their level.
func (u *User) Tier() string {
	switch {
	case u.Level >= 51:
		return "diamond"
	case u.Level >= 31:
		return "platinum"
	case u.Level >= 16:
		return "gold"
	case u.Level >= 6:
		return "silver"
	default:
		return "bronze"
	}
}
