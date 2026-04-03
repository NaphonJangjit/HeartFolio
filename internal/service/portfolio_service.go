package service

import (
	"context"

	"github.com/NaphonJangjit/HeartFolio/internal/model"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type PortfolioService struct {
	userSvc  *UserService
	badgeSvc *BadgeService
	questSvc *QuestService
	moodSvc  *MoodService
}

func NewPortfolioService(userSvc *UserService, badgeSvc *BadgeService, questSvc *QuestService, moodSvc *MoodService) *PortfolioService {
	return &PortfolioService{
		userSvc:  userSvc,
		badgeSvc: badgeSvc,
		questSvc: questSvc,
		moodSvc:  moodSvc,
	}
}

// GetPortfolio builds the full emotional portfolio for a user.
func (s *PortfolioService) GetPortfolio(ctx context.Context, userID bson.ObjectID) (*model.Portfolio, error) {
	user, err := s.userSvc.GetByID(ctx, userID.Hex())
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrNotFound
	}

	skillStats, err := s.badgeSvc.GetDetailedSkillStats(ctx, userID)
	if err != nil {
		return nil, err
	}

	userBadges, err := s.badgeSvc.GetUserBadges(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Quest summary
	allQuests, err := s.questSvc.GetUserQuests(ctx, userID, "")
	if err != nil {
		return nil, err
	}
	var summary model.QuestSummary
	for _, uq := range allQuests {
		switch uq.Status {
		case model.QuestStatusActive:
			summary.TotalStarted++
		case model.QuestStatusCompleted:
			summary.TotalStarted++
			summary.TotalCompleted++
			summary.TotalEXPEarned += uq.EXPEarned
		case model.QuestStatusAbandoned:
			summary.TotalStarted++
			summary.TotalAbandoned++
		}
	}

	// Recent moods (last 10)
	recentMoods, err := s.moodSvc.GetRecent(ctx, userID, 10)
	if err != nil {
		recentMoods = []*model.MoodEntry{}
	}

	return &model.Portfolio{
		UserID:       userID.Hex(),
		Email:        user.Email,
		Level:        user.Level,
		EXP:          user.EXP,
		Tier:         user.Tier(),
		TotalBadges:  int32(len(userBadges)),
		SkillStats:   skillStats,
		QuestSummary: summary,
		RecentMoods:  recentMoods,
	}, nil
}
