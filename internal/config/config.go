package config

import (
	"os"
	"strconv"
	"time"
	"strings"
)

type Config struct {
	DBHost        string
	DBPort        string
	DBUser        string
	DBPass        string
	DBName        string
	DBSSL         string
	Port          string
	SheetID       string
	SheetTab      string
	SheetInterval time.Duration

	// S3 / Object storage for uploads
	S3Bucket       string
	S3Region       string
	S3Endpoint     string
	S3AccessKey    string
	S3SecretKey    string
	S3UsePathStyle bool
	S3BaseURL      string
	MaxUploadMB    int
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func Load() Config {
	// interval seconds
	intervalSec, _ := strconv.Atoi(env("SHEET_REFRESH_SEC", "300"))
	maxUploadMB, _ := strconv.Atoi(env("MAX_UPLOAD_MB", "10"))
	return Config{
		DBHost:        env("DB_HOST", "localhost"),
		DBPort:        env("DB_PORT", "5432"),
		DBUser:        env("DB_USER", "postgres"),
		DBPass:        env("DB_PASSWORD", "postgres"),
		DBName:        env("DB_NAME", "relief"),
		DBSSL:         env("DB_SSLMODE", "disable"),
		Port:          env("PORT", "8080"),
		SheetID:       env("SHEET_ID", ""),
		SheetTab:      env("SHEET_TAB", ""),
		SheetInterval: time.Duration(intervalSec) * time.Second,

		S3Bucket:       env("S3_BUCKET", ""),
		S3Region:       env("S3_REGION", "auto"),
		S3Endpoint:     env("S3_ENDPOINT", ""),
		S3AccessKey:    env("S3_ACCESS_KEY_ID", ""),
		S3SecretKey:    env("S3_SECRET_ACCESS_KEY", ""),
		S3UsePathStyle: strings.EqualFold(env("S3_USE_PATH_STYLE", "false"), "true"),
		S3BaseURL:      env("S3_BASE_URL", ""), // optional CDN or website URL
		MaxUploadMB:    maxUploadMB,
	}
}
