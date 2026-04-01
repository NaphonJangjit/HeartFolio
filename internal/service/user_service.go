package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/NaphonJangjit/HeartFolio/internal/db/mongo"
	redisDB "github.com/NaphonJangjit/HeartFolio/internal/db/redis"
	"github.com/NaphonJangjit/HeartFolio/internal/middleware"
	"github.com/NaphonJangjit/HeartFolio/internal/model"
	"go.mongodb.org/mongo-driver/v2/bson"
	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	repo      *mongo.Repository[model.User]
	redisRepo *redisDB.Repository
	jwtSecret []byte
}

func NewUserService(repo *mongo.Repository[model.User], redisRepo *redisDB.Repository, jwtSecret []byte) *UserService {
	return &UserService{repo: repo, redisRepo: redisRepo, jwtSecret: jwtSecret}
}

func (s *UserService) Register(ctx context.Context, email, password string) (*model.User, string, error) {
	if email == "" || password == "" {
		return nil, "", errors.New("email and password required")
	}

	existing, err := s.repo.FindOne(ctx, bson.M{"email": email})
	if err != nil {
		return nil, "", fmt.Errorf("database error: %w", err)
	}
	if existing != nil {
		return nil, "", ErrUserAlreadyExists
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", fmt.Errorf("failed to hash password: %w", err)
	}

	user := &model.User{
		Email:    email,
		Password: string(hashed),
		Role:     "user",
		EXP:      0,
		Level:    1,
	}
	_, err = s.repo.InsertOne(ctx, user)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create user: %w", err)
	}

	token, err := middleware.GenerateToken(s.jwtSecret, user.ID.Hex(), user.Email, user.Role)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate token: %w", err)
	}
	return user, token, nil
}

func (s *UserService) Login(ctx context.Context, email, password string) (*model.User, string, error) {
	if email == "" || password == "" {
		return nil, "", errors.New("email and password required")
	}

	user, err := s.repo.FindOne(ctx, bson.M{"email": email})
	if err != nil {
		return nil, "", fmt.Errorf("database error: %w", err)
	}
	if user == nil {
		return nil, "", ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, "", ErrInvalidCredentials
	}

	token, err := middleware.GenerateToken(s.jwtSecret, user.ID.Hex(), user.Email, user.Role)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate token: %w", err)
	}
	return user, token, nil
}

func (s *UserService) GetByID(ctx context.Context, userID string) (*model.User, error) {
	if s.redisRepo == nil {
		return s.repo.FindByID(ctx, userID)
	}

	cacheKey := fmt.Sprintf("user:%s", userID)
	userJSON, err := s.redisRepo.GetOrElse(ctx, cacheKey,
		func() (string, error) {
			user, err := s.repo.FindByID(ctx, userID)
			if err != nil {
				return "", err
			}
			if user == nil {
				return "", errors.New("user not found")
			}
			jsonBytes, err := json.Marshal(user)
			if err != nil {
				return "", err
			}
			return string(jsonBytes), nil
		},
		5*time.Minute,
	)
	if err != nil {
		return nil, err
	}

	var user model.User
	if err := json.Unmarshal([]byte(userJSON), &user); err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *UserService) AddEXP(ctx context.Context, userID bson.ObjectID, amount int64) (int64, int32, error) {
	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return 0, 0, err
	}
	if user == nil {
		return 0, 0, errors.New("user not found")
	}

	newEXP := user.EXP + amount
	newLevel := calculateLevel(newEXP)

	update := bson.M{
		"$set": bson.M{
			"exp":   newEXP,
			"level": newLevel,
		},
	}
	_, err = s.repo.UpdateOne(ctx, bson.M{"_id": userID}, update)
	if err != nil {
		return 0, 0, err
	}

	if s.redisRepo != nil {
		cacheKey := fmt.Sprintf("user:%s", userID.Hex())
		_ = s.redisRepo.Del(ctx, cacheKey)
	}

	return newEXP, newLevel, nil
}

func calculateLevel(exp int64) int32 {
	// Every 100 EXP = 1 level, starting at level 1
	level := int32(exp/100) + 1
	if level < 1 {
		level = 1
	}
	return level
}
