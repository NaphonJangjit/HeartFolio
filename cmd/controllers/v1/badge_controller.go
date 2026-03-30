package v1

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	imongo "github.com/NaphonJangjit/HeartFolio/internal/db/mongo"
	"github.com/NaphonJangjit/HeartFolio/internal/webhttp"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type Badge struct {
	ID          bson.ObjectID `bson:"_id,omitempty" json:"id"`
	Name        string        `bson:"name" json:"name"`
	Description string        `bson:"description" json:"description"`
	Category    string        `bson:"category" json:"category"`
	MaxLevel    int32         `bson:"max_level" json:"max_level"`
	Criteria    string        `bson:"criteria" json:"criteria"`
	Icon        string        `bson:"icon,omitempty" json:"icon"`
	CreatedAt   time.Time     `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time     `bson:"updated_at" json:"updated_at"`
}

type UserBadge struct {
	ID          bson.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID      bson.ObjectID `bson:"user_id" json:"user_id"`
	BadgeID     bson.ObjectID `bson:"badge_id" json:"badge_id"`
	Level       int32         `bson:"level" json:"level"`
	EarnedAt    time.Time     `bson:"earned_at" json:"earned_at"`
	LastUpdated time.Time     `bson:"last_updated" json:"last_updated"`
}

type BadgeController struct {
	badgeRepo     *imongo.Repository[Badge]
	userBadgeRepo *imongo.Repository[UserBadge]
	userRepo      *imongo.Repository[User]
}

func NewBadgeController(
	badgeRepo *imongo.Repository[Badge],
	userBadgeRepo *imongo.Repository[UserBadge],
	userRepo *imongo.Repository[User],
) *BadgeController {
	return &BadgeController{
		badgeRepo:     badgeRepo,
		userBadgeRepo: userBadgeRepo,
		userRepo:      userRepo,
	}
}

func EnsureBadgeIndexes(badgeColl *mongo.Collection, userBadgeColl *mongo.Collection) error {
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

func (c *BadgeController) AwardBadge(ctx context.Context, userID, badgeID bson.ObjectID, level int32) (bool, error) {
	_, err := c.userRepo.FindByID(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("user not found: %w", err)
	}
	badge, err := c.badgeRepo.FindByID(ctx, badgeID)
	if err != nil {
		return false, fmt.Errorf("badge not found: %w", err)
	}
	if badge == nil {
		return false, errors.New("badge not found")
	}
	if level > badge.MaxLevel {
		level = badge.MaxLevel
	}

	filter := bson.M{"user_id": userID, "badge_id": badgeID}
	existing, err := c.userBadgeRepo.FindOne(ctx, filter)
	if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
		return false, err
	}
	now := time.Now()
	if existing == nil {
		ub := &UserBadge{
			UserID:      userID,
			BadgeID:     badgeID,
			Level:       level,
			EarnedAt:    now,
			LastUpdated: now,
		}
		_, err := c.userBadgeRepo.InsertOne(ctx, ub)
		if err != nil {
			return false, err
		}
		return true, nil
	}
	if level > existing.Level {
		update := bson.M{
			"$set": bson.M{
				"level":        level,
				"last_updated": now,
			},
		}
		_, err := c.userBadgeRepo.UpdateOne(ctx, filter, update)
		if err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

func (c *BadgeController) GetUserBadges(ctx context.Context, userID bson.ObjectID) ([]map[string]interface{}, error) {
	filter := bson.M{"user_id": userID}
	userBadges, err := c.userBadgeRepo.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	if len(userBadges) == 0 {
		return []map[string]interface{}{}, nil
	}

	badgeIDs := make([]bson.ObjectID, len(userBadges))
	for i, ub := range userBadges {
		badgeIDs[i] = ub.BadgeID
	}
	badgeFilter := bson.M{"_id": bson.M{"$in": badgeIDs}}
	badges, err := c.badgeRepo.Find(ctx, badgeFilter)
	if err != nil {
		return nil, err
	}
	badgeMap := make(map[bson.ObjectID]*Badge)
	for _, b := range badges {
		badgeMap[b.ID] = b
	}

	var result []map[string]interface{}
	for _, ub := range userBadges {
		badge := badgeMap[ub.BadgeID]
		if badge == nil {
			continue
		}
		item := map[string]interface{}{
			"badge": map[string]interface{}{
				"id":          badge.ID.Hex(),
				"name":        badge.Name,
				"description": badge.Description,
				"category":    badge.Category,
				"max_level":   badge.MaxLevel,
				"icon":        badge.Icon,
			},
			"level":        ub.Level,
			"earned_at":    ub.EarnedAt,
			"last_updated": ub.LastUpdated,
		}
		result = append(result, item)
	}
	return result, nil
}

func (c *BadgeController) GetSkillStats(ctx context.Context, userID bson.ObjectID) (map[string]int32, error) {
	filter := bson.M{"user_id": userID}
	userBadges, err := c.userBadgeRepo.Find(ctx, filter)
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
	badgeFilter := bson.M{"_id": bson.M{"$in": badgeIDs}}
	badges, err := c.badgeRepo.Find(ctx, badgeFilter)
	if err != nil {
		return nil, err
	}
	badgeMap := make(map[bson.ObjectID]*Badge)
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

func (c *BadgeController) AwardBadgeByTrigger(ctx context.Context, userID bson.ObjectID, badgeName string, level int32) (bool, error) {
	filter := bson.M{"name": badgeName}
	badge, err := c.badgeRepo.FindOne(ctx, filter)
	if err != nil {
		return false, fmt.Errorf("badge not found: %w", err)
	}
	if badge == nil {
		return false, errors.New("badge not found")
	}
	return c.AwardBadge(ctx, userID, badge.ID, level)
}

func (c *BadgeController) ListBadges(w http.ResponseWriter, r *http.Request) {
	badges, err := c.badgeRepo.Find(r.Context(), bson.M{})
	if err != nil {
		webhttp.ErrorJSON(w, http.StatusInternalServerError, "failed to list badges")
		return
	}
	webhttp.JSON(w, http.StatusOK, badges)
}

func (c *BadgeController) GetBadge(w http.ResponseWriter, r *http.Request) {
	idStr := webhttp.Param(r, "id")
	id, err := bson.ObjectIDFromHex(idStr)
	if err != nil {
		webhttp.ErrorJSON(w, http.StatusBadRequest, "invalid badge id")
		return
	}
	badge, err := c.badgeRepo.FindByID(r.Context(), id)
	if err != nil {
		webhttp.ErrorJSON(w, http.StatusInternalServerError, "failed to fetch badge")
		return
	}
	if badge == nil {
		webhttp.ErrorJSON(w, http.StatusNotFound, "badge not found")
		return
	}
	webhttp.JSON(w, http.StatusOK, badge)
}

func (c *BadgeController) AwardBadgeToSelf(w http.ResponseWriter, r *http.Request) {
	userIDHex := r.Context().Value("userID").(string)
	userID, err := bson.ObjectIDFromHex(userIDHex)
	if err != nil {
		webhttp.ErrorJSON(w, http.StatusBadRequest, "invalid user id")
		return
	}
	badgeIDStr := webhttp.Param(r, "id")
	badgeID, err := bson.ObjectIDFromHex(badgeIDStr)
	if err != nil {
		webhttp.ErrorJSON(w, http.StatusBadRequest, "invalid badge id")
		return
	}
	var req struct {
		Level int32 `json:"level"`
	}
	if err := webhttp.ReadJSON(r, &req); err != nil {
		webhttp.ErrorJSON(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Level <= 0 {
		req.Level = 1
	}
	ok, err := c.AwardBadge(r.Context(), userID, badgeID, req.Level)
	if err != nil {
		webhttp.ErrorJSON(w, http.StatusInternalServerError, "failed to award badge")
		return
	}
	if !ok {
		webhttp.JSON(w, http.StatusOK, map[string]string{"message": "badge already owned at same or higher level"})
		return
	}
	webhttp.JSON(w, http.StatusOK, map[string]string{"message": "badge awarded"})
}

func (c *BadgeController) GetUserBadgesByID(w http.ResponseWriter, r *http.Request) {
	userIDStr := webhttp.Param(r, "userID")
	userID, err := bson.ObjectIDFromHex(userIDStr)
	if err != nil {
		webhttp.ErrorJSON(w, http.StatusBadRequest, "invalid user id")
		return
	}
	badges, err := c.GetUserBadges(r.Context(), userID)
	if err != nil {
		webhttp.ErrorJSON(w, http.StatusInternalServerError, "failed to fetch badges")
		return
	}
	webhttp.JSON(w, http.StatusOK, badges)
}

func (c *BadgeController) GetUserSkillStatsByID(w http.ResponseWriter, r *http.Request) {
	userIDStr := webhttp.Param(r, "userID")
	userID, err := bson.ObjectIDFromHex(userIDStr)
	if err != nil {
		webhttp.ErrorJSON(w, http.StatusBadRequest, "invalid user id")
		return
	}
	stats, err := c.GetSkillStats(r.Context(), userID)
	if err != nil {
		webhttp.ErrorJSON(w, http.StatusInternalServerError, "failed to fetch stats")
		return
	}
	webhttp.JSON(w, http.StatusOK, stats)
}

func (c *BadgeController) GetMyBadges(w http.ResponseWriter, r *http.Request) {
	userIDHex := r.Context().Value("userID").(string)
	userID, err := bson.ObjectIDFromHex(userIDHex)
	if err != nil {
		webhttp.ErrorJSON(w, http.StatusBadRequest, "invalid user id")
		return
	}
	badges, err := c.GetUserBadges(r.Context(), userID)
	if err != nil {
		webhttp.ErrorJSON(w, http.StatusInternalServerError, "failed to fetch badges")
		return
	}
	webhttp.JSON(w, http.StatusOK, badges)
}

func (c *BadgeController) GetMySkillStats(w http.ResponseWriter, r *http.Request) {
	userIDHex := r.Context().Value("userID").(string)
	userID, err := bson.ObjectIDFromHex(userIDHex)
	if err != nil {
		webhttp.ErrorJSON(w, http.StatusBadRequest, "invalid user id")
		return
	}
	stats, err := c.GetSkillStats(r.Context(), userID)
	if err != nil {
		webhttp.ErrorJSON(w, http.StatusInternalServerError, "failed to fetch stats")
		return
	}
	webhttp.JSON(w, http.StatusOK, stats)
}

func (c *BadgeController) Router() *webhttp.Router {
	r := webhttp.New()

	r.Get("/", c.ListBadges)
	r.Get("/{id}", c.GetBadge)

	authGroup := r.Group("")
	authGroup.Use(AuthMiddleware)
	authGroup.Post("/{id}/award", c.AwardBadgeToSelf)

	r.Get("/users/{userID}/badges", c.GetUserBadgesByID)
	r.Get("/users/{userID}/skill-stats", c.GetUserSkillStatsByID)

	me := r.Group("/me")
	me.Use(AuthMiddleware)
	me.Get("/badges", c.GetMyBadges)
	me.Get("/skill-stats", c.GetMySkillStats)

	return r
}
