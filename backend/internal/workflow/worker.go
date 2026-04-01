package workflow

import (
	"time"

	"content-creator-imm/internal/service"
)

// RunWorker executes a single worker's LLM call with streaming and SSE forwarding.
func RunWorker(wd WorkerDef, input WorkerInput, stageID string, sse SSEWriter) (WorkerOutput, error) {
	sse.SendWorkerStart(stageID, wd.Name, wd.DisplayName)
	start := time.Now()

	content, err := service.StreamClaude(input.SystemPrompt, input.UserPrompt, func(token string) bool {
		sse.SendWorkerToken(wd.Name, token)
		return true
	})
	duration := time.Since(start)

	if err != nil {
		sse.SendWorkerDone(wd.Name)
		return WorkerOutput{}, err
	}

	sse.SendWorkerDone(wd.Name)
	return WorkerOutput{
		Name:     wd.Name,
		Content:  content,
		Duration: duration,
	}, nil
}

// RunWorkerNonStream executes a single worker's LLM call without streaming.
func RunWorkerNonStream(wd WorkerDef, input WorkerInput, stageID string, sse SSEWriter) (WorkerOutput, error) {
	sse.SendWorkerStart(stageID, wd.Name, wd.DisplayName)
	start := time.Now()

	content, err := service.CallClaude(input.SystemPrompt, input.UserPrompt, input.MaxTokens)
	duration := time.Since(start)

	if err != nil {
		sse.SendWorkerDone(wd.Name)
		return WorkerOutput{}, err
	}

	sse.SendWorkerDone(wd.Name)
	return WorkerOutput{
		Name:     wd.Name,
		Content:  content,
		Duration: duration,
	}, nil
}
