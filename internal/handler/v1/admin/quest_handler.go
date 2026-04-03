package admin

import (
	"net/http"

	"github.com/NaphonJangjit/HeartFolio/internal/model"
	"github.com/NaphonJangjit/HeartFolio/internal/service"
	"github.com/NaphonJangjit/HeartFolio/internal/webhttp"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type QuestHandler struct {
	svc *service.QuestService
}

func NewQuestHandler(svc *service.QuestService) *QuestHandler {
	return &QuestHandler{svc: svc}
}

func questResource(q *model.Quest) webhttp.Resource {
	return webhttp.Resource{
		Type: "quests",
		ID:   q.ID.Hex(),
		Attributes: map[string]interface{}{
			"name":               q.Name,
			"description":        q.Description,
			"category":           q.Category,
			"exp_reward":         q.EXPReward,
			"mood_tags":          q.MoodTags,
			"difficulty":         q.Difficulty,
			"estimated_duration": q.EstimatedDuration,
			"badge_level":        q.BadgeLevel,
			"is_active":          q.IsActive,
			"created_at":         q.CreatedAt,
			"updated_at":         q.UpdatedAt,
		},
	}
}

func (h *QuestHandler) CreateQuest(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name              string   `json:"name"`
		Description       string   `json:"description"`
		Category          string   `json:"category"`
		EXPReward         int64    `json:"exp_reward"`
		MoodTags          []string `json:"mood_tags"`
		Difficulty        string   `json:"difficulty"`
		EstimatedDuration int32    `json:"estimated_duration"`
		BadgeID           string   `json:"badge_id"`
		BadgeLevel        int32    `json:"badge_level"`
	}
	if err := webhttp.ReadJSON(r, &req); err != nil {
		webhttp.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		webhttp.RespondError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.EXPReward <= 0 {
		webhttp.RespondError(w, http.StatusBadRequest, "exp_reward must be positive")
		return
	}

	quest := &model.Quest{
		Name:              req.Name,
		Description:       req.Description,
		Category:          req.Category,
		EXPReward:         req.EXPReward,
		MoodTags:          req.MoodTags,
		Difficulty:        req.Difficulty,
		EstimatedDuration: req.EstimatedDuration,
		BadgeLevel:        req.BadgeLevel,
		IsActive:          true,
	}

	if req.BadgeID != "" {
		badgeOID, err := bson.ObjectIDFromHex(req.BadgeID)
		if err != nil {
			webhttp.RespondError(w, http.StatusBadRequest, "invalid badge_id")
			return
		}
		quest.BadgeID = badgeOID
	}

	if err := h.svc.Create(r.Context(), quest); err != nil {
		webhttp.RespondError(w, http.StatusInternalServerError, "failed to create quest")
		return
	}
	webhttp.RespondOne(w, http.StatusCreated, questResource(quest))
}

func (h *QuestHandler) ListQuests(w http.ResponseWriter, r *http.Request) {
	quests, err := h.svc.ListAll(r.Context())
	if err != nil {
		webhttp.RespondError(w, http.StatusInternalServerError, "failed to list quests")
		return
	}
	resources := make([]webhttp.Resource, len(quests))
	for i, q := range quests {
		resources[i] = questResource(q)
	}
	webhttp.RespondManyPaginated(w, http.StatusOK, resources, webhttp.ParsePage(r))
}

func (h *QuestHandler) GetQuest(w http.ResponseWriter, r *http.Request) {
	id, err := bson.ObjectIDFromHex(webhttp.Param(r, "id"))
	if err != nil {
		webhttp.RespondError(w, http.StatusBadRequest, "invalid quest id")
		return
	}
	quest, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		webhttp.RespondError(w, http.StatusInternalServerError, "failed to fetch quest")
		return
	}
	if quest == nil {
		webhttp.RespondError(w, http.StatusNotFound, "quest not found")
		return
	}
	webhttp.RespondOne(w, http.StatusOK, questResource(quest))
}

func (h *QuestHandler) UpdateQuest(w http.ResponseWriter, r *http.Request) {
	id, err := bson.ObjectIDFromHex(webhttp.Param(r, "id"))
	if err != nil {
		webhttp.RespondError(w, http.StatusBadRequest, "invalid quest id")
		return
	}

	var req struct {
		Name              string   `json:"name"`
		Description       string   `json:"description"`
		Category          string   `json:"category"`
		EXPReward         int64    `json:"exp_reward"`
		MoodTags          []string `json:"mood_tags"`
		Difficulty        string   `json:"difficulty"`
		EstimatedDuration int32    `json:"estimated_duration"`
		BadgeID           string   `json:"badge_id"`
		BadgeLevel        int32    `json:"badge_level"`
		IsActive          *bool    `json:"is_active"`
	}
	if err := webhttp.ReadJSON(r, &req); err != nil {
		webhttp.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	update := bson.M{
		"name":               req.Name,
		"description":        req.Description,
		"category":           req.Category,
		"exp_reward":         req.EXPReward,
		"mood_tags":          req.MoodTags,
		"difficulty":         req.Difficulty,
		"estimated_duration": req.EstimatedDuration,
		"badge_level":        req.BadgeLevel,
	}
	if req.IsActive != nil {
		update["is_active"] = *req.IsActive
	}
	if req.BadgeID != "" {
		badgeOID, err := bson.ObjectIDFromHex(req.BadgeID)
		if err != nil {
			webhttp.RespondError(w, http.StatusBadRequest, "invalid badge_id")
			return
		}
		update["badge_id"] = badgeOID
	}

	result, err := h.svc.Update(r.Context(), id, update)
	if err != nil {
		webhttp.RespondError(w, http.StatusInternalServerError, "failed to update quest")
		return
	}
	if result.MatchedCount == 0 {
		webhttp.RespondError(w, http.StatusNotFound, "quest not found")
		return
	}
	webhttp.RespondOne(w, http.StatusOK, webhttp.Resource{
		Type:       "quests",
		ID:         id.Hex(),
		Attributes: map[string]string{"message": "quest updated"},
	})
}

func (h *QuestHandler) DeleteQuest(w http.ResponseWriter, r *http.Request) {
	id, err := bson.ObjectIDFromHex(webhttp.Param(r, "id"))
	if err != nil {
		webhttp.RespondError(w, http.StatusBadRequest, "invalid quest id")
		return
	}

	result, err := h.svc.Delete(r.Context(), id)
	if err != nil {
		webhttp.RespondError(w, http.StatusInternalServerError, "failed to delete quest")
		return
	}
	if result.DeletedCount == 0 {
		webhttp.RespondError(w, http.StatusNotFound, "quest not found")
		return
	}
	webhttp.RespondOne(w, http.StatusOK, webhttp.Resource{
		Type:       "quests",
		ID:         id.Hex(),
		Attributes: map[string]string{"message": "quest deleted"},
	})
}

func (h *QuestHandler) Router() *webhttp.Router {
	r := webhttp.New()
	r.Get("/", h.ListQuests)
	r.Post("/", h.CreateQuest)
	r.Get("/{id}", h.GetQuest)
	r.Put("/{id}", h.UpdateQuest)
	r.Delete("/{id}", h.DeleteQuest)
	return r
}
