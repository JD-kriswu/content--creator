package workflow

import "time"

type StageType string

const (
	StageParallel StageType = "parallel"
	StageSerial   StageType = "serial"
	StageHuman    StageType = "human"
)

type WorkerDef struct {
	Name         string  `yaml:"name"`
	DisplayName  string  `yaml:"display_name"`
	SystemPrompt string  `yaml:"system"`
	UserPromptTpl string `yaml:"user"`
	MaxTokens    int     `yaml:"max_tokens"`
	Temperature  float64 `yaml:"temperature"`
	OutputFormat string  `yaml:"output_format"`
}

type SynthDef struct {
	Name         string  `yaml:"name"`
	DisplayName  string  `yaml:"display_name"`
	SystemPrompt string  `yaml:"system"`
	UserPromptTpl string `yaml:"user"`
	MaxTokens    int     `yaml:"max_tokens"`
	Temperature  float64 `yaml:"temperature"`
	OutputFormat string  `yaml:"output_format"`
}

type StageDef struct {
	ID          string    `yaml:"id"`
	DisplayName string    `yaml:"display_name"`
	Type        StageType `yaml:"type"`
	WorkerNames []string  `yaml:"workers"`
	SynthPath   string    `yaml:"synth_prompt"`
	HumanPrompt string   `yaml:"prompt"`
	Options     []string  `yaml:"options"`
	Workers  []WorkerDef `yaml:"-"`
	SynthDef *SynthDef   `yaml:"-"`
}

type WorkflowDef struct {
	Type        string            `yaml:"type"`
	DisplayName string            `yaml:"display_name"`
	Meta        map[string]any    `yaml:"meta"`
	Stages      []StageDef        `yaml:"stages"`
}

type WorkerOutput struct {
	Name     string
	Content  string
	Tokens   int
	Duration time.Duration
}

type StageOutput struct {
	StageID string
	Workers []WorkerOutput
	Summary string
}

type SharedContext struct {
	OriginalText string
	SourceURL    string
	UserStyle    string
	WorkflowMeta map[string]any
}

type WorkflowContext struct {
	Shared       SharedContext
	StageOutputs map[string]*StageOutput
	HumanInputs  map[string]string
}

func NewWorkflowContext(shared SharedContext) *WorkflowContext {
	return &WorkflowContext{
		Shared:       shared,
		StageOutputs: make(map[string]*StageOutput),
		HumanInputs:  make(map[string]string),
	}
}

type WorkerInput struct {
	SystemPrompt string
	UserPrompt   string
	MaxTokens    int
	Temperature  float64
}

type WorkflowInput struct {
	Text      string
	SourceURL string
	UserStyle string
	UserID    uint
}
