package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

type Loader struct {
	basePath string
	devMode  bool
	mu       sync.RWMutex
	cache    map[string]*WorkflowDef
}

func NewLoader(basePath string, devMode bool) *Loader {
	return &Loader{
		basePath: basePath,
		devMode:  devMode,
		cache:    make(map[string]*WorkflowDef),
	}
}

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

func (l *Loader) Reload(workflowType string) (*WorkflowDef, error) {
	return l.loadFromDisk(workflowType)
}

func (l *Loader) loadFromDisk(workflowType string) (*WorkflowDef, error) {
	dir := filepath.Join(l.basePath, workflowType)

	wfPath := filepath.Join(dir, "workflow.yaml")
	data, err := os.ReadFile(wfPath)
	if err != nil {
		return nil, fmt.Errorf("read workflow.yaml: %w", err)
	}
	var def WorkflowDef
	if err := yaml.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("parse workflow.yaml: %w", err)
	}

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

		if stage.SynthPath != "" {
			sd, err := l.loadSynthDef(dir, stage.SynthPath)
			if err != nil {
				return nil, fmt.Errorf("stage %s synth: %w", stage.ID, err)
			}
			stage.SynthDef = sd
		}
	}

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
