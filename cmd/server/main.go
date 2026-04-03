package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/NaphonJangjit/HeartFolio/internal/config"
	"github.com/NaphonJangjit/HeartFolio/internal/db/mongo"
	redisDB "github.com/NaphonJangjit/HeartFolio/internal/db/redis"
	"github.com/NaphonJangjit/HeartFolio/internal/handler/swagger"
	v1 "github.com/NaphonJangjit/HeartFolio/internal/handler/v1"
	"github.com/NaphonJangjit/HeartFolio/internal/handler/v1/admin"
	"github.com/NaphonJangjit/HeartFolio/internal/middleware"
	"github.com/NaphonJangjit/HeartFolio/internal/model"
	"github.com/NaphonJangjit/HeartFolio/internal/service"
	"github.com/NaphonJangjit/HeartFolio/internal/webhttp"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg := config.Load()
	if len(cfg.JWTSecret) == 0 {
		slog.Error("JWT_SECRET environment variable not set")
		os.Exit(1)
	}

	mongoClient, err := mongo.Connect(cfg.MongoURI)
	if err != nil {
		slog.Error("failed to connect to MongoDB", "error", err)
		os.Exit(1)
	}
	defer mongoClient.Close()

	db := mongoClient.Database(cfg.DBName)

	// Collections
	usersColl := db.Collection("users")
	badgesColl := db.Collection("badges")
	userBadgesColl := db.Collection("user_badges")
	questsColl := db.Collection("quests")
	userQuestsColl := db.Collection("user_quests")
	moodsColl := db.Collection("moods")

	// Repositories
	userRepo := mongo.NewRepository[model.User](usersColl)
	badgeRepo := mongo.NewRepository[model.Badge](badgesColl)
	userBadgeRepo := mongo.NewRepository[model.UserBadge](userBadgesColl)
	questRepo := mongo.NewRepository[model.Quest](questsColl)
	userQuestRepo := mongo.NewRepository[model.UserQuest](userQuestsColl)
	moodRepo := mongo.NewRepository[model.MoodEntry](moodsColl)

	// Redis (optional)
	var redisRepo *redisDB.Repository
	redisClient, err := redisDB.Connect(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	if err != nil {
		slog.Warn("Redis connection failed, caching disabled", "error", err)
	} else {
		defer redisClient.Close()
		redisRepo = redisDB.NewRepository(redisClient.GetClient())
	}

	// Services
	userSvc := service.NewUserService(userRepo, redisRepo, cfg.JWTSecret)
	badgeSvc := service.NewBadgeService(badgeRepo, userBadgeRepo, userRepo)
	questSvc := service.NewQuestService(questRepo, userQuestRepo, badgeSvc, userSvc)
	moodSvc := service.NewMoodService(moodRepo)
	portfolioSvc := service.NewPortfolioService(userSvc, badgeSvc, questSvc, moodSvc)

	// Ensure indexes
	if err := userSvc.EnsureIndexes(usersColl.Col); err != nil {
		slog.Warn("failed to create user indexes", "error", err)
	}
	if err := badgeSvc.EnsureIndexes(badgesColl.Col, userBadgesColl.Col); err != nil {
		slog.Warn("failed to create badge indexes", "error", err)
	}
	if err := questSvc.EnsureIndexes(questsColl.Col, userQuestsColl.Col); err != nil {
		slog.Warn("failed to create quest indexes", "error", err)
	}
	if err := moodSvc.EnsureIndexes(moodsColl.Col); err != nil {
		slog.Warn("failed to create mood indexes", "error", err)
	}

	// Router
	app := webhttp.New()
	app.Use(webhttp.CORS(webhttp.DefaultCORSConfig()))

	app.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("pong"))
	})

	app.Register("/api/v1", v1API(cfg.JWTSecret, userSvc, badgeSvc, questSvc, moodSvc, portfolioSvc))

	// Swagger UI at /docs
	app.Register("/docs", swagger.Handler())

	// Graceful shutdown
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      app,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("server starting", "port", cfg.Port, "swagger", "/docs/")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	slog.Info("shutting down", "signal", sig.String())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("forced shutdown", "error", err)
	}
	slog.Info("server stopped")
}

func v1API(
	jwtSecret []byte,
	userSvc *service.UserService,
	badgeSvc *service.BadgeService,
	questSvc *service.QuestService,
	moodSvc *service.MoodService,
	portfolioSvc *service.PortfolioService,
) *webhttp.Router {
	r := webhttp.New()

	// Public + user routes
	userHandler := v1.NewUserHandler(userSvc, portfolioSvc, moodSvc, jwtSecret)
	r.Register("/users", userHandler.Router())

	badgeHandler := v1.NewBadgeHandler(badgeSvc, userSvc, jwtSecret)
	r.Register("/badges", badgeHandler.Router())

	questHandler := v1.NewQuestHandler(questSvc, jwtSecret)
	r.Register("/quests", questHandler.Router())

	// Admin routes
	adminRouter := webhttp.New()
	adminRouter.Use(middleware.Auth(jwtSecret))
	adminRouter.Use(middleware.RequireAdmin)

	adminBadgeHandler := admin.NewBadgeHandler(badgeSvc)
	adminRouter.Register("/badges", adminBadgeHandler.Router())

	adminQuestHandler := admin.NewQuestHandler(questSvc)
	adminRouter.Register("/quests", adminQuestHandler.Router())

	r.Register("/admin", adminRouter)

	return r
}
