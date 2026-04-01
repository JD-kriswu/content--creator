package workflow

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"content-creator-imm/internal/model"
	"content-creator-imm/internal/repository"
	"content-creator-imm/internal/service"
)

// Engine orchestrates a workflow execution: loads the definition, runs stages,
// persists progress, and handles human-in-the-loop resume.
type Engine struct {
	loader     *Loader
	sse        SSEWriter
	def        *WorkflowDef
	ctx        *WorkflowContext
	input      WorkflowInput
	workflowID uint
}

// NewEngine creates a new workflow engine.
func NewEngine(loader *Loader, sse SSEWriter) *Engine {
	return &Engine{
		loader: loader,
		sse:    sse,
	}
}

// WorkflowID returns the database ID of the current workflow.
func (e *Engine) WorkflowID() uint {
	return e.workflowID
}

// Start loads a workflow definition, creates a DB record, and begins execution.
func (e *Engine) Start(workflowType string, input WorkflowInput) error {
	def, err := e.loader.Load(workflowType)
	if err != nil {
		return fmt.Errorf("load workflow %s: %w", workflowType, err)
	}
	e.def = def
	e.input = input

	shared := SharedContext{
		OriginalText: input.Text,
		SourceURL:    input.SourceURL,
		UserStyle:    input.UserStyle,
		WorkflowMeta: def.Meta,
	}
	e.ctx = NewWorkflowContext(shared)

	// Create DB workflow record
	inputJSON, _ := json.Marshal(input)
	wf := &model.Workflow{
		UserID:    input.UserID,
		Type:      workflowType,
		Status:    "running",
		InputJSON: string(inputJSON),
	}
	if err := repository.CreateWorkflow(wf); err != nil {
		return fmt.Errorf("create workflow record: %w", err)
	}
	e.workflowID = wf.ID

	return e.runStages(0)
}

// Resume restores a paused workflow and continues from the appropriate stage.
func (e *Engine) Resume(workflowID uint, humanInput string) error {
	wf, err := repository.GetWorkflow(workflowID)
	if err != nil {
		return fmt.Errorf("get workflow %d: %w", workflowID, err)
	}
	if wf.Status != "paused" {
		return fmt.Errorf("workflow %d is not paused (status=%s)", workflowID, wf.Status)
	}
	e.workflowID = workflowID

	// Restore input
	if err := json.Unmarshal([]byte(wf.InputJSON), &e.input); err != nil {
		return fmt.Errorf("restore input: %w", err)
	}

	// Reload workflow definition
	def, err := e.loader.Load(wf.Type)
	if err != nil {
		return fmt.Errorf("load workflow %s: %w", wf.Type, err)
	}
	e.def = def

	// Restore context from DB
	if wf.ContextJSON != "" {
		var shared SharedContext
		if err := json.Unmarshal([]byte(wf.ContextJSON), &shared); err != nil {
			return fmt.Errorf("restore context: %w", err)
		}
		e.ctx = NewWorkflowContext(shared)
	} else {
		shared := SharedContext{
			OriginalText: e.input.Text,
			SourceURL:    e.input.SourceURL,
			UserStyle:    e.input.UserStyle,
			WorkflowMeta: def.Meta,
		}
		e.ctx = NewWorkflowContext(shared)
	}

	// Restore stage outputs from DB
	if err := e.restoreStageOutputs(); err != nil {
		return fmt.Errorf("restore stage outputs: %w", err)
	}

	// Determine resume point based on human input
	resumeIdx, humanStageID := e.resolveResumeStage(humanInput)

	// Store the human input
	if humanStageID != "" {
		e.ctx.HumanInputs[humanStageID] = humanInput
	}

	// Update workflow status back to running
	wf.Status = "running"
	_ = repository.UpdateWorkflow(wf)

	return e.runStages(resumeIdx)
}

// runStages iterates through stages starting from startIdx.
func (e *Engine) runStages(startIdx int) error {
	for i := startIdx; i < len(e.def.Stages); i++ {
		stage := e.def.Stages[i]

		switch stage.Type {
		case StageParallel:
			out, err := ExecuteParallelStage(e.ctx, stage, e.sse)
			if err != nil {
				e.handleStageError(stage.ID, err)
				return err
			}
			e.ctx.StageOutputs[stage.ID] = out
			e.persistStageOutput(stage, i, out)
			e.saveCheckpoint()

			// If this is a similarity stage, send similarity data via SSE
			if stage.ID == "similarity" && out.Summary != "" {
				e.sendSimilarityResult(out.Summary)
			}

		case StageSerial:
			out, err := ExecuteSerialStage(e.ctx, stage, e.sse)
			if err != nil {
				e.handleStageError(stage.ID, err)
				return err
			}
			e.ctx.StageOutputs[stage.ID] = out
			e.persistStageOutput(stage, i, out)
			e.saveCheckpoint()

			// If this is a similarity stage, send similarity data via SSE
			if stage.ID == "similarity" && out.Summary != "" {
				e.sendSimilarityResult(out.Summary)
			}

		case StageHuman:
			err := ExecuteHumanStage(e.ctx, stage, e.sse)
			if err == ErrWaitingHuman {
				e.pauseWorkflow(stage.ID, i)
				return ErrWaitingHuman
			}
			if err != nil {
				e.handleStageError(stage.ID, err)
				return err
			}
		}
	}

	return e.finish()
}

// finish completes the workflow: for viral_script, saves the script and sends complete SSE.
func (e *Engine) finish() error {
	// Find the draft from the last serial/parallel stage output
	var draft string
	var similarityScore float64

	// Look for the create stage output (the draft)
	if out, ok := e.ctx.StageOutputs["create"]; ok {
		draft = out.Summary
	}
	// Fallback: use the last stage's summary
	if draft == "" {
		for i := len(e.def.Stages) - 1; i >= 0; i-- {
			sid := e.def.Stages[i].ID
			if out, ok := e.ctx.StageOutputs[sid]; ok && out.Summary != "" {
				draft = out.Summary
				break
			}
		}
	}

	// Extract similarity score
	if out, ok := e.ctx.StageOutputs["similarity"]; ok && out.Summary != "" {
		similarityScore = parseSimilarityScore(out.Summary)
	}

	// Save script if we have a draft
	if draft != "" {
		script, err := service.SaveScriptFromWorkflow(
			e.input.UserID,
			e.input.SourceURL,
			draft,
			similarityScore,
		)
		if err != nil {
			e.sse.SendError(fmt.Sprintf("保存稿件失败: %v", err))
			e.markWorkflowFailed(err)
			return err
		}

		// Update workflow record
		e.markWorkflowCompleted(script.ID)
		e.sse.SendComplete(script.ID)
	} else {
		e.markWorkflowCompleted(0)
		e.sse.SendComplete(0)
	}

	return nil
}

// --- Persistence helpers ---

// persistStageOutput saves stage and worker results to the database.
func (e *Engine) persistStageOutput(stage StageDef, sequence int, out *StageOutput) {
	now := time.Now()
	ws := &model.WorkflowStage{
		WorkflowID: e.workflowID,
		StageID:    stage.ID,
		Type:       string(stage.Type),
		Sequence:   sequence,
		Status:     "completed",
		OutputJSON: toJSON(out),
		StartedAt:  &now,
		EndedAt:    &now,
	}
	_ = repository.CreateWorkflowStage(ws)

	for _, wo := range out.Workers {
		ww := &model.WorkflowWorker{
			StageID:    ws.ID,
			WorkflowID: e.workflowID,
			WorkerName: wo.Name,
			Status:     "completed",
			OutputJSON: wo.Content,
			TokensUsed: wo.Tokens,
			DurationMs: wo.Duration.Milliseconds(),
			StartedAt:  &now,
			EndedAt:    &now,
		}
		_ = repository.CreateWorkflowWorker(ww)
	}
}

// saveCheckpoint persists current workflow context to the DB for resume.
func (e *Engine) saveCheckpoint() {
	contextJSON, _ := json.Marshal(e.ctx.Shared)
	wf, err := repository.GetWorkflow(e.workflowID)
	if err != nil {
		return
	}
	wf.ContextJSON = string(contextJSON)
	_ = repository.UpdateWorkflow(wf)
}

// restoreStageOutputs rebuilds StageOutputs from DB records.
func (e *Engine) restoreStageOutputs() error {
	stages, err := repository.GetWorkflowStages(e.workflowID)
	if err != nil {
		return err
	}

	for _, s := range stages {
		if s.Status != "completed" {
			continue
		}

		// Try to restore from OutputJSON first
		if s.OutputJSON != "" {
			var out StageOutput
			if err := json.Unmarshal([]byte(s.OutputJSON), &out); err == nil {
				e.ctx.StageOutputs[s.StageID] = &out
				continue
			}
		}

		// Fallback: rebuild from workers
		workers, err := repository.GetWorkflowWorkersByStage(s.ID)
		if err != nil {
			continue
		}

		out := &StageOutput{StageID: s.StageID}
		for _, w := range workers {
			out.Workers = append(out.Workers, WorkerOutput{
				Name:     w.WorkerName,
				Content:  w.OutputJSON,
				Tokens:   w.TokensUsed,
				Duration: time.Duration(w.DurationMs) * time.Millisecond,
			})
		}
		// Use last worker content as summary if only one worker
		if len(out.Workers) == 1 {
			out.Summary = out.Workers[0].Content
		}
		e.ctx.StageOutputs[s.StageID] = out
	}

	return nil
}

// handleStageError marks the workflow as failed and sends an error SSE event.
func (e *Engine) handleStageError(stageID string, err error) {
	e.sse.SendError(fmt.Sprintf("阶段 %s 失败: %v", stageID, err))
	e.markWorkflowFailed(err)
}

// pauseWorkflow marks the workflow as paused at the given stage.
func (e *Engine) pauseWorkflow(stageID string, stageIdx int) {
	wf, err := repository.GetWorkflow(e.workflowID)
	if err != nil {
		return
	}
	now := time.Now()
	wf.Status = "paused"
	wf.PausedAt = &now

	// Save checkpoint data including which stage we paused at
	checkpoint := map[string]any{
		"paused_stage_id":  stageID,
		"paused_stage_idx": stageIdx,
	}
	checkpointJSON, _ := json.Marshal(checkpoint)
	wf.OutputJSON = string(checkpointJSON)

	contextJSON, _ := json.Marshal(e.ctx.Shared)
	wf.ContextJSON = string(contextJSON)

	_ = repository.UpdateWorkflow(wf)
}

func (e *Engine) markWorkflowCompleted(scriptID uint) {
	wf, err := repository.GetWorkflow(e.workflowID)
	if err != nil {
		return
	}
	wf.Status = "completed"
	if scriptID > 0 {
		outputJSON, _ := json.Marshal(map[string]any{"script_id": scriptID})
		wf.OutputJSON = string(outputJSON)
	}
	_ = repository.UpdateWorkflow(wf)
}

func (e *Engine) markWorkflowFailed(stageErr error) {
	wf, err := repository.GetWorkflow(e.workflowID)
	if err != nil {
		return
	}
	wf.Status = "failed"
	wf.Error = stageErr.Error()
	_ = repository.UpdateWorkflow(wf)
}

// resolveResumeStage determines which stage index to resume from based on human input.
// Returns (stageIndex, humanStageID).
func (e *Engine) resolveResumeStage(humanInput string) (int, string) {
	// Find the paused human stage
	var pausedIdx int
	var humanStageID string

	wf, err := repository.GetWorkflow(e.workflowID)
	if err == nil && wf.OutputJSON != "" {
		var checkpoint map[string]any
		if err := json.Unmarshal([]byte(wf.OutputJSON), &checkpoint); err == nil {
			if idx, ok := checkpoint["paused_stage_idx"]; ok {
				if idxFloat, ok := idx.(float64); ok {
					pausedIdx = int(idxFloat)
				}
			}
			if sid, ok := checkpoint["paused_stage_id"]; ok {
				if sidStr, ok := sid.(string); ok {
					humanStageID = sidStr
				}
			}
		}
	}

	// If we couldn't find checkpoint info, scan for the first human stage
	if humanStageID == "" {
		for i, s := range e.def.Stages {
			if s.Type == StageHuman {
				pausedIdx = i
				humanStageID = s.ID
				break
			}
		}
	}

	input := strings.TrimSpace(humanInput)

	// Parse user choice:
	// 1 or 确认 → continue from next stage after human
	// 2 or 调整 → re-run from "create" stage
	// 3 or 更换素材 → re-run from "research" stage
	// 4 or 重新 → full restart from stage 0
	switch {
	case input == "1" || strings.Contains(input, "确认"):
		return pausedIdx + 1, humanStageID

	case input == "2" || strings.Contains(input, "调整"):
		// Find "create" stage
		for i, s := range e.def.Stages {
			if s.ID == "create" {
				return i, humanStageID
			}
		}
		return pausedIdx + 1, humanStageID

	case input == "3" || strings.Contains(input, "更换素材"):
		// Find "research" stage
		for i, s := range e.def.Stages {
			if s.ID == "research" {
				return i, humanStageID
			}
		}
		return 0, humanStageID

	case input == "4" || strings.Contains(input, "重新"):
		return 0, humanStageID

	default:
		// Default: continue from next stage
		return pausedIdx + 1, humanStageID
	}
}

// sendSimilarityResult parses similarity JSON and sends it via SSE.
func (e *Engine) sendSimilarityResult(summary string) {
	var data any
	if err := json.Unmarshal([]byte(summary), &data); err == nil {
		e.sse.SendSimilarity(data)
	} else {
		// Try to extract JSON from the summary
		start := strings.Index(summary, "{")
		end := strings.LastIndex(summary, "}")
		if start >= 0 && end > start {
			if err := json.Unmarshal([]byte(summary[start:end+1]), &data); err == nil {
				e.sse.SendSimilarity(data)
			}
		}
	}
}

// parseSimilarityScore extracts the similarity score from the similarity stage output.
func parseSimilarityScore(summary string) float64 {
	var data map[string]any

	raw := summary
	// Try to find JSON in the summary
	start := strings.Index(summary, "{")
	end := strings.LastIndex(summary, "}")
	if start >= 0 && end > start {
		raw = summary[start : end+1]
	}

	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return 0
	}

	// Try common field names
	for _, key := range []string{"similarity", "similarity_score", "score", "overall_similarity"} {
		if v, ok := data[key]; ok {
			switch val := v.(type) {
			case float64:
				return val
			case string:
				var f float64
				if _, err := fmt.Sscanf(val, "%f", &f); err == nil {
					return f
				}
			}
		}
	}
	return 0
}

// toJSON marshals v to JSON string, returning "" on error.
func toJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}
