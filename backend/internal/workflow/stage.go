package workflow

import (
	"fmt"
	"sync"

	"content-creator-imm/internal/service"
)

// ErrWaitingHuman is returned when a human stage is waiting for user input.
var ErrWaitingHuman = fmt.Errorf("waiting for human input")

// ExecuteParallelStage runs all workers concurrently, then optionally synthesizes results.
func ExecuteParallelStage(ctx *WorkflowContext, stage StageDef, sse SSEWriter) (*StageOutput, error) {
	sse.SendStageStart(stage.ID, stage.DisplayName, stage.Type)

	outputs := make([]WorkerOutput, len(stage.Workers))
	errs := make([]error, len(stage.Workers))

	var wg sync.WaitGroup
	for i, wd := range stage.Workers {
		wg.Add(1)
		go func(idx int, w WorkerDef) {
			defer wg.Done()
			input := BuildWorkerInput(ctx, w)
			out, err := RunWorker(w, input, stage.ID, sse)
			outputs[idx] = out
			errs[idx] = err
		}(i, wd)
	}
	wg.Wait()

	// Check for errors
	for i, err := range errs {
		if err != nil {
			sse.SendStageDone(stage.ID)
			return nil, fmt.Errorf("worker %s: %w", stage.Workers[i].Name, err)
		}
	}

	stageOut := &StageOutput{
		StageID: stage.ID,
		Workers: outputs,
	}

	// Run synth if defined
	if stage.SynthDef != nil {
		synthInput := BuildSynthInput(ctx, *stage.SynthDef, stage.ID, outputs)
		sse.SendSynthStart(stage.ID)

		synthContent, err := service.StreamClaude(synthInput.SystemPrompt, synthInput.UserPrompt, func(token string) bool {
			sse.SendSynthToken(token)
			return true
		})
		if err != nil {
			sse.SendSynthDone(stage.ID)
			sse.SendStageDone(stage.ID)
			return nil, fmt.Errorf("synth %s: %w", stage.SynthDef.Name, err)
		}

		sse.SendSynthDone(stage.ID)
		stageOut.Summary = synthContent
	}

	sse.SendStageDone(stage.ID)
	return stageOut, nil
}

// ExecuteSerialStage runs a single worker. Uses non-streaming if maxTokens <= 256.
func ExecuteSerialStage(ctx *WorkflowContext, stage StageDef, sse SSEWriter) (*StageOutput, error) {
	if len(stage.Workers) == 0 {
		return nil, fmt.Errorf("serial stage %s has no workers", stage.ID)
	}

	sse.SendStageStart(stage.ID, stage.DisplayName, stage.Type)

	wd := stage.Workers[0]
	input := BuildWorkerInput(ctx, wd)

	var out WorkerOutput
	var err error

	if wd.MaxTokens > 0 && wd.MaxTokens <= 256 {
		out, err = RunWorkerNonStream(wd, input, stage.ID, sse)
	} else {
		out, err = RunWorker(wd, input, stage.ID, sse)
	}

	if err != nil {
		sse.SendStageDone(stage.ID)
		return nil, fmt.Errorf("worker %s: %w", wd.Name, err)
	}

	stageOut := &StageOutput{
		StageID: stage.ID,
		Workers: []WorkerOutput{out},
		Summary: out.Content,
	}

	sse.SendStageDone(stage.ID)
	return stageOut, nil
}

// ExecuteHumanStage sends outline and action options via SSE, then returns ErrWaitingHuman.
func ExecuteHumanStage(ctx *WorkflowContext, stage StageDef, sse SSEWriter) error {
	sse.SendStageStart(stage.ID, stage.DisplayName, stage.Type)

	// Find the last stage output's Summary and parse outline from it
	var lastSummary string
	for _, so := range ctx.StageOutputs {
		lastSummary = so.Summary
	}

	if lastSummary != "" {
		outlineData, _ := service.ParseOutlineFromAnalysis(lastSummary)
		if outlineData != nil {
			sse.SendOutline(outlineData)
		}
	}

	sse.SendAction(stage.HumanPrompt, stage.Options)
	return ErrWaitingHuman
}
