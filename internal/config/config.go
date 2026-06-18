package config

import (
	"os"
	"strings"
)

type Config struct {
	Host            string
	Port            string
	DatabaseURL     string
	StateAPIToken   string
	CORSOrigins     []string
	RedisURL        string
	InstanceID      string
	IngestMode      string
	ConsumerEnabled bool
}

func Load() Config {
	host := trimQuotes(strings.TrimSpace(os.Getenv("HOST")))
	if host == "" {
		host = "0.0.0.0"
	}

	port := trimQuotes(strings.TrimSpace(os.Getenv("PORT")))
	if port == "" {
		port = "8080"
	}

	dbURL := NormalizeDatabaseURL(os.Getenv("DATABASE_URL"))
	if dbURL == "" {
		dbURL = "postgres://streamer:streamer@localhost:5432/streamer?sslmode=disable"
	}

	token := trimQuotes(strings.TrimSpace(os.Getenv("STATE_API_TOKEN")))
	if token == "" {
		token = "dev-token"
	}

	cors := os.Getenv("CORS_ORIGINS")
	origins := []string{"http://localhost:5173", "http://127.0.0.1:5173"}
	if cors != "" {
		origins = splitCSV(cors)
	}

	instanceID := os.Getenv("INSTANCE_ID")
	if instanceID == "" {
		host, _ := os.Hostname()
		if host != "" {
			instanceID = host
		} else {
			instanceID = "state-api"
		}
	}

	ingestMode := strings.ToLower(strings.TrimSpace(os.Getenv("INGEST_MODE")))
	if ingestMode == "" {
		ingestMode = "sync"
	}

	consumerEnabled := true
	if v := strings.ToLower(strings.TrimSpace(os.Getenv("CONSUMER_ENABLED"))); v == "false" || v == "0" {
		consumerEnabled = false
	}

	return Config{
		Host:            host,
		Port:            port,
		DatabaseURL:     dbURL,
		StateAPIToken:   token,
		CORSOrigins:     origins,
		RedisURL:        strings.TrimSpace(os.Getenv("REDIS_URL")),
		InstanceID:      instanceID,
		IngestMode:      ingestMode,
		ConsumerEnabled: consumerEnabled,
	}
}

func splitCSV(s string) []string {
	s = trimQuotes(strings.TrimSpace(s))
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = trimQuotes(strings.TrimSpace(p))
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func trimQuotes(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return strings.TrimSpace(s[1 : len(s)-1])
		}
	}
	return s
}
