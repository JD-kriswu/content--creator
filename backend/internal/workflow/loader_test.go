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
	if len(def.Stages) != 9 {
		t.Fatalf("expected 9 stages, got %d", len(def.Stages))
	}

	// Stage 0: research (parallel, only viral_decoder now)
	s := def.Stages[0]
	if s.ID != "research" || s.Type != StageParallel {
		t.Errorf("stage 0: expected research/parallel, got %s/%s", s.ID, s.Type)
	}
	if len(s.Workers) != 1 {
		t.Errorf("stage 0: expected 1 worker (viral_decoder only), got %d", len(s.Workers))
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
	if s.SynthDef != nil {
		t.Error("stage 0: synth def should be nil (no synth for single worker)")
	}

	// Stage 1: material_check (serial)
	mcs := def.Stages[1]
	if mcs.ID != "material_check" || mcs.Type != StageSerial {
		t.Errorf("stage 1: expected material_check/serial, got %s/%s", mcs.ID, mcs.Type)
	}

	// Stage 2: material_curator (serial with skip_if)
	mcs2 := def.Stages[2]
	if mcs2.ID != "material_curator" || mcs2.Type != StageSerial {
		t.Errorf("stage 2: expected material_curator/serial, got %s/%s", mcs2.ID, mcs2.Type)
	}
	if mcs2.SkipIf == "" {
		t.Error("stage 2: skip_if should be set")
	}

	// Stage 5: confirm_outline (human)
	h := def.Stages[5]
	if h.ID != "confirm_outline" || h.Type != StageHuman {
		t.Errorf("stage 5: expected confirm_outline/human, got %s/%s", h.ID, h.Type)
	}
	if len(h.Options) != 4 {
		t.Errorf("stage 5: expected 4 options, got %d", len(h.Options))
	}
}
