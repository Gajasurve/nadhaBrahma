package main

import (
	"fmt"
	"strings"
)

// ANSI helpers (kept minimal; terminals handle these fine).
const (
	cReset  = "\033[0m"
	cBold   = "\033[1m"
	cDim    = "\033[2m"
	cSaffron = "\033[38;5;208m"
	cGold   = "\033[38;5;220m"
	cCyan   = "\033[36m"
	cGreen  = "\033[32m"
)

const rule = "────────────────────────────────────────────────────────────"

// renderResult prints the kriti's meaning, the well-known canonical version as an
// anchor, then the gem we surfaced beyond it, then the other renditions considered.
func renderResult(seed SeedKriti, gem, known *Recording, h Hook, peakRatio float64, alts []*Recording) {
	raga := seed.Raga
	if raga == "" {
		raga = "—"
	}

	fmt.Printf("\n%s%s%s\n", cDim, rule, cReset)
	fmt.Printf("%s%s♪  %s%s  %s—  %s%s\n", cBold, cSaffron, seed.Title, cReset, cDim, seed.Composer, cReset)
	fmt.Printf("   %s%s · Raga %s · %s%s\n", cDim, seed.Deity, raga, seed.Language, cReset)

	if h.Meaning != "" {
		fmt.Printf("\n   %s\n", wrap(h.Meaning, 3))
	}

	sameAsGem := known != nil && known.VideoID == gem.VideoID

	// ── the known version (anchor) ──
	if known != nil && !sameAsGem {
		fmt.Printf("\n   %s◆ Known version%s %s(the one most people hear)%s\n", cCyan, cReset, cDim, cReset)
		fmt.Printf("     %s%s%s  %s· %s views · %s%s\n",
			cBold, known.Artist, cReset, cDim, humanize(known.Views), dur(known.Duration), cReset)
		fmt.Printf("     %s%s%s\n", cDim, watchURL(known.VideoID, 0), cReset)
	}

	// ── the gem ──
	if sameAsGem {
		fmt.Printf("\n%s✦ The well-known version is also our top pick%s\n", cGold, cReset)
	} else {
		fmt.Printf("\n%s✦ The gem we found%s\n", cGold, cReset)
	}
	fmt.Printf("%s▶  %s%s%s   %s%s views · %s likes · %s%s\n",
		cGold, cBold, gem.Artist, cReset,
		cDim, humanize(gem.Views), humanize(gem.Likes), dur(gem.Duration), cReset)

	if h.Why != "" {
		fmt.Printf("   %sWhy this one:%s %s\n", cCyan, cReset, wrapInline(h.Why, 17))
	}
	fmt.Printf("   %sgem %s%.2f%s%s · %s%s\n", cDim, cGreen, gem.GemScore, cReset, cDim, gem.EmotionNote, cReset)

	if peakRatio >= 1.3 {
		fmt.Printf("   %s✦ personal peak — %.1f× this artist's own average%s\n", cGold, peakRatio, cReset)
	}
	if h.ListenFor != "" {
		fmt.Printf("   %sListen for:%s %s\n", cCyan, cReset, h.ListenFor)
	}

	url := watchURL(gem.VideoID, gem.MagicTS)
	if gem.MagicTS > 0 {
		fmt.Printf("   %sStart at %s%s%s  →  %s%s\n", cDim, cBold, mmss(gem.MagicTS), cReset, url, cReset)
	} else {
		fmt.Printf("   %s→ %s%s\n", cDim, url, cReset)
	}

	renderAlternates(alts)
	fmt.Printf("%s%s%s\n", cDim, rule, cReset)
}

// renderAlternates lists the runner-up renditions with their links, so you can
// compare — the gem is chosen for you, but every version is one keystroke away.
func renderAlternates(alts []*Recording) {
	if len(alts) == 0 {
		return
	}
	fmt.Printf("\n   %sother renditions considered:%s\n", cDim, cReset)
	for _, a := range alts {
		fmt.Printf("     %sgem %.2f%s  %s%s%s  %s· %s views · %s%s\n",
			cGreen, a.GemScore, cReset, cBold, a.Artist, cReset,
			cDim, humanize(a.Views), dur(a.Duration), cReset)
		fmt.Printf("        %s%s%s\n", cDim, watchURL(a.VideoID, a.MagicTS), cReset)
	}
}

// renderBreakdown prints the sub-score explanation ([w] command).
func renderBreakdown(r *Recording) {
	b := r.Breakdown
	fmt.Printf("\n   %swhy this scored %.2f:%s\n", cDim, r.GemScore, cReset)
	fmt.Printf("     emotion (crowd)    %s\n", bar(b.Emotion))
	fmt.Printf("     passion (comments) %s\n", bar(b.Passion))
	fmt.Printf("     engagement (likes) %s\n", bar(b.Engagement))
	fmt.Printf("     heat (replayed)    %s\n", bar(b.Heat))
	fmt.Printf("     elaboration (len)  %s\n", bar(b.Elaboration))
	if b.LiveBonus > 1.0 {
		fmt.Printf("     %slive bonus ×%.2f%s\n", cGold, b.LiveBonus, cReset)
	}
	if b.Hidden > 1.0 {
		fmt.Printf("     %shidden-gem bonus ×%.2f (under-exposed)%s\n", cGold, b.Hidden, cReset)
	}
}

func bar(v float64) string {
	n := int(v*20 + 0.5)
	if n > 20 {
		n = 20
	}
	return fmt.Sprintf("%s%s%s %.2f", cGreen, strings.Repeat("█", n), cReset+cDim, v) + cReset
}

func watchURL(id string, ts int) string {
	if ts > 0 {
		return fmt.Sprintf("https://youtu.be/%s?t=%d", id, ts)
	}
	return "https://youtu.be/" + id
}

func dur(secs int) string {
	if secs <= 0 {
		return "?"
	}
	return mmss(secs)
}

func mmss(secs int) string {
	if secs >= 3600 {
		return fmt.Sprintf("%d:%02d:%02d", secs/3600, (secs%3600)/60, secs%60)
	}
	return fmt.Sprintf("%d:%02d", secs/60, secs%60)
}

func humanize(n int64) string {
	f := float64(n)
	switch {
	case f >= 1e6:
		return fmt.Sprintf("%.1fM", f/1e6)
	case f >= 1e3:
		return fmt.Sprintf("%.1fK", f/1e3)
	default:
		return fmt.Sprintf("%d", n)
	}
}

// wrap indents wrapped lines by `indent` spaces, ~66 chars wide.
func wrap(s string, indent int) string {
	return wrapInline(s, indent)
}

func wrapInline(s string, indent int) string {
	const width = 66
	words := strings.Fields(s)
	var b strings.Builder
	line := 0
	pad := strings.Repeat(" ", indent)
	for i, w := range words {
		if line > 0 && line+len(w)+1 > width {
			b.WriteString("\n" + pad)
			line = 0
		} else if i > 0 && line > 0 {
			b.WriteString(" ")
			line++
		}
		b.WriteString(w)
		line += len(w)
	}
	return b.String()
}
