package service

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"content-creator-imm/config"
	"content-creator-imm/internal/model"
	"content-creator-imm/internal/repository"
)

type SessionState int

const (
	StateIdle      SessionState = iota // waiting for input
	StateAnalyzing                     // running 5-role analysis
	StateAwaiting                      // waiting for user to confirm outline
	StateWriting                       // writing final draft
	StateComplete                      // done
)

type OutlineData struct {
	Elements   []string     `json:"elements"`
	Materials  []string     `json:"materials"`
	Outline    []OutlinePart `json:"outline"`
	Estimated  string       `json:"estimated_similarity"`
	Strategy   string       `json:"strategy"`
}

type OutlinePart struct {
	Part     string `json:"part"`
	Duration string `json:"duration"`
	Content  string `json:"content"`
	Emotion  string `json:"emotion"`
}

type ChatSession struct {
	ID           string
	UserID       uint
	State        SessionState
	OriginalText string
	SourceURL    string
	AnalysisFull string
	OutlineJSON  string
	OutlineData  *OutlineData
	FinalDraft   string
	UserNote     string   // adjustment note from user
	CreatedAt    time.Time
	Mu           sync.Mutex
}

var (
	sessions   = map[string]*ChatSession{}
	sessionsMu sync.RWMutex
)

func GetOrCreateSession(userID uint) *ChatSession {
	sessionsMu.Lock()
	defer sessionsMu.Unlock()
	key := fmt.Sprintf("u%d", userID)
	s, ok := sessions[key]
	if !ok {
		s = &ChatSession{
			ID:        key,
			UserID:    userID,
			State:     StateIdle,
			CreatedAt: time.Now(),
		}
		sessions[key] = s
	}
	return s
}

// ResetSession clears session state for a new conversation.
func ResetSession(userID uint) {
	sessionsMu.Lock()
	defer sessionsMu.Unlock()
	key := fmt.Sprintf("u%d", userID)
	sessions[key] = &ChatSession{
		ID:        key,
		UserID:    userID,
		State:     StateIdle,
		CreatedAt: time.Now(),
	}
}

// ParseOutlineFromAnalysis extracts JSON from the analysis text between markers.
func ParseOutlineFromAnalysis(text string) (*OutlineData, string) {
	start := strings.Index(text, "---OUTLINE_START---")
	end := strings.Index(text, "---OUTLINE_END---")
	if start < 0 || end < 0 || end <= start {
		return nil, ""
	}
	raw := strings.TrimSpace(text[start+len("---OUTLINE_START---") : end])
	var data OutlineData
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return nil, raw
	}
	return &data, raw
}

// SaveScript saves the final script to local file and records metadata in DB.
func SaveScript(userID uint, s *ChatSession, similarityScore, viralScore float64) (*model.Script, error) {
	// Save content to local file
	dir := config.C.StoragePath
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	filename := fmt.Sprintf("%d_%d.md", userID, time.Now().UnixMilli())
	path := filepath.Join(dir, filename)

	content := fmt.Sprintf("# 口播稿\n\n生成时间: %s\n来源: %s\n\n---\n\n%s",
		time.Now().Format("2006-01-02 15:04:05"),
		s.SourceURL,
		s.FinalDraft,
	)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return nil, err
	}

	// Extract title from first line of draft
	title := extractTitle(s.FinalDraft)

	script := &model.Script{
		UserID:          userID,
		Title:           title,
		SourceURL:       s.SourceURL,
		ContentPath:     path,
		SimilarityScore: similarityScore,
		ViralScore:      viralScore,
	}
	if err := repository.CreateScript(script); err != nil {
		return nil, err
	}
	return script, nil
}

func extractTitle(text string) string {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) > 5 && len(line) < 60 {
			// Remove markdown heading markers
			line = strings.TrimLeft(line, "#")
			return strings.TrimSpace(line)
		}
	}
	return "口播稿 " + time.Now().Format("01-02 15:04")
}
