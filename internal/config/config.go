package config

import "os"

type Config struct {
	JWTSecret     []byte
	MongoURI      string
	DBName        string
	RedisAddr     string
	RedisPassword string
	RedisDB       int
	Port          string
}

func Load() *Config {
	return &Config{
		JWTSecret:     []byte(getEnv("JWT_SECRET", "")),
		MongoURI:      getEnv("MONGODB_URI", "mongodb://localhost:27017"),
		DBName:        getEnv("DB_NAME", "heartfolio"),
		RedisAddr:     getEnv("REDIS_ADDR", "127.0.0.1:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       0,
		Port:          getEnv("PORT", "8080"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
