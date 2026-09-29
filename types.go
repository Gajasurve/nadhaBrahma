package main

// Candidate is a lightweight search hit (from the fast flat search).
type Candidate struct {
	VideoID  string
	Title    string
	Views    int64
	Duration int // seconds
}

// Comment is one YouTube comment we pulled for emotion scoring.
type Comment struct {
	Text  string `json:"text"`
	Likes int    `json:"likes"`
}

// HeatPoint is one slice of YouTube's "most replayed" graph.
type HeatPoint struct {
	Start float64 `json:"start"` // seconds
	Value float64 `json:"value"` // 0..1 replay intensity
}

// Recording is a fully-scored rendition — the unit we cache and rank.
type Recording struct {
	VideoID    string      `json:"video_id"`
	Kriti      string      `json:"kriti"`
	Deity      string      `json:"deity"`
	Title      string      `json:"title"`
	Artist     string      `json:"artist"` // channel/uploader
	Views      int64       `json:"views"`
	Likes      int64       `json:"likes"`
	CommentCnt int64       `json:"comment_cnt"`
	Duration   int         `json:"duration"` // seconds
	UploadDate string      `json:"upload_date"`
	Heatmap    []HeatPoint `json:"heatmap"`
	Comments   []Comment   `json:"comments"`

	// Derived / scored fields
	EmotionScore float64      `json:"emotion_score"` // 0..1 from Groq comment reading
	EmotionNote  string       `json:"emotion_note"`  // short Groq rationale
	GemScore     float64      `json:"gem_score"`     // 0..1 final
	MagicTS      int          `json:"magic_ts"`      // seconds — where to start listening
	Breakdown    GemBreakdown `json:"breakdown"`
}

// GemBreakdown records each sub-score so the CLI can explain "why this one".
type GemBreakdown struct {
	Passion     float64 `json:"passion"`     // comment density
	Engagement  float64 `json:"engagement"`  // like ratio
	Heat        float64 `json:"heat"`        // sustained replay intensity
	Elaboration float64 `json:"elaboration"` // duration → manodharma room
	Emotion     float64 `json:"emotion"`     // Groq
	LiveBonus   float64 `json:"live_bonus"`
	Hidden      float64 `json:"hidden"` // hidden-gem bonus (rewards low exposure)
}
