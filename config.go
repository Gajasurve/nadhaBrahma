package main

import (
	"os"
	"path/filepath"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds everything the CLI needs. Loaded from .env + environment.
// No credit-card-bearing service is required: yt-dlp is free, Groq has a free tier.
type Config struct {
	GroqKey   string
	GroqModel string

	DBPath string // local SQLite gem cache

	// Cost/volume guardrails — these keep API + network work bounded.
	MaxCandidates   int // results per search query (we run a few query angles)
	MetaDepth       int // candidates we metadata-screen for free (no comments) to find hidden gems
	CommentDepth    int // top-N (by metric score) we fetch comments for (the expensive step)
	MaxComments     int // comments pulled per video
	MinDurationSecs int // ignore clips shorter than this (usually junk/shorts)
	MaxViews        int64 // if >0, skip renditions above this view count ("deep cuts")
}

// loadConfig reads .env (next to the binary, then cwd) and the environment.
func loadConfig() *Config {
	// Best-effort .env load; env vars still win if already set.
	if exe, err := os.Executable(); err == nil {
		_ = godotenv.Load(filepath.Join(filepath.Dir(exe), ".env"))
	}
	_ = godotenv.Load(".env")

	c := &Config{
		GroqKey:         os.Getenv("GROQ_API_KEY"),
		GroqModel:       envOr("GROQ_MODEL", "openai/gpt-oss-120b"),
		DBPath:          envOr("DB_PATH", defaultDBPath()),
		MaxCandidates:   envInt("MAX_CANDIDATES", 10),
		MetaDepth:       envInt("META_DEPTH", 9),
		CommentDepth:    envInt("COMMENT_DEPTH", 4),
		MaxComments:     envInt("MAX_COMMENTS", 30),
		MinDurationSecs: envInt("MIN_DURATION_SECS", 180),
		MaxViews:        int64(envInt("MAX_VIEWS", 0)),
	}
	return c
}

// defaultDBPath keeps everything local, inside the project's ./data folder so the
// cache is visible and the corpus compounds. No remote store is used.
func defaultDBPath() string {
	dir := "data"
	_ = os.MkdirAll(dir, 0o755)
	return filepath.Join(dir, "gems.db")
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
