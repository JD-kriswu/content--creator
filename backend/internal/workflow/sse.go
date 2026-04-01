package workflow

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"
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

func (g *GinSSEWriter) SendComplete(scriptID uint) {
	g.send(map[string]any{"type": "complete", "scriptId": scriptID})
}

func (g *GinSSEWriter) SendError(message string) {
	g.send(map[string]any{"type": "error", "message": message})
}
