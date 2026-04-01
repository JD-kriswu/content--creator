package workflow

import (
	"fmt"
	"strings"
)

func interpolate(tpl string, vars map[string]string) string {
	result := tpl
	for key, val := range vars {
		result = strings.ReplaceAll(result, "{{"+key+"}}", val)
	}
	for {
		start := strings.Index(result, "{{")
		if start == -1 {
			break
		}
		end := strings.Index(result[start:], "}}")
		if end == -1 {
			break
		}
		result = result[:start] + result[start+end+2:]
	}
	return result
}

func BuildWorkerInput(ctx *WorkflowContext, wd WorkerDef) WorkerInput {
	vars := buildVarsMap(ctx)
	return WorkerInput{
		SystemPrompt: wd.SystemPrompt,
		UserPrompt:   interpolate(wd.UserPromptTpl, vars),
		MaxTokens:    wd.MaxTokens,
		Temperature:  wd.Temperature,
	}
}

func BuildSynthInput(ctx *WorkflowContext, sd SynthDef, stageID string, workerOutputs []WorkerOutput) WorkerInput {
	vars := buildVarsMap(ctx)
	for _, wo := range workerOutputs {
		key := fmt.Sprintf("stage.%s.worker.%s.output", stageID, wo.Name)
		vars[key] = wo.Content
	}
	return WorkerInput{
		SystemPrompt: sd.SystemPrompt,
		UserPrompt:   interpolate(sd.UserPromptTpl, vars),
		MaxTokens:    sd.MaxTokens,
		Temperature:  sd.Temperature,
	}
}

func buildVarsMap(ctx *WorkflowContext) map[string]string {
	vars := map[string]string{
		"original_text": ctx.Shared.OriginalText,
		"source_url":    ctx.Shared.SourceURL,
		"user_style":    ctx.Shared.UserStyle,
	}

	for k, v := range ctx.Shared.WorkflowMeta {
		vars[fmt.Sprintf("workflow.meta.%s", k)] = fmt.Sprintf("%v", v)
	}

	for stageID, output := range ctx.StageOutputs {
		vars[fmt.Sprintf("stage.%s.summary", stageID)] = output.Summary
		for _, w := range output.Workers {
			vars[fmt.Sprintf("stage.%s.worker.%s.output", stageID, w.Name)] = w.Content
		}
	}

	for humanID, input := range ctx.HumanInputs {
		vars[fmt.Sprintf("human.%s.input", humanID)] = input
	}

	return vars
}
