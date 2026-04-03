package v1

import (
	"net/http"

	"github.com/NaphonJangjit/HeartFolio/internal/middleware"
	"github.com/NaphonJangjit/HeartFolio/internal/model"
	"github.com/NaphonJangjit/HeartFolio/internal/service"
	"github.com/NaphonJangjit/HeartFolio/internal/webhttp"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func skillStatResource(ss model.SkillStat) webhttp.Resource {
	return webhttp.Resource{
		Type: "skill-stats",
		ID:   ss.Category,
		Attributes: map[string]interface{}{
			"category":    ss.Category,
			"total_level": ss.TotalLevel,
			"badge_count": ss.BadgeCount,
			"badges":      ss.Badges,
		},
	}
}

type BadgeHandler struct {
	svc       *service.BadgeService
	userSvc   *service.UserService
	jwtSecret []byte
}

func NewBadgeHandler(svc *service.BadgeService, userSvc *service.UserService, jwtSecret []byte) *BadgeHandler {
	return &BadgeHandler{svc: svc, userSvc: userSvc, jwtSecret: jwtSecret}
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

func userBadgeResource(ub model.UserBadgeDetail) webhttp.Resource {
	return webhttp.Resource{
		Type: "user-badges",
		ID:   ub.UserBadgeID,
		Attributes: map[string]interface{}{
			"level":        ub.Level,
			"earned_at":    ub.EarnedAt,
			"last_updated": ub.LastUpdated,
		},
		Relationships: map[string]webhttp.Relationship{
			"badge": {Data: badgeResource(ub.Badge)},
		},
	}
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
	webhttp.RespondManyPaginated(w, http.StatusOK, resources, webhttp.ParsePage(r))
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

func (h *BadgeHandler) AwardBadgeToSelf(w http.ResponseWriter, r *http.Request) {
	userID, err := bson.ObjectIDFromHex(middleware.UserIDFromContext(r.Context()))
	if err != nil {
		webhttp.RespondError(w, http.StatusBadRequest, "invalid user id")
		return
	}
	badgeID, err := bson.ObjectIDFromHex(webhttp.Param(r, "id"))
	if err != nil {
		webhttp.RespondError(w, http.StatusBadRequest, "invalid badge id")
		return
	}

	var req struct {
		Level int32 `json:"level"`
	}
	if err := webhttp.ReadJSON(r, &req); err != nil {
		webhttp.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Level <= 0 {
		req.Level = 1
	}

	ok, err := h.svc.AwardBadge(r.Context(), userID, badgeID, req.Level)
	if err != nil {
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

func (h *BadgeHandler) GetUserBadgesByID(w http.ResponseWriter, r *http.Request) {
	userID, err := bson.ObjectIDFromHex(webhttp.Param(r, "userID"))
	if err != nil {
		webhttp.RespondError(w, http.StatusBadRequest, "invalid user id")
		return
	}
	user, err := h.userSvc.GetByID(r.Context(), userID.Hex())
	if err != nil || user == nil {
		webhttp.RespondError(w, http.StatusNotFound, "user not found")
		return
	}
	if !user.Privacy.ShowBadges {
		webhttp.RespondError(w, http.StatusForbidden, "this user's badges are private")
		return
	}
	h.respondUserBadges(w, r, userID)
}

func (h *BadgeHandler) GetUserSkillStatsByID(w http.ResponseWriter, r *http.Request) {
	userID, err := bson.ObjectIDFromHex(webhttp.Param(r, "userID"))
	if err != nil {
		webhttp.RespondError(w, http.StatusBadRequest, "invalid user id")
		return
	}
	user, err := h.userSvc.GetByID(r.Context(), userID.Hex())
	if err != nil || user == nil {
		webhttp.RespondError(w, http.StatusNotFound, "user not found")
		return
	}
	if !user.Privacy.ShowStats {
		webhttp.RespondError(w, http.StatusForbidden, "this user's stats are private")
		return
	}
	h.respondSkillStats(w, r, userID)
}

func (h *BadgeHandler) GetMyBadges(w http.ResponseWriter, r *http.Request) {
	userID, err := bson.ObjectIDFromHex(middleware.UserIDFromContext(r.Context()))
	if err != nil {
		webhttp.RespondError(w, http.StatusBadRequest, "invalid user id")
		return
	}
	h.respondUserBadges(w, r, userID)
}

func (h *BadgeHandler) GetMySkillStats(w http.ResponseWriter, r *http.Request) {
	userID, err := bson.ObjectIDFromHex(middleware.UserIDFromContext(r.Context()))
	if err != nil {
		webhttp.RespondError(w, http.StatusBadRequest, "invalid user id")
		return
	}
	h.respondSkillStats(w, r, userID)
}

func (h *BadgeHandler) respondUserBadges(w http.ResponseWriter, r *http.Request, userID bson.ObjectID) {
	details, err := h.svc.GetUserBadges(r.Context(), userID)
	if err != nil {
		webhttp.RespondError(w, http.StatusInternalServerError, "failed to fetch badges")
		return
	}
	resources := make([]webhttp.Resource, len(details))
	for i, d := range details {
		resources[i] = userBadgeResource(d)
	}
	webhttp.RespondManyPaginated(w, http.StatusOK, resources, webhttp.ParsePage(r))
}

func (h *BadgeHandler) respondSkillStats(w http.ResponseWriter, r *http.Request, userID bson.ObjectID) {
	stats, err := h.svc.GetDetailedSkillStats(r.Context(), userID)
	if err != nil {
		webhttp.RespondError(w, http.StatusInternalServerError, "failed to fetch stats")
		return
	}
	resources := make([]webhttp.Resource, len(stats))
	for i, ss := range stats {
		resources[i] = skillStatResource(ss)
	}
	webhttp.RespondMany(w, http.StatusOK, resources)
}

func (h *BadgeHandler) Router() *webhttp.Router {
	r := webhttp.New()

	r.Get("/", h.ListBadges)
	r.Get("/{id}", h.GetBadge)

	authGroup := r.Group("")
	authGroup.Use(middleware.Auth(h.jwtSecret))
	authGroup.Post("/{id}/award", h.AwardBadgeToSelf)

	r.Get("/users/{userID}/badges", h.GetUserBadgesByID)
	r.Get("/users/{userID}/skill-stats", h.GetUserSkillStatsByID)

	me := r.Group("/me")
	me.Use(middleware.Auth(h.jwtSecret))
	me.Get("/badges", h.GetMyBadges)
	me.Get("/skill-stats", h.GetMySkillStats)

	return r
}
