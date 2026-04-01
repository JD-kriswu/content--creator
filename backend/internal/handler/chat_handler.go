package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"content-creator-imm/internal/model"
	"content-creator-imm/internal/repository"
	"content-creator-imm/internal/service"
	"content-creator-imm/internal/workflow"

	"github.com/gin-gonic/gin"
)

// wfLoader is the package-level workflow loader, set by main via SetWorkflowLoader.
var wfLoader *workflow.Loader

// SetWorkflowLoader sets the package-level workflow loader (called from main).
func SetWorkflowLoader(l *workflow.Loader) {
	wfLoader = l
}

// formatUserStyle formats a UserStyle into a readable string for the workflow input.
func formatUserStyle(userID uint) string {
	style, err := repository.GetStyleByUserID(userID)
	if err != nil || style == nil {
		return "用户暂无风格档案，请使用通用爆款风格。"
	}
	var parts []string
	if style.LanguageStyle != "" {
		parts = append(parts, "- 语言风格："+style.LanguageStyle)
	}
	if style.EmotionTone != "" {
		parts = append(parts, "- 情绪基调："+style.EmotionTone)
	}
	if style.OpeningStyle != "" {
		parts = append(parts, "- 典型开场："+style.OpeningStyle)
	}
	if style.ClosingStyle != "" {
		parts = append(parts, "- 典型结尾："+style.ClosingStyle)
	}
	if style.Catchphrases != "" {
		parts = append(parts, "- 标志性元素："+style.Catchphrases)
	}
	if len(parts) == 0 {
		return "用户暂无风格档案，请使用通用爆款风格。"
	}
	return strings.Join(parts, "\n")
}

// GetSession returns current session state
func GetSession(c *gin.Context) {
	userID := c.GetUint("userID")
	sess := service.GetOrCreateSession(userID)
	c.JSON(http.StatusOK, gin.H{
		"session_id": sess.ID,
		"state":      sess.State,
	})
}

// ResetSession resets the session to idle
func ResetSession(c *gin.Context) {
	userID := c.GetUint("userID")
	sess := service.GetOrCreateSession(userID)
	sess.Mu.Lock()
	// Flush in-progress conversation before resetting
	if sess.ConvID != 0 && sess.State != service.StateComplete {
		service.FlushConversation(sess, 0, nil)
	}
	sess.Mu.Unlock()
	service.ResetSession(userID)

	// Create a placeholder conversation for the new session so it appears in the list immediately
	newSess := service.GetOrCreateSession(userID)
	service.EnsureConversation(newSess, "新会话")

	c.JSON(http.StatusOK, gin.H{"message": "session reset", "conv_id": newSess.ConvID})
}

// SendMessage handles user messages with SSE streaming response.
// It delegates all pipeline logic to the workflow engine.
func SendMessage(c *gin.Context) {
	userID := c.GetUint("userID")

	var req struct {
		Message string `json:"message" binding:"required"`
		ConvID  uint   `json:"conv_id"` // optional: client's current conversation ID
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	sseW := workflow.NewGinSSEWriter(c.Writer, c.Writer)

	sess := service.GetOrCreateSession(userID)
	sess.Mu.Lock()
	defer sess.Mu.Unlock()

	// If the frontend sent a conv_id that differs from session, try to recover workflow state
	if req.ConvID != 0 && sess.ConvID != req.ConvID {
		conv, err := repository.GetConversation(req.ConvID, userID)
		if err != nil {
			sseW.SendError("会话不存在或无权访问")
			return
		}
		sess.ConvID = conv.ID
		sess.ActiveWorkflowID = 0
		sess.StoredMsgs = nil
		if conv.State == 1 {
			sess.SetState(service.StateComplete)
		} else {
			sess.SetState(service.StateIdle)
		}
	}

	message := strings.TrimSpace(req.Message)

	if sess.ActiveWorkflowID == 0 {
		// --- New workflow ---

		// Extract text from URL if needed
		var text, sourceURL string
		if service.IsURL(message) {
			sseW.SendInfo("正在提取链接内容...")
			extracted, err := service.ExtractURL(message)
			if err != nil {
				sseW.SendError("链接提取失败：" + err.Error() + "\n\n请直接粘贴文案内容")
				return
			}
			sourceURL = message
			text = extracted
			sseW.SendInfo(fmt.Sprintf("✅ 已提取 %d 字", len(text)))
		} else {
			text = message
		}

		// Load user style
		userStyle := formatUserStyle(userID)

		// Build workflow input
		input := workflow.WorkflowInput{
			Text:      text,
			SourceURL: sourceURL,
			UserStyle: userStyle,
			UserID:    userID,
		}

		// Create and start engine
		engine := workflow.NewEngine(wfLoader, sseW)
		err := engine.Start("viral_script", input)

		// Save the workflow ID to session
		wfID := engine.WorkflowID()
		if err == workflow.ErrWaitingHuman {
			// Workflow paused for human input
			sess.ActiveWorkflowID = wfID
			sess.SetState(service.StateAwaiting)
		} else if err != nil {
			// Workflow failed — engine already sent SSE error
			sess.ActiveWorkflowID = 0
			sess.SetState(service.StateIdle)
		} else {
			// Workflow completed
			sess.ActiveWorkflowID = 0
			sess.SetState(service.StateComplete)
		}
	} else {
		// --- Resume paused workflow ---
		engine := workflow.NewEngine(wfLoader, sseW)
		err := engine.Resume(sess.ActiveWorkflowID, message)

		if err == workflow.ErrWaitingHuman {
			// Still paused (e.g. user sent adjustment note, engine re-paused)
			sess.SetState(service.StateAwaiting)
		} else if err != nil {
			// Workflow failed
			sess.ActiveWorkflowID = 0
			sess.SetState(service.StateIdle)
		} else {
			// Workflow completed
			sess.ActiveWorkflowID = 0
			sess.SetState(service.StateComplete)
		}
	}
}

// GetScripts returns script list
func GetScripts(c *gin.Context) {
	userID := c.GetUint("userID")
	page := 1
	limit := 20
	scripts, total, err := repository.ListScripts(userID, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"scripts": scripts, "total": total})
}

// GetScript returns a single script with content
func GetScript(c *gin.Context) {
	userID := c.GetUint("userID")
	var params struct {
		ID uint `uri:"id"`
	}
	if err := c.ShouldBindUri(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	script, err := repository.GetScript(params.ID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "稿件不存在"})
		return
	}

	// Read content from file
	var content string
	if script.ContentPath != "" {
		data, err := readScriptFile(script.ContentPath)
		if err == nil {
			content = string(data)
		}
	}

	c.JSON(http.StatusOK, gin.H{"script": script, "content": content})
}

func readScriptFile(path string) ([]byte, error) {
	return readFile(path)
}

// GetProfile returns user profile + style
func GetProfile(c *gin.Context) {
	userID := c.GetUint("userID")
	style, _ := repository.GetStyleByUserID(userID)
	c.JSON(http.StatusOK, gin.H{"style": style})
}

// UpdateStyle updates user style profile
func UpdateStyle(c *gin.Context) {
	userID := c.GetUint("userID")
	var s model.UserStyle
	if err := c.ShouldBindJSON(&s); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	s.UserID = userID
	if err := repository.UpsertStyle(&s); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "风格档案已更新"})
}

// GetConversations returns conversation list for current user
func GetConversations(c *gin.Context) {
	userID := c.GetUint("userID")
	list, err := repository.ListConversations(userID, 50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"conversations": list})
}

// GetConversationDetail returns a conversation with its messages from the Message table
func GetConversationDetail(c *gin.Context) {
	userID := c.GetUint("userID")
	var params struct {
		ID uint `uri:"id"`
	}
	if err := c.ShouldBindUri(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	conv, err := repository.GetConversation(params.ID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
		return
	}

	msgs, _ := repository.ListMessagesByConvID(conv.ID)
	storedMsgs := make([]service.StoredMsg, 0, len(msgs))
	for _, m := range msgs {
		sm := service.StoredMsg{
			Role:    m.Role,
			Type:    m.Type,
			Content: m.Content,
			Step:    m.Step,
			Name:    m.Name,
		}
		if m.DataJSON != "" && m.DataJSON != "null" {
			sm.Data = json.RawMessage(m.DataJSON)
		}
		if m.OptionsJSON != "" && m.OptionsJSON != "null" {
			var opts []string
			_ = json.Unmarshal([]byte(m.OptionsJSON), &opts)
			sm.Options = opts
		}
		storedMsgs = append(storedMsgs, sm)
	}
	msgsJSON, _ := json.Marshal(storedMsgs)
	c.JSON(http.StatusOK, gin.H{"conversation": conv, "messages": string(msgsJSON)})
}
