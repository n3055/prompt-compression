// Package config handles application configuration via environment variables.
package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all application configuration.
type Config struct {
	Server     ServerConfig
	Compressor CompressorConfig
	RateLimit  RateLimitConfig
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Port            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

// CompressorConfig holds compression engine tuning parameters.
type CompressorConfig struct {
	// MaxKeywords is the maximum number of keywords to extract per message.
	MaxKeywords int
	// MinWordLength is the minimum character length for a word to be considered.
	MinWordLength int
	// PositionWeightBoost is the multiplier for words in first/last sentences.
	PositionWeightBoost float64
	// LengthBonusThreshold is the character count above which words get a bonus.
	LengthBonusThreshold int
	// LengthBonusMultiplier is the score multiplier for long words.
	LengthBonusMultiplier float64
}

// RateLimitConfig holds rate limiting settings.
type RateLimitConfig struct {
	RequestsPerSecond float64
	BurstSize         int
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port:            envOrDefault("SERVER_PORT", "8080"),
			ReadTimeout:     envDurationOrDefault("SERVER_READ_TIMEOUT", 15*time.Second),
			WriteTimeout:    envDurationOrDefault("SERVER_WRITE_TIMEOUT", 15*time.Second),
			ShutdownTimeout: envDurationOrDefault("SERVER_SHUTDOWN_TIMEOUT", 10*time.Second),
		},
		Compressor: CompressorConfig{
			MaxKeywords:           envIntOrDefault("COMPRESSOR_MAX_KEYWORDS", 8),
			MinWordLength:         envIntOrDefault("COMPRESSOR_MIN_WORD_LENGTH", 3),
			PositionWeightBoost:   envFloatOrDefault("COMPRESSOR_POSITION_BOOST", 1.5),
			LengthBonusThreshold:  envIntOrDefault("COMPRESSOR_LENGTH_THRESHOLD", 5),
			LengthBonusMultiplier: envFloatOrDefault("COMPRESSOR_LENGTH_BONUS", 1.2),
		},
		RateLimit: RateLimitConfig{
			RequestsPerSecond: envFloatOrDefault("RATE_LIMIT_RPS", 10.0),
			BurstSize:         envIntOrDefault("RATE_LIMIT_BURST", 20),
		},
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envIntOrDefault(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func envFloatOrDefault(key string, fallback float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return fallback
}

func envDurationOrDefault(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
