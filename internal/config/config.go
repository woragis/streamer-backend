package config

import (
	"os"
	"strings"
)

type Config struct {
	Port          string
	DatabaseURL   string
	StateAPIToken string
	CORSOrigins   []string
	RedisURL      string
	InstanceID    string
}

func Load() Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "./data/state.db"
	}

	token := os.Getenv("STATE_API_TOKEN")
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

	return Config{
		Port:          port,
		DatabaseURL:   dbURL,
		StateAPIToken: token,
		CORSOrigins:   origins,
		RedisURL:      strings.TrimSpace(os.Getenv("REDIS_URL")),
		InstanceID:    instanceID,
	}
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
