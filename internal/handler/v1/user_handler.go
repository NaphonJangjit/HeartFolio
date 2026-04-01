package v1

import (
	"errors"
	"net/http"

	"github.com/NaphonJangjit/HeartFolio/internal/middleware"
	"github.com/NaphonJangjit/HeartFolio/internal/service"
	"github.com/NaphonJangjit/HeartFolio/internal/webhttp"
)

type UserHandler struct {
	svc       *service.UserService
	jwtSecret []byte
}

func NewUserHandler(svc *service.UserService, jwtSecret []byte) *UserHandler {
	return &UserHandler{svc: svc, jwtSecret: jwtSecret}
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
			"email": user.Email,
			"role":  user.Role,
			"exp":   user.EXP,
			"level": user.Level,
		},
	})
}

func (h *UserHandler) Router() *webhttp.Router {
	r := webhttp.New()
	r.Post("/register", h.Register)
	r.Post("/login", h.Login)

	protected := r.Group("")
	protected.Use(middleware.Auth(h.jwtSecret))
	protected.Get("/me", h.Me)

	return r
}
