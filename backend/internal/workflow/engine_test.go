package workflow

import (
	"testing"
)

type MockSSEWriter struct {
	Events []map[string]any
}

func (m *MockSSEWriter) SendStageStart(stageID, stageName string, stageType StageType) {
	m.Events = append(m.Events, map[string]any{"type": "stage_start", "stage_id": stageID})
}
func (m *MockSSEWriter) SendStageDone(stageID string) {
	m.Events = append(m.Events, map[string]any{"type": "stage_done", "stage_id": stageID})
}
func (m *MockSSEWriter) SendWorkerStart(stageID, workerName, workerDisplay string) {
	m.Events = append(m.Events, map[string]any{"type": "worker_start", "worker_name": workerName})
}
func (m *MockSSEWriter) SendWorkerToken(workerName, content string) {}
func (m *MockSSEWriter) SendWorkerDone(workerName string) {
	m.Events = append(m.Events, map[string]any{"type": "worker_done", "worker_name": workerName})
}
func (m *MockSSEWriter) SendSynthStart(stageID string)        {}
func (m *MockSSEWriter) SendSynthToken(content string)         {}
func (m *MockSSEWriter) SendSynthDone(stageID string)          {}
func (m *MockSSEWriter) SendStep(step int, name string)        {}
func (m *MockSSEWriter) SendInfo(content string)               {}
func (m *MockSSEWriter) SendOutline(data any)                  {}
func (m *MockSSEWriter) SendAction(_ string, options []string) {}
func (m *MockSSEWriter) SendSimilarity(data any)               {}
func (m *MockSSEWriter) SendComplete(scriptID uint)            {}
func (m *MockSSEWriter) SendError(message string)              {}

func TestContextAndInterpolationIntegration(t *testing.T) {
	ctx := NewWorkflowContext(SharedContext{
		OriginalText: "test text",
		UserStyle:    "casual",
		WorkflowMeta: map[string]any{"max_similarity": 30},
	})

	wd := WorkerDef{
		SystemPrompt:  "You are a test worker",
		UserPromptTpl: "Text: {{original_text}}, Style: {{user_style}}, Max: {{workflow.meta.max_similarity}}",
		MaxTokens:     100,
		Temperature:   0.5,
	}

	input := BuildWorkerInput(ctx, wd)

	if input.UserPrompt != "Text: test text, Style: casual, Max: 30" {
		t.Errorf("unexpected prompt: %s", input.UserPrompt)
	}
	if input.SystemPrompt != "You are a test worker" {
		t.Errorf("unexpected system prompt: %s", input.SystemPrompt)
	}
	if input.MaxTokens != 100 {
		t.Errorf("unexpected max tokens: %d", input.MaxTokens)
	}
}

func TestNewWorkflowContextInitialized(t *testing.T) {
	ctx := NewWorkflowContext(SharedContext{OriginalText: "hello"})
	if ctx.StageOutputs == nil {
		t.Error("StageOutputs should be initialized")
	}
	if ctx.HumanInputs == nil {
		t.Error("HumanInputs should be initialized")
	}
}

func TestMockSSEWriterImplementsInterface(t *testing.T) {
	var _ SSEWriter = &MockSSEWriter{}
}
