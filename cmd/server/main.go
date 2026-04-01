package main

import (
	"log"
	"net/http"

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
	cfg := config.Load()
	if len(cfg.JWTSecret) == 0 {
		log.Fatal("JWT_SECRET environment variable not set")
	}

	mongoClient, err := mongo.Connect(cfg.MongoURI)
	if err != nil {
		log.Fatal("Failed to connect to MongoDB:", err)
	}
	defer mongoClient.Close()

	db := mongoClient.Database(cfg.DBName)

	// Collections
	usersColl := db.Collection("users")
	badgesColl := db.Collection("badges")
	userBadgesColl := db.Collection("user_badges")
	questsColl := db.Collection("quests")
	userQuestsColl := db.Collection("user_quests")

	// Repositories
	userRepo := mongo.NewRepository[model.User](usersColl)
	badgeRepo := mongo.NewRepository[model.Badge](badgesColl)
	userBadgeRepo := mongo.NewRepository[model.UserBadge](userBadgesColl)
	questRepo := mongo.NewRepository[model.Quest](questsColl)
	userQuestRepo := mongo.NewRepository[model.UserQuest](userQuestsColl)

	// Redis (optional)
	var redisRepo *redisDB.Repository
	redisClient, err := redisDB.Connect(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	if err != nil {
		log.Printf("Warning: Redis connection failed: %v. Caching disabled.", err)
	} else {
		defer redisClient.Close()
		redisRepo = redisDB.NewRepository(redisClient.GetClient())
	}

	// Services
	userSvc := service.NewUserService(userRepo, redisRepo, cfg.JWTSecret)
	badgeSvc := service.NewBadgeService(badgeRepo, userBadgeRepo, userRepo)
	questSvc := service.NewQuestService(questRepo, userQuestRepo, badgeSvc, userSvc)

	// Ensure indexes
	if err := badgeSvc.EnsureIndexes(badgesColl.Col, userBadgesColl.Col); err != nil {
		log.Printf("Warning: failed to create badge indexes: %v", err)
	}
	if err := questSvc.EnsureIndexes(questsColl.Col, userQuestsColl.Col); err != nil {
		log.Printf("Warning: failed to create quest indexes: %v", err)
	}

	// Router
	app := webhttp.New()

	app.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("pong"))
	})

	app.Register("/api/v1", v1API(cfg.JWTSecret, userSvc, badgeSvc, questSvc))

	// Swagger UI at /docs
	app.Register("/docs", swagger.Handler())

	log.Printf("Server starting on :%s", cfg.Port)
	log.Println("Swagger UI available at /docs/")
	if err := http.ListenAndServe(":"+cfg.Port, app); err != nil {
		log.Fatal(err)
	}
}

func v1API(
	jwtSecret []byte,
	userSvc *service.UserService,
	badgeSvc *service.BadgeService,
	questSvc *service.QuestService,
) *webhttp.Router {
	r := webhttp.New()

	// Public + user routes
	userHandler := v1.NewUserHandler(userSvc, jwtSecret)
	r.Register("/users", userHandler.Router())

	badgeHandler := v1.NewBadgeHandler(badgeSvc, jwtSecret)
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
