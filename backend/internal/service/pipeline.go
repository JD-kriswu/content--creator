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

// StoredMsg is the persisted format of a single chat message.
type StoredMsg struct {
	Role    string          `json:"role"`              // user / assistant
	Type    string          `json:"type"`              // text/step/info/outline/action/similarity/complete/error
	Content string          `json:"content,omitempty"` // for text/info/error
	Data    json.RawMessage `json:"data,omitempty"`    // for outline/similarity
	Options []string        `json:"options,omitempty"` // for action
	Step    int             `json:"step,omitempty"`
	Name    string          `json:"name,omitempty"`
}

type ChatSession struct {
	ID               string
	UserID           uint
	State            SessionState
	StateChangedAt   time.Time
	ConvID           uint          // current Conversation DB id
	ActiveWorkflowID uint          // non-zero when a workflow is running/paused
	StoredMsgs       []StoredMsg   // accumulated messages for persistence
	OriginalText     string
	SourceURL        string
	AnalysisFull     string
	OutlineJSON      string
	OutlineData      *OutlineData
	FinalDraft       string
	UserNote         string
	CreatedAt        time.Time
	Mu               sync.Mutex
}

func (s *ChatSession) SetState(state SessionState) {
	s.State = state
	s.StateChangedAt = time.Now()
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
			ID:             key,
			UserID:         userID,
			State:          StateIdle,
			StateChangedAt: time.Now(),
			CreatedAt:      time.Now(),
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
		ID:             key,
		UserID:         userID,
		State:          StateIdle,
		StateChangedAt: time.Now(),
		CreatedAt:      time.Now(),
	}
}

// FlushConversation updates conversation state and script_id in DB.
func FlushConversation(sess *ChatSession, state int, scriptID *uint) {
	if sess.ConvID == 0 {
		return
	}
	updates := map[string]interface{}{"state": state}
	if scriptID != nil {
		updates["script_id"] = *scriptID
	}
	_ = repository.UpdateConversationMeta(sess.ConvID, updates)
}

// EnsureConversation creates a Conversation record if one doesn't exist yet.
// Returns the conversation ID.
func EnsureConversation(sess *ChatSession, title string) uint {
	if sess.ConvID != 0 {
		return sess.ConvID
	}
	conv := &model.Conversation{
		UserID: sess.UserID,
		Title:  title,
		State:  0,
	}
	if err := repository.CreateConversation(conv); err == nil {
		sess.ConvID = conv.ID
	}
	return sess.ConvID
}

// AddMsg appends a message to the session's in-memory store.
func (s *ChatSession) AddMsg(msg StoredMsg) {
	s.StoredMsgs = append(s.StoredMsgs, msg)
}

// PersistMsg saves a single message to the Message table immediately.
func PersistMsg(convID uint, msg StoredMsg) {
	if convID == 0 {
		return
	}
	dataStr := ""
	if msg.Data != nil {
		dataStr = string(msg.Data)
	}
	optStr := ""
	if len(msg.Options) > 0 {
		b, _ := json.Marshal(msg.Options)
		optStr = string(b)
	}
	m := &model.Message{
		ConversationID: convID,
		Role:           msg.Role,
		Type:           msg.Type,
		Content:        msg.Content,
		DataJSON:       dataStr,
		OptionsJSON:    optStr,
		Step:           msg.Step,
		Name:           msg.Name,
	}
	_ = repository.CreateMessage(m)
}


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

	cleanDraft := StripQualityCheck(s.FinalDraft)

	content := fmt.Sprintf("# 口播稿\n\n生成时间: %s\n来源: %s\n\n---\n\n%s",
		time.Now().Format("2006-01-02 15:04:05"),
		s.SourceURL,
		cleanDraft,
	)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return nil, err
	}

	// Extract title from first line of clean draft
	title := extractTitle(cleanDraft)

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

	// Link conversation to script and mark completed
	if s.ConvID != 0 {
		FlushConversation(s, 1, &script.ID)
	}

	return script, nil
}

func extractTitle(text string) string {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		line = strings.TrimLeft(line, "#")
		line = strings.TrimSpace(line)
		if len([]rune(line)) >= 6 && len([]rune(line)) <= 30 {
			return line
		}
	}
	// fallback: take first non-empty line, truncate
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			runes := []rune(line)
			if len(runes) > 20 {
				return string(runes[:20]) + "..."
			}
			return line
		}
	}
	return "口播稿 " + time.Now().Format("01-02 15:04")
}

// StripQualityCheck removes the ---QUALITY_CHECK_START--- ... ---QUALITY_CHECK_END--- block.
func StripQualityCheck(text string) string {
	const startMark = "---QUALITY_CHECK_START---"
	idx := strings.Index(text, startMark)
	if idx < 0 {
		return strings.TrimSpace(text)
	}
	return strings.TrimRight(text[:idx], "\n\r ")
}

// SaveScriptFromWorkflow saves a script produced by the workflow engine (no ChatSession dependency).
func SaveScriptFromWorkflow(userID uint, sourceURL, draft string, similarityScore float64) (*model.Script, error) {
	dir := config.C.StoragePath
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	filename := fmt.Sprintf("%d_%d.md", userID, time.Now().UnixMilli())
	path := filepath.Join(dir, filename)

	cleanDraft := StripQualityCheck(draft)

	content := fmt.Sprintf("# 口播稿\n\n生成时间: %s\n来源: %s\n\n---\n\n%s",
		time.Now().Format("2006-01-02 15:04:05"),
		sourceURL,
		cleanDraft,
	)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return nil, err
	}

	title := extractTitle(cleanDraft)

	script := &model.Script{
		UserID:          userID,
		Title:           title,
		SourceURL:       sourceURL,
		ContentPath:     path,
		SimilarityScore: similarityScore,
	}
	if err := repository.CreateScript(script); err != nil {
		return nil, err
	}

	return script, nil
}
