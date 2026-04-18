package workflow

import "time"

// InputType 定义用户输入类型，用于路由决策
type InputType string

const (
	InputTypeOriginalScript      InputType = "original_script"       // URL 或完整文案，需完整流程
	InputTypeIdea                InputType = "idea"                  // 简短想法/观点，直接生成大纲
	InputTypeOutline             InputType = "outline"               // 用户已提供大纲，直接撰写
	InputTypeDraft               InputType = "draft"                 // 草稿基础上改写润色
	InputTypeScriptWithMaterial  InputType = "script_with_material"  // 原稿 + 用户已有素材
	InputTypeScriptWithOutline   InputType = "script_with_outline"   // 原稿 + 用户已提供大纲
)

type StageType string

const (
	StageParallel StageType = "parallel"
	StageSerial   StageType = "serial"
	StageHuman    StageType = "human"
)

type WorkerDef struct {
	Name          string  `yaml:"name"`
	DisplayName   string  `yaml:"display_name"`
	SystemPrompt  string  `yaml:"system"`
	UserPromptTpl string  `yaml:"user"`
	MaxTokens     int     `yaml:"max_tokens"`
	Temperature   float64 `yaml:"temperature"`
	OutputFormat  string  `yaml:"output_format"`
	SilentOutput  bool    `yaml:"silent_output"` // true 时不流式发送 worker_token，静默执行
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
	SkipIf      string    `yaml:"skip_if"` // Condition expression, e.g. "{{stage.material_check.need_material}} == false"
	Workers     []WorkerDef `yaml:"-"`
	SynthDef    *SynthDef   `yaml:"-"`
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
	OriginalText       string
	SourceURL          string
	UserStyle          string
	WorkflowMeta       map[string]any
	CourseContext      string // 课程内容，用于卖点生成
	FeedbackConstraint string // 用户反馈约束
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
	Text              string
	SourceURL         string
	UserStyle         string
	UserID            uint
	ConvID            uint
	InputType         InputType // 输入类型，用于路由决策
	CourseContext     string    // 课程内容，用于卖点生成
	FeedbackConstraint string    // 用户反馈约束，用于重跑时注入 prompt
}
