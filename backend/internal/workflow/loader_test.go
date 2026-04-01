package workflow

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoaderLoadsViralScript(t *testing.T) {
	wd, _ := os.Getwd()
	base := filepath.Join(wd, "..", "..", "workflows")
	if _, err := os.Stat(filepath.Join(base, "viral_script", "workflow.yaml")); os.IsNotExist(err) {
		t.Skipf("workflows directory not found at %s, skipping", base)
	}

	loader := NewLoader(base, true)
	def, err := loader.Load("viral_script")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if def.Type != "viral_script" {
		t.Errorf("expected type viral_script, got %s", def.Type)
	}
	if def.DisplayName != "口播稿改写" {
		t.Errorf("expected display_name 口播稿改写, got %s", def.DisplayName)
	}
	if len(def.Stages) != 6 {
		t.Fatalf("expected 6 stages, got %d", len(def.Stages))
	}

	s := def.Stages[0]
	if s.ID != "research" || s.Type != StageParallel {
		t.Errorf("stage 0: expected research/parallel, got %s/%s", s.ID, s.Type)
	}
	if len(s.Workers) != 3 {
		t.Errorf("stage 0: expected 3 workers, got %d", len(s.Workers))
	}
	if s.Workers[0].Name != "viral_decoder" {
		t.Errorf("stage 0 worker 0: expected viral_decoder, got %s", s.Workers[0].Name)
	}
	if s.Workers[0].SystemPrompt == "" {
		t.Error("stage 0 worker 0: system prompt is empty")
	}
	if s.Workers[0].UserPromptTpl == "" {
		t.Error("stage 0 worker 0: user prompt template is empty")
	}
	if s.SynthDef == nil {
		t.Error("stage 0: synth def should not be nil")
	}

	h := def.Stages[3]
	if h.ID != "confirm_outline" || h.Type != StageHuman {
		t.Errorf("stage 3: expected confirm_outline/human, got %s/%s", h.ID, h.Type)
	}
	if len(h.Options) != 4 {
		t.Errorf("stage 3: expected 4 options, got %d", len(h.Options))
	}
}
