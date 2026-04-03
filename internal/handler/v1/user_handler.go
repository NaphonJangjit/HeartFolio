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

type UserHandler struct {
	svc          *service.UserService
	portfolioSvc *service.PortfolioService
	moodSvc      *service.MoodService
	jwtSecret    []byte
}

func NewUserHandler(svc *service.UserService, portfolioSvc *service.PortfolioService, moodSvc *service.MoodService, jwtSecret []byte) *UserHandler {
	return &UserHandler{svc: svc, portfolioSvc: portfolioSvc, moodSvc: moodSvc, jwtSecret: jwtSecret}
}

func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := webhttp.ReadJSON(r, &req); err != nil {
		webhttp.RespondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	user, token, err := h.svc.Register(r.Context(), req.Email, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserAlreadyExists):
			webhttp.RespondError(w, http.StatusConflict, "user already exists")
		case errors.Is(err, service.ErrInvalidEmail):
			webhttp.RespondError(w, http.StatusBadRequest, "invalid email format")
		case errors.Is(err, service.ErrPasswordTooWeak):
			webhttp.RespondError(w, http.StatusBadRequest, "password must be at least 8 characters")
		case req.Email == "" || req.Password == "":
			webhttp.RespondError(w, http.StatusBadRequest, "email and password required")
		default:
			webhttp.RespondError(w, http.StatusInternalServerError, "failed to create user")
		}
		return
	}

	webhttp.RespondOne(w, http.StatusCreated, webhttp.Resource{
		Type: "users",
		ID:   user.ID.Hex(),
		Attributes: map[string]interface{}{
			"email": user.Email,
			"role":  user.Role,
			"token": token,
		},
	})
}

func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := webhttp.ReadJSON(r, &req); err != nil {
		webhttp.RespondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	user, token, err := h.svc.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidCredentials):
			webhttp.RespondError(w, http.StatusUnauthorized, "invalid credentials")
		case req.Email == "" || req.Password == "":
			webhttp.RespondError(w, http.StatusBadRequest, "email and password required")
		default:
			webhttp.RespondError(w, http.StatusInternalServerError, "database error")
		}
		return
	}

	webhttp.RespondOne(w, http.StatusOK, webhttp.Resource{
		Type: "users",
		ID:   user.ID.Hex(),
		Attributes: map[string]interface{}{
			"email": user.Email,
			"role":  user.Role,
			"token": token,
		},
	})
}

func (h *UserHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())

	user, err := h.svc.GetByID(r.Context(), userID)
	if err != nil {
		webhttp.RespondError(w, http.StatusInternalServerError, "failed to fetch user")
		return
	}
	if user == nil {
		webhttp.RespondError(w, http.StatusNotFound, "user not found")
		return
	}

	webhttp.RespondOne(w, http.StatusOK, webhttp.Resource{
		Type: "users",
		ID:   user.ID.Hex(),
		Attributes: map[string]interface{}{
			"email":   user.Email,
			"role":    user.Role,
			"exp":     user.EXP,
			"level":   user.Level,
			"tier":    user.Tier(),
			"privacy": user.Privacy,
		},
	})
}

func (h *UserHandler) UpdatePrivacy(w http.ResponseWriter, r *http.Request) {
	userID, err := bson.ObjectIDFromHex(middleware.UserIDFromContext(r.Context()))
	if err != nil {
		webhttp.RespondError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	var req model.PrivacySettings
	if err := webhttp.ReadJSON(r, &req); err != nil {
		webhttp.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.svc.UpdatePrivacy(r.Context(), userID, req); err != nil {
		webhttp.RespondError(w, http.StatusInternalServerError, "failed to update privacy")
		return
	}

	webhttp.RespondOne(w, http.StatusOK, webhttp.Resource{
		Type: "privacy-settings",
		ID:   userID.Hex(),
		Attributes: map[string]interface{}{
			"show_badges": req.ShowBadges,
			"show_stats":  req.ShowStats,
			"show_quests": req.ShowQuests,
		},
	})
}

func (h *UserHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID, err := bson.ObjectIDFromHex(middleware.UserIDFromContext(r.Context()))
	if err != nil {
		webhttp.RespondError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	var req struct {
		Email string `json:"email"`
	}
	if err := webhttp.ReadJSON(r, &req); err != nil {
		webhttp.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Email != "" {
		if err := service.ValidateEmail(req.Email); err != nil {
			webhttp.RespondError(w, http.StatusBadRequest, "invalid email format")
			return
		}
	}

	user, err := h.svc.UpdateProfile(r.Context(), userID, req.Email)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserAlreadyExists):
			webhttp.RespondError(w, http.StatusConflict, "email already in use")
		default:
			webhttp.RespondError(w, http.StatusInternalServerError, "failed to update profile")
		}
		return
	}

	webhttp.RespondOne(w, http.StatusOK, webhttp.Resource{
		Type: "users",
		ID:   user.ID.Hex(),
		Attributes: map[string]interface{}{
			"email":   user.Email,
			"role":    user.Role,
			"exp":     user.EXP,
			"level":   user.Level,
			"tier":    user.Tier(),
			"privacy": user.Privacy,
		},
	})
}

func (h *UserHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	userID, err := bson.ObjectIDFromHex(middleware.UserIDFromContext(r.Context()))
	if err != nil {
		webhttp.RespondError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	var req struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := webhttp.ReadJSON(r, &req); err != nil {
		webhttp.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.OldPassword == "" || req.NewPassword == "" {
		webhttp.RespondError(w, http.StatusBadRequest, "old_password and new_password are required")
		return
	}
	if len(req.NewPassword) < 8 {
		webhttp.RespondError(w, http.StatusBadRequest, "new_password must be at least 8 characters")
		return
	}

	err = h.svc.ChangePassword(r.Context(), userID, req.OldPassword, req.NewPassword)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidCredentials):
			webhttp.RespondError(w, http.StatusUnauthorized, "incorrect old password")
		case errors.Is(err, service.ErrNotFound):
			webhttp.RespondError(w, http.StatusNotFound, "user not found")
		default:
			webhttp.RespondError(w, http.StatusInternalServerError, "failed to change password")
		}
		return
	}

	webhttp.RespondOne(w, http.StatusOK, webhttp.Resource{
		Type:       "users",
		ID:         userID.Hex(),
		Attributes: map[string]string{"message": "password changed"},
	})
}

func (h *UserHandler) SubmitMood(w http.ResponseWriter, r *http.Request) {
	userID, err := bson.ObjectIDFromHex(middleware.UserIDFromContext(r.Context()))
	if err != nil {
		webhttp.RespondError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	var req struct {
		Mood string `json:"mood"`
		Note string `json:"note"`
	}
	if err := webhttp.ReadJSON(r, &req); err != nil {
		webhttp.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Mood == "" {
		webhttp.RespondError(w, http.StatusBadRequest, "mood is required")
		return
	}
	if err := service.ValidateMood(req.Mood); err != nil {
		webhttp.RespondError(w, http.StatusBadRequest, "invalid mood value; allowed: happy, sad, anxious, angry, lonely, burnout, stressed, calm, motivated, confused, hopeful, grateful, overwhelmed, content, frustrated, excited")
		return
	}

	entry, err := h.moodSvc.Submit(r.Context(), userID, req.Mood, req.Note)
	if err != nil {
		webhttp.RespondError(w, http.StatusInternalServerError, "failed to submit mood")
		return
	}

	webhttp.RespondOne(w, http.StatusCreated, webhttp.Resource{
		Type: "mood-entries",
		ID:   entry.ID.Hex(),
		Attributes: map[string]interface{}{
			"mood":       entry.Mood,
			"note":       entry.Note,
			"created_at": entry.CreatedAt,
		},
	})
}

func (h *UserHandler) GetMoodHistory(w http.ResponseWriter, r *http.Request) {
	userID, err := bson.ObjectIDFromHex(middleware.UserIDFromContext(r.Context()))
	if err != nil {
		webhttp.RespondError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	entries, err := h.moodSvc.GetHistory(r.Context(), userID)
	if err != nil {
		webhttp.RespondError(w, http.StatusInternalServerError, "failed to fetch mood history")
		return
	}

	resources := make([]webhttp.Resource, len(entries))
	for i, e := range entries {
		resources[i] = webhttp.Resource{
			Type: "mood-entries",
			ID:   e.ID.Hex(),
			Attributes: map[string]interface{}{
				"mood":       e.Mood,
				"note":       e.Note,
				"created_at": e.CreatedAt,
			},
		}
	}
	webhttp.RespondManyPaginated(w, http.StatusOK, resources, webhttp.ParsePage(r))
}

func (h *UserHandler) GetMyPortfolio(w http.ResponseWriter, r *http.Request) {
	userID, err := bson.ObjectIDFromHex(middleware.UserIDFromContext(r.Context()))
	if err != nil {
		webhttp.RespondError(w, http.StatusBadRequest, "invalid user id")
		return
	}
	h.respondPortfolio(w, r, userID)
}

func (h *UserHandler) GetUserPortfolio(w http.ResponseWriter, r *http.Request) {
	userID, err := bson.ObjectIDFromHex(webhttp.Param(r, "userID"))
	if err != nil {
		webhttp.RespondError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	// Check privacy: fetch user to see if portfolio is public
	user, err := h.svc.GetByID(r.Context(), userID.Hex())
	if err != nil {
		webhttp.RespondError(w, http.StatusInternalServerError, "failed to fetch user")
		return
	}
	if user == nil {
		webhttp.RespondError(w, http.StatusNotFound, "user not found")
		return
	}
	if !user.Privacy.ShowBadges && !user.Privacy.ShowStats {
		webhttp.RespondError(w, http.StatusForbidden, "this user's portfolio is private")
		return
	}

	h.respondPortfolio(w, r, userID)
}

func (h *UserHandler) respondPortfolio(w http.ResponseWriter, r *http.Request, userID bson.ObjectID) {
	portfolio, err := h.portfolioSvc.GetPortfolio(r.Context(), userID)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			webhttp.RespondError(w, http.StatusNotFound, "user not found")
			return
		}
		webhttp.RespondError(w, http.StatusInternalServerError, "failed to build portfolio")
		return
	}

	// Build skill stats resources
	skillStatsResources := make([]webhttp.Resource, len(portfolio.SkillStats))
	for i, ss := range portfolio.SkillStats {
		skillStatsResources[i] = webhttp.Resource{
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

	// Build mood resources
	moodResources := make([]webhttp.Resource, len(portfolio.RecentMoods))
	for i, m := range portfolio.RecentMoods {
		moodResources[i] = webhttp.Resource{
			Type: "mood-entries",
			ID:   m.ID.Hex(),
			Attributes: map[string]interface{}{
				"mood":       m.Mood,
				"note":       m.Note,
				"created_at": m.CreatedAt,
			},
		}
	}

	webhttp.RespondOne(w, http.StatusOK, webhttp.Resource{
		Type: "portfolios",
		ID:   portfolio.UserID,
		Attributes: map[string]interface{}{
			"email":         portfolio.Email,
			"level":         portfolio.Level,
			"exp":           portfolio.EXP,
			"tier":          portfolio.Tier,
			"total_badges":  portfolio.TotalBadges,
			"quest_summary": portfolio.QuestSummary,
		},
		Relationships: map[string]webhttp.Relationship{
			"skill_stats":  {Data: skillStatsResources},
			"recent_moods": {Data: moodResources},
		},
	})
}

func (h *UserHandler) Router() *webhttp.Router {
	r := webhttp.New()
	r.Post("/register", h.Register)
	r.Post("/login", h.Login)

	// Public portfolio (respects privacy)
	r.Get("/{userID}/portfolio", h.GetUserPortfolio)

	protected := r.Group("")
	protected.Use(middleware.Auth(h.jwtSecret))
	protected.Get("/me", h.Me)
	protected.Patch("/me", h.UpdateProfile)
	protected.Patch("/me/privacy", h.UpdatePrivacy)
	protected.Post("/me/password", h.ChangePassword)
	protected.Post("/me/mood", h.SubmitMood)
	protected.Get("/me/moods", h.GetMoodHistory)
	protected.Get("/me/portfolio", h.GetMyPortfolio)

	return r
}
