package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	imongo "github.com/NaphonJangjit/HeartFolio/internal/db/mongo"
	redisDB "github.com/NaphonJangjit/HeartFolio/internal/db/redis"
	"github.com/NaphonJangjit/HeartFolio/internal/middleware"
	"github.com/NaphonJangjit/HeartFolio/internal/model"
	"go.mongodb.org/mongo-driver/v2/bson"
	mgDriver "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	repo      *imongo.Repository[model.User]
	redisRepo *redisDB.Repository
	jwtSecret []byte
}

func NewUserService(repo *imongo.Repository[model.User], redisRepo *redisDB.Repository, jwtSecret []byte) *UserService {
	return &UserService{repo: repo, redisRepo: redisRepo, jwtSecret: jwtSecret}
}

func (s *UserService) EnsureIndexes(usersColl *mgDriver.Collection) error {
	ctx := context.Background()
	_, err := usersColl.Indexes().CreateOne(ctx, mgDriver.IndexModel{
		Keys:    bson.D{{Key: "email", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return fmt.Errorf("failed to create user email unique index: %w", err)
	}
	return nil
}

func (s *UserService) Register(ctx context.Context, email, password string) (*model.User, string, error) {
	if email == "" || password == "" {
		return nil, "", errors.New("email and password required")
	}
	if err := ValidateEmail(email); err != nil {
		return nil, "", err
	}
	if err := ValidatePassword(password); err != nil {
		return nil, "", err
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
		Privacy:  model.DefaultPrivacy(),
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
	newLevel := CalculateLevel(newEXP)

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

func (s *UserService) UpdatePrivacy(ctx context.Context, userID bson.ObjectID, privacy model.PrivacySettings) error {
	update := bson.M{
		"$set": bson.M{
			"privacy": privacy,
		},
	}
	_, err := s.repo.UpdateOne(ctx, bson.M{"_id": userID}, update)
	if err != nil {
		return err
	}
	if s.redisRepo != nil {
		cacheKey := fmt.Sprintf("user:%s", userID.Hex())
		_ = s.redisRepo.Del(ctx, cacheKey)
	}
	return nil
}

// UpdateProfile updates the user's email.
func (s *UserService) UpdateProfile(ctx context.Context, userID bson.ObjectID, email string) (*model.User, error) {
	if email != "" {
		existing, err := s.repo.FindOne(ctx, bson.M{"email": email, "_id": bson.M{"$ne": userID}})
		if err != nil {
			return nil, fmt.Errorf("database error: %w", err)
		}
		if existing != nil {
			return nil, ErrUserAlreadyExists
		}
	}

	update := bson.M{}
	if email != "" {
		update["email"] = email
	}
	if len(update) == 0 {
		return s.repo.FindByID(ctx, userID)
	}

	_, err := s.repo.UpdateOne(ctx, bson.M{"_id": userID}, bson.M{"$set": update})
	if err != nil {
		return nil, err
	}

	if s.redisRepo != nil {
		cacheKey := fmt.Sprintf("user:%s", userID.Hex())
		_ = s.redisRepo.Del(ctx, cacheKey)
	}

	return s.repo.FindByID(ctx, userID)
}

// ChangePassword verifies the old password and sets a new one.
func (s *UserService) ChangePassword(ctx context.Context, userID bson.ObjectID, oldPassword, newPassword string) error {
	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrNotFound
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(oldPassword)); err != nil {
		return ErrInvalidCredentials
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	_, err = s.repo.UpdateOne(ctx, bson.M{"_id": userID}, bson.M{"$set": bson.M{"password": string(hashed)}})
	if err != nil {
		return err
	}

	if s.redisRepo != nil {
		cacheKey := fmt.Sprintf("user:%s", userID.Hex())
		_ = s.redisRepo.Del(ctx, cacheKey)
	}
	return nil
}

func CalculateLevel(exp int64) int32 {
	// Every 100 EXP = 1 level, starting at level 1
	level := int32(exp/100) + 1
	if level < 1 {
		level = 1
	}
	return level
}
