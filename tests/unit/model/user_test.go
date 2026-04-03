package model_test

import (
	"testing"

	"github.com/NaphonJangjit/HeartFolio/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestUserTier(t *testing.T) {
	tests := []struct {
		name  string
		level int32
		tier  string
	}{
		{"level 0 is bronze", 0, "bronze"},
		{"level 1 is bronze", 1, "bronze"},
		{"level 5 is bronze", 5, "bronze"},
		{"level 6 is silver", 6, "silver"},
		{"level 10 is silver", 10, "silver"},
		{"level 15 is silver", 15, "silver"},
		{"level 16 is gold", 16, "gold"},
		{"level 30 is gold", 30, "gold"},
		{"level 31 is platinum", 31, "platinum"},
		{"level 50 is platinum", 50, "platinum"},
		{"level 51 is diamond", 51, "diamond"},
		{"level 100 is diamond", 100, "diamond"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &model.User{Level: tt.level}
			assert.Equal(t, tt.tier, u.Tier())
		})
	}
}

func TestDefaultPrivacy(t *testing.T) {
	p := model.DefaultPrivacy()
	assert.True(t, p.ShowBadges, "ShowBadges should default to true")
	assert.True(t, p.ShowStats, "ShowStats should default to true")
	assert.True(t, p.ShowQuests, "ShowQuests should default to true")
}
