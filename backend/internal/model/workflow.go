package model

import "time"

// Workflow represents a workflow execution instance.
type Workflow struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	UserID      uint       `gorm:"index;not null" json:"user_id"`
	Type        string     `gorm:"size:64;index" json:"type"`
	Status      string     `gorm:"size:32;index" json:"status"` // pending/running/paused/completed/failed
	InputJSON   string     `gorm:"column:input_json;type:text" json:"-"`
	ContextJSON string     `gorm:"column:context_json;type:longtext" json:"-"`
	OutputJSON  string     `gorm:"column:output_json;type:longtext" json:"-"`
	ConvID      *uint      `gorm:"index" json:"conv_id,omitempty"`
	Error       string     `gorm:"type:text" json:"error,omitempty"`
	PausedAt    *time.Time `json:"paused_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// WorkflowStage represents a single stage execution within a workflow.
type WorkflowStage struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	WorkflowID uint       `gorm:"index;not null" json:"workflow_id"`
	StageID    string     `gorm:"size:64;index" json:"stage_id"`
	Type       string     `gorm:"size:32" json:"type"` // parallel/serial/human
	Sequence   int        `json:"sequence"`
	Status     string     `gorm:"size:32" json:"status"` // pending/running/completed/failed
	InputJSON  string     `gorm:"column:input_json;type:text" json:"-"`
	OutputJSON string     `gorm:"column:output_json;type:longtext" json:"-"`
	StartedAt  *time.Time `json:"started_at,omitempty"`
	EndedAt    *time.Time `json:"ended_at,omitempty"`
}

// WorkflowWorker represents a single worker execution within a stage.
type WorkflowWorker struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	StageID    uint       `gorm:"index;not null" json:"stage_id"`
	WorkflowID uint       `gorm:"index;not null" json:"workflow_id"`
	WorkerName string     `gorm:"size:64" json:"worker_name"`
	Role       string     `gorm:"size:64" json:"role"`
	Status     string     `gorm:"size:32" json:"status"` // pending/running/completed/failed
	InputJSON  string     `gorm:"column:input_json;type:text" json:"-"`
	OutputJSON string     `gorm:"column:output_json;type:longtext" json:"-"`
	TokensUsed int        `json:"tokens_used"`
	DurationMs int64      `json:"duration_ms"`
	StartedAt  *time.Time `json:"started_at,omitempty"`
	EndedAt    *time.Time `json:"ended_at,omitempty"`
}
