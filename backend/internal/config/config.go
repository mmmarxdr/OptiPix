package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port               string
	MaxUploadSize      int64
	TempDir            string
	CorsOrigin         string
	SVGOPath           string
	MaxConcurrency     int
	RateLimitPerMinute int
}

func Load() *Config {
	return &Config{
		Port:               getEnv("PORT", "8090"),
		MaxUploadSize:      getEnvInt64("MAX_UPLOAD_SIZE", 52428800),
		TempDir:            getEnv("TEMP_DIR", "/tmp/optipix"),
		CorsOrigin:         getEnv("CORS_ORIGIN", "*"),
		SVGOPath:           getEnv("SVGO_PATH", "svgo"),
		MaxConcurrency:     int(getEnvInt64("MAX_CONCURRENCY", 4)),
		RateLimitPerMinute: int(getEnvInt64("RATE_LIMIT_PER_MINUTE", 60)),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func getEnvInt64(key string, fallback int64) int64 {
	if value, exists := os.LookupEnv(key); exists {
		if v, err := strconv.ParseInt(value, 10, 64); err == nil {
			return v
		}
	}
	return fallback
}
