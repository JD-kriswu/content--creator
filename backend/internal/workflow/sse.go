package workflow

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"content-creator-imm/internal/model"
	"content-creator-imm/internal/repository"
)

type SSEWriter interface {
	SendStageStart(stageID, stageName string, stageType StageType)
	SendStageDone(stageID string)
	SendWorkerStart(stageID, workerName, workerDisplay string)
	SendWorkerToken(workerName, content string)
	SendWorkerDone(workerName string)
	SendSynthStart(stageID string)
	SendSynthToken(content string)
	SendSynthDone(stageID string)
	SendStep(step int, name string)
	SendInfo(content string)
	SendOutline(data any)
	SendAction(prompt string, options []string)
	SendSimilarity(data any)
	SendFinalDraft(content string)
	SendComplete(scriptID uint)
	SendError(message string)
}

type GinSSEWriter struct {
	w       io.Writer
	flusher interface{ Flush() }
	mu      sync.Mutex
}

func NewGinSSEWriter(w io.Writer, flusher interface{ Flush() }) *GinSSEWriter {
	return &GinSSEWriter{w: w, flusher: flusher}
}

func (g *GinSSEWriter) send(data any) {
	b, _ := json.Marshal(data)
	g.mu.Lock()
	fmt.Fprintf(g.w, "data: %s\n\n", b)
	g.flusher.Flush()
	g.mu.Unlock()
}

func (g *GinSSEWriter) SendStageStart(stageID, stageName string, stageType StageType) {
	g.send(map[string]any{"type": "stage_start", "stage_id": stageID, "stage_name": stageName, "stage_type": stageType})
}

func (g *GinSSEWriter) SendStageDone(stageID string) {
	g.send(map[string]any{"type": "stage_done", "stage_id": stageID})
}

func (g *GinSSEWriter) SendWorkerStart(stageID, workerName, workerDisplay string) {
	g.send(map[string]any{"type": "worker_start", "stage_id": stageID, "worker_name": workerName, "worker_display": workerDisplay})
}

func (g *GinSSEWriter) SendWorkerToken(workerName, content string) {
	g.send(map[string]any{"type": "worker_token", "worker_name": workerName, "content": content})
}

func (g *GinSSEWriter) SendWorkerDone(workerName string) {
	g.send(map[string]any{"type": "worker_done", "worker_name": workerName})
}

func (g *GinSSEWriter) SendSynthStart(stageID string) {
	g.send(map[string]any{"type": "synth_start", "stage_id": stageID})
}

func (g *GinSSEWriter) SendSynthToken(content string) {
	g.send(map[string]any{"type": "synth_token", "content": content})
}

func (g *GinSSEWriter) SendSynthDone(stageID string) {
	g.send(map[string]any{"type": "synth_done", "stage_id": stageID})
}

func (g *GinSSEWriter) SendStep(step int, name string) {
	g.send(map[string]any{"type": "step", "step": step, "name": name})
}

func (g *GinSSEWriter) SendInfo(content string) {
	g.send(map[string]any{"type": "info", "content": content})
}

func (g *GinSSEWriter) SendOutline(data any) {
	g.send(map[string]any{"type": "outline", "data": data})
}

func (g *GinSSEWriter) SendAction(prompt string, options []string) {
	g.send(map[string]any{"type": "action", "options": options})
}

func (g *GinSSEWriter) SendSimilarity(data any) {
	g.send(map[string]any{"type": "similarity", "data": data})
}

func (g *GinSSEWriter) SendFinalDraft(content string) {
	g.send(map[string]any{"type": "final_draft", "content": content})
}

func (g *GinSSEWriter) SendComplete(scriptID uint) {
	g.send(map[string]any{"type": "complete", "scriptId": scriptID})
}

func (g *GinSSEWriter) SendError(message string) {
	g.send(map[string]any{"type": "error", "message": message})
}

// MessageSavingSSEWriter wraps SSEWriter and saves messages to database
type MessageSavingSSEWriter struct {
	inner    SSEWriter
	convID   uint
	mu       sync.Mutex
	messages []model.Message
}

func NewMessageSavingSSEWriter(inner SSEWriter, convID uint) *MessageSavingSSEWriter {
	return &MessageSavingSSEWriter{inner: inner, convID: convID}
}

func (m *MessageSavingSSEWriter) saveMsg(msgType, content string, dataJSON, optionsJSON string, step int, name string) {
	if m.convID == 0 {
		return
	}
	msg := model.Message{
		ConversationID: m.convID,
		Role:           "assistant",
		Type:           msgType,
		Content:        content,
		DataJSON:       dataJSON,
		OptionsJSON:    optionsJSON,
		Step:           step,
		Name:           name,
	}
	m.mu.Lock()
	m.messages = append(m.messages, msg)
	m.mu.Unlock()
	repository.CreateMessage(&msg)
}

func (m *MessageSavingSSEWriter) SendStageStart(stageID, stageName string, stageType StageType) {
	m.saveMsg("info", "▶ "+stageName, "", "", 0, "")
	m.inner.SendStageStart(stageID, stageName, stageType)
}

func (m *MessageSavingSSEWriter) SendStageDone(stageID string) {
	m.inner.SendStageDone(stageID)
}

func (m *MessageSavingSSEWriter) SendWorkerStart(stageID, workerName, workerDisplay string) {
	m.saveMsg("step", workerDisplay, "", "", 0, "")
	m.inner.SendWorkerStart(stageID, workerName, workerDisplay)
}

func (m *MessageSavingSSEWriter) SendWorkerToken(workerName, content string) {
	m.inner.SendWorkerToken(workerName, content)
}

func (m *MessageSavingSSEWriter) SendWorkerDone(workerName string) {
	m.inner.SendWorkerDone(workerName)
}

func (m *MessageSavingSSEWriter) SendSynthStart(stageID string) {
	m.saveMsg("info", "综合分析...", "", "", 0, "")
	m.inner.SendSynthStart(stageID)
}

func (m *MessageSavingSSEWriter) SendSynthToken(content string) {
	m.inner.SendSynthToken(content)
}

func (m *MessageSavingSSEWriter) SendSynthDone(stageID string) {
	m.inner.SendSynthDone(stageID)
}

func (m *MessageSavingSSEWriter) SendStep(step int, name string) {
	m.saveMsg("step", "", "", "", step, name)
	m.inner.SendStep(step, name)
}

func (m *MessageSavingSSEWriter) SendInfo(content string) {
	m.saveMsg("info", content, "", "", 0, "")
	m.inner.SendInfo(content)
}

func (m *MessageSavingSSEWriter) SendOutline(data any) {
	dataJSON, _ := json.Marshal(data)
	m.saveMsg("outline", "", string(dataJSON), "", 0, "")
	m.inner.SendOutline(data)
}

func (m *MessageSavingSSEWriter) SendAction(prompt string, options []string) {
	optionsJSON, _ := json.Marshal(options)
	m.saveMsg("action", prompt, "", string(optionsJSON), 0, "")
	m.inner.SendAction(prompt, options)
}

func (m *MessageSavingSSEWriter) SendSimilarity(data any) {
	dataJSON, _ := json.Marshal(data)
	m.saveMsg("similarity", "", string(dataJSON), "", 0, "")
	m.inner.SendSimilarity(data)
}

func (m *MessageSavingSSEWriter) SendFinalDraft(content string) {
	// Save the final draft content as a text message for conversation history
	m.saveMsg("text", content, "", "", 0, "")
	m.inner.SendFinalDraft(content)
}

func (m *MessageSavingSSEWriter) SendComplete(scriptID uint) {
	m.inner.SendComplete(scriptID)
}

func (m *MessageSavingSSEWriter) SendError(message string) {
	m.saveMsg("error", message, "", "", 0, "")
	m.inner.SendError(message)
}
