package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// ytdlpAvailable reports whether the yt-dlp binary is on PATH.
func ytdlpAvailable() bool {
	_, err := exec.LookPath("yt-dlp")
	return err == nil
}

// searchCandidates runs a fast flat search — metadata only, no per-video extraction.
// This is the cheapest call and gives us enough (views, duration) to screen.
func searchCandidates(ctx context.Context, query string, max int) ([]Candidate, error) {
	cmd := exec.CommandContext(ctx,
		"yt-dlp", fmt.Sprintf("ytsearch%d:%s", max, query),
		"--flat-playlist", "--dump-json",
		"--no-warnings", "--quiet", "--ignore-errors",
	)
	out, err := cmd.Output()
	if err != nil && len(out) == 0 {
		return nil, fmt.Errorf("yt-dlp search failed: %w", err)
	}

	var cands []Candidate
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		var d struct {
			ID       string  `json:"id"`
			Title    string  `json:"title"`
			Views    int64   `json:"view_count"`
			Duration float64 `json:"duration"`
		}
		if json.Unmarshal([]byte(line), &d) != nil || d.ID == "" {
			continue
		}
		cands = append(cands, Candidate{
			VideoID:  d.ID,
			Title:    d.Title,
			Views:    d.Views,
			Duration: int(d.Duration),
		})
	}
	return cands, nil
}

// fetchFull assembles a Recording from two free yt-dlp calls:
//  1. metadata + heatmap (no comments) → gives the TRUE comment_count (needed for
//     passion density; a capped comment fetch would report only the sample size).
//  2. a capped, top-sorted comment sample → for AI emotion scoring.
// Only run on screened candidates.
func fetchFull(ctx context.Context, videoID string, maxComments int) (*Recording, error) {
	r, err := fetchMeta(ctx, videoID)
	if err != nil {
		return nil, err
	}
	// Comments are best-effort: if they fail, we still have a scoreable recording.
	if cs, err := fetchComments(ctx, videoID, maxComments); err == nil {
		r.Comments = cs
	}
	return r, nil
}

// fetchMeta pulls metadata + heatmap only (fast, no comment download).
func fetchMeta(ctx context.Context, videoID string) (*Recording, error) {
	url := "https://www.youtube.com/watch?v=" + videoID
	cmd := exec.CommandContext(ctx,
		"yt-dlp", url,
		"--skip-download", "--dump-single-json",
		"--no-warnings", "--quiet", "--ignore-errors",
	)
	out, err := cmd.Output()
	if err != nil && len(out) == 0 {
		return nil, fmt.Errorf("yt-dlp meta failed: %w", err)
	}

	var d struct {
		ID         string  `json:"id"`
		Title      string  `json:"title"`
		Channel    string  `json:"channel"`
		Uploader   string  `json:"uploader"`
		Views      int64   `json:"view_count"`
		Likes      int64   `json:"like_count"`
		CommentCnt int64   `json:"comment_count"`
		Duration   float64 `json:"duration"`
		UploadDate string  `json:"upload_date"`
		Heatmap    []struct {
			Start float64 `json:"start_time"`
			Value float64 `json:"value"`
		} `json:"heatmap"`
	}
	if err := json.Unmarshal(out, &d); err != nil {
		return nil, fmt.Errorf("yt-dlp meta parse: %w", err)
	}

	artist := d.Channel
	if artist == "" {
		artist = d.Uploader
	}
	r := &Recording{
		VideoID:    d.ID,
		Title:      d.Title,
		Artist:     artist,
		Views:      d.Views,
		Likes:      d.Likes,
		CommentCnt: d.CommentCnt,
		Duration:   int(d.Duration),
		UploadDate: d.UploadDate,
	}
	for _, h := range d.Heatmap {
		r.Heatmap = append(r.Heatmap, HeatPoint{Start: h.Start, Value: h.Value})
	}
	return r, nil
}

// fetchComments pulls up to maxComments top-sorted comments.
func fetchComments(ctx context.Context, videoID string, maxComments int) ([]Comment, error) {
	url := "https://www.youtube.com/watch?v=" + videoID
	cmd := exec.CommandContext(ctx,
		"yt-dlp", url,
		"--skip-download", "--write-comments",
		"--extractor-args", fmt.Sprintf("youtube:comment_sort=top;max_comments=%d,all,0,0", maxComments),
		"--dump-single-json",
		"--no-warnings", "--quiet", "--ignore-errors",
	)
	out, err := cmd.Output()
	if err != nil && len(out) == 0 {
		return nil, fmt.Errorf("yt-dlp comments failed: %w", err)
	}
	var d struct {
		Comments []struct {
			Text  string `json:"text"`
			Likes int    `json:"like_count"`
		} `json:"comments"`
	}
	if err := json.Unmarshal(out, &d); err != nil {
		return nil, fmt.Errorf("yt-dlp comments parse: %w", err)
	}
	var cs []Comment
	for _, c := range d.Comments {
		t := strings.TrimSpace(c.Text)
		if t != "" {
			cs = append(cs, Comment{Text: t, Likes: c.Likes})
		}
	}
	return cs, nil
}

// ctxTimeout is a small helper for per-call deadlines.
func ctxTimeout(parent context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, d)
}
