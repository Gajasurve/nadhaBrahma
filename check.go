package main

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// runCheck verifies the environment before you rely on it: yt-dlp, Groq, cache.
func runCheck(cfg *Config) error {
	fmt.Printf("%s%sNādabrahma — environment check%s\n\n", cBold, cSaffron, cReset)

	ok := true

	// yt-dlp
	if path, err := exec.LookPath("yt-dlp"); err == nil {
		ver := "?"
		if out, e := exec.Command("yt-dlp", "--version").Output(); e == nil {
			ver = strings.TrimSpace(string(out))
		}
		pass(fmt.Sprintf("yt-dlp  %s (%s)", ver, path))
	} else {
		fail("yt-dlp not found — install: pip install -U yt-dlp")
		ok = false
	}

	// Groq
	if cfg.GroqKey == "" {
		fail("GROQ_API_KEY not set — add it to .env (comment scoring + hooks need it)")
		ok = false
	} else {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		err := groqPing(ctx, cfg)
		cancel()
		if err == nil {
			pass(fmt.Sprintf("Groq    key OK, model %q", cfg.GroqModel))
		} else {
			fail(fmt.Sprintf("Groq    %v", err))
			ok = false
		}
	}

	// cache
	if store, err := openStore(cfg.DBPath); err == nil {
		store.Close()
		pass("cache   " + cfg.DBPath)
	} else {
		fail("cache   " + err.Error())
		ok = false
	}

	// guardrail summary
	fmt.Printf("\n%sguardrails:%s per kriti = 1 search + up to %d comment-fetches (top %d of %d results).\n",
		cDim, cReset, cfg.CommentDepth, cfg.CommentDepth, cfg.MaxCandidates)
	fmt.Printf("%s           yt-dlp is free (no per-video cost). Only Groq tokens are used, per-kriti, free tier.%s\n", cDim, cReset)

	fmt.Println()
	if ok {
		fmt.Printf("%s✓ ready — run: nadabrahma discover%s\n", cGreen, cReset)
		return nil
	}
	return fmt.Errorf("some checks failed")
}

func pass(msg string) { fmt.Printf("  %s✓%s %s\n", cGreen, cReset, msg) }
func fail(msg string) { fmt.Printf("  %s✗%s %s\n", "\033[31m", cReset, msg) }
