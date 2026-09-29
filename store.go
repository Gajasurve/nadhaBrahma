package main

import (
	"database/sql"
	"encoding/json"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Store is the local gem cache. Every scored recording lands here so a video is
// never fetched or re-scored twice — the corpus compounds and stays free.
type Store struct {
	db *sql.DB
}

func openStore(path string) (*Store, error) {
	db, err := sql.Open("sqlite3", path+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, err
	}
	s := &Store{db: db}
	if err := s.init(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) init() error {
	_, err := s.db.Exec(`
	CREATE TABLE IF NOT EXISTS recordings (
		video_id     TEXT PRIMARY KEY,
		kriti        TEXT,
		deity        TEXT,
		title        TEXT,
		artist       TEXT,
		views        INTEGER,
		likes        INTEGER,
		comment_cnt  INTEGER,
		duration     INTEGER,
		upload_date  TEXT,
		heatmap      TEXT,
		comments     TEXT,
		emotion      REAL,
		emotion_note TEXT,
		gem_score    REAL,
		magic_ts     INTEGER,
		breakdown    TEXT,
		scored_at    TEXT
	);
	CREATE INDEX IF NOT EXISTS idx_rec_kriti  ON recordings(kriti);
	CREATE INDEX IF NOT EXISTS idx_rec_artist ON recordings(artist);

	CREATE TABLE IF NOT EXISTS history (
		id    INTEGER PRIMARY KEY AUTOINCREMENT,
		kind  TEXT,   -- 'deity' | 'kriti' | 'video'
		value TEXT,
		ts    TEXT
	);
	`)
	return err
}

func (s *Store) Close() error { return s.db.Close() }

// getRecording returns a cached, already-scored rendition if present.
func (s *Store) getRecording(videoID string) (*Recording, bool) {
	row := s.db.QueryRow(`SELECT video_id,kriti,deity,title,artist,views,likes,comment_cnt,
		duration,upload_date,heatmap,comments,emotion,emotion_note,gem_score,magic_ts,breakdown
		FROM recordings WHERE video_id=?`, videoID)

	var r Recording
	var heat, comments, breakdown string
	err := row.Scan(&r.VideoID, &r.Kriti, &r.Deity, &r.Title, &r.Artist, &r.Views, &r.Likes,
		&r.CommentCnt, &r.Duration, &r.UploadDate, &heat, &comments, &r.EmotionScore,
		&r.EmotionNote, &r.GemScore, &r.MagicTS, &breakdown)
	if err != nil {
		return nil, false
	}
	_ = json.Unmarshal([]byte(heat), &r.Heatmap)
	_ = json.Unmarshal([]byte(comments), &r.Comments)
	_ = json.Unmarshal([]byte(breakdown), &r.Breakdown)
	return &r, true
}

func (s *Store) saveRecording(r *Recording) error {
	heat, _ := json.Marshal(r.Heatmap)
	comments, _ := json.Marshal(r.Comments)
	breakdown, _ := json.Marshal(r.Breakdown)
	_, err := s.db.Exec(`INSERT OR REPLACE INTO recordings
		(video_id,kriti,deity,title,artist,views,likes,comment_cnt,duration,upload_date,
		 heatmap,comments,emotion,emotion_note,gem_score,magic_ts,breakdown,scored_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		r.VideoID, r.Kriti, r.Deity, r.Title, r.Artist, r.Views, r.Likes, r.CommentCnt,
		r.Duration, r.UploadDate, string(heat), string(comments), r.EmotionScore,
		r.EmotionNote, r.GemScore, r.MagicTS, string(breakdown),
		time.Now().Format(time.RFC3339))
	return err
}

// artistBaseline returns the mean gem score across an artist's cached recordings,
// and the count. Used to flag a "personal peak" (a rendition well above their norm).
func (s *Store) artistBaseline(artist string) (mean float64, n int) {
	row := s.db.QueryRow(`SELECT AVG(gem_score), COUNT(*) FROM recordings WHERE artist=?`, artist)
	var avg sql.NullFloat64
	_ = row.Scan(&avg, &n)
	if avg.Valid {
		mean = avg.Float64
	}
	return
}

// ── history (anti-repeat) ──

func (s *Store) addHistory(kind, value string) {
	_, _ = s.db.Exec(`INSERT INTO history(kind,value,ts) VALUES(?,?,?)`,
		kind, value, time.Now().Format(time.RFC3339))
}

// recentHistory returns the last n values of a kind (most recent first).
func (s *Store) recentHistory(kind string, n int) map[string]bool {
	out := map[string]bool{}
	rows, err := s.db.Query(`SELECT value FROM history WHERE kind=? ORDER BY id DESC LIMIT ?`, kind, n)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var v string
		if rows.Scan(&v) == nil {
			out[v] = true
		}
	}
	return out
}
