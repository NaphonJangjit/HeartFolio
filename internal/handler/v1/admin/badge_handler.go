package admin

import (
	"errors"
	"net/http"

	"github.com/NaphonJangjit/HeartFolio/internal/model"
	"github.com/NaphonJangjit/HeartFolio/internal/service"
	"github.com/NaphonJangjit/HeartFolio/internal/webhttp"
	"go.mongodb.org/mongo-driver/v2/bson"
	mgDriver "go.mongodb.org/mongo-driver/v2/mongo"
)

type BadgeHandler struct {
	svc *service.BadgeService
}

func NewBadgeHandler(svc *service.BadgeService) *BadgeHandler {
	return &BadgeHandler{svc: svc}
}

func badgeResource(b *model.Badge) webhttp.Resource {
	return webhttp.Resource{
		Type: "badges",
		ID:   b.ID.Hex(),
		Attributes: map[string]interface{}{
			"name":        b.Name,
			"description": b.Description,
			"category":    b.Category,
			"max_level":   b.MaxLevel,
			"criteria":    b.Criteria,
			"icon":        b.Icon,
			"created_at":  b.CreatedAt,
			"updated_at":  b.UpdatedAt,
		},
	}
}

func (h *BadgeHandler) CreateBadge(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Category    string `json:"category"`
		MaxLevel    int32  `json:"max_level"`
		Criteria    string `json:"criteria"`
		Icon        string `json:"icon"`
	}
	if err := webhttp.ReadJSON(r, &req); err != nil {
		webhttp.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		webhttp.RespondError(w, http.StatusBadRequest, "name is required")
		return
	}

	badge := &model.Badge{
		Name:        req.Name,
		Description: req.Description,
		Category:    req.Category,
		MaxLevel:    req.MaxLevel,
		Criteria:    req.Criteria,
		Icon:        req.Icon,
	}
	if err := h.svc.Create(r.Context(), badge); err != nil {
		webhttp.RespondError(w, http.StatusInternalServerError, "failed to create badge")
		return
	}
	webhttp.RespondOne(w, http.StatusCreated, badgeResource(badge))
}

func (h *BadgeHandler) ListBadges(w http.ResponseWriter, r *http.Request) {
	badges, err := h.svc.List(r.Context())
	if err != nil {
		webhttp.RespondError(w, http.StatusInternalServerError, "failed to list badges")
		return
	}
	resources := make([]webhttp.Resource, len(badges))
	for i, b := range badges {
		resources[i] = badgeResource(b)
	}
	webhttp.RespondMany(w, http.StatusOK, resources)
}

func (h *BadgeHandler) GetBadge(w http.ResponseWriter, r *http.Request) {
	id, err := bson.ObjectIDFromHex(webhttp.Param(r, "id"))
	if err != nil {
		webhttp.RespondError(w, http.StatusBadRequest, "invalid badge id")
		return
	}
	badge, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		webhttp.RespondError(w, http.StatusInternalServerError, "failed to fetch badge")
		return
	}
	if badge == nil {
		webhttp.RespondError(w, http.StatusNotFound, "badge not found")
		return
	}
	webhttp.RespondOne(w, http.StatusOK, badgeResource(badge))
}

func (h *BadgeHandler) UpdateBadge(w http.ResponseWriter, r *http.Request) {
	id, err := bson.ObjectIDFromHex(webhttp.Param(r, "id"))
	if err != nil {
		webhttp.RespondError(w, http.StatusBadRequest, "invalid badge id")
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
		webhttp.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	update := bson.M{
		"name":        req.Name,
		"description": req.Description,
		"category":    req.Category,
		"max_level":   req.MaxLevel,
		"criteria":    req.Criteria,
		"icon":        req.Icon,
	}
	result, err := h.svc.Update(r.Context(), id, update)
	if err != nil {
		webhttp.RespondError(w, http.StatusInternalServerError, "failed to update badge")
		return
	}
	if result.MatchedCount == 0 {
		webhttp.RespondError(w, http.StatusNotFound, "badge not found")
		return
	}
	webhttp.RespondOne(w, http.StatusOK, webhttp.Resource{
		Type:       "badges",
		ID:         id.Hex(),
		Attributes: map[string]string{"message": "badge updated"},
	})
}

func (h *BadgeHandler) DeleteBadge(w http.ResponseWriter, r *http.Request) {
	id, err := bson.ObjectIDFromHex(webhttp.Param(r, "id"))
	if err != nil {
		webhttp.RespondError(w, http.StatusBadRequest, "invalid badge id")
		return
	}

	result, err := h.svc.Delete(r.Context(), id)
	if err != nil {
		webhttp.RespondError(w, http.StatusInternalServerError, "failed to delete badge")
		return
	}
	if result.DeletedCount == 0 {
		webhttp.RespondError(w, http.StatusNotFound, "badge not found")
		return
	}
	webhttp.RespondOne(w, http.StatusOK, webhttp.Resource{
		Type:       "badges",
		ID:         id.Hex(),
		Attributes: map[string]string{"message": "badge deleted"},
	})
}

func (h *BadgeHandler) AwardBadgeToUser(w http.ResponseWriter, r *http.Request) {
	badgeID, err := bson.ObjectIDFromHex(webhttp.Param(r, "badgeId"))
	if err != nil {
		webhttp.RespondError(w, http.StatusBadRequest, "invalid badge id")
		return
	}

	var req struct {
		UserID string `json:"user_id"`
		Level  int32  `json:"level"`
	}
	if err := webhttp.ReadJSON(r, &req); err != nil {
		webhttp.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	userID, err := bson.ObjectIDFromHex(req.UserID)
	if err != nil {
		webhttp.RespondError(w, http.StatusBadRequest, "invalid user id")
		return
	}
	if req.Level <= 0 {
		req.Level = 1
	}

	ok, err := h.svc.AwardBadge(r.Context(), userID, badgeID, req.Level)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) || errors.Is(err, mgDriver.ErrNoDocuments) {
			webhttp.RespondError(w, http.StatusNotFound, "badge or user not found")
			return
		}
		webhttp.RespondError(w, http.StatusInternalServerError, "failed to award badge")
		return
	}

	msg := "badge awarded"
	if !ok {
		msg = "badge already owned at same or higher level"
	}
	webhttp.RespondOne(w, http.StatusOK, webhttp.Resource{
		Type:       "user-badges",
		Attributes: map[string]string{"message": msg},
	})
}

func (h *BadgeHandler) Router() *webhttp.Router {
	r := webhttp.New()
	r.Get("/", h.ListBadges)
	r.Post("/", h.CreateBadge)
	r.Get("/{id}", h.GetBadge)
	r.Put("/{id}", h.UpdateBadge)
	r.Delete("/{id}", h.DeleteBadge)
	r.Post("/{badgeId}/award", h.AwardBadgeToUser)
	return r
}
