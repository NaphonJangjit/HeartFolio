package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	imongo "github.com/NaphonJangjit/HeartFolio/internal/db/mongo"
	"github.com/NaphonJangjit/HeartFolio/internal/model"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type BadgeService struct {
	badgeRepo     *imongo.Repository[model.Badge]
	userBadgeRepo *imongo.Repository[model.UserBadge]
	userRepo      *imongo.Repository[model.User]
}

func NewBadgeService(
	badgeRepo *imongo.Repository[model.Badge],
	userBadgeRepo *imongo.Repository[model.UserBadge],
	userRepo *imongo.Repository[model.User],
) *BadgeService {
	return &BadgeService{
		badgeRepo:     badgeRepo,
		userBadgeRepo: userBadgeRepo,
		userRepo:      userRepo,
	}
}

func (s *BadgeService) EnsureIndexes(badgeColl, userBadgeColl *mongo.Collection) error {
	ctx := context.Background()

	_, err := badgeColl.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "name", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return fmt.Errorf("failed to create badge name index: %w", err)
	}

	_, err = userBadgeColl.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "user_id", Value: 1}, {Key: "badge_id", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return fmt.Errorf("failed to create user_badge unique index: %w", err)
	}

	_, err = userBadgeColl.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "user_id", Value: 1}},
	})
	if err != nil {
		return fmt.Errorf("failed to create user_badge user index: %w", err)
	}

	return nil
}

func (s *BadgeService) Create(ctx context.Context, badge *model.Badge) error {
	now := time.Now()
	badge.CreatedAt = now
	badge.UpdatedAt = now
	_, err := s.badgeRepo.InsertOne(ctx, badge)
	return err
}

func (s *BadgeService) GetByID(ctx context.Context, id bson.ObjectID) (*model.Badge, error) {
	return s.badgeRepo.FindByID(ctx, id)
}

func (s *BadgeService) List(ctx context.Context) ([]*model.Badge, error) {
	return s.badgeRepo.Find(ctx, bson.M{})
}

func (s *BadgeService) Update(ctx context.Context, id bson.ObjectID, update bson.M) (*mongo.UpdateResult, error) {
	update["updated_at"] = time.Now()
	return s.badgeRepo.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": update})
}

func (s *BadgeService) Delete(ctx context.Context, id bson.ObjectID) (*mongo.DeleteResult, error) {
	result, err := s.badgeRepo.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return nil, err
	}
	// Clean up user badges referencing this badge
	_, _ = s.userBadgeRepo.DeleteOne(ctx, bson.M{"badge_id": id})
	return result, nil
}

func (s *BadgeService) AwardBadge(ctx context.Context, userID, badgeID bson.ObjectID, level int32) (bool, error) {
	_, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("user not found: %w", err)
	}
	badge, err := s.badgeRepo.FindByID(ctx, badgeID)
	if err != nil {
		return false, fmt.Errorf("badge not found: %w", err)
	}
	if badge == nil {
		return false, ErrNotFound
	}
	if level > badge.MaxLevel {
		level = badge.MaxLevel
	}

	filter := bson.M{"user_id": userID, "badge_id": badgeID}
	existing, err := s.userBadgeRepo.FindOne(ctx, filter)
	if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
		return false, err
	}
	now := time.Now()
	if existing == nil {
		ub := &model.UserBadge{
			UserID:      userID,
			BadgeID:     badgeID,
			Level:       level,
			EarnedAt:    now,
			LastUpdated: now,
		}
		_, err := s.userBadgeRepo.InsertOne(ctx, ub)
		return err == nil, err
	}
	if level > existing.Level {
		update := bson.M{
			"$set": bson.M{
				"level":        level,
				"last_updated": now,
			},
		}
		_, err := s.userBadgeRepo.UpdateOne(ctx, filter, update)
		return err == nil, err
	}
	return false, nil
}

func (s *BadgeService) AwardBadgeByName(ctx context.Context, userID bson.ObjectID, badgeName string, level int32) (bool, error) {
	badge, err := s.badgeRepo.FindOne(ctx, bson.M{"name": badgeName})
	if err != nil {
		return false, fmt.Errorf("badge not found: %w", err)
	}
	if badge == nil {
		return false, ErrNotFound
	}
	return s.AwardBadge(ctx, userID, badge.ID, level)
}

func (s *BadgeService) GetUserBadges(ctx context.Context, userID bson.ObjectID) ([]model.UserBadgeDetail, error) {
	filter := bson.M{"user_id": userID}
	userBadges, err := s.userBadgeRepo.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	if len(userBadges) == 0 {
		return []model.UserBadgeDetail{}, nil
	}

	badgeIDs := make([]bson.ObjectID, len(userBadges))
	for i, ub := range userBadges {
		badgeIDs[i] = ub.BadgeID
	}
	badges, err := s.badgeRepo.Find(ctx, bson.M{"_id": bson.M{"$in": badgeIDs}})
	if err != nil {
		return nil, err
	}
	badgeMap := make(map[bson.ObjectID]*model.Badge)
	for _, b := range badges {
		badgeMap[b.ID] = b
	}

	var result []model.UserBadgeDetail
	for _, ub := range userBadges {
		badge := badgeMap[ub.BadgeID]
		if badge == nil {
			continue
		}
		result = append(result, model.UserBadgeDetail{
			UserBadgeID: ub.ID.Hex(),
			Badge:       badge,
			Level:       ub.Level,
			EarnedAt:    ub.EarnedAt,
			LastUpdated: ub.LastUpdated,
		})
	}
	return result, nil
}

func (s *BadgeService) GetSkillStats(ctx context.Context, userID bson.ObjectID) (map[string]int32, error) {
	filter := bson.M{"user_id": userID}
	userBadges, err := s.userBadgeRepo.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	if len(userBadges) == 0 {
		return map[string]int32{}, nil
	}

	badgeIDs := make([]bson.ObjectID, len(userBadges))
	for i, ub := range userBadges {
		badgeIDs[i] = ub.BadgeID
	}
	badges, err := s.badgeRepo.Find(ctx, bson.M{"_id": bson.M{"$in": badgeIDs}})
	if err != nil {
		return nil, err
	}
	badgeMap := make(map[bson.ObjectID]*model.Badge)
	for _, b := range badges {
		badgeMap[b.ID] = b
	}

	scores := make(map[string]int32)
	for _, ub := range userBadges {
		badge := badgeMap[ub.BadgeID]
		if badge == nil {
			continue
		}
		scores[badge.Category] += ub.Level
	}
	return scores, nil
}
