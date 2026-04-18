package workflow

import (
	"encoding/json"
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
		"original_text":       ctx.Shared.OriginalText,
		"source_url":          ctx.Shared.SourceURL,
		"user_style":          ctx.Shared.UserStyle,
		"course_context":      ctx.Shared.CourseContext,
		"feedback_constraint": ctx.Shared.FeedbackConstraint,
	}

	for k, v := range ctx.Shared.WorkflowMeta {
		vars[fmt.Sprintf("workflow.meta.%s", k)] = fmt.Sprintf("%v", v)
	}

	for stageID, output := range ctx.StageOutputs {
		vars[fmt.Sprintf("stage.%s.summary", stageID)] = output.Summary
		for _, w := range output.Workers {
			vars[fmt.Sprintf("stage.%s.worker.%s.output", stageID, w.Name)] = w.Content
			// Extract JSON fields from worker output
			extractJSONFields(vars, fmt.Sprintf("stage.%s.worker.%s.output", stageID, w.Name), w.Content)
		}
		// Extract JSON fields from summary
		extractJSONFields(vars, fmt.Sprintf("stage.%s.summary", stageID), output.Summary)
	}

	for humanID, input := range ctx.HumanInputs {
		vars[fmt.Sprintf("human.%s.input", humanID)] = input
	}

	return vars
}

// extractJSONFields parses JSON content and adds nested field references to vars.
// e.g., from {"need_material": true} adds "stage.X.summary.need_material" = "true"
func extractJSONFields(vars map[string]string, baseKey, content string) {
	if content == "" {
		return
	}

	// Try to find JSON in the content
	raw := content
	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")
	if start >= 0 && end > start {
		raw = content[start : end+1]
	}

	var data map[string]any
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return // Not valid JSON, skip
	}

	for k, v := range data {
		switch val := v.(type) {
		case bool:
			vars[fmt.Sprintf("%s.%s", baseKey, k)] = fmt.Sprintf("%v", val)
		case string:
			vars[fmt.Sprintf("%s.%s", baseKey, k)] = val
		case float64:
			vars[fmt.Sprintf("%s.%s", baseKey, k)] = fmt.Sprintf("%v", val)
		case map[string]any:
			// Nested object, extract its fields
			for nk, nv := range val {
				switch nval := nv.(type) {
				case bool:
					vars[fmt.Sprintf("%s.%s.%s", baseKey, k, nk)] = fmt.Sprintf("%v", nval)
				case string:
					vars[fmt.Sprintf("%s.%s.%s", baseKey, k, nk)] = nval
				case float64:
					vars[fmt.Sprintf("%s.%s.%s", baseKey, k, nk)] = fmt.Sprintf("%v", nval)
				}
			}
		}
	}
}
