package feishu

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"content-creator-imm/internal/service"
	"content-creator-imm/internal/workflow"
)

// FeishuSSEWriter implements workflow.SSEWriter interface,
// converting SSE events into Feishu Card updates.
type FeishuSSEWriter struct {
	api        *service.FeishuAPI
	chatID     string
	messageID  string
	stageName  string
	content    strings.Builder
	throttleMs int
	lastUpdate time.Time
	mu         sync.Mutex
}

// NewFeishuSSEWriter creates a new FeishuSSEWriter instance.
// throttleMs controls the minimum interval between card updates (in milliseconds).
func NewFeishuSSEWriter(chatID, appID, appSecret string, throttleMs int) *FeishuSSEWriter {
	return &FeishuSSEWriter{
		api:        service.NewFeishuAPI(appID, appSecret),
		chatID:     chatID,
		throttleMs: throttleMs,
	}
}

// Init creates the initial card and returns the message ID.
func (w *FeishuSSEWriter) Init() error {
	card := w.buildCard()
	cardJSON, _ := json.Marshal(card)
	msgID, err := w.api.CreateCard(w.chatID, string(cardJSON))
	if err != nil {
		return err
	}
	w.messageID = msgID
	return nil
}

// --- SSEWriter interface implementation ---

func (w *FeishuSSEWriter) SendStageStart(stageID, stageName string, stageType workflow.StageType) {
	w.mu.Lock()
	w.stageName = stageName
	w.content.Reset()
	w.mu.Unlock()
	w.updateCard()
}

func (w *FeishuSSEWriter) SendStageDone(stageID string) {
	w.forceUpdate()
}

func (w *FeishuSSEWriter) SendWorkerStart(stageID, workerName, workerDisplay string) {
	w.updateCard()
}

func (w *FeishuSSEWriter) SendWorkerToken(workerName, content string) {
	w.mu.Lock()
	w.content.WriteString(content)
	w.mu.Unlock()
	w.updateCard()
}

func (w *FeishuSSEWriter) SendWorkerDone(workerName string) {}

func (w *FeishuSSEWriter) SendSynthStart(stageID string) {}

func (w *FeishuSSEWriter) SendSynthToken(content string) {
	w.mu.Lock()
	w.content.WriteString(content)
	w.mu.Unlock()
	w.updateCard()
}

func (w *FeishuSSEWriter) SendSynthDone(stageID string) {}

func (w *FeishuSSEWriter) SendStep(step int, name string) {}

func (w *FeishuSSEWriter) SendInfo(content string) {}

func (w *FeishuSSEWriter) SendOutline(data any) {
	w.forceUpdate()
}

func (w *FeishuSSEWriter) SendAction(prompt string, options []string) {
	w.forceUpdate()
}

func (w *FeishuSSEWriter) SendSimilarity(data any) {
	w.forceUpdate()
}

func (w *FeishuSSEWriter) SendFinalDraft(content string) {
	w.mu.Lock()
	w.content.Reset()
	w.content.WriteString(content)
	w.mu.Unlock()
	w.forceUpdate()
}

func (w *FeishuSSEWriter) SendComplete(scriptID uint) {
	w.mu.Lock()
	w.stageName = "完成"
	w.mu.Unlock()
	w.forceUpdate()
}

func (w *FeishuSSEWriter) SendError(message string) {
	w.mu.Lock()
	w.stageName = "错误"
	w.content.Reset()
	w.content.WriteString(fmt.Sprintf("❌ %s", message))
	w.mu.Unlock()
	w.forceUpdate()
}

// --- internal methods ---

func (w *FeishuSSEWriter) updateCard() {
	w.mu.Lock()
	now := time.Now()
	if now.Sub(w.lastUpdate) < time.Duration(w.throttleMs)*time.Millisecond {
		w.mu.Unlock()
		return
	}
	w.mu.Unlock()
	w.forceUpdate()
}

func (w *FeishuSSEWriter) forceUpdate() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.messageID == "" {
		return
	}

	card := w.buildCard()
	cardJSON, _ := json.Marshal(card)
	w.api.UpdateCard(w.messageID, string(cardJSON))
	w.lastUpdate = time.Now()
}

func (w *FeishuSSEWriter) buildCard() Card {
	template := "blue"
	title := "口播稿创作"
	if w.stageName == "完成" {
		template = "green"
		title = "创作完成"
	} else if w.stageName == "错误" {
		template = "red"
		title = "创作失败"
	} else if w.stageName != "" {
		title = fmt.Sprintf("口播稿创作 - %s", w.stageName)
	}

	elements := []CardElement{}
	if w.content.Len() > 0 {
		text := w.content.String()
		if len(text) > 4000 {
			text = text[:4000] + "..."
		}
		elements = append(elements, CardElement{
			Tag:  "div",
			Text: &CardText{Tag: "lark_md", Content: text},
		})
	}

	return Card{
		Config: CardConfig{WideScreenMode: true},
		Header: CardHeader{
			Title:    CardText{Tag: "plain_text", Content: title},
			Template: template,
		},
		Elements: elements,
	}
}