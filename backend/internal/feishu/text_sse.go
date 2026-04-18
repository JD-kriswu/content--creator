package feishu

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"content-creator-imm/internal/service"
	"content-creator-imm/internal/workflow"
)

// FeishuTextSSEWriter implements workflow.SSEWriter interface,
// sending progress updates as text messages to Feishu chat.
type FeishuTextSSEWriter struct {
	api          *service.FeishuAPI
	chatID       string
	lastSent     strings.Builder
	stageName    string
	throttleMs   int
	lastUpdate   time.Time
	pendingLines []string
	mu           sync.Mutex
}

// NewFeishuTextSSEWriter creates a new text-based SSE writer.
func NewFeishuTextSSEWriter(chatID, appID, appSecret string, throttleMs int) *FeishuTextSSEWriter {
	return &FeishuTextSSEWriter{
		api:        service.NewFeishuAPI(appID, appSecret),
		chatID:     chatID,
		throttleMs: throttleMs,
	}
}

// --- SSEWriter interface implementation ---

func (w *FeishuTextSSEWriter) SendStageStart(stageID, stageName string, stageType workflow.StageType) {
	w.mu.Lock()
	w.stageName = stageName
	w.pendingLines = append(w.pendingLines, fmt.Sprintf("▶ 开始：%s", stageName))
	w.mu.Unlock()
	w.flush()
}

func (w *FeishuTextSSEWriter) SendStageDone(stageID string) {
	w.mu.Lock()
	w.pendingLines = append(w.pendingLines, fmt.Sprintf("✓ %s 完成", w.stageName))
	w.mu.Unlock()
	w.flush()
}

func (w *FeishuTextSSEWriter) SendWorkerStart(stageID, workerName, workerDisplay string) {
	w.mu.Lock()
	w.pendingLines = append(w.pendingLines, fmt.Sprintf("  • %s", workerDisplay))
	w.mu.Unlock()
	w.flush()
}

func (w *FeishuTextSSEWriter) SendWorkerToken(workerName, content string) {
	w.mu.Lock()
	w.lastSent.WriteString(content)
	// Send content updates periodically
	if len(w.lastSent.String()) > 200 {
		n := len(w.lastSent.String())
	if n > 200 {
		n = 200
	}
	w.pendingLines = append(w.pendingLines, w.lastSent.String()[:n])
		w.lastSent.Reset()
	}
	w.mu.Unlock()
	w.flush()
}

func (w *FeishuTextSSEWriter) SendWorkerDone(workerName string) {
	// Send remaining content
	w.mu.Lock()
	if w.lastSent.Len() > 0 {
		text := w.lastSent.String()
		if len(text) > 4000 {
			text = text[:4000] + "..."
		}
		w.pendingLines = append(w.pendingLines, text)
		w.lastSent.Reset()
	}
	w.mu.Unlock()
	w.flush()
}

func (w *FeishuTextSSEWriter) SendSynthStart(stageID string) {}

func (w *FeishuTextSSEWriter) SendSynthToken(content string) {
	w.SendWorkerToken("synth", content)
}

func (w *FeishuTextSSEWriter) SendSynthDone(stageID string) {
	w.SendWorkerDone("synth")
}

func (w *FeishuTextSSEWriter) SendStep(step int, name string) {
	w.mu.Lock()
	w.pendingLines = append(w.pendingLines, fmt.Sprintf("步骤 %d：%s", step, name))
	w.mu.Unlock()
	w.flush()
}

func (w *FeishuTextSSEWriter) SendInfo(content string) {
	w.mu.Lock()
	w.pendingLines = append(w.pendingLines, fmt.Sprintf("ℹ️ %s", content))
	w.mu.Unlock()
	w.flush()
}

func (w *FeishuTextSSEWriter) SendOutline(data any) {
	w.mu.Lock()
	w.pendingLines = append(w.pendingLines, "📝 已生成大纲，等待您确认...")
	w.mu.Unlock()
	w.flush()
}

func (w *FeishuTextSSEWriter) SendAction(prompt string, options []string) {
	w.mu.Lock()
	var msg string
	msg = prompt + "\n请回复数字选择：\n"
	for i, opt := range options {
		msg += fmt.Sprintf("%d. %s\n", i+1, opt)
	}
	w.pendingLines = append(w.pendingLines, msg)
	w.mu.Unlock()
	w.flush()
}

func (w *FeishuTextSSEWriter) SendSimilarity(data any) {
	w.mu.Lock()
	w.pendingLines = append(w.pendingLines, "📊 相似度检测完成")
	w.mu.Unlock()
	w.flush()
}

func (w *FeishuTextSSEWriter) SendFinalDraft(content string) {
	w.mu.Lock()
	// Truncate if too long for a single message
	text := content
	if len(text) > 4000 {
		text = text[:4000] + "...\n\n(内容过长，已截断)"
	}
	w.pendingLines = append(w.pendingLines, fmt.Sprintf("📄 最终稿件：\n\n%s", text))
	w.mu.Unlock()
	w.flush()
}

func (w *FeishuTextSSEWriter) SendComplete(scriptID uint) {
	w.mu.Lock()
	w.pendingLines = append(w.pendingLines, "✅ 口播稿创作完成！已保存到您的账户。")
	w.mu.Unlock()
	w.flush()
}

func (w *FeishuTextSSEWriter) SendError(message string) {
	w.mu.Lock()
	w.pendingLines = append(w.pendingLines, fmt.Sprintf("❌ 错误：%s", message))
	w.mu.Unlock()
	w.flush()
}

// --- internal methods ---

func (w *FeishuTextSSEWriter) flush() {
	w.mu.Lock()
	defer w.mu.Unlock()

	now := time.Now()
	if now.Sub(w.lastUpdate) < time.Duration(w.throttleMs)*time.Millisecond {
		return
	}

	if len(w.pendingLines) == 0 {
		return
	}

	// Combine pending lines into one message
	text := strings.Join(w.pendingLines, "\n")
	w.pendingLines = nil

	// Send text message
	_, err := w.api.SendText(w.chatID, text)
	if err != nil {
		// Log error but don't fail the workflow
		fmt.Printf("[FeishuTextSSE] send error: %v\n", err)
	}

	w.lastUpdate = now
}