package workflow

import (
	"testing"
)

func TestInterpolateBasicVars(t *testing.T) {
	vars := map[string]string{
		"original_text": "Hello world",
		"user_style":    "口语化",
	}
	tpl := "原文：{{original_text}}\n风格：{{user_style}}"
	got := interpolate(tpl, vars)
	want := "原文：Hello world\n风格：口语化"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestInterpolateStageVars(t *testing.T) {
	vars := map[string]string{
		"stage.research.summary":                     "汇总结果",
		"stage.research.worker.viral_decoder.output": "解构输出",
	}
	tpl := "汇总：{{stage.research.summary}}\n解构：{{stage.research.worker.viral_decoder.output}}"
	got := interpolate(tpl, vars)
	want := "汇总：汇总结果\n解构：解构输出"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestInterpolateMissingVar(t *testing.T) {
	vars := map[string]string{}
	tpl := "文本：{{original_text}}"
	got := interpolate(tpl, vars)
	want := "文本："
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestBuildWorkerInput(t *testing.T) {
	ctx := NewWorkflowContext(SharedContext{
		OriginalText: "测试原文",
		UserStyle:    "幽默风格",
		WorkflowMeta: map[string]any{"max_similarity": 30},
	})
	ctx.StageOutputs["research"] = &StageOutput{
		StageID: "research",
		Summary: "研究汇总",
		Workers: []WorkerOutput{
			{Name: "viral_decoder", Content: "解构输出"},
		},
	}
	ctx.HumanInputs["confirm_outline"] = "确认，开始撰写"

	wd := WorkerDef{
		SystemPrompt:  "你是专家",
		UserPromptTpl: "原文：{{original_text}}\n汇总：{{stage.research.summary}}\n用户：{{human.confirm_outline.input}}",
		MaxTokens:     2000,
		Temperature:   0.3,
	}

	input := BuildWorkerInput(ctx, wd)
	if input.SystemPrompt != "你是专家" {
		t.Errorf("system prompt mismatch")
	}
	if input.MaxTokens != 2000 {
		t.Errorf("max tokens mismatch")
	}
	want := "原文：测试原文\n汇总：研究汇总\n用户：确认，开始撰写"
	if input.UserPrompt != want {
		t.Errorf("user prompt:\ngot:  %q\nwant: %q", input.UserPrompt, want)
	}
}
