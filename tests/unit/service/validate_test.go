package service_test

import (
	"testing"

	"github.com/NaphonJangjit/HeartFolio/internal/service"
	"github.com/stretchr/testify/assert"
)

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name  string
		email string
		valid bool
	}{
		{"standard email", "user@example.com", true},
		{"email with tags and subdomain", "test.user+tag@domain.co.th", true},
		{"minimal valid email", "a@b.cd", true},
		{"empty string", "", false},
		{"missing @", "not-an-email", false},
		{"missing local part", "@missing-local.com", false},
		{"missing domain", "missing-domain@", false},
		{"dot after @", "missing@.com", false},
		{"spaces in email", "spaces in@email.com", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.ValidateEmail(tt.email)
			if tt.valid {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, service.ErrInvalidEmail)
			}
		})
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		valid    bool
	}{
		{"exactly 8 chars", "12345678", true},
		{"long password", "longpasswordhere", true},
		{"too short", "short", false},
		{"7 chars", "1234567", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.ValidatePassword(tt.password)
			if tt.valid {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, service.ErrPasswordTooWeak)
			}
		})
	}
}

func TestValidateMood(t *testing.T) {
	tests := []struct {
		name  string
		mood  string
		valid bool
	}{
		{"lowercase happy", "happy", true},
		{"lowercase anxious", "anxious", true},
		{"lowercase burnout", "burnout", true},
		{"uppercase HAPPY", "HAPPY", true},
		{"mixed case Calm", "Calm", true},
		{"invalid mood", "invalid", false},
		{"empty string", "", false},
		{"nonexistent rage", "rage", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.ValidateMood(tt.mood)
			if tt.valid {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, service.ErrInvalidMood)
			}
		})
	}
}

func TestCalculateLevel(t *testing.T) {
	tests := []struct {
		name  string
		exp   int64
		level int32
	}{
		{"0 EXP", 0, 1},
		{"50 EXP", 50, 1},
		{"99 EXP", 99, 1},
		{"100 EXP", 100, 2},
		{"199 EXP", 199, 2},
		{"200 EXP", 200, 3},
		{"999 EXP", 999, 10},
		{"1000 EXP", 1000, 11},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.level, service.CalculateLevel(tt.exp))
		})
	}
}
