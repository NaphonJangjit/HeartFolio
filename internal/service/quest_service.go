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

type QuestService struct {
	questRepo     *imongo.Repository[model.Quest]
	userQuestRepo *imongo.Repository[model.UserQuest]
	badgeService  *BadgeService
	userService   *UserService
}

func NewQuestService(
	questRepo *imongo.Repository[model.Quest],
	userQuestRepo *imongo.Repository[model.UserQuest],
	badgeService *BadgeService,
	userService *UserService,
) *QuestService {
	return &QuestService{
		questRepo:     questRepo,
		userQuestRepo: userQuestRepo,
		badgeService:  badgeService,
		userService:   userService,
	}
}

func (s *QuestService) EnsureIndexes(questColl, userQuestColl *mongo.Collection) error {
	ctx := context.Background()

	_, err := questColl.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "name", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return fmt.Errorf("failed to create quest name index: %w", err)
	}

	_, err = questColl.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "mood_tags", Value: 1}},
	})
	if err != nil {
		return fmt.Errorf("failed to create quest mood_tags index: %w", err)
	}

	_, err = questColl.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "is_active", Value: 1}},
	})
	if err != nil {
		return fmt.Errorf("failed to create quest is_active index: %w", err)
	}

	_, err = userQuestColl.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "status", Value: 1}},
	})
	if err != nil {
		return fmt.Errorf("failed to create user_quest compound index: %w", err)
	}

	_, err = userQuestColl.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "user_id", Value: 1}, {Key: "quest_id", Value: 1}, {Key: "status", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return fmt.Errorf("failed to create user_quest unique index: %w", err)
	}

	return nil
}

// Create creates a new quest in the library (admin operation).
func (s *QuestService) Create(ctx context.Context, quest *model.Quest) error {
	now := time.Now()
	quest.CreatedAt = now
	quest.UpdatedAt = now
	quest.IsActive = true
	_, err := s.questRepo.InsertOne(ctx, quest)
	return err
}

func (s *QuestService) GetByID(ctx context.Context, id bson.ObjectID) (*model.Quest, error) {
	return s.questRepo.FindByID(ctx, id)
}

func (s *QuestService) List(ctx context.Context) ([]*model.Quest, error) {
	return s.questRepo.Find(ctx, bson.M{"is_active": true})
}

func (s *QuestService) ListAll(ctx context.Context) ([]*model.Quest, error) {
	return s.questRepo.Find(ctx, bson.M{})
}

func (s *QuestService) Update(ctx context.Context, id bson.ObjectID, update bson.M) (*mongo.UpdateResult, error) {
	update["updated_at"] = time.Now()
	return s.questRepo.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": update})
}

func (s *QuestService) Delete(ctx context.Context, id bson.ObjectID) (*mongo.DeleteResult, error) {
	return s.questRepo.DeleteOne(ctx, bson.M{"_id": id})
}

// ListByMood returns active quests matching a given mood tag (Adaptive Quests).
func (s *QuestService) ListByMood(ctx context.Context, mood string) ([]*model.Quest, error) {
	filter := bson.M{
		"is_active": true,
		"mood_tags": mood,
	}
	return s.questRepo.Find(ctx, filter)
}

// StartQuest assigns a quest to a user, setting it to active status.
func (s *QuestService) StartQuest(ctx context.Context, userID, questID bson.ObjectID) (*model.UserQuest, error) {
	quest, err := s.questRepo.FindByID(ctx, questID)
	if err != nil {
		return nil, err
	}
	if quest == nil {
		return nil, ErrNotFound
	}
	if !quest.IsActive {
		return nil, ErrNotFound
	}

	// Check if user already has this quest active
	existing, err := s.userQuestRepo.FindOne(ctx, bson.M{
		"user_id":  userID,
		"quest_id": questID,
		"status":   model.QuestStatusActive,
	})
	if err == nil && existing != nil {
		return nil, ErrQuestAlreadyActive
	}

	uq := &model.UserQuest{
		UserID:    userID,
		QuestID:   questID,
		Status:    model.QuestStatusActive,
		StartedAt: time.Now(),
		EXPEarned: 0,
	}
	_, err = s.userQuestRepo.InsertOne(ctx, uq)
	if err != nil {
		return nil, err
	}
	return uq, nil
}

// CompleteQuest marks a quest as completed, awards EXP, and optionally upgrades a badge.
func (s *QuestService) CompleteQuest(ctx context.Context, userID, userQuestID bson.ObjectID) (*model.UserQuest, error) {
	uq, err := s.userQuestRepo.FindByID(ctx, userQuestID)
	if err != nil {
		return nil, err
	}
	if uq == nil {
		return nil, ErrNotFound
	}
	if uq.UserID != userID {
		return nil, ErrNotFound
	}
	if uq.Status != model.QuestStatusActive {
		return nil, ErrQuestNotActive
	}

	quest, err := s.questRepo.FindByID(ctx, uq.QuestID)
	if err != nil {
		return nil, err
	}
	if quest == nil {
		return nil, ErrNotFound
	}

	now := time.Now()
	update := bson.M{
		"$set": bson.M{
			"status":       model.QuestStatusCompleted,
			"completed_at": now,
			"exp_earned":   quest.EXPReward,
		},
	}
	_, err = s.userQuestRepo.UpdateOne(ctx, bson.M{"_id": userQuestID}, update)
	if err != nil {
		return nil, err
	}

	// Award EXP to user (Reward Loop)
	_, _, err = s.userService.AddEXP(ctx, userID, quest.EXPReward)
	if err != nil {
		return nil, fmt.Errorf("failed to add EXP: %w", err)
	}

	// If quest has an associated badge, try to award/upgrade it
	if !quest.BadgeID.IsZero() && quest.BadgeLevel > 0 {
		_, _ = s.badgeService.AwardBadge(ctx, userID, quest.BadgeID, quest.BadgeLevel)
	}

	uq.Status = model.QuestStatusCompleted
	uq.CompletedAt = &now
	uq.EXPEarned = quest.EXPReward
	return uq, nil
}

// AbandonQuest marks a quest as abandoned. No EXP is awarded.
func (s *QuestService) AbandonQuest(ctx context.Context, userID, userQuestID bson.ObjectID) error {
	uq, err := s.userQuestRepo.FindByID(ctx, userQuestID)
	if err != nil {
		return err
	}
	if uq == nil || uq.UserID != userID {
		return ErrNotFound
	}
	if uq.Status != model.QuestStatusActive {
		return ErrQuestNotActive
	}

	_, err = s.userQuestRepo.UpdateOne(ctx, bson.M{"_id": userQuestID}, bson.M{
		"$set": bson.M{"status": model.QuestStatusAbandoned},
	})
	return err
}

// GetUserQuests returns all quests for a user, optionally filtered by status.
func (s *QuestService) GetUserQuests(ctx context.Context, userID bson.ObjectID, status string) ([]*model.UserQuest, error) {
	filter := bson.M{"user_id": userID}
	if status != "" {
		filter["status"] = status
	}
	return s.userQuestRepo.Find(ctx, filter)
}

// GetUserQuestDetails returns user quests enriched with quest information.
func (s *QuestService) GetUserQuestDetails(ctx context.Context, userID bson.ObjectID, status string) ([]model.UserQuestDetail, error) {
	userQuests, err := s.GetUserQuests(ctx, userID, status)
	if err != nil {
		return nil, err
	}
	if len(userQuests) == 0 {
		return []model.UserQuestDetail{}, nil
	}

	questIDs := make([]bson.ObjectID, len(userQuests))
	for i, uq := range userQuests {
		questIDs[i] = uq.QuestID
	}
	quests, err := s.questRepo.Find(ctx, bson.M{"_id": bson.M{"$in": questIDs}})
	if err != nil {
		return nil, err
	}
	questMap := make(map[bson.ObjectID]*model.Quest)
	for _, q := range quests {
		questMap[q.ID] = q
	}

	var result []model.UserQuestDetail
	for _, uq := range userQuests {
		quest := questMap[uq.QuestID]
		if quest == nil {
			continue
		}
		result = append(result, model.UserQuestDetail{
			UserQuestID: uq.ID.Hex(),
			Quest:       quest,
			Status:      uq.Status,
			StartedAt:   uq.StartedAt,
			CompletedAt: uq.CompletedAt,
			EXPEarned:   uq.EXPEarned,
		})
	}
	return result, nil
}
