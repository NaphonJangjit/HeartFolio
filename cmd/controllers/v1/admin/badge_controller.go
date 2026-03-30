package admin

import (
	"errors"
	"net/http"
	"time"

	"github.com/NaphonJangjit/HeartFolio/cmd/controllers/v1"
	imongo "github.com/NaphonJangjit/HeartFolio/internal/db/mongo"
	"github.com/NaphonJangjit/HeartFolio/internal/webhttp"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type AdminBadgeController struct {
	badgeRepo     *imongo.Repository[v1.Badge]
	userBadgeRepo *imongo.Repository[v1.UserBadge]
	userRepo      *imongo.Repository[v1.User]
}

func NewAdminBadgeController(
	badgeRepo *imongo.Repository[v1.Badge],
	userBadgeRepo *imongo.Repository[v1.UserBadge],
	userRepo *imongo.Repository[v1.User],
) *AdminBadgeController {
	return &AdminBadgeController{
		badgeRepo:     badgeRepo,
		userBadgeRepo: userBadgeRepo,
		userRepo:      userRepo,
	}
}

func (c *AdminBadgeController) CreateBadge(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Category    string `json:"category"`
		MaxLevel    int32  `json:"max_level"`
		Criteria    string `json:"criteria"`
		Icon        string `json:"icon"`
	}
	if err := webhttp.ReadJSON(r, &req); err != nil {
		webhttp.ErrorJSON(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		webhttp.ErrorJSON(w, http.StatusBadRequest, "name is required")
		return
	}
	now := time.Now()
	badge := &v1.Badge{
		Name:        req.Name,
		Description: req.Description,
		Category:    req.Category,
		MaxLevel:    req.MaxLevel,
		Criteria:    req.Criteria,
		Icon:        req.Icon,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	_, err := c.badgeRepo.InsertOne(r.Context(), badge)
	if err != nil {
		webhttp.ErrorJSON(w, http.StatusInternalServerError, "failed to create badge")
		return
	}
	webhttp.JSON(w, http.StatusCreated, badge)
}

func (c *AdminBadgeController) ListBadges(w http.ResponseWriter, r *http.Request) {
	badges, err := c.badgeRepo.Find(r.Context(), bson.M{})
	if err != nil {
		webhttp.ErrorJSON(w, http.StatusInternalServerError, "failed to list badges")
		return
	}
	webhttp.JSON(w, http.StatusOK, badges)
}

func (c *AdminBadgeController) GetBadge(w http.ResponseWriter, r *http.Request) {
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

func (c *AdminBadgeController) UpdateBadge(w http.ResponseWriter, r *http.Request) {
	idStr := webhttp.Param(r, "id")
	id, err := bson.ObjectIDFromHex(idStr)
	if err != nil {
		webhttp.ErrorJSON(w, http.StatusBadRequest, "invalid badge id")
		return
	}
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Category    string `json:"category"`
		MaxLevel    int32  `json:"max_level"`
		Criteria    string `json:"criteria"`
		Icon        string `json:"icon"`
	}
	if err := webhttp.ReadJSON(r, &req); err != nil {
		webhttp.ErrorJSON(w, http.StatusBadRequest, "invalid request body")
		return
	}
	update := bson.M{
		"$set": bson.M{
			"name":        req.Name,
			"description": req.Description,
			"category":    req.Category,
			"max_level":   req.MaxLevel,
			"criteria":    req.Criteria,
			"icon":        req.Icon,
			"updated_at":  time.Now(),
		},
	}
	result, err := c.badgeRepo.UpdateOne(r.Context(), bson.M{"_id": id}, update)
	if err != nil {
		webhttp.ErrorJSON(w, http.StatusInternalServerError, "failed to update badge")
		return
	}
	if result.MatchedCount == 0 {
		webhttp.ErrorJSON(w, http.StatusNotFound, "badge not found")
		return
	}
	webhttp.JSON(w, http.StatusOK, map[string]string{"message": "badge updated"})
}

func (c *AdminBadgeController) DeleteBadge(w http.ResponseWriter, r *http.Request) {
	idStr := webhttp.Param(r, "id")
	id, err := bson.ObjectIDFromHex(idStr)
	if err != nil {
		webhttp.ErrorJSON(w, http.StatusBadRequest, "invalid badge id")
		return
	}
	result, err := c.badgeRepo.DeleteOne(r.Context(), bson.M{"_id": id})
	if err != nil {
		webhttp.ErrorJSON(w, http.StatusInternalServerError, "failed to delete badge")
		return
	}
	if result.DeletedCount == 0 {
		webhttp.ErrorJSON(w, http.StatusNotFound, "badge not found")
		return
	}
	_, _ = c.userBadgeRepo.DeleteOne(r.Context(), bson.M{"badge_id": id})
	webhttp.JSON(w, http.StatusOK, map[string]string{"message": "badge deleted"})
}

func (c *AdminBadgeController) AwardBadgeToUser(w http.ResponseWriter, r *http.Request) {
	badgeIDStr := webhttp.Param(r, "badgeId")
	badgeID, err := bson.ObjectIDFromHex(badgeIDStr)
	if err != nil {
		webhttp.ErrorJSON(w, http.StatusBadRequest, "invalid badge id")
		return
	}
	var req struct {
		UserID string `json:"user_id"`
		Level  int32  `json:"level"`
	}
	if err := webhttp.ReadJSON(r, &req); err != nil {
		webhttp.ErrorJSON(w, http.StatusBadRequest, "invalid request body")
		return
	}
	userID, err := bson.ObjectIDFromHex(req.UserID)
	if err != nil {
		webhttp.ErrorJSON(w, http.StatusBadRequest, "invalid user id")
		return
	}
	if req.Level <= 0 {
		req.Level = 1
	}
	_, err = c.userRepo.FindByID(r.Context(), userID)
	if err != nil {
		webhttp.ErrorJSON(w, http.StatusBadRequest, "user not found")
		return
	}
	badge, err := c.badgeRepo.FindByID(r.Context(), badgeID)
	if err != nil || badge == nil {
		webhttp.ErrorJSON(w, http.StatusNotFound, "badge not found")
		return
	}
	if req.Level > badge.MaxLevel {
		req.Level = badge.MaxLevel
	}
	filter := bson.M{"user_id": userID, "badge_id": badgeID}
	existing, err := c.userBadgeRepo.FindOne(r.Context(), filter)
	if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
		webhttp.ErrorJSON(w, http.StatusInternalServerError, "database error")
		return
	}
	now := time.Now()
	if existing == nil {
		ub := &v1.UserBadge{
			UserID:      userID,
			BadgeID:     badgeID,
			Level:       req.Level,
			EarnedAt:    now,
			LastUpdated: now,
		}
		_, err = c.userBadgeRepo.InsertOne(r.Context(), ub)
	} else if req.Level > existing.Level {
		update := bson.M{
			"$set": bson.M{
				"level":        req.Level,
				"last_updated": now,
			},
		}
		_, err = c.userBadgeRepo.UpdateOne(r.Context(), filter, update)
	} else {
		webhttp.JSON(w, http.StatusOK, map[string]string{"message": "badge already owned at same or higher level"})
		return
	}
	if err != nil {
		webhttp.ErrorJSON(w, http.StatusInternalServerError, "failed to award badge")
		return
	}
	webhttp.JSON(w, http.StatusOK, map[string]string{"message": "badge awarded"})
}

func (c *AdminBadgeController) Router() *webhttp.Router {
	r := webhttp.New()
	r.Get("/", c.ListBadges)
	r.Post("/", c.CreateBadge)
	r.Get("/{id}", c.GetBadge)
	r.Put("/{id}", c.UpdateBadge)
	r.Delete("/{id}", c.DeleteBadge)
	r.Post("/{badgeId}/award", c.AwardBadgeToUser)
	return r
}