package main

import (
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Gem scoring weights. Emotion (Groq reading real comments) leads, because raw
// popularity is a poor proxy for "unskippable". Everything is normalised to 0..1.
const (
	wEmotion     = 0.30
	wPassion     = 0.24 // comment density — people compelled to speak
	wEngagement  = 0.18 // like ratio — quality, harder to game than views
	wHeat        = 0.16 // sustained "most replayed" intensity
	wElaboration = 0.12 // duration → room for manodharma (alapana/neraval/swaram)
)

var (
	liveRe = regexp.MustCompile(`(?i)\b(live|sabha|academy|festival|concert|kutcheri|katcheri)\b`)
	yearRe = regexp.MustCompile(`\b(19|20)\d{2}\b`)
	tsRe   = regexp.MustCompile(`\b(\d{1,2}):([0-5]\d)\b`)
)

// computeMetrics fills the metric sub-scores and the "start here" timestamp.
// (Emotion is scored separately by Groq and combined in finalizeGem.)
func computeMetrics(r *Recording) {
	views := float64(maxI64(r.Views, 1))

	likeRatio := float64(r.Likes) / views
	commentRatio := float64(r.CommentCnt) / views

	r.Breakdown.Engagement = clamp01(likeRatio / 0.045)     // ~4.5% likes = excellent
	r.Breakdown.Passion = clamp01(commentRatio / 0.0015)    // comment density
	r.Breakdown.Elaboration = clamp01(float64(r.Duration) / 1500) // ~25 min = full
	r.Breakdown.Heat = heatScore(r.Heatmap)

	r.Breakdown.LiveBonus = 1.0
	if liveRe.MatchString(r.Title) || yearRe.MatchString(r.Title) {
		r.Breakdown.LiveBonus = 1.15
	}
	r.Breakdown.Hidden = hiddenBonus(r.Views)

	r.MagicTS = magicTimestamp(r)
}

// finalizeGem combines metric sub-scores with the Groq emotion score, lifted by
// the live and hidden-gem bonuses.
func finalizeGem(r *Recording) {
	r.Breakdown.Emotion = r.EmotionScore
	base := wEmotion*r.Breakdown.Emotion +
		wPassion*r.Breakdown.Passion +
		wEngagement*r.Breakdown.Engagement +
		wHeat*r.Breakdown.Heat +
		wElaboration*r.Breakdown.Elaboration
	r.GemScore = clamp01(base * r.Breakdown.LiveBonus * r.Breakdown.Hidden)
}

// metricScore is the emotion-free score used to CHOOSE which candidates deserve
// the expensive comment/emotion pass. It is deliberately view-independent (ratios,
// heat, elaboration) and lifted by the hidden-gem bonus, so buried renditions with
// strong signals rise above popular-but-lukewarm ones — that's how we find gems.
func metricScore(r *Recording) float64 {
	const norm = wPassion + wEngagement + wHeat + wElaboration
	base := (wPassion*r.Breakdown.Passion +
		wEngagement*r.Breakdown.Engagement +
		wHeat*r.Breakdown.Heat +
		wElaboration*r.Breakdown.Elaboration) / norm
	return clamp01(base * r.Breakdown.LiveBonus * r.Breakdown.Hidden)
}

// hiddenBonus rewards under-exposed renditions: ~1.35 at a few thousand views,
// tapering to 1.0 by ~1M. It only ever multiplies an already quality-gated score,
// so it surfaces buried gems without rewarding low-quality obscurity.
func hiddenBonus(views int64) float64 {
	if views < 1 {
		views = 1
	}
	exposure := clamp01((math.Log10(float64(views)) - 3) / 3) // 1e3→0 … 1e6→1
	return 1.0 + 0.35*(1-exposure)
}

// heatScore = mean of the top replay values, ignoring the intro spike (always ~1.0).
func heatScore(hm []HeatPoint) float64 {
	if len(hm) == 0 {
		return 0.45 // neutral when YouTube exposes no heatmap
	}
	var vals []float64
	for _, h := range hm {
		if h.Start < 30 { // skip the guaranteed intro replay bump
			continue
		}
		vals = append(vals, h.Value)
	}
	if len(vals) == 0 {
		return 0.45
	}
	sort.Sort(sort.Reverse(sort.Float64Slice(vals)))
	n := 10
	if len(vals) < n {
		n = len(vals)
	}
	sum := 0.0
	for i := 0; i < n; i++ {
		sum += vals[i]
	}
	return clamp01(sum / float64(n))
}

// magicTimestamp finds where to start listening: the strongest replay spike past
// the intro, else the most-cited timestamp in comments.
func magicTimestamp(r *Recording) int {
	best, bestVal := 0, -1.0
	for _, h := range r.Heatmap {
		if h.Start < 45 {
			continue
		}
		if h.Value > bestVal {
			bestVal, best = h.Value, int(h.Start)
		}
	}
	if best > 0 {
		return best
	}
	return topCommentTimestamp(r.Comments)
}

// topCommentTimestamp returns the most frequently mentioned mm:ss in comments.
func topCommentTimestamp(comments []Comment) int {
	freq := map[int]int{}
	for _, c := range comments {
		for _, m := range tsRe.FindAllStringSubmatch(c.Text, -1) {
			mm, _ := strconv.Atoi(m[1])
			ss, _ := strconv.Atoi(m[2])
			secs := mm*60 + ss
			if secs > 30 { // ignore trivial early stamps
				freq[secs]++
			}
		}
	}
	best, bestN := 0, 0
	for secs, n := range freq {
		if n > bestN {
			bestN, best = n, secs
		}
	}
	return best
}

// looksLikeGoodCandidate filters obvious junk before we spend a full extraction.
func looksLikeGoodCandidate(c Candidate, minDur int) bool {
	if c.Duration > 0 && c.Duration < minDur {
		return false
	}
	t := strings.ToLower(c.Title)
	for _, bad := range []string{"private video", "deleted video", "#shorts", "whatsapp status", "trailer", "teaser"} {
		if strings.Contains(t, bad) {
			return false
		}
	}
	return true
}

func clamp01(x float64) float64 { return math.Max(0, math.Min(1, x)) }

func maxI64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
