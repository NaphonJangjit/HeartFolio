package main

import (
	"log"
	"net/http"
	"os"

	v1 "github.com/NaphonJangjit/HeartFolio/cmd/controllers/v1"
	"github.com/NaphonJangjit/HeartFolio/internal/db/mongo"
	redisDB "github.com/NaphonJangjit/HeartFolio/internal/db/redis"
	"github.com/NaphonJangjit/HeartFolio/internal/webhttp"
)

func main() {
	// 1. Load configuration
	jwtSecret := []byte(os.Getenv("JWT_SECRET"))
	if len(jwtSecret) == 0 {
		log.Fatal("JWT_SECRET environment variable not set")
	}
	v1.SetJWTSecret(jwtSecret)

	mongoURI := os.Getenv("MONGODB_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}
	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		dbName = "heartfolio"
	}

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "127.0.0.1:6379"
	}
	redisPassword := os.Getenv("REDIS_PASSWORD")
	redisDBIndex := 0

	mongoClient, err := mongo.Connect(mongoURI)
	if err != nil {
		log.Fatal("Failed to connect to MongoDB:", err)
	}
	defer mongoClient.Close()

	db := mongoClient.Database(dbName)
	usersColl := db.Collection("users")
	userRepo := mongo.NewRepository[v1.User](usersColl)

	var redisRepo *redisDB.Repository
	redisClient, err := redisDB.Connect(redisAddr, redisPassword, redisDBIndex)
	if err != nil {
		log.Printf("Warning: Redis connection failed: %v. Caching disabled.", err)
		redisRepo = nil
	} else {
		defer redisClient.Close()
		redisRepo = redisDB.NewRepository(redisClient.GetClient())
	}

	app := webhttp.New()

	app.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("pong"))
	})

	app.Register("/api/v1", v1API(userRepo, redisRepo))

	log.Println("Server starting on :8080")
	if err := http.ListenAndServe(":8080", app); err != nil {
		log.Fatal(err)
	}
}

func v1API(userRepo *mongo.Repository[v1.User], redisRepo *redisDB.Repository) *webhttp.Router {
	r := webhttp.New()

	userCtrl := v1.NewUserController(userRepo, redisRepo)
	r.Register("/users", userCtrl.Router())


	return r
}