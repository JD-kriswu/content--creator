# Claw SaaS Workflow Engine Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Transform content-creator-imm from a single-LLM-call 5-role simulation into a real multi-Agent workflow engine with YAML-based prompt configuration, parallel Worker execution, and business-agnostic data model.

**Architecture:** Introduce a `workflow/` package as the engine core (types, loader, context builder, worker, stage executor, engine). Workflow definitions live in YAML files under `backend/workflows/`. The chat handler delegates to the engine instead of directly managing state. Frontend extends SSE handling to show parallel Worker progress.

**Tech Stack:** Go 1.22, Gin, GORM/MySQL, YAML (`gopkg.in/yaml.v3`), React 18, TypeScript, Tailwind v4

**Spec:** `docs/superpowers/specs/2026-04-02-claw-saas-workflow-engine-design.md`

---

## File Structure

### New Files (Backend)

| File | Responsibility |
|------|----------------|
| `backend/internal/workflow/types.go` | WorkflowDef, StageDef, WorkerDef, StageType, WorkflowContext, StageOutput, WorkerOutput, WorkerInput |
| `backend/internal/workflow/loader.go` | Load WorkflowDef from YAML files, dev-mode hot reload |
| `backend/internal/workflow/loader_test.go` | Tests for YAML loading and validation |
| `backend/internal/workflow/context.go` | ContextBuilder: variable interpolation `{{var}}`, buildWorkerInput |
| `backend/internal/workflow/context_test.go` | Tests for variable interpolation |
| `backend/internal/workflow/worker.go` | Run single Worker: LLM streaming call with SSE forwarding |
| `backend/internal/workflow/stage.go` | Stage executors: executeParallel, executeSerial, executeHuman |
| `backend/internal/workflow/engine.go` | Engine: Start/Resume/Cancel, stage loop, business hook (SaveScript) |
| `backend/internal/workflow/engine_test.go` | Integration test with mock LLM |
| `backend/internal/workflow/sse.go` | SSEWriter interface + implementation wrapping gin.Context |
| `backend/internal/model/workflow.go` | Workflow, WorkflowStage, WorkflowWorker GORM models |
| `backend/internal/repository/workflow_repo.go` | CRUD for workflow/stage/worker records |
| `backend/workflows/viral_script/workflow.yaml` | Stage orchestration definition |
| `backend/workflows/viral_script/prompts/viral_decoder.yaml` | 爆款解构师 prompt |
| `backend/workflows/viral_script/prompts/style_architect.yaml` | 风格建模师 prompt |
| `backend/workflows/viral_script/prompts/material_curator.yaml` | 素材补齐师 prompt |
| `backend/workflows/viral_script/prompts/creative_agent.yaml` | 创作代理 prompt |
| `backend/workflows/viral_script/prompts/optimization_agent.yaml` | 优化代理 prompt |
| `backend/workflows/viral_script/prompts/draft_writer.yaml` | 终稿撰写 prompt |
| `backend/workflows/viral_script/prompts/similarity_checker.yaml` | 相似度检测 prompt |
| `backend/workflows/viral_script/synth/research_synth.yaml` | 研究组汇总 prompt |
| `frontend/src/components/create/WorkerPanel.tsx` | Single Worker streaming output (title + status + collapsible content) |
| `frontend/src/components/create/ParallelStageView.tsx` | Grid layout for parallel Workers |
| `frontend/src/components/create/StageProgress.tsx` | Stage progress bar |

### Modified Files

| File | Changes |
|------|---------|
| `backend/go.mod` | Add `gopkg.in/yaml.v3` dependency |
| `backend/internal/db/db.go` | AutoMigrate 3 new models + 2 field additions |
| `backend/internal/model/conversation.go` | Add WorkflowType, WorkflowID fields |
| `backend/internal/model/message.go` | Add StageID, WorkerName fields |
| `backend/internal/handler/chat_handler.go` | Replace state machine with Engine.Start/Resume |
| `backend/internal/service/pipeline.go` | Keep session struct (simplified), remove analysis/draft logic |
| `backend/main.go` | Initialize WorkflowLoader, inject into handler |
| `frontend/src/lib/sse.ts` | Add stage_start, worker_start, worker_token, worker_done, synth_* events |
| `frontend/src/components/create/MessageList.tsx` | Add ChatMsg types for stage/worker, render ParallelStageView |
| `frontend/src/pages/Dashboard.tsx` | Extend reducer with STAGE_START, WORKER_START, WORKER_TOKEN, WORKER_DONE actions |

---

## Task 1: Workflow Type Definitions

**Files:**
- Create: `backend/internal/workflow/types.go`

- [ ] **Step 1: Create the workflow package and type definitions**

```go
// backend/internal/workflow/types.go
package workflow

import "time"

// StageType defines how a stage executes its workers.
type StageType string

const (
	StageParallel StageType = "parallel"
	StageSerial   StageType = "serial"
	StageHuman    StageType = "human"
)

// WorkerDef defines a single Worker agent loaded from YAML.
type WorkerDef struct {
	Name         string  `yaml:"name"`
	DisplayName  string  `yaml:"display_name"`
	SystemPrompt string  `yaml:"system"`
	UserPromptTpl string `yaml:"user"`
	MaxTokens    int     `yaml:"max_tokens"`
	Temperature  float64 `yaml:"temperature"`
	OutputFormat string  `yaml:"output_format"`
}

// SynthDef defines the synthesis prompt for a parallel stage.
type SynthDef struct {
	Name         string  `yaml:"name"`
	DisplayName  string  `yaml:"display_name"`
	SystemPrompt string  `yaml:"system"`
	UserPromptTpl string `yaml:"user"`
	MaxTokens    int     `yaml:"max_tokens"`
	Temperature  float64 `yaml:"temperature"`
	OutputFormat string  `yaml:"output_format"`
}

// StageDef defines a single stage in a workflow.
type StageDef struct {
	ID          string    `yaml:"id"`
	DisplayName string    `yaml:"display_name"`
	Type        StageType `yaml:"type"`
	WorkerNames []string  `yaml:"workers"`  // references to prompt YAML file names
	SynthPath   string    `yaml:"synth_prompt"` // relative path to synth YAML
	HumanPrompt string   `yaml:"prompt"`    // for human stages
	Options     []string  `yaml:"options"`   // for human stages

	// Populated by loader (not from YAML directly)
	Workers  []WorkerDef `yaml:"-"`
	SynthDef *SynthDef   `yaml:"-"`
}

// WorkflowDef is the complete workflow definition loaded from YAML.
type WorkflowDef struct {
	Type        string            `yaml:"type"`
	DisplayName string            `yaml:"display_name"`
	Meta        map[string]any    `yaml:"meta"`
	Stages      []StageDef        `yaml:"stages"`
}

// --- Runtime types ---

// WorkerOutput holds the result of a single Worker execution.
type WorkerOutput struct {
	Name     string
	Content  string
	Tokens   int
	Duration time.Duration
}

// StageOutput holds the combined results of a stage.
type StageOutput struct {
	StageID string
	Workers []WorkerOutput
	Summary string // LLM-synthesized summary for parallel stages; raw output for serial
}

// SharedContext is the Layer 1 context shared by all Workers.
type SharedContext struct {
	OriginalText string
	SourceURL    string
	UserStyle    string // pre-formatted style text
	WorkflowMeta map[string]any
}

// WorkflowContext is the runtime context container.
type WorkflowContext struct {
	Shared       SharedContext
	StageOutputs map[string]*StageOutput // stageID → output
	HumanInputs  map[string]string       // humanStageID → user input
}

// NewWorkflowContext creates an initialized context.
func NewWorkflowContext(shared SharedContext) *WorkflowContext {
	return &WorkflowContext{
		Shared:       shared,
		StageOutputs: make(map[string]*StageOutput),
		HumanInputs:  make(map[string]string),
	}
}

// WorkerInput is the fully-resolved input for a Worker's LLM call.
type WorkerInput struct {
	SystemPrompt string
	UserPrompt   string
	MaxTokens    int
	Temperature  float64
}

// WorkflowInput is what the API handler passes to Engine.Start.
type WorkflowInput struct {
	Text      string
	SourceURL string
	UserStyle string // pre-formatted
	UserID    uint
}
```

- [ ] **Step 2: Verify it compiles**

Run: `cd /data/code/content_creator_imm/backend && go build ./internal/workflow/`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
cd /data/code/content_creator_imm
git add backend/internal/workflow/types.go
git commit -m "feat(workflow): add core type definitions for workflow engine"
```

---

## Task 2: YAML Workflow Definitions

**Files:**
- Create: `backend/workflows/viral_script/workflow.yaml`
- Create: `backend/workflows/viral_script/prompts/viral_decoder.yaml`
- Create: `backend/workflows/viral_script/prompts/style_architect.yaml`
- Create: `backend/workflows/viral_script/prompts/material_curator.yaml`
- Create: `backend/workflows/viral_script/prompts/creative_agent.yaml`
- Create: `backend/workflows/viral_script/prompts/optimization_agent.yaml`
- Create: `backend/workflows/viral_script/prompts/draft_writer.yaml`
- Create: `backend/workflows/viral_script/prompts/similarity_checker.yaml`
- Create: `backend/workflows/viral_script/synth/research_synth.yaml`

- [ ] **Step 1: Create workflow.yaml**

```yaml
# backend/workflows/viral_script/workflow.yaml
type: viral_script
display_name: 口播稿改写
meta:
  max_similarity: 30
  word_count: "300-600"

stages:
  - id: research
    display_name: 研究分析
    type: parallel
    workers:
      - viral_decoder
      - style_architect
      - material_curator
    synth_prompt: synth/research_synth.yaml

  - id: create
    display_name: 大纲创作
    type: serial
    workers:
      - creative_agent

  - id: optimize
    display_name: 优化审查
    type: serial
    workers:
      - optimization_agent

  - id: confirm_outline
    display_name: 确认大纲
    type: human
    prompt: "请确认大纲：1-确认开始撰写 2-调整大纲 3-更换素材 4-重新分析"
    options:
      - "确认，开始撰写"
      - "调整大纲"
      - "更换素材"
      - "重新分析"

  - id: write
    display_name: 撰写终稿
    type: serial
    workers:
      - draft_writer

  - id: similarity
    display_name: 相似度检测
    type: serial
    workers:
      - similarity_checker
```

- [ ] **Step 2: Create all 7 worker prompt YAML files and 1 synth YAML file**

Copy prompt content from the design spec section 4. Each file follows the pattern:

```yaml
# backend/workflows/viral_script/prompts/{name}.yaml
name: {name}
display_name: {中文名}
max_tokens: {N}
temperature: {T}
output_format: {markdown|json|text}

system: |
  {system prompt content}

user: |
  {user prompt template with {{variables}}}
```

Files to create (content from spec — copy verbatim from `docs/superpowers/specs/2026-04-02-claw-saas-workflow-engine-design.md` sections 4.3–4.10):

1. `prompts/viral_decoder.yaml` — 爆款解构师, max_tokens:2000, temp:0.3
2. `prompts/style_architect.yaml` — 风格建模师, max_tokens:1500, temp:0.3
3. `prompts/material_curator.yaml` — 素材补齐师, max_tokens:1500, temp:0.5
4. `prompts/creative_agent.yaml` — 创作代理, max_tokens:3000, temp:0.7
5. `prompts/optimization_agent.yaml` — 优化代理, max_tokens:3000, temp:0.3
6. `prompts/draft_writer.yaml` — 终稿撰写, max_tokens:4000, temp:0.8
7. `prompts/similarity_checker.yaml` — 相似度检测, max_tokens:256, temp:0.1
8. `synth/research_synth.yaml` — 研究汇总, max_tokens:3000, temp:0.3

- [ ] **Step 3: Commit**

```bash
cd /data/code/content_creator_imm
git add backend/workflows/
git commit -m "feat(workflow): add viral_script YAML workflow and prompt definitions"
```

---

## Task 3: Workflow Loader

**Files:**
- Modify: `backend/go.mod` (add yaml dependency)
- Create: `backend/internal/workflow/loader.go`
- Create: `backend/internal/workflow/loader_test.go`

- [ ] **Step 1: Add yaml.v3 dependency**

Run: `cd /data/code/content_creator_imm/backend && go get gopkg.in/yaml.v3`

- [ ] **Step 2: Write the loader test**

```go
// backend/internal/workflow/loader_test.go
package workflow

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoaderLoadsViralScript(t *testing.T) {
	// Find the workflows directory relative to test
	wd, _ := os.Getwd()
	// Walk up to backend/, then into workflows/
	base := filepath.Join(wd, "..", "..", "workflows")
	if _, err := os.Stat(filepath.Join(base, "viral_script", "workflow.yaml")); os.IsNotExist(err) {
		t.Skipf("workflows directory not found at %s, skipping", base)
	}

	loader := NewLoader(base, true) // devMode = true
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

	// Stage 1: research (parallel, 3 workers, has synth)
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

	// Stage 4: human
	h := def.Stages[3]
	if h.ID != "confirm_outline" || h.Type != StageHuman {
		t.Errorf("stage 3: expected confirm_outline/human, got %s/%s", h.ID, h.Type)
	}
	if len(h.Options) != 4 {
		t.Errorf("stage 3: expected 4 options, got %d", len(h.Options))
	}
}
```

- [ ] **Step 3: Run the test to verify it fails**

Run: `cd /data/code/content_creator_imm/backend && go test ./internal/workflow/ -run TestLoaderLoads -v`
Expected: Compilation error — `NewLoader` not defined

- [ ] **Step 4: Implement the loader**

```go
// backend/internal/workflow/loader.go
package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

// Loader loads WorkflowDef from YAML files.
type Loader struct {
	basePath string
	devMode  bool
	mu       sync.RWMutex
	cache    map[string]*WorkflowDef
}

// NewLoader creates a Loader. In devMode, YAML is re-read on every Load call.
func NewLoader(basePath string, devMode bool) *Loader {
	return &Loader{
		basePath: basePath,
		devMode:  devMode,
		cache:    make(map[string]*WorkflowDef),
	}
}

// Load returns the WorkflowDef for the given type, reading from disk or cache.
func (l *Loader) Load(workflowType string) (*WorkflowDef, error) {
	if !l.devMode {
		l.mu.RLock()
		if cached, ok := l.cache[workflowType]; ok {
			l.mu.RUnlock()
			return cached, nil
		}
		l.mu.RUnlock()
	}
	return l.loadFromDisk(workflowType)
}

// Reload forces a re-read from disk and updates the cache.
func (l *Loader) Reload(workflowType string) (*WorkflowDef, error) {
	return l.loadFromDisk(workflowType)
}

func (l *Loader) loadFromDisk(workflowType string) (*WorkflowDef, error) {
	dir := filepath.Join(l.basePath, workflowType)

	// 1. Read workflow.yaml
	wfPath := filepath.Join(dir, "workflow.yaml")
	data, err := os.ReadFile(wfPath)
	if err != nil {
		return nil, fmt.Errorf("read workflow.yaml: %w", err)
	}
	var def WorkflowDef
	if err := yaml.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("parse workflow.yaml: %w", err)
	}

	// 2. For each stage, load worker prompt YAMLs
	for i := range def.Stages {
		stage := &def.Stages[i]
		stage.Workers = make([]WorkerDef, 0, len(stage.WorkerNames))

		for _, name := range stage.WorkerNames {
			wd, err := l.loadWorkerDef(dir, name)
			if err != nil {
				return nil, fmt.Errorf("stage %s worker %s: %w", stage.ID, name, err)
			}
			stage.Workers = append(stage.Workers, *wd)
		}

		// 3. Load synth prompt if specified
		if stage.SynthPath != "" {
			sd, err := l.loadSynthDef(dir, stage.SynthPath)
			if err != nil {
				return nil, fmt.Errorf("stage %s synth: %w", stage.ID, err)
			}
			stage.SynthDef = sd
		}
	}

	// Cache
	l.mu.Lock()
	l.cache[workflowType] = &def
	l.mu.Unlock()

	return &def, nil
}

func (l *Loader) loadWorkerDef(dir, name string) (*WorkerDef, error) {
	path := filepath.Join(dir, "prompts", name+".yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var wd WorkerDef
	if err := yaml.Unmarshal(data, &wd); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &wd, nil
}

func (l *Loader) loadSynthDef(dir, relPath string) (*SynthDef, error) {
	path := filepath.Join(dir, relPath)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var sd SynthDef
	if err := yaml.Unmarshal(data, &sd); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &sd, nil
}
```

- [ ] **Step 5: Run the test to verify it passes**

Run: `cd /data/code/content_creator_imm/backend && go test ./internal/workflow/ -run TestLoaderLoads -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
cd /data/code/content_creator_imm
git add backend/go.mod backend/go.sum backend/internal/workflow/loader.go backend/internal/workflow/loader_test.go
git commit -m "feat(workflow): implement YAML workflow loader with hot-reload"
```

---

## Task 4: Context Builder (Variable Interpolation)

**Files:**
- Create: `backend/internal/workflow/context.go`
- Create: `backend/internal/workflow/context_test.go`

- [ ] **Step 1: Write the context builder test**

```go
// backend/internal/workflow/context_test.go
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
		"stage.research.summary":                       "汇总结果",
		"stage.research.worker.viral_decoder.output":   "解构输出",
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
	// Missing vars should be replaced with empty string
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
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd /data/code/content_creator_imm/backend && go test ./internal/workflow/ -run TestInterpolate -v`
Expected: Compilation error — `interpolate` not defined

- [ ] **Step 3: Implement context builder**

```go
// backend/internal/workflow/context.go
package workflow

import (
	"fmt"
	"strings"
)

// interpolate replaces {{var.path}} placeholders with values from the vars map.
// Missing vars are replaced with empty string.
func interpolate(tpl string, vars map[string]string) string {
	result := tpl
	for key, val := range vars {
		result = strings.ReplaceAll(result, "{{"+key+"}}", val)
	}
	// Remove any remaining unresolved placeholders
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

// BuildWorkerInput resolves a WorkerDef's prompt template against the current WorkflowContext.
func BuildWorkerInput(ctx *WorkflowContext, wd WorkerDef) WorkerInput {
	vars := buildVarsMap(ctx)
	return WorkerInput{
		SystemPrompt: wd.SystemPrompt,
		UserPrompt:   interpolate(wd.UserPromptTpl, vars),
		MaxTokens:    wd.MaxTokens,
		Temperature:  wd.Temperature,
	}
}

// BuildSynthInput resolves a SynthDef's prompt template, injecting worker outputs from the current stage.
func BuildSynthInput(ctx *WorkflowContext, sd SynthDef, stageID string, workerOutputs []WorkerOutput) WorkerInput {
	vars := buildVarsMap(ctx)
	// Also add current stage worker outputs (they may not be in StageOutputs yet)
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

	// Workflow meta
	for k, v := range ctx.Shared.WorkflowMeta {
		vars[fmt.Sprintf("workflow.meta.%s", k)] = fmt.Sprintf("%v", v)
	}

	// Stage outputs
	for stageID, output := range ctx.StageOutputs {
		vars[fmt.Sprintf("stage.%s.summary", stageID)] = output.Summary
		for _, w := range output.Workers {
			vars[fmt.Sprintf("stage.%s.worker.%s.output", stageID, w.Name)] = w.Content
		}
	}

	// Human inputs
	for humanID, input := range ctx.HumanInputs {
		vars[fmt.Sprintf("human.%s.input", humanID)] = input
	}

	return vars
}
```

- [ ] **Step 4: Run all context tests**

Run: `cd /data/code/content_creator_imm/backend && go test ./internal/workflow/ -run "TestInterpolate|TestBuildWorker" -v`
Expected: All PASS

- [ ] **Step 5: Commit**

```bash
cd /data/code/content_creator_imm
git add backend/internal/workflow/context.go backend/internal/workflow/context_test.go
git commit -m "feat(workflow): implement context builder with variable interpolation"
```

---

## Task 5: SSE Writer Interface

**Files:**
- Create: `backend/internal/workflow/sse.go`

- [ ] **Step 1: Define the SSE writer interface and implementation**

```go
// backend/internal/workflow/sse.go
package workflow

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"
)

// SSEWriter sends SSE events to the client. Thread-safe.
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

// GinSSEWriter writes SSE events to a gin response writer.
type GinSSEWriter struct {
	w       io.Writer
	flusher interface{ Flush() }
	mu      sync.Mutex
}

// NewGinSSEWriter wraps a gin.Context's writer for SSE output.
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
```

- [ ] **Step 2: Verify it compiles**

Run: `cd /data/code/content_creator_imm/backend && go build ./internal/workflow/`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
cd /data/code/content_creator_imm
git add backend/internal/workflow/sse.go
git commit -m "feat(workflow): add SSE writer interface for stage/worker events"
```

---

## Task 6: Worker Execution

**Files:**
- Create: `backend/internal/workflow/worker.go`

- [ ] **Step 1: Implement Worker runner**

The Worker calls the existing `service.StreamClaude` / `service.CallClaude` functions, forwarding tokens to SSE.

```go
// backend/internal/workflow/worker.go
package workflow

import (
	"strings"
	"time"

	"content_creator_imm/internal/service"
)

// RunWorker executes a single Worker's LLM call, streaming tokens to SSE.
// Returns the full output and duration.
func RunWorker(wd WorkerDef, input WorkerInput, stageID string, sse SSEWriter) (WorkerOutput, error) {
	sse.SendWorkerStart(stageID, wd.Name, wd.DisplayName)

	start := time.Now()
	var fullContent strings.Builder

	content, err := service.StreamClaude(input.SystemPrompt, input.UserPrompt, func(token string) bool {
		fullContent.WriteString(token)
		sse.SendWorkerToken(wd.Name, token)
		return true
	})
	if err != nil {
		sse.SendWorkerDone(wd.Name)
		return WorkerOutput{Name: wd.Name}, err
	}

	duration := time.Since(start)
	sse.SendWorkerDone(wd.Name)

	// Use the content returned by StreamClaude (it accumulates internally too)
	output := content
	if fullContent.Len() > 0 {
		output = fullContent.String()
	}

	return WorkerOutput{
		Name:     wd.Name,
		Content:  output,
		Duration: duration,
	}, nil
}

// RunWorkerNonStream executes a Worker with a non-streaming LLM call (e.g., similarity check).
func RunWorkerNonStream(wd WorkerDef, input WorkerInput, stageID string, sse SSEWriter) (WorkerOutput, error) {
	sse.SendWorkerStart(stageID, wd.Name, wd.DisplayName)

	start := time.Now()
	content, err := service.CallClaude(input.SystemPrompt, input.UserPrompt, input.MaxTokens)
	duration := time.Since(start)

	sse.SendWorkerDone(wd.Name)

	if err != nil {
		return WorkerOutput{Name: wd.Name}, err
	}

	return WorkerOutput{
		Name:     wd.Name,
		Content:  content,
		Duration: duration,
	}, nil
}
```

- [ ] **Step 2: Verify it compiles**

Run: `cd /data/code/content_creator_imm/backend && go build ./internal/workflow/`
Expected: No errors (imports `service.StreamClaude` which exists)

- [ ] **Step 3: Commit**

```bash
cd /data/code/content_creator_imm
git add backend/internal/workflow/worker.go
git commit -m "feat(workflow): implement Worker LLM execution with SSE forwarding"
```

---

## Task 7: Stage Executors

**Files:**
- Create: `backend/internal/workflow/stage.go`

- [ ] **Step 1: Implement the three stage executor functions**

```go
// backend/internal/workflow/stage.go
package workflow

import (
	"fmt"
	"sync"

	"content_creator_imm/internal/service"
)

// ErrWaitingHuman is returned by executeHumanStage to signal the engine should pause.
var ErrWaitingHuman = fmt.Errorf("waiting for human input")

// ExecuteParallelStage runs all Workers concurrently, then synthesizes results.
func ExecuteParallelStage(ctx *WorkflowContext, stage StageDef, sse SSEWriter) (*StageOutput, error) {
	results := make([]WorkerOutput, len(stage.Workers))
	errs := make([]error, len(stage.Workers))
	var wg sync.WaitGroup

	for i, wd := range stage.Workers {
		wg.Add(1)
		go func(idx int, wd WorkerDef) {
			defer wg.Done()
			input := BuildWorkerInput(ctx, wd)
			out, err := RunWorker(wd, input, stage.ID, sse)
			results[idx] = out
			errs[idx] = err
		}(i, wd)
	}
	wg.Wait()

	// Check for errors
	for i, err := range errs {
		if err != nil {
			return nil, fmt.Errorf("worker %s failed: %w", stage.Workers[i].Name, err)
		}
	}

	// Synthesize if SynthDef is present
	var summary string
	if stage.SynthDef != nil {
		sse.SendSynthStart(stage.ID)
		synthInput := BuildSynthInput(ctx, *stage.SynthDef, stage.ID, results)

		var err error
		summary, err = service.StreamClaude(synthInput.SystemPrompt, synthInput.UserPrompt, func(token string) bool {
			sse.SendSynthToken(token)
			return true
		})
		if err != nil {
			return nil, fmt.Errorf("synth failed: %w", err)
		}
		sse.SendSynthDone(stage.ID)
	} else {
		// No synth: concatenate worker outputs
		for _, r := range results {
			summary += fmt.Sprintf("## %s\n%s\n\n", r.Name, r.Content)
		}
	}

	return &StageOutput{
		StageID: stage.ID,
		Workers: results,
		Summary: summary,
	}, nil
}

// ExecuteSerialStage runs a single Worker.
func ExecuteSerialStage(ctx *WorkflowContext, stage StageDef, sse SSEWriter) (*StageOutput, error) {
	if len(stage.Workers) == 0 {
		return nil, fmt.Errorf("serial stage %s has no workers", stage.ID)
	}
	wd := stage.Workers[0]
	input := BuildWorkerInput(ctx, wd)

	// Use non-streaming for similarity checker (small max_tokens, json output)
	var out WorkerOutput
	var err error
	if wd.MaxTokens <= 256 {
		out, err = RunWorkerNonStream(wd, input, stage.ID, sse)
	} else {
		out, err = RunWorker(wd, input, stage.ID, sse)
	}
	if err != nil {
		return nil, err
	}

	return &StageOutput{
		StageID: stage.ID,
		Workers: []WorkerOutput{out},
		Summary: out.Content,
	}, nil
}

// ExecuteHumanStage sends the outline and action options to the client, then returns ErrWaitingHuman.
func ExecuteHumanStage(ctx *WorkflowContext, stage StageDef, sse SSEWriter) error {
	// Send the last stage's output as outline data
	// Parse outline JSON from the previous stage's output
	lastOutput := findLastStageOutput(ctx)
	if lastOutput != nil {
		outlineData := service.ParseOutlineFromAnalysis(lastOutput.Summary)
		if outlineData != nil {
			sse.SendOutline(outlineData)
		}
	}
	sse.SendAction(stage.HumanPrompt, stage.Options)
	return ErrWaitingHuman
}

func findLastStageOutput(ctx *WorkflowContext) *StageOutput {
	// Return the most recently added stage output
	// Since maps are unordered, we track this differently in the engine
	// For now, check common stage IDs in order
	for _, id := range []string{"optimize", "create", "research"} {
		if out, ok := ctx.StageOutputs[id]; ok {
			return out
		}
	}
	return nil
}
```

- [ ] **Step 2: Verify it compiles**

Run: `cd /data/code/content_creator_imm/backend && go build ./internal/workflow/`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
cd /data/code/content_creator_imm
git add backend/internal/workflow/stage.go
git commit -m "feat(workflow): implement parallel, serial, and human stage executors"
```

---

## Task 8: Database Models and Migration

**Files:**
- Create: `backend/internal/model/workflow.go`
- Modify: `backend/internal/model/conversation.go`
- Modify: `backend/internal/model/message.go`
- Modify: `backend/internal/db/db.go`
- Create: `backend/internal/repository/workflow_repo.go`

- [ ] **Step 1: Create workflow models**

```go
// backend/internal/model/workflow.go
package model

import "time"

// Workflow is a workflow execution record (business-agnostic).
type Workflow struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	UserID      uint       `gorm:"index;not null" json:"user_id"`
	Type        string     `gorm:"size:64;index;not null" json:"type"`
	Status      string     `gorm:"size:20;index" json:"status"` // pending/running/paused/completed/failed
	InputJSON   string     `gorm:"type:text" json:"-"`
	ContextJSON string     `gorm:"type:text" json:"-"`
	OutputJSON  string     `gorm:"type:text" json:"-"`
	ConvID      *uint      `json:"conv_id,omitempty"`
	Error       string     `gorm:"type:text" json:"error,omitempty"`
	PausedAt    string     `gorm:"size:64" json:"-"` // stage ID where paused for human input
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// WorkflowStage is a stage execution record.
type WorkflowStage struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	WorkflowID uint       `gorm:"index;not null" json:"workflow_id"`
	StageID    string     `gorm:"size:64;not null" json:"stage_id"`
	Type       string     `gorm:"size:20" json:"type"`
	Sequence   int        `json:"sequence"`
	Status     string     `gorm:"size:20" json:"status"`
	InputJSON  string     `gorm:"type:text" json:"-"`
	OutputJSON string     `gorm:"type:text" json:"-"`
	StartedAt  *time.Time `json:"started_at,omitempty"`
	EndedAt    *time.Time `json:"ended_at,omitempty"`
}

// WorkflowWorker is a worker execution record.
type WorkflowWorker struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	StageID    uint       `gorm:"index;not null" json:"stage_id"`
	WorkflowID uint       `gorm:"index;not null" json:"workflow_id"`
	WorkerName string     `gorm:"size:64;not null" json:"worker_name"`
	Role       string     `gorm:"size:128" json:"role"`
	Status     string     `gorm:"size:20" json:"status"`
	InputJSON  string     `gorm:"type:text" json:"-"`
	OutputJSON string     `gorm:"type:text" json:"-"`
	TokensUsed int        `json:"tokens_used"`
	DurationMs int        `json:"duration_ms"`
	StartedAt  *time.Time `json:"started_at,omitempty"`
	EndedAt    *time.Time `json:"ended_at,omitempty"`
}
```

- [ ] **Step 2: Add fields to existing Conversation model**

In `backend/internal/model/conversation.go`, add after the `State` field:

```go
	WorkflowType string `gorm:"size:64;index" json:"workflow_type,omitempty"`
	WorkflowID   *uint  `json:"workflow_id,omitempty"`
```

- [ ] **Step 3: Add fields to existing Message model**

In `backend/internal/model/message.go`, add after the `Name` field:

```go
	StageID    string `gorm:"size:64;index" json:"stage_id,omitempty"`
	WorkerName string `gorm:"size:64" json:"worker_name,omitempty"`
```

- [ ] **Step 4: Update AutoMigrate in db.go**

In `backend/internal/db/db.go`, add the 3 new models to the AutoMigrate call:

```go
	db.AutoMigrate(
		&model.User{},
		&model.UserStyle{},
		&model.Script{},
		&model.Conversation{},
		&model.Message{},
		&model.Workflow{},       // new
		&model.WorkflowStage{},  // new
		&model.WorkflowWorker{}, // new
	)
```

- [ ] **Step 5: Create workflow repository**

```go
// backend/internal/repository/workflow_repo.go
package repository

import (
	"content_creator_imm/internal/db"
	"content_creator_imm/internal/model"
)

func CreateWorkflow(w *model.Workflow) error {
	return db.DB.Create(w).Error
}

func UpdateWorkflow(w *model.Workflow) error {
	return db.DB.Save(w).Error
}

func GetWorkflow(id uint) (*model.Workflow, error) {
	var w model.Workflow
	err := db.DB.First(&w, id).Error
	return &w, err
}

func CreateWorkflowStage(s *model.WorkflowStage) error {
	return db.DB.Create(s).Error
}

func UpdateWorkflowStage(s *model.WorkflowStage) error {
	return db.DB.Save(s).Error
}

func CreateWorkflowWorker(w *model.WorkflowWorker) error {
	return db.DB.Create(w).Error
}

func UpdateWorkflowWorker(w *model.WorkflowWorker) error {
	return db.DB.Save(w).Error
}

func GetActiveWorkflow(userID uint) (*model.Workflow, error) {
	var w model.Workflow
	err := db.DB.Where("user_id = ? AND status IN ('running','paused')", userID).
		Order("created_at DESC").First(&w).Error
	return &w, err
}
```

- [ ] **Step 6: Verify backend compiles**

Run: `cd /data/code/content_creator_imm/backend && go build .`
Expected: No errors

- [ ] **Step 7: Commit**

```bash
cd /data/code/content_creator_imm
git add backend/internal/model/workflow.go backend/internal/model/conversation.go backend/internal/model/message.go backend/internal/db/db.go backend/internal/repository/workflow_repo.go
git commit -m "feat(db): add workflow/stage/worker models and repository"
```

---

## Task 9: Workflow Engine

**Files:**
- Create: `backend/internal/workflow/engine.go`

This is the orchestrator that ties everything together.

- [ ] **Step 1: Implement the Engine**

```go
// backend/internal/workflow/engine.go
package workflow

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"content_creator_imm/internal/db"
	"content_creator_imm/internal/model"
	"content_creator_imm/internal/repository"
	"content_creator_imm/internal/service"
)

// Engine orchestrates a workflow execution.
type Engine struct {
	loader *Loader
	sse    SSEWriter
	wfCtx  *WorkflowContext
	wfDef  *WorkflowDef
	wfID   uint // DB Workflow.ID
	userID uint
}

// NewEngine creates an engine for a new workflow.
func NewEngine(loader *Loader, sse SSEWriter) *Engine {
	return &Engine{loader: loader, sse: sse}
}

// Start loads the workflow definition, creates DB records, and executes stages.
func (e *Engine) Start(workflowType string, input WorkflowInput) error {
	// 1. Load workflow definition
	def, err := e.loader.Load(workflowType)
	if err != nil {
		e.sse.SendError(fmt.Sprintf("加载工作流失败: %v", err))
		return err
	}
	e.wfDef = def
	e.userID = input.UserID

	// 2. Build shared context
	shared := SharedContext{
		OriginalText: input.Text,
		SourceURL:    input.SourceURL,
		UserStyle:    input.UserStyle,
		WorkflowMeta: def.Meta,
	}
	e.wfCtx = NewWorkflowContext(shared)

	// 3. Create DB workflow record
	ctxJSON, _ := json.Marshal(shared)
	wf := &model.Workflow{
		UserID:      input.UserID,
		Type:        workflowType,
		Status:      "running",
		InputJSON:   input.Text,
		ContextJSON: string(ctxJSON),
	}
	if err := repository.CreateWorkflow(wf); err != nil {
		log.Printf("failed to create workflow record: %v", err)
	}
	e.wfID = wf.ID

	// 4. Execute stages
	return e.runStages(0)
}

// Resume continues execution after a human stage.
func (e *Engine) Resume(workflowID uint, humanInput string) error {
	// 1. Load workflow record
	wf, err := repository.GetWorkflow(workflowID)
	if err != nil {
		e.sse.SendError("无法恢复工作流")
		return err
	}
	e.wfID = wf.ID
	e.userID = wf.UserID

	// 2. Restore workflow definition
	def, err := e.loader.Load(wf.Type)
	if err != nil {
		return err
	}
	e.wfDef = def

	// 3. Restore context from DB
	var shared SharedContext
	json.Unmarshal([]byte(wf.ContextJSON), &shared)
	e.wfCtx = NewWorkflowContext(shared)

	// Restore stage outputs from DB
	e.restoreStageOutputs()

	// 4. Record human input
	e.wfCtx.HumanInputs[wf.PausedAt] = humanInput

	// 5. Determine resume point based on user choice
	resumeIdx := e.resolveResumeStage(wf.PausedAt, humanInput)

	// Update workflow status
	wf.Status = "running"
	wf.PausedAt = ""
	repository.UpdateWorkflow(wf)

	return e.runStages(resumeIdx)
}

func (e *Engine) runStages(startIdx int) error {
	for i := startIdx; i < len(e.wfDef.Stages); i++ {
		stage := e.wfDef.Stages[i]

		// Send stage_start + step events
		e.sse.SendStageStart(stage.ID, stage.DisplayName, stage.Type)
		e.sse.SendStep(i+1, stage.DisplayName)

		var err error
		switch stage.Type {
		case StageParallel:
			output, execErr := ExecuteParallelStage(e.wfCtx, stage, e.sse)
			if execErr != nil {
				e.handleStageError(stage.ID, execErr)
				return execErr
			}
			e.wfCtx.StageOutputs[stage.ID] = output
			e.persistStageOutput(stage, output, i)

		case StageSerial:
			output, execErr := ExecuteSerialStage(e.wfCtx, stage, e.sse)
			if execErr != nil {
				e.handleStageError(stage.ID, execErr)
				return execErr
			}
			e.wfCtx.StageOutputs[stage.ID] = output
			e.persistStageOutput(stage, output, i)

			// Special handling: send similarity data for similarity stage
			if stage.ID == "similarity" {
				e.handleSimilarityOutput(output)
			}

		case StageHuman:
			err = ExecuteHumanStage(e.wfCtx, stage, e.sse)
			if err == ErrWaitingHuman {
				// Persist checkpoint
				e.saveCheckpoint(stage.ID)
				return nil // Normal pause — not an error
			}
		}

		if err != nil {
			e.handleStageError(stage.ID, err)
			return err
		}

		e.sse.SendStageDone(stage.ID)
	}

	// All stages complete — finalize
	return e.finish()
}

func (e *Engine) finish() error {
	// Save script (business logic for viral_script workflow)
	if e.wfDef.Type == "viral_script" {
		return e.saveViralScript()
	}
	return nil
}

func (e *Engine) saveViralScript() error {
	writeOutput, ok := e.wfCtx.StageOutputs["write"]
	if !ok {
		e.sse.SendError("终稿数据丢失")
		return fmt.Errorf("write stage output missing")
	}
	draft := service.StripQualityCheck(writeOutput.Summary)

	// Parse similarity
	simOutput, ok := e.wfCtx.StageOutputs["similarity"]
	var simScore float64
	if ok {
		simScore = parseSimilarityTotal(simOutput.Summary)
	}

	// Save script file + DB record
	scriptID, err := service.SaveScript(e.userID, e.wfCtx.Shared.OriginalText, e.wfCtx.Shared.SourceURL, draft, simScore)
	if err != nil {
		e.sse.SendError(fmt.Sprintf("保存失败: %v", err))
		return err
	}

	// Update workflow as completed
	wf, _ := repository.GetWorkflow(e.wfID)
	if wf != nil {
		wf.Status = "completed"
		wf.OutputJSON = draft
		repository.UpdateWorkflow(wf)
	}

	e.sse.SendComplete(scriptID)
	return nil
}

func (e *Engine) handleStageError(stageID string, err error) {
	log.Printf("stage %s error: %v", stageID, err)
	e.sse.SendError(fmt.Sprintf("阶段 %s 执行失败: %v", stageID, err))
	wf, _ := repository.GetWorkflow(e.wfID)
	if wf != nil {
		wf.Status = "failed"
		wf.Error = err.Error()
		repository.UpdateWorkflow(wf)
	}
}

func (e *Engine) saveCheckpoint(stageID string) {
	ctxJSON, _ := json.Marshal(e.wfCtx.Shared)
	wf, _ := repository.GetWorkflow(e.wfID)
	if wf != nil {
		wf.Status = "paused"
		wf.PausedAt = stageID
		wf.ContextJSON = string(ctxJSON)
		repository.UpdateWorkflow(wf)
	}
}

func (e *Engine) persistStageOutput(stage StageDef, output *StageOutput, seq int) {
	now := time.Now()
	dbStage := &model.WorkflowStage{
		WorkflowID: e.wfID,
		StageID:    stage.ID,
		Type:       string(stage.Type),
		Sequence:   seq,
		Status:     "completed",
		OutputJSON: output.Summary,
		StartedAt:  &now,
		EndedAt:    &now,
	}
	repository.CreateWorkflowStage(dbStage)

	for _, wo := range output.Workers {
		dbWorker := &model.WorkflowWorker{
			StageID:    dbStage.ID,
			WorkflowID: e.wfID,
			WorkerName: wo.Name,
			Status:     "completed",
			OutputJSON: wo.Content,
			DurationMs: int(wo.Duration.Milliseconds()),
			StartedAt:  &now,
			EndedAt:    &now,
		}
		repository.CreateWorkflowWorker(dbWorker)
	}
}

func (e *Engine) restoreStageOutputs() {
	// Query all completed stages + workers for this workflow
	var stages []model.WorkflowStage
	db.DB.Where("workflow_id = ? AND status = 'completed'", e.wfID).
		Order("sequence ASC").Find(&stages)

	for _, s := range stages {
		var workers []model.WorkflowWorker
		db.DB.Where("stage_id = ?", s.ID).Find(&workers)

		wo := make([]WorkerOutput, 0, len(workers))
		for _, w := range workers {
			wo = append(wo, WorkerOutput{
				Name:    w.WorkerName,
				Content: w.OutputJSON,
			})
		}
		e.wfCtx.StageOutputs[s.StageID] = &StageOutput{
			StageID: s.StageID,
			Workers: wo,
			Summary: s.OutputJSON,
		}
	}
}

func (e *Engine) resolveResumeStage(pausedStageID, humanInput string) int {
	choice := strings.TrimSpace(humanInput)

	// Find the paused stage index
	pausedIdx := -1
	for i, s := range e.wfDef.Stages {
		if s.ID == pausedStageID {
			pausedIdx = i
			break
		}
	}

	// Determine resume point based on user choice
	switch {
	case choice == "1" || strings.Contains(choice, "确认"):
		// Continue from next stage after human
		return pausedIdx + 1
	case choice == "2" || strings.Contains(choice, "调整"):
		// Re-run from create stage with user note
		return e.findStageIndex("create")
	case choice == "3" || strings.Contains(choice, "素材") || strings.Contains(choice, "更换"):
		// Re-run from research
		e.wfCtx.StageOutputs = make(map[string]*StageOutput)
		return e.findStageIndex("research")
	case choice == "4" || strings.Contains(choice, "重新"):
		// Full restart
		e.wfCtx.StageOutputs = make(map[string]*StageOutput)
		return 0
	default:
		// Treat as adjustment note, continue from create
		e.wfCtx.HumanInputs[pausedStageID] = choice
		return e.findStageIndex("create")
	}
}

func (e *Engine) findStageIndex(stageID string) int {
	for i, s := range e.wfDef.Stages {
		if s.ID == stageID {
			return i
		}
	}
	return 0
}

func (e *Engine) handleSimilarityOutput(output *StageOutput) {
	// Try to parse similarity JSON and send as SSE event
	var simData map[string]any
	if err := json.Unmarshal([]byte(output.Summary), &simData); err == nil {
		e.sse.SendSimilarity(simData)
	}
}

func parseSimilarityTotal(raw string) float64 {
	var data struct {
		Total float64 `json:"total"`
	}
	// Try to find JSON in the output
	raw = strings.TrimSpace(raw)
	if json.Unmarshal([]byte(raw), &data) == nil {
		return data.Total
	}
	return 0
}
```

- [ ] **Step 2: Note — `service.SaveScript` and `service.StripQualityCheck` need refactoring**

The current `SaveScript` in `pipeline.go` takes a `*ChatSession`. We need to extract it into a standalone function. Add this to `pipeline.go` (or a new file) — a function with this signature:

```go
// SaveScript saves the final draft to file + DB. Returns scriptID.
func SaveScript(userID uint, originalText, sourceURL, draft string, similarity float64) (uint, error)
```

Also export `StripQualityCheck`:

```go
func StripQualityCheck(text string) string {
    return stripQualityCheck(text)
}
```

These are small refactors of existing code in `pipeline.go`.

- [ ] **Step 3: Verify backend compiles**

Run: `cd /data/code/content_creator_imm/backend && go build .`
Expected: No errors (after the pipeline.go refactors above)

- [ ] **Step 4: Commit**

```bash
cd /data/code/content_creator_imm
git add backend/internal/workflow/engine.go backend/internal/service/pipeline.go
git commit -m "feat(workflow): implement workflow engine with stage loop and resume"
```

---

## Task 10: Refactor Chat Handler

**Files:**
- Modify: `backend/internal/handler/chat_handler.go`
- Modify: `backend/internal/service/pipeline.go`
- Modify: `backend/main.go`

- [ ] **Step 1: Add WorkflowLoader to main.go initialization**

In `backend/main.go`, after `db.Init()`, add:

```go
	// Initialize workflow loader
	wfLoader := workflow.NewLoader("workflows", config.C.Port == "3004") // devMode in dev
```

Pass `wfLoader` to the chat handler setup (the exact injection depends on how handlers are registered — add it as a package-level variable or pass to handler struct).

- [ ] **Step 2: Add session tracking fields to ChatSession**

In `backend/internal/service/pipeline.go`, add to the `ChatSession` struct:

```go
	ActiveWorkflowID uint  // non-zero if a workflow is running/paused
```

- [ ] **Step 3: Rewrite SendMessage handler**

Replace the core of `SendMessage` in `chat_handler.go`. The new flow:

```go
func SendMessage(c *gin.Context) {
	userID := c.GetUint("userID")
	var req struct {
		Message string `json:"message"`
		ConvID  *uint  `json:"conv_id,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}

	// SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	sse := workflow.NewGinSSEWriter(c.Writer, c.Writer)
	sess := service.GetOrCreateSession(userID)
	sess.Mu.Lock()

	if sess.ActiveWorkflowID == 0 {
		// New workflow
		sess.Mu.Unlock()

		// Extract text or URL
		text := req.Message
		sourceURL := ""
		if service.IsURL(text) {
			sourceURL = text
			extracted, err := service.ExtractURL(text)
			if err != nil {
				sse.SendError(fmt.Sprintf("提取失败: %v", err))
				return
			}
			text = extracted
			sse.SendInfo(fmt.Sprintf("已提取 %d 字", len([]rune(text))))
		}

		// Load user style
		userStyle := loadUserStyleFormatted(userID)

		// Create conversation
		service.EnsureConversation(sess, req.Message)

		input := workflow.WorkflowInput{
			Text:      text,
			SourceURL: sourceURL,
			UserStyle: userStyle,
			UserID:    userID,
		}

		engine := workflow.NewEngine(wfLoader, sse)
		sess.ActiveWorkflowID = 0 // Will be set by engine

		// Run synchronously (SSE streams in real-time)
		if err := engine.Start("viral_script", input); err != nil && err != workflow.ErrWaitingHuman {
			log.Printf("workflow error: %v", err)
		}

		// Save the workflow ID for resume
		sess.Mu.Lock()
		sess.ActiveWorkflowID = engine.WorkflowID()
		sess.Mu.Unlock()

	} else {
		// Resume paused workflow
		wfID := sess.ActiveWorkflowID
		sess.Mu.Unlock()

		engine := workflow.NewEngine(wfLoader, sse)
		if err := engine.Resume(wfID, req.Message); err != nil {
			log.Printf("resume error: %v", err)
		}

		// Check if workflow completed
		wf, _ := repository.GetWorkflow(wfID)
		if wf != nil && (wf.Status == "completed" || wf.Status == "failed") {
			sess.Mu.Lock()
			sess.ActiveWorkflowID = 0
			sess.Mu.Unlock()
		}
	}
}
```

- [ ] **Step 4: Add `WorkflowID()` getter to Engine**

In `engine.go`, add:

```go
func (e *Engine) WorkflowID() uint {
	return e.wfID
}
```

- [ ] **Step 5: Verify backend compiles**

Run: `cd /data/code/content_creator_imm/backend && go build .`
Expected: No errors

- [ ] **Step 6: Commit**

```bash
cd /data/code/content_creator_imm
git add backend/main.go backend/internal/handler/chat_handler.go backend/internal/service/pipeline.go backend/internal/workflow/engine.go
git commit -m "feat(handler): refactor chat handler to use workflow engine"
```

---

## Task 11: Frontend SSE Types Extension

**Files:**
- Modify: `frontend/src/lib/sse.ts`

- [ ] **Step 1: Extend SSEEvent union type**

```typescript
// frontend/src/lib/sse.ts
export type SSEEvent =
  | { type: 'token'; content: string }
  | { type: 'step'; step: number; name: string }
  | { type: 'info'; content: string }
  | { type: 'outline'; data: unknown }
  | { type: 'action'; options: string[] }
  | { type: 'similarity'; data: unknown }
  | { type: 'final_draft'; content: string }
  | { type: 'complete'; scriptId: number }
  | { type: 'error'; message: string }
  // New workflow engine events
  | { type: 'stage_start'; stage_id: string; stage_name: string; stage_type: 'parallel' | 'serial' | 'human' }
  | { type: 'stage_done'; stage_id: string }
  | { type: 'worker_start'; stage_id: string; worker_name: string; worker_display: string }
  | { type: 'worker_token'; worker_name: string; content: string }
  | { type: 'worker_done'; worker_name: string }
  | { type: 'synth_start'; stage_id: string }
  | { type: 'synth_token'; content: string }
  | { type: 'synth_done'; stage_id: string }

export function parseSSELine(line: string): SSEEvent | null {
  if (!line.startsWith('data: ')) return null
  try {
    return JSON.parse(line.slice(6)) as SSEEvent
  } catch {
    return null
  }
}
```

- [ ] **Step 2: Verify types compile**

Run: `cd /data/code/content_creator_imm/frontend && npx tsc --noEmit`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
cd /data/code/content_creator_imm
git add frontend/src/lib/sse.ts
git commit -m "feat(frontend): extend SSE types for workflow stage/worker events"
```

---

## Task 12: Frontend Parallel Worker Components

**Files:**
- Create: `frontend/src/components/create/WorkerPanel.tsx`
- Create: `frontend/src/components/create/ParallelStageView.tsx`
- Create: `frontend/src/components/create/StageProgress.tsx`

- [ ] **Step 1: Create WorkerPanel component**

```tsx
// frontend/src/components/create/WorkerPanel.tsx
import { useState } from 'react'

interface WorkerPanelProps {
  name: string
  displayName: string
  content: string
  status: 'running' | 'done'
}

export function WorkerPanel({ displayName, content, status }: WorkerPanelProps) {
  const [expanded, setExpanded] = useState(false)

  return (
    <div className="border rounded-lg p-3 bg-white dark:bg-gray-800">
      <div className="flex items-center justify-between cursor-pointer" onClick={() => setExpanded(!expanded)}>
        <span className="font-medium text-sm">{displayName}</span>
        <div className="flex items-center gap-2">
          {status === 'running' ? (
            <span className="text-xs text-blue-500 animate-pulse">输出中...</span>
          ) : (
            <span className="text-xs text-green-500">完成</span>
          )}
          <span className="text-xs text-gray-400">{expanded ? '收起' : '展开'}</span>
        </div>
      </div>
      {expanded && content && (
        <div className="mt-2 text-sm text-gray-600 dark:text-gray-300 whitespace-pre-wrap max-h-60 overflow-y-auto border-t pt-2">
          {content}
        </div>
      )}
    </div>
  )
}
```

- [ ] **Step 2: Create ParallelStageView component**

```tsx
// frontend/src/components/create/ParallelStageView.tsx
import { WorkerPanel } from './WorkerPanel'

export interface WorkerStream {
  name: string
  displayName: string
  content: string
  status: 'running' | 'done'
}

interface ParallelStageViewProps {
  stageName: string
  workers: WorkerStream[]
  synthContent?: string
  synthStatus?: 'running' | 'done'
}

export function ParallelStageView({ stageName, workers, synthContent, synthStatus }: ParallelStageViewProps) {
  const doneCount = workers.filter(w => w.status === 'done').length

  return (
    <div className="border rounded-xl p-4 bg-gray-50 dark:bg-gray-900 space-y-3">
      <div className="flex items-center justify-between">
        <span className="font-semibold text-sm">{stageName}</span>
        <span className="text-xs text-gray-500">{doneCount}/{workers.length} 完成</span>
      </div>
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-2">
        {workers.map(w => (
          <WorkerPanel key={w.name} {...w} />
        ))}
      </div>
      {synthContent !== undefined && (
        <div className="border-t pt-2">
          <span className="text-xs font-medium text-gray-500">
            {synthStatus === 'running' ? '汇总分析中...' : '汇总完成'}
          </span>
          {synthStatus === 'done' && synthContent && (
            <div className="mt-1 text-sm text-gray-600 whitespace-pre-wrap max-h-40 overflow-y-auto">
              {synthContent}
            </div>
          )}
        </div>
      )}
    </div>
  )
}
```

- [ ] **Step 3: Create StageProgress component**

```tsx
// frontend/src/components/create/StageProgress.tsx
interface StageProgressProps {
  currentStep: number
  totalSteps: number
  stageName: string
}

export function StageProgress({ currentStep, totalSteps, stageName }: StageProgressProps) {
  const percent = Math.round((currentStep / totalSteps) * 100)

  return (
    <div className="flex items-center gap-3 py-2">
      <div className="flex-1 h-2 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden">
        <div
          className="h-full bg-blue-500 rounded-full transition-all duration-500"
          style={{ width: `${percent}%` }}
        />
      </div>
      <span className="text-xs text-gray-500 whitespace-nowrap">
        {currentStep}/{totalSteps} {stageName}
      </span>
    </div>
  )
}
```

- [ ] **Step 4: Verify types compile**

Run: `cd /data/code/content_creator_imm/frontend && npx tsc --noEmit`
Expected: No errors

- [ ] **Step 5: Commit**

```bash
cd /data/code/content_creator_imm
git add frontend/src/components/create/WorkerPanel.tsx frontend/src/components/create/ParallelStageView.tsx frontend/src/components/create/StageProgress.tsx
git commit -m "feat(frontend): add WorkerPanel, ParallelStageView, StageProgress components"
```

---

## Task 13: Frontend Dashboard State Extension

**Files:**
- Modify: `frontend/src/pages/Dashboard.tsx`
- Modify: `frontend/src/components/create/MessageList.tsx`

- [ ] **Step 1: Extend DashState and reducer**

Add to `DashState`:

```typescript
  currentStage: { id: string; name: string; type: string } | null
  activeWorkers: Map<string, WorkerStream>
  synthContent: string
  synthStatus: 'idle' | 'running' | 'done'
  currentStep: number
  totalSteps: number
```

Add new reducer actions:

```typescript
  | { type: 'STAGE_START'; stage_id: string; stage_name: string; stage_type: string }
  | { type: 'STAGE_DONE'; stage_id: string }
  | { type: 'WORKER_START'; stage_id: string; worker_name: string; worker_display: string }
  | { type: 'WORKER_TOKEN'; worker_name: string; content: string }
  | { type: 'WORKER_DONE'; worker_name: string }
  | { type: 'SYNTH_START' }
  | { type: 'SYNTH_TOKEN'; content: string }
  | { type: 'SYNTH_DONE' }
```

Add reducer cases:

```typescript
case 'STAGE_START': {
  return {
    ...state,
    currentStage: { id: action.stage_id, name: action.stage_name, type: action.stage_type },
    activeWorkers: new Map(),
    synthContent: '',
    synthStatus: 'idle' as const,
    currentStep: state.currentStep + 1,
  }
}
case 'STAGE_DONE':
  // If parallel stage just finished, add a ParallelStageView message
  if (state.currentStage?.type === 'parallel' && state.activeWorkers.size > 0) {
    const workersSnapshot = Array.from(state.activeWorkers.values())
    return {
      ...state,
      messages: [...state.messages, {
        id: `${Date.now()}-pstage`,
        type: 'parallel_stage' as ChatMsg['type'],
        workers: workersSnapshot,
        synthContent: state.synthContent,
      }],
      activeWorkers: new Map(),
    }
  }
  return state

case 'WORKER_START': {
  const workers = new Map(state.activeWorkers)
  workers.set(action.worker_name, {
    name: action.worker_name,
    displayName: action.worker_display,
    content: '',
    status: 'running',
  })
  return { ...state, activeWorkers: workers }
}

case 'WORKER_TOKEN': {
  const workers = new Map(state.activeWorkers)
  const w = workers.get(action.worker_name)
  if (w) {
    workers.set(action.worker_name, { ...w, content: w.content + action.content })
  }
  return { ...state, activeWorkers: workers }
}

case 'WORKER_DONE': {
  const workers = new Map(state.activeWorkers)
  const w = workers.get(action.worker_name)
  if (w) {
    workers.set(action.worker_name, { ...w, status: 'done' })
  }
  return { ...state, activeWorkers: workers }
}

case 'SYNTH_START':
  return { ...state, synthStatus: 'running' as const, synthContent: '' }

case 'SYNTH_TOKEN':
  return { ...state, synthContent: state.synthContent + action.content }

case 'SYNTH_DONE':
  return { ...state, synthStatus: 'done' as const }
```

- [ ] **Step 2: Add SSE event handling in runSSE**

In the `switch (event.type)` block inside `runSSE`, add:

```typescript
case 'stage_start':
  dispatch({ type: 'STAGE_START', stage_id: event.stage_id, stage_name: event.stage_name, stage_type: event.stage_type })
  break
case 'stage_done':
  dispatch({ type: 'STAGE_DONE', stage_id: event.stage_id })
  break
case 'worker_start':
  dispatch({ type: 'WORKER_START', stage_id: event.stage_id, worker_name: event.worker_name, worker_display: event.worker_display })
  break
case 'worker_token':
  // For serial stages, also forward to APPEND_TOKEN for the main stream
  if (state.currentStage?.type !== 'parallel') {
    dispatch({ type: 'APPEND_TOKEN', content: event.content })
  }
  dispatch({ type: 'WORKER_TOKEN', worker_name: event.worker_name, content: event.content })
  break
case 'worker_done':
  dispatch({ type: 'WORKER_DONE', worker_name: event.worker_name })
  break
case 'synth_start':
  dispatch({ type: 'SYNTH_START' })
  break
case 'synth_token':
  dispatch({ type: 'SYNTH_TOKEN', content: event.content })
  break
case 'synth_done':
  dispatch({ type: 'SYNTH_DONE' })
  break
```

- [ ] **Step 3: Update MessageList to render parallel_stage messages**

In `MessageList.tsx`, add the `parallel_stage` type to `ChatMsg`:

```typescript
export interface ChatMsg {
  id: string
  type: 'user' | 'ai' | 'step' | 'info' | 'action' | 'similarity' | 'error' | 'outline' | 'parallel_stage'
  content?: string
  options?: string[]
  data?: unknown
  streaming?: boolean
  workers?: WorkerStream[]
  synthContent?: string
}
```

Add a render case for `parallel_stage`:

```tsx
case 'parallel_stage':
  return (
    <ParallelStageView
      stageName={msg.content ?? '研究分析'}
      workers={msg.workers ?? []}
      synthContent={msg.synthContent}
      synthStatus="done"
    />
  )
```

Import the component at the top:

```typescript
import { ParallelStageView, type WorkerStream } from './ParallelStageView'
```

- [ ] **Step 4: Verify types compile and build**

Run: `cd /data/code/content_creator_imm/frontend && npx tsc --noEmit && npm run build`
Expected: No errors

- [ ] **Step 5: Commit**

```bash
cd /data/code/content_creator_imm
git add frontend/src/pages/Dashboard.tsx frontend/src/components/create/MessageList.tsx
git commit -m "feat(frontend): extend Dashboard state for multi-worker parallel display"
```

---

## Task 14: Integration Test and Smoke Test

**Files:**
- Create: `backend/internal/workflow/engine_test.go`

- [ ] **Step 1: Write a mock-based integration test**

```go
// backend/internal/workflow/engine_test.go
package workflow

import (
	"testing"
)

// MockSSEWriter captures SSE events for testing.
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
func (m *MockSSEWriter) SendSynthStart(stageID string)  {}
func (m *MockSSEWriter) SendSynthToken(content string)   {}
func (m *MockSSEWriter) SendSynthDone(stageID string)    {}
func (m *MockSSEWriter) SendStep(step int, name string)  {}
func (m *MockSSEWriter) SendInfo(content string)         {}
func (m *MockSSEWriter) SendOutline(data any)            {}
func (m *MockSSEWriter) SendAction(_ string, options []string) {}
func (m *MockSSEWriter) SendSimilarity(data any)         {}
func (m *MockSSEWriter) SendComplete(scriptID uint)      {}
func (m *MockSSEWriter) SendError(message string)        {}

func TestContextAndInterpolation(t *testing.T) {
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
}
```

- [ ] **Step 2: Run the test**

Run: `cd /data/code/content_creator_imm/backend && go test ./internal/workflow/ -v`
Expected: All tests PASS

- [ ] **Step 3: Manual smoke test**

```bash
# Start the backend
cd /data/code/content_creator_imm/backend && go run .

# In another terminal, test with curl
TOKEN=$(curl -s http://localhost:3004/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@test.com","password":"Test1234"}' | python3 -c "import sys,json; print(json.load(sys.stdin)['token'])")

# Send a message and observe SSE events
curl -N http://localhost:3004/api/chat/message \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"message":"https://example.com/some-video"}'
```

Expected: SSE events stream with `stage_start`, `worker_start`, `worker_token`, `worker_done`, `stage_done` events.

- [ ] **Step 4: Commit test**

```bash
cd /data/code/content_creator_imm
git add backend/internal/workflow/engine_test.go
git commit -m "test(workflow): add mock SSE writer and integration test"
```

---

## Task 15: Final Cleanup and Documentation

**Files:**
- Modify: `backend/internal/service/pipeline.go` — remove deprecated analysis/draft functions
- Modify: `backend/internal/service/prompts.go` — add deprecation comment (keep for reference)

- [ ] **Step 1: Clean up pipeline.go**

Remove `handleIdle`, `handleAwaiting`, `writeFinalDraft` flow from pipeline.go. Keep:
- `ChatSession` struct (with new `ActiveWorkflowID` field)
- `GetOrCreateSession`, `ResetSession`
- `EnsureConversation`, `FlushConversation`
- `PersistMsg`
- `SaveScript` (refactored to standalone)
- `ParseOutlineFromAnalysis`
- `StripQualityCheck` (exported)

Remove from pipeline.go:
- All the StoredMsg-based analysis flow (now handled by Engine)

- [ ] **Step 2: Add deprecation note to prompts.go**

```go
// prompts.go — DEPRECATED: Prompts have been migrated to YAML files in backend/workflows/.
// This file is kept for reference only. The active prompts are in:
//   workflows/viral_script/prompts/*.yaml
//   workflows/viral_script/synth/*.yaml
```

- [ ] **Step 3: Verify full build**

Run:
```bash
cd /data/code/content_creator_imm/backend && go build .
cd /data/code/content_creator_imm/frontend && npx tsc --noEmit && npm run build
```
Expected: Both pass

- [ ] **Step 4: Commit**

```bash
cd /data/code/content_creator_imm
git add -A
git commit -m "refactor: clean up deprecated pipeline code, mark prompts.go as deprecated"
```

---

## Summary

| Task | Description | Files |
|------|-------------|-------|
| 1 | Workflow type definitions | `workflow/types.go` |
| 2 | YAML workflow + prompt files | `workflows/viral_script/**/*.yaml` |
| 3 | YAML loader with tests | `workflow/loader.go`, `loader_test.go` |
| 4 | Context builder with tests | `workflow/context.go`, `context_test.go` |
| 5 | SSE writer interface | `workflow/sse.go` |
| 6 | Worker execution | `workflow/worker.go` |
| 7 | Stage executors | `workflow/stage.go` |
| 8 | DB models + migration | `model/workflow.go`, repos, db.go |
| 9 | Workflow engine | `workflow/engine.go` |
| 10 | Chat handler refactor | `chat_handler.go`, `main.go`, `pipeline.go` |
| 11 | Frontend SSE types | `sse.ts` |
| 12 | Frontend components | `WorkerPanel`, `ParallelStageView`, `StageProgress` |
| 13 | Dashboard state extension | `Dashboard.tsx`, `MessageList.tsx` |
| 14 | Integration test | `engine_test.go` + smoke test |
| 15 | Cleanup + docs | `pipeline.go`, `prompts.go` |
