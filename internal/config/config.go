package config

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port                     string
	MemoryMB                 int
	MaxClients               int
	MaxValueBytes            int
	GlobalRateLimitPerSecond int
	IPRateLimitPerSecond     int
	ReadTimeout              time.Duration
	WriteTimeout             time.Duration
	IdleTimeout              time.Duration
}

// Load reads .env values first, then converts environment variables into the
// typed config used by the rest of the application.
func Load(path string) Config {
	loadDotEnv(path)

	return Config{
		Port:                     envString("BLINKDB_PORT", "6379"),
		MemoryMB:                 envInt("BLINKDB_MEMORY_MB", 256),
		MaxClients:               envInt("BLINKDB_MAX_CLIENTS", 10000),
		MaxValueBytes:            envInt("BLINKDB_MAX_VALUE_BYTES", 1048576),
		GlobalRateLimitPerSecond: envInt("BLINKDB_GLOBAL_RATE_LIMIT_PER_SECOND", 50000),
		IPRateLimitPerSecond:     envInt("BLINKDB_IP_RATE_LIMIT_PER_SECOND", 1000),
		ReadTimeout:              envDuration("BLINKDB_READ_TIMEOUT", 30*time.Second),
		WriteTimeout:             envDuration("BLINKDB_WRITE_TIMEOUT", 5*time.Second),
		IdleTimeout:              envDuration("BLINKDB_IDLE_TIMEOUT", 30*time.Second),
	}
}

// loadDotEnv keeps local development simple: go run ./cmd/server can use the
// same .env file as Docker Compose. Real environment variables win over .env,
// so Docker or shell overrides still work.
func loadDotEnv(path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if key != "" && os.Getenv(key) == "" {
			_ = os.Setenv(key, value)
		}
	}

	if err := scanner.Err(); err != nil {
		return
	}
}

// envString returns a string config value or a safe default when it is missing.
func envString(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

// envInt parses numeric config like memory MB, max clients, and rate limits.
func envInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

// envDuration accepts Go durations like "5s" and also plain seconds like "5".
func envDuration(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := time.ParseDuration(value)
	if err == nil {
		return parsed
	}

	seconds, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return time.Duration(seconds) * time.Second
}
