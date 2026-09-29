package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
)

const groqEndpoint = "https://api.groq.com/openai/v1/chat/completions"

// ── cost accounting ───────────────────────────────────────────────────────────
// Groq's published rates for openai/gpt-oss-120b (USD per 1M tokens, approximate).
// You are on the free tier, so real spend is ₹0 — this just shows the equivalent.
const (
	groqInPerM  = 0.15
	groqOutPerM = 0.75
	usdToINR    = 83.5 // approximate
)

var (
	groqInTokens  int64
	groqOutTokens int64
	groqCalls     int64
)

// groqUsage returns cumulative token counts and call count for this process run.
func groqUsage() (inTok, outTok, calls int64) {
	return atomic.LoadInt64(&groqInTokens), atomic.LoadInt64(&groqOutTokens), atomic.LoadInt64(&groqCalls)
}

// groqINR converts token counts to the equivalent paid-tier cost in rupees.
func groqINR(inTok, outTok int64) float64 {
	usd := float64(inTok)/1e6*groqInPerM + float64(outTok)/1e6*groqOutPerM
	return usd * usdToINR
}

const musicologistSystem = `You are a Carnatic musicologist specialising in Telugu and Sanskrit compositions
(Tyagaraja, Muthuswami Dikshitar, Syama Sastri, Annamacharya, Kshetrayya and similar).
You are precise, never invent recordings, and always reply with valid JSON only — no markdown, no backticks.`

type groqMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type groqRequest struct {
	Model           string        `json:"model"`
	Messages        []groqMessage `json:"messages"`
	MaxTokens       int           `json:"max_tokens"`
	Temperature     float64       `json:"temperature"`
	ReasoningEffort string        `json:"reasoning_effort,omitempty"`
}

// groqChat sends one completion and returns the (fence-stripped) content.
func groqChat(ctx context.Context, cfg *Config, system, user string, maxTokens int) (string, error) {
	if cfg.GroqKey == "" {
		return "", fmt.Errorf("GROQ_API_KEY not set")
	}
	reqBody := groqRequest{
		Model: cfg.GroqModel,
		Messages: []groqMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
		MaxTokens:   maxTokens,
		Temperature: 0.4,
	}
	// gpt-oss models reason before answering; keep that cheap and leave room for
	// the JSON answer within max_tokens.
	if strings.Contains(cfg.GroqModel, "gpt-oss") {
		reqBody.ReasoningEffort = "low"
	}
	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", groqEndpoint, bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.GroqKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	var out struct {
		Choices []struct {
			Message groqMessage `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int64 `json:"prompt_tokens"`
			CompletionTokens int64 `json:"completion_tokens"`
		} `json:"usage"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", fmt.Errorf("groq parse: %s", strings.TrimSpace(string(raw)))
	}
	if out.Error != nil {
		return "", fmt.Errorf("groq: %s", out.Error.Message)
	}
	if len(out.Choices) == 0 {
		return "", fmt.Errorf("groq: empty response")
	}
	atomic.AddInt64(&groqInTokens, out.Usage.PromptTokens)
	atomic.AddInt64(&groqOutTokens, out.Usage.CompletionTokens)
	atomic.AddInt64(&groqCalls, 1)
	return stripFences(out.Choices[0].Message.Content), nil
}

// groqPing verifies the key + model with a tiny call (used by `check`).
func groqPing(ctx context.Context, cfg *Config) error {
	_, err := groqChat(ctx, cfg, "Reply with the single word OK.", "ping", 5)
	return err
}

// EmotionResult is Groq's read of a recording's comments.
type EmotionResult struct {
	Score float64 `json:"score"` // 0..1 — density of transcendence markers
	Note  string  `json:"note"`  // short human-readable rationale
}

// scoreComments asks Groq to read real comments and judge how "unskippable" the
// crowd finds this rendition — the strongest gem signal.
func scoreComments(ctx context.Context, cfg *Config, kriti, artist string, comments []Comment) (EmotionResult, error) {
	if len(comments) == 0 {
		return EmotionResult{}, fmt.Errorf("no comments")
	}
	var b strings.Builder
	for i, c := range comments {
		if i >= 30 {
			break
		}
		fmt.Fprintf(&b, "- (%d likes) %s\n", c.Likes, oneLine(c.Text, 200))
	}
	user := fmt.Sprintf(`Kriti: "%s" — rendition attributed to %s.
Below are real YouTube comments on this recording.

Rate how strongly the audience reacts to THIS rendition as transcendent/unskippable.
Look for markers: goosebumps, tears, "again and again", "no words", "which sabha/year",
"took me to another world", devotional overwhelm — weighted by how many comments show them.
Ignore generic praise ("nice song") and off-topic chatter.

Comments:
%s

Return ONLY JSON: {"score": 0.0-1.0, "note": "<=12 word reason"}`, kriti, artist, b.String())

	raw, err := groqChat(ctx, cfg, musicologistSystem, user, 500)
	if err != nil {
		return EmotionResult{}, err
	}
	var res EmotionResult
	if err := json.Unmarshal([]byte(raw), &res); err != nil {
		return EmotionResult{}, fmt.Errorf("emotion parse: %s", raw)
	}
	if res.Score < 0 {
		res.Score = 0
	}
	if res.Score > 1 {
		res.Score = 1
	}
	return res, nil
}

// Hook is the framing shown before you press play.
type Hook struct {
	Meaning   string `json:"meaning"`    // what the kriti is saying
	Why       string `json:"why"`        // why this recording is the one
	ListenFor string `json:"listen_for"` // the mood / what to feel
}

// writeHook composes the pre-play framing from the kriti's meaning + the winning
// recording's real crowd signal.
func writeHook(ctx context.Context, cfg *Config, seed SeedKriti, r *Recording) (Hook, error) {
	var topComments strings.Builder
	for i, c := range r.Comments {
		if i >= 8 {
			break
		}
		fmt.Fprintf(&topComments, "- %s\n", oneLine(c.Text, 160))
	}
	user := fmt.Sprintf(`Kriti: "%s" by %s | Deity: %s | Raga: %s | Language: %s
Known meaning: %s

Chosen rendition: "%s" by %s
Signal: %d views, %d likes, %d comments, %d min. Crowd emotion note: %q.
Top comments:
%s

Write the framing a listener reads BEFORE pressing play. Be specific and moving, not generic.
- meaning: 2-3 sentences on what this kriti is truly saying (the bhava, the imagery).
- why: 1-2 sentences on why THIS rendition is the one, grounded in the signal above.
- listen_for: one short line — the mood to surrender to.

Return ONLY JSON: {"meaning":"...","why":"...","listen_for":"..."}`,
		seed.Title, seed.Composer, seed.Deity, seed.Raga, seed.Language, seed.Gloss,
		r.Title, r.Artist, r.Views, r.Likes, r.CommentCnt, r.Duration/60,
		r.EmotionNote, topComments.String())

	raw, err := groqChat(ctx, cfg, musicologistSystem, user, 900)
	if err != nil {
		return Hook{}, err
	}
	var h Hook
	if err := json.Unmarshal([]byte(raw), &h); err != nil {
		return Hook{}, fmt.Errorf("hook parse: %s", raw)
	}
	return h, nil
}

func stripFences(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}

func oneLine(s string, max int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > max {
		s = s[:max]
	}
	return s
}
