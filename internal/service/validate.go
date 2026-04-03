package service

import (
	"errors"
	"regexp"
	"strings"
)

var emailRegexp = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

var validMoods = map[string]bool{
	"happy": true, "sad": true, "anxious": true, "angry": true,
	"lonely": true, "burnout": true, "stressed": true, "calm": true,
	"motivated": true, "confused": true, "hopeful": true, "grateful": true,
	"overwhelmed": true, "content": true, "frustrated": true, "excited": true,
}

var (
	ErrInvalidEmail    = errors.New("invalid email format")
	ErrPasswordTooWeak = errors.New("password must be at least 8 characters")
	ErrInvalidMood     = errors.New("invalid mood value")
)

// ValidateEmail checks email format.
func ValidateEmail(email string) error {
	if !emailRegexp.MatchString(email) {
		return ErrInvalidEmail
	}
	return nil
}

// ValidatePassword checks minimum password strength.
func ValidatePassword(password string) error {
	if len(password) < 8 {
		return ErrPasswordTooWeak
	}
	return nil
}

// ValidateMood checks if the mood string is in the allowed set.
func ValidateMood(mood string) error {
	if !validMoods[strings.ToLower(mood)] {
		return ErrInvalidMood
	}
	return nil
}

// ValidMoods returns the list of allowed mood values.
func ValidMoods() []string {
	moods := make([]string, 0, len(validMoods))
	for m := range validMoods {
		moods = append(moods, m)
	}
	return moods
}
