package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/NaphonJangjit/HeartFolio/internal/db/mongo"
	redisDB "github.com/NaphonJangjit/HeartFolio/internal/db/redis"
	"github.com/NaphonJangjit/HeartFolio/internal/webhttp"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID       bson.ObjectID `bson:"_id,omitempty" json:"id"`
	Email    string        `bson:"email" json:"email"`
	Password string        `bson:"password" json:"-"`
	Role     string        `bson:"role" json:"role"`
}

var jwtSecret []byte

const tokenExpiry = 24 * time.Hour

func SetJWTSecret(secret []byte) {
	jwtSecret = secret
}

type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

func generateToken(user *User) (string, error) {
	claims := Claims{
		UserID: user.ID.Hex(),
		Email:  user.Email,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func validateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return jwtSecret, nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	return nil, jwt.ErrTokenInvalidClaims
}

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			webhttp.ErrorJSON(w, http.StatusUnauthorized, "missing authorization header")
			return
		}

		tokenString := ""
		_, err := fmt.Sscanf(authHeader, "Bearer %s", &tokenString)
		if err != nil || tokenString == "" {
			webhttp.ErrorJSON(w, http.StatusUnauthorized, "invalid authorization header format")
			return
		}

		claims, err := validateToken(tokenString)
		if err != nil {
			webhttp.ErrorJSON(w, http.StatusUnauthorized, "invalid or expired token")
			return
		}

		ctx := context.WithValue(r.Context(), "userID", claims.UserID)
		ctx = context.WithValue(ctx, "email", claims.Email)
		ctx = context.WithValue(ctx, "role", claims.Role)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func AdminAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role := r.Context().Value("role")
		if role == nil || role != "admin" {
			webhttp.ErrorJSON(w, http.StatusForbidden, "admin access required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

type UserController struct {
	repo      *mongo.Repository[User]
	redisRepo *redisDB.Repository
}

func NewUserController(repo *mongo.Repository[User], redisRepo *redisDB.Repository) *UserController {
	return &UserController{repo: repo, redisRepo: redisRepo}
}

func (c *UserController) Register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := webhttp.ReadJSON(r, &req); err != nil {
		webhttp.ErrorJSON(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Email == "" || req.Password == "" {
		webhttp.ErrorJSON(w, http.StatusBadRequest, "email and password required")
		return
	}

	existing, err := c.repo.FindOne(r.Context(), bson.M{"email": req.Email})
	if err != nil {
		webhttp.ErrorJSON(w, http.StatusInternalServerError, "database error")
		return
	}
	if existing != nil {
		webhttp.ErrorJSON(w, http.StatusConflict, "user already exists")
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		webhttp.ErrorJSON(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	user := &User{
		Email:    req.Email,
		Password: string(hashed),
		Role:     "user",
	}
	_, err = c.repo.InsertOne(r.Context(), user)
	if err != nil {
		webhttp.ErrorJSON(w, http.StatusInternalServerError, "failed to create user")
		return
	}

	token, err := generateToken(user)
	if err != nil {
		webhttp.ErrorJSON(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	webhttp.JSON(w, http.StatusCreated, map[string]string{
		"token": token,
		"email": user.Email,
		"id":    user.ID.Hex(),
		"role":  user.Role,
	})
}

func (c *UserController) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := webhttp.ReadJSON(r, &req); err != nil {
		webhttp.ErrorJSON(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Email == "" || req.Password == "" {
		webhttp.ErrorJSON(w, http.StatusBadRequest, "email and password required")
		return
	}

	user, err := c.repo.FindOne(r.Context(), bson.M{"email": req.Email})
	if err != nil {
		webhttp.ErrorJSON(w, http.StatusInternalServerError, "database error")
		return
	}
	if user == nil {
		webhttp.ErrorJSON(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		webhttp.ErrorJSON(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	token, err := generateToken(user)
	if err != nil {
		webhttp.ErrorJSON(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	webhttp.JSON(w, http.StatusOK, map[string]string{
		"token": token,
		"email": user.Email,
		"id":    user.ID.Hex(),
		"role":  user.Role,
	})
}

func (c *UserController) Me(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("userID").(string)

	if c.redisRepo == nil {
		user, err := c.repo.FindByID(r.Context(), userID)
		if err != nil {
			webhttp.ErrorJSON(w, http.StatusInternalServerError, "failed to fetch user")
			return
		}
		if user == nil {
			webhttp.ErrorJSON(w, http.StatusNotFound, "user not found")
			return
		}
		webhttp.JSON(w, http.StatusOK, map[string]string{
			"id":    user.ID.Hex(),
			"email": user.Email,
			"role":  user.Role,
		})
		return
	}

	cacheKey := fmt.Sprintf("user:%s", userID)
	userJSON, err := c.redisRepo.GetOrElse(r.Context(), cacheKey,
		func() (string, error) {
			user, err := c.repo.FindByID(r.Context(), userID)
			if err != nil {
				return "", err
			}
			if user == nil {
				return "", fmt.Errorf("user not found")
			}
			data := map[string]string{
				"id":    user.ID.Hex(),
				"email": user.Email,
				"role":  user.Role,
			}
			jsonBytes, err := json.Marshal(data)
			if err != nil {
				return "", err
			}
			return string(jsonBytes), nil
		},
		5*time.Minute,
	)
	if err != nil {
		webhttp.ErrorJSON(w, http.StatusInternalServerError, "failed to get user data")
		return
	}

	var userData map[string]string
	if err := json.Unmarshal([]byte(userJSON), &userData); err != nil {
		webhttp.ErrorJSON(w, http.StatusInternalServerError, "failed to parse user data")
		return
	}

	webhttp.JSON(w, http.StatusOK, userData)
}

func (c *UserController) Router() *webhttp.Router {
	r := webhttp.New()
	r.Post("/register", c.Register)
	r.Post("/login", c.Login)
	protected := r.Group("")
	protected.Use(AuthMiddleware)
	protected.Get("/me", c.Me)
	return r
}