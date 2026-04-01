package repository

import (
	"content-creator-imm/internal/db"
	"content-creator-imm/internal/model"
)

// CreateWorkflow inserts a new workflow record.
func CreateWorkflow(w *model.Workflow) error {
	return db.DB.Create(w).Error
}

// UpdateWorkflow saves all fields of the workflow.
func UpdateWorkflow(w *model.Workflow) error {
	return db.DB.Save(w).Error
}

// GetWorkflow retrieves a workflow by ID.
func GetWorkflow(id uint) (*model.Workflow, error) {
	var w model.Workflow
	if err := db.DB.First(&w, id).Error; err != nil {
		return nil, err
	}
	return &w, nil
}

// GetActiveWorkflow finds a running or paused workflow for the given user.
func GetActiveWorkflow(userID uint) (*model.Workflow, error) {
	var w model.Workflow
	err := db.DB.Where("user_id = ? AND status IN ?", userID, []string{"running", "paused"}).
		Order("created_at DESC").
		First(&w).Error
	if err != nil {
		return nil, err
	}
	return &w, nil
}

// CreateWorkflowStage inserts a new workflow stage record.
func CreateWorkflowStage(s *model.WorkflowStage) error {
	return db.DB.Create(s).Error
}

// UpdateWorkflowStage saves all fields of the workflow stage.
func UpdateWorkflowStage(s *model.WorkflowStage) error {
	return db.DB.Save(s).Error
}

// CreateWorkflowWorker inserts a new workflow worker record.
func CreateWorkflowWorker(w *model.WorkflowWorker) error {
	return db.DB.Create(w).Error
}

// UpdateWorkflowWorker saves all fields of the workflow worker.
func UpdateWorkflowWorker(w *model.WorkflowWorker) error {
	return db.DB.Save(w).Error
}
