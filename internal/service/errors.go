package service

import "errors"

var (
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrNotFound          = errors.New("not found")
	ErrBadgeMaxLevel     = errors.New("badge already at max level")
	ErrQuestAlreadyActive = errors.New("quest already active for user")
	ErrQuestNotActive    = errors.New("quest is not in active status")
)
