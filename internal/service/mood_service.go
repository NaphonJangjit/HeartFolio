package service

import (
	"context"
	"fmt"
	"time"

	imongo "github.com/NaphonJangjit/HeartFolio/internal/db/mongo"
	"github.com/NaphonJangjit/HeartFolio/internal/model"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type MoodService struct {
	moodRepo *imongo.Repository[model.MoodEntry]
}

func NewMoodService(moodRepo *imongo.Repository[model.MoodEntry]) *MoodService {
	return &MoodService{moodRepo: moodRepo}
}

func (s *MoodService) EnsureIndexes(moodColl *mongo.Collection) error {
	ctx := context.Background()
	_, err := moodColl.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "created_at", Value: -1}},
	})
	if err != nil {
		return fmt.Errorf("failed to create mood index: %w", err)
	}
	return nil
}

// Submit records a new mood entry for the user.
func (s *MoodService) Submit(ctx context.Context, userID bson.ObjectID, mood, note string) (*model.MoodEntry, error) {
	entry := &model.MoodEntry{
		UserID:    userID,
		Mood:      mood,
		Note:      note,
		CreatedAt: time.Now(),
	}
	_, err := s.moodRepo.InsertOne(ctx, entry)
	if err != nil {
		return nil, err
	}
	return entry, nil
}

// GetRecent returns the most recent mood entries for a user (up to limit).
func (s *MoodService) GetRecent(ctx context.Context, userID bson.ObjectID, limit int) ([]*model.MoodEntry, error) {
	return s.moodRepo.FindWithOptions(ctx, bson.M{"user_id": userID},
		options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}).SetLimit(int64(limit)),
	)
}

// GetHistory returns all mood entries for a user, ordered newest first.
func (s *MoodService) GetHistory(ctx context.Context, userID bson.ObjectID) ([]*model.MoodEntry, error) {
	return s.moodRepo.FindWithOptions(ctx, bson.M{"user_id": userID},
		options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}),
	)
}
