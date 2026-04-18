package handler

import (
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

// promptsDir is the base directory for prompt YAML files
// Relative to the backend working directory
const promptsDir = "workflows/viral_script"

// PromptMeta contains metadata about a prompt file
type PromptMeta struct {
	Name        string `yaml:"name"`
	DisplayName string `yaml:"display_name"`
}

// PromptFile represents a prompt file with its path and metadata
type PromptFile struct {
	Path        string `json:"path"`         // Relative path like "prompts/viral_decoder.yaml"
	Name        string `json:"name"`         // Prompt name from YAML
	DisplayName string `json:"display_name"` // Display name from YAML
	Content     string `json:"content"`      // Full YAML content
}

// GetPrompts returns list of all prompt files with their content
func GetPrompts(c *gin.Context) {
	basePath := promptsDir
	files := []PromptFile{}

	// Add charter file first (if exists)
	charterPath := filepath.Join(basePath, "_charter.yaml")
	if content, err := ioutil.ReadFile(charterPath); err == nil {
		var meta PromptMeta
		if err := yaml.Unmarshal(content, &meta); err != nil {
			meta.Name = "_charter"
			meta.DisplayName = "系统宪章"
		}
		files = append(files, PromptFile{
			Path:        "_charter.yaml",
			Name:        meta.Name,
			DisplayName: meta.DisplayName,
			Content:     string(content),
		})
	}

	// Walk through prompts directory
	promptsPath := filepath.Join(basePath, "prompts")
	if entries, err := ioutil.ReadDir(promptsPath); err == nil {
		for _, entry := range entries {
			if !strings.HasSuffix(entry.Name(), ".yaml") {
				continue
			}

			fullPath := filepath.Join(promptsPath, entry.Name())
			content, err := ioutil.ReadFile(fullPath)
			if err != nil {
				continue
			}

			// Parse YAML to extract metadata
			var meta PromptMeta
			if err := yaml.Unmarshal(content, &meta); err != nil {
				// If parsing fails, use filename as name
				meta.Name = strings.TrimSuffix(entry.Name(), ".yaml")
				meta.DisplayName = meta.Name
			}

			files = append(files, PromptFile{
				Path:        "prompts/" + entry.Name(),
				Name:        meta.Name,
				DisplayName: meta.DisplayName,
				Content:     string(content),
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{"prompts": files})
}

// UpdatePrompt updates a specific prompt file
func UpdatePrompt(c *gin.Context) {
	var req struct {
		Path    string `json:"path" binding:"required"`    // Relative path like "prompts/viral_decoder.yaml"
		Content string `json:"content" binding:"required"` // Full YAML content
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate path to prevent directory traversal
	if strings.Contains(req.Path, "..") || strings.HasPrefix(req.Path, "/") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid path"})
		return
	}

	// Allow: _charter.yaml, prompts/*.yaml
	isValidPath := req.Path == "_charter.yaml" || strings.HasPrefix(req.Path, "prompts/")
	if !isValidPath {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path must be _charter.yaml or in prompts/ directory"})
		return
	}
	if !strings.HasSuffix(req.Path, ".yaml") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file must be .yaml"})
		return
	}

	// Validate YAML syntax
	var meta PromptMeta
	if err := yaml.Unmarshal([]byte(req.Content), &meta); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid YAML: " + err.Error()})
		return
	}

	fullPath := filepath.Join(promptsDir, req.Path)

	// Ensure the directory exists
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create directory"})
		return
	}

	// Write the file
	if err := ioutil.WriteFile(fullPath, []byte(req.Content), 0644); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to write file"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "prompt updated",
		"path":         req.Path,
		"name":         meta.Name,
		"display_name": meta.DisplayName,
	})
}