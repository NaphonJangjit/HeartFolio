package v1

import (
	"errors"
	"net/http"

	"github.com/NaphonJangjit/HeartFolio/internal/middleware"
	"github.com/NaphonJangjit/HeartFolio/internal/model"
	"github.com/NaphonJangjit/HeartFolio/internal/service"
	"github.com/NaphonJangjit/HeartFolio/internal/webhttp"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type QuestHandler struct {
	svc       *service.QuestService
	jwtSecret []byte
}

func NewQuestHandler(svc *service.QuestService, jwtSecret []byte) *QuestHandler {
	return &QuestHandler{svc: svc, jwtSecret: jwtSecret}
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

func userQuestResource(uq *model.UserQuest) webhttp.Resource {
	return webhttp.Resource{
		Type: "user-quests",
		ID:   uq.ID.Hex(),
		Attributes: map[string]interface{}{
			"status":       uq.Status,
			"started_at":   uq.StartedAt,
			"completed_at": uq.CompletedAt,
			"exp_earned":   uq.EXPEarned,
		},
	}
}

func userQuestDetailResource(d model.UserQuestDetail) webhttp.Resource {
	return webhttp.Resource{
		Type: "user-quests",
		ID:   d.UserQuestID,
		Attributes: map[string]interface{}{
			"status":       d.Status,
			"started_at":   d.StartedAt,
			"completed_at": d.CompletedAt,
			"exp_earned":   d.EXPEarned,
		},
		Relationships: map[string]webhttp.Relationship{
			"quest": {Data: questResource(d.Quest)},
		},
	}
}

func (h *QuestHandler) ListQuests(w http.ResponseWriter, r *http.Request) {
	quests, err := h.svc.List(r.Context())
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

func (h *QuestHandler) ListByMood(w http.ResponseWriter, r *http.Request) {
	mood := webhttp.Param(r, "mood")
	if mood == "" {
		webhttp.RespondError(w, http.StatusBadRequest, "mood parameter required")
		return
	}
	quests, err := h.svc.ListByMood(r.Context(), mood)
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

func (h *QuestHandler) StartQuest(w http.ResponseWriter, r *http.Request) {
	userID, err := bson.ObjectIDFromHex(middleware.UserIDFromContext(r.Context()))
	if err != nil {
		webhttp.RespondError(w, http.StatusBadRequest, "invalid user id")
		return
	}
	questID, err := bson.ObjectIDFromHex(webhttp.Param(r, "id"))
	if err != nil {
		webhttp.RespondError(w, http.StatusBadRequest, "invalid quest id")
		return
	}

	uq, err := h.svc.StartQuest(r.Context(), userID, questID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrNotFound):
			webhttp.RespondError(w, http.StatusNotFound, "quest not found")
		case errors.Is(err, service.ErrQuestAlreadyActive):
			webhttp.RespondError(w, http.StatusConflict, "quest already active")
		default:
			webhttp.RespondError(w, http.StatusInternalServerError, "failed to start quest")
		}
		return
	}
	webhttp.RespondOne(w, http.StatusCreated, userQuestResource(uq))
}

func (h *QuestHandler) CompleteQuest(w http.ResponseWriter, r *http.Request) {
	userID, err := bson.ObjectIDFromHex(middleware.UserIDFromContext(r.Context()))
	if err != nil {
		webhttp.RespondError(w, http.StatusBadRequest, "invalid user id")
		return
	}
	uqID, err := bson.ObjectIDFromHex(webhttp.Param(r, "userQuestID"))
	if err != nil {
		webhttp.RespondError(w, http.StatusBadRequest, "invalid user quest id")
		return
	}

	uq, err := h.svc.CompleteQuest(r.Context(), userID, uqID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrNotFound):
			webhttp.RespondError(w, http.StatusNotFound, "quest not found")
		case errors.Is(err, service.ErrQuestNotActive):
			webhttp.RespondError(w, http.StatusBadRequest, "quest is not active")
		default:
			webhttp.RespondError(w, http.StatusInternalServerError, "failed to complete quest")
		}
		return
	}
	webhttp.RespondOne(w, http.StatusOK, userQuestResource(uq))
}

func (h *QuestHandler) AbandonQuest(w http.ResponseWriter, r *http.Request) {
	userID, err := bson.ObjectIDFromHex(middleware.UserIDFromContext(r.Context()))
	if err != nil {
		webhttp.RespondError(w, http.StatusBadRequest, "invalid user id")
		return
	}
	uqID, err := bson.ObjectIDFromHex(webhttp.Param(r, "userQuestID"))
	if err != nil {
		webhttp.RespondError(w, http.StatusBadRequest, "invalid user quest id")
		return
	}

	err = h.svc.AbandonQuest(r.Context(), userID, uqID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrNotFound):
			webhttp.RespondError(w, http.StatusNotFound, "quest not found")
		case errors.Is(err, service.ErrQuestNotActive):
			webhttp.RespondError(w, http.StatusBadRequest, "quest is not active")
		default:
			webhttp.RespondError(w, http.StatusInternalServerError, "failed to abandon quest")
		}
		return
	}
	webhttp.RespondOne(w, http.StatusOK, webhttp.Resource{
		Type:       "user-quests",
		ID:         uqID.Hex(),
		Attributes: map[string]string{"status": "abandoned"},
	})
}

func (h *QuestHandler) GetMyQuests(w http.ResponseWriter, r *http.Request) {
	userID, err := bson.ObjectIDFromHex(middleware.UserIDFromContext(r.Context()))
	if err != nil {
		webhttp.RespondError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	status := r.URL.Query().Get("status")
	details, err := h.svc.GetUserQuestDetails(r.Context(), userID, status)
	if err != nil {
		webhttp.RespondError(w, http.StatusInternalServerError, "failed to fetch quests")
		return
	}
	resources := make([]webhttp.Resource, len(details))
	for i, d := range details {
		resources[i] = userQuestDetailResource(d)
	}
	webhttp.RespondManyPaginated(w, http.StatusOK, resources, webhttp.ParsePage(r))
}

func (h *QuestHandler) Router() *webhttp.Router {
	r := webhttp.New()

	r.Get("/", h.ListQuests)
	r.Get("/{id}", h.GetQuest)
	r.Get("/mood/{mood}", h.ListByMood)

	auth := r.Group("")
	auth.Use(middleware.Auth(h.jwtSecret))
	auth.Post("/{id}/start", h.StartQuest)
	auth.Post("/{userQuestID}/complete", h.CompleteQuest)
	auth.Post("/{userQuestID}/abandon", h.AbandonQuest)

	me := r.Group("/me")
	me.Use(middleware.Auth(h.jwtSecret))
	me.Get("/quests", h.GetMyQuests)

	return r
}
