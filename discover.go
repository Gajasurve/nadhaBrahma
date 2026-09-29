package main

import (
	"bufio"
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

// runDiscover is the deity-shuffle gem feed — the heart of v1.
// deityFilter (optional) pins the feed to one deity, e.g. "Rama".
// deep enables "deep cuts": a view ceiling so only lesser-known renditions surface.
func runDiscover(cfg *Config, deityFilter string, deep bool) error {
	if !ytdlpAvailable() {
		return fmt.Errorf("yt-dlp not found on PATH — install it first (pip install -U yt-dlp)")
	}
	if deep && cfg.MaxViews == 0 {
		cfg.MaxViews = 150000 // default "deep cut" ceiling
	}

	store, err := openStore(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("open cache: %w", err)
	}
	defer store.Close()

	// Groq is core but we degrade gracefully so a bad/absent key still shows gems.
	groqOK := false
	if cfg.GroqKey != "" {
		ctx, cancel := ctxTimeout(context.Background(), 20*time.Second)
		if err := groqPing(ctx, cfg); err == nil {
			groqOK = true
		} else {
			fmt.Printf("%s⚠  Groq unavailable (%v) — ranking on metrics only, no AI hooks.%s\n", cDim, err, cReset)
		}
		cancel()
	} else {
		fmt.Printf("%s⚠  GROQ_API_KEY not set — ranking on metrics only. Run `nadabrahma check`.%s\n", cDim, cReset)
	}

	mode := ""
	if cfg.MaxViews > 0 {
		mode = fmt.Sprintf(" %s(deep cuts: under %s views)%s", cGold, humanize(cfg.MaxViews), cReset)
	}
	fmt.Printf("%s%sNādabrahma%s %s— deity-shuffle gem feed.%s%s\n%s[enter] next · [o] open · [w] why · [q] quit%s\n",
		cBold, cSaffron, cReset, cDim, cReset, mode, cDim, cReset)

	in := bufio.NewScanner(os.Stdin)
	for {
		deity := pickDeity(store, deityFilter)
		seed, ok := pickKriti(store, deity)
		if !ok {
			store.addHistory("deity", deity)
			continue
		}

		fmt.Printf("\n%s… seeking a %s gem: %q%s\n", cDim, deity, seed.Title, cReset)

		bi, bo, bc := groqUsage() // snapshot to measure this gem's Groq cost
		rec, known, alts, hook, peak, err := findGem(context.Background(), cfg, store, seed, groqOK)
		if err != nil || rec == nil {
			fmt.Printf("%s   (no strong rendition found%s — rotating)%s\n", cDim, err2s(err), cReset)
			store.addHistory("kriti", seed.Title)
			continue
		}

		renderResult(seed, rec, known, hook, peak, alts)

		// Groq cost accounting for this gem (and running total).
		ai, ao, ac := groqUsage()
		di, do, dc := ai-bi, ao-bo, ac-bc
		fmt.Printf("%s   groq: this gem %d calls · %d in + %d out tok ≈ ₹%.4f  |  run total ₹%.4f  (free tier: ₹0)%s\n",
			cDim, dc, di, do, groqINR(di, do), groqINR(ai, ao), cReset)

		store.addHistory("deity", deity)
		store.addHistory("kriti", seed.Title)
		store.addHistory("video", rec.VideoID)

		// interaction
		next := false
		for !next {
			fmt.Print("› ")
			if !in.Scan() {
				return nil // EOF / ctrl-D
			}
			switch strings.ToLower(strings.TrimSpace(in.Text())) {
			case "", "n":
				next = true
			case "q":
				printGroqTotal()
				fmt.Println("🙏")
				return nil
			case "o":
				openBrowser(watchURL(rec.VideoID, rec.MagicTS))
			case "w":
				renderBreakdown(rec)
			default:
				next = true
			}
		}
	}
}

// findGem finds, for a kriti, both the well-known canonical version (the anchor
// most people already hear) and the gem(s) our scoring surfaces beyond it:
//  1. gather candidates from several search angles (studio, live, raga).
//  2. the most-viewed candidate is the "known version" — force-kept as the anchor.
//  3. metadata-screen up to MetaDepth for FREE and rank by a view-INDEPENDENT metric
//     score + hidden-gem bonus; spend the emotion pass only on the top CommentDepth.
// Returns: gem (our pick), known (the famous anchor), alts (other scored renditions).
func findGem(ctx context.Context, cfg *Config, store *Store, seed SeedKriti, groqOK bool) (*Recording, *Recording, []*Recording, Hook, float64, error) {
	cands := gatherCandidates(ctx, cfg, seed)
	if len(cands) == 0 {
		return nil, nil, nil, Hook{}, 0, fmt.Errorf("no candidates")
	}

	// The most-viewed rendition is the canonical "known version" — our anchor.
	knownCand := cands[0]
	for _, c := range cands {
		if c.Views > knownCand.Views {
			knownCand = c
		}
	}

	// Sample a spread across the whole result list so buried renditions get a look,
	// then guarantee the known version is in the pool so we can always show it.
	pool := selectSpread(cands, cfg.MetaDepth)
	inPool := false
	for _, c := range pool {
		if c.VideoID == knownCand.VideoID {
			inPool = true
			break
		}
	}
	if !inPool {
		pool = append([]Candidate{knownCand}, pool...)
	}

	// ── free metadata pass (parallel): score everything WITHOUT emotion ──
	metaScored := make([]*Recording, len(pool))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 3)
	for i, c := range pool {
		wg.Add(1)
		go func(i int, c Candidate) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			if r, ok := store.getRecording(c.VideoID); ok {
				metaScored[i] = r // already fully scored (incl. emotion)
				return
			}
			mctx, mcancel := ctxTimeout(ctx, 30*time.Second)
			r, err := fetchMeta(mctx, c.VideoID)
			mcancel()
			if err != nil {
				return
			}
			r.Kriti, r.Deity = seed.Title, seed.Deity
			computeMetrics(r)
			metaScored[i] = r
		}(i, c)
	}
	wg.Wait()

	var allMeta []*Recording
	for _, r := range metaScored {
		if r != nil {
			allMeta = append(allMeta, r)
		}
	}
	if len(allMeta) == 0 {
		return nil, nil, nil, Hook{}, 0, fmt.Errorf("no renditions extracted")
	}

	// The known version's full record (artist, likes) — fall back to search data.
	var known *Recording
	for _, r := range allMeta {
		if r.VideoID == knownCand.VideoID {
			known = r
			break
		}
	}
	if known == nil {
		known = &Recording{VideoID: knownCand.VideoID, Title: knownCand.Title,
			Artist: "—", Views: knownCand.Views, Duration: knownCand.Duration}
	}

	// Rank by emotion-free metric score (view-independent + hidden bonus): this is
	// what promotes under-exposed, high-passion renditions into the emotion pass.
	sort.Slice(allMeta, func(i, j int) bool {
		return metricScore(allMeta[i]) > metricScore(allMeta[j])
	})
	commentSet := allMeta
	if len(commentSet) > cfg.CommentDepth {
		commentSet = commentSet[:cfg.CommentDepth]
	}

	// ── expensive comment + emotion pass, only on the metric leaders ──
	var scored []*Recording
	for _, r := range commentSet {
		if len(r.Comments) == 0 && r.EmotionScore == 0 { // not yet emotion-scored
			cctx, ccancel := ctxTimeout(ctx, 30*time.Second)
			if cs, err := fetchComments(cctx, r.VideoID, cfg.MaxComments); err == nil {
				r.Comments = cs
			}
			ccancel()
			if groqOK && len(r.Comments) > 0 {
				ectx, ecancel := ctxTimeout(ctx, 25*time.Second)
				if res, err := scoreComments(ectx, cfg, seed.Title, r.Artist, r.Comments); err == nil {
					r.EmotionScore = res.Score
					r.EmotionNote = res.Note
				}
				ecancel()
			}
			finalizeGem(r)
			_ = store.saveRecording(r)
		}
		scored = append(scored, r)
	}

	sort.Slice(scored, func(i, j int) bool { return scored[i].GemScore > scored[j].GemScore })
	winner := scored[0]

	// alts = other scored renditions, minus the winner and the known anchor.
	var alts []*Recording
	for _, r := range scored {
		if r.VideoID == winner.VideoID || r.VideoID == known.VideoID {
			continue
		}
		alts = append(alts, r)
	}

	// personal-peak flag vs this artist's cached average
	peak := 0.0
	if mean, n := store.artistBaseline(winner.Artist); n >= 3 && mean > 0 {
		peak = winner.GemScore / mean
	}

	// hook
	hook := fallbackHook(seed, winner)
	if groqOK {
		hctx, hcancel := ctxTimeout(ctx, 30*time.Second)
		if h, err := writeHook(hctx, cfg, seed, winner); err == nil {
			hook = h
		}
		hcancel()
	}
	return winner, known, alts, hook, peak, nil
}

// gatherCandidates searches several angles so the pool isn't just the popular
// top-of-search results: studio, live/concert, and raga-specific queries.
func gatherCandidates(ctx context.Context, cfg *Config, seed SeedKriti) []Candidate {
	queries := []string{
		fmt.Sprintf("%s %s carnatic", seed.Title, seed.Composer),
		fmt.Sprintf("%s carnatic live concert", seed.Title),
	}
	if seed.Raga != "" {
		queries = append(queries, fmt.Sprintf("%s %s carnatic", seed.Title, seed.Raga))
	} else {
		queries = append(queries, fmt.Sprintf("%s carnatic rare", seed.Title))
	}

	seen := map[string]bool{}
	var out []Candidate
	for _, q := range queries {
		sctx, cancel := ctxTimeout(ctx, 45*time.Second)
		cands, err := searchCandidates(sctx, q, cfg.MaxCandidates)
		cancel()
		if err != nil {
			continue
		}
		for _, c := range cands {
			if seen[c.VideoID] || !looksLikeGoodCandidate(c, cfg.MinDurationSecs) {
				continue
			}
			if cfg.MaxViews > 0 && c.Views > cfg.MaxViews {
				continue // deep-cuts mode: skip the well-known, highly-viewed versions
			}
			seen[c.VideoID] = true
			out = append(out, c)
		}
	}
	return out
}

// selectSpread picks up to n candidates spread evenly across the whole pool, so we
// sample deep results (where gems hide), not just the first few.
func selectSpread(cands []Candidate, n int) []Candidate {
	if len(cands) <= n {
		return cands
	}
	out := make([]Candidate, 0, n)
	step := float64(len(cands)) / float64(n)
	for i := 0; i < n; i++ {
		out = append(out, cands[int(float64(i)*step)])
	}
	return out
}

// fallbackHook is used when Groq is unavailable — meaning from the curated seed.
func fallbackHook(seed SeedKriti, r *Recording) Hook {
	why := fmt.Sprintf("Top-ranked of the renditions found — %s views, %s likes.",
		humanize(r.Views), humanize(r.Likes))
	return Hook{Meaning: seed.Gloss, Why: why, ListenFor: seed.Deity + " bhava"}
}

// pickDeity chooses a deity. If filter names a known deity, it is pinned;
// otherwise a fresh one is chosen, avoiding the last few for variety.
func pickDeity(store *Store, filter string) string {
	all := deities()
	if filter != "" {
		for _, d := range all {
			if strings.EqualFold(d, filter) {
				return d
			}
		}
		fmt.Printf("%s(unknown deity %q — shuffling all)%s\n", cDim, filter, cReset)
	}
	recent := store.recentHistory("deity", 3)
	var fresh []string
	for _, d := range all {
		if !recent[d] {
			fresh = append(fresh, d)
		}
	}
	if len(fresh) == 0 {
		fresh = all
	}
	return fresh[rand.Intn(len(fresh))]
}

// pickKriti chooses a kriti for the deity, avoiding recently shown titles.
func pickKriti(store *Store, deity string) (SeedKriti, bool) {
	ks := kritisFor(deity)
	if len(ks) == 0 {
		return SeedKriti{}, false
	}
	recent := store.recentHistory("kriti", 20)
	var fresh []SeedKriti
	for _, k := range ks {
		if !recent[k.Title] {
			fresh = append(fresh, k)
		}
	}
	if len(fresh) == 0 {
		fresh = ks
	}
	return fresh[rand.Intn(len(fresh))], true
}

func openBrowser(url string) {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
	case "windows":
		cmd, args = "rundll32", []string{"url.dll,FileProtocolHandler"}
	default:
		cmd = "xdg-open"
	}
	args = append(args, url)
	if err := exec.Command(cmd, args...).Start(); err != nil {
		fmt.Printf("%s   open: %s%s\n", cDim, url, cReset)
	}
}

// printGroqTotal shows the run's cumulative Groq usage and rupee-equivalent.
func printGroqTotal() {
	in, out, calls := groqUsage()
	if calls == 0 {
		return
	}
	fmt.Printf("%s─ Groq this run: %d calls · %d in + %d out = %d tokens ≈ ₹%.4f (free tier: you paid ₹0)%s\n",
		cDim, calls, in, out, in+out, groqINR(in, out), cReset)
}

func err2s(err error) string {
	if err == nil {
		return ""
	}
	return " " + err.Error()
}
