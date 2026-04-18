package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"content-creator-imm/internal/feishu"
	"content-creator-imm/internal/model"
	"content-creator-imm/internal/repository"
	"content-creator-imm/internal/service"
	"content-creator-imm/internal/workflow"

	"github.com/gin-gonic/gin"
)

// wfLoader is the package-level workflow loader, set by main via SetWorkflowLoader.
var wfLoader *workflow.Loader

// feishuRouter is the package-level feishu router, set by main via SetFeishuRouter.
var feishuRouter *feishu.Router

// SetWorkflowLoader sets the package-level workflow loader (called from main).
func SetWorkflowLoader(l *workflow.Loader) {
	wfLoader = l
}

// SetFeishuRouter sets the package-level feishu router (called from main).
func SetFeishuRouter(r *feishu.Router) {
	feishuRouter = r
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
// IMPORTANT: We only hold the session lock briefly when reading/updating state,
// NOT during the entire request (including LLM API calls) to avoid blocking.
func SendMessage(c *gin.Context) {
	userID := c.GetUint("userID")

	var req struct {
		Message string `json:"message" binding:"required"`
		ConvID  uint   `json:"conv_id"` // optional: client's current conversation ID
		Mock    bool   `json:"mock"`    // optional: mock SSE for E2E testing
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

	// Force write headers to start streaming immediately
	c.Writer.WriteHeaderNow()

	baseSSEW := workflow.NewGinSSEWriter(c.Writer, c.Writer)

	// Mock SSE for E2E testing
	if req.Mock {
		handleMockSSE(baseSSEW, userID)
		return
	}

	sess := service.GetOrCreateSession(userID)

	// --- Phase 1: Read session state (brief lock) ---
	sess.Mu.Lock()
	activeWorkflowID := sess.ActiveWorkflowID
	convID := sess.ConvID

	// Try to recover paused workflow from DB if session lost it
	if activeWorkflowID == 0 {
		pausedWf, err := repository.GetActiveWorkflow(userID)
		if err == nil && pausedWf.Status == "paused" {
			// Found a paused workflow in DB, recover session state
			activeWorkflowID = pausedWf.ID
			sess.ActiveWorkflowID = pausedWf.ID
			// Restore convID from workflow record
			if pausedWf.ConvID != nil && *pausedWf.ConvID > 0 {
				convID = *pausedWf.ConvID
				sess.ConvID = convID
			} else {
				// Fallback: extract from InputJSON
				var input workflow.WorkflowInput
				if jsonErr := json.Unmarshal([]byte(pausedWf.InputJSON), &input); jsonErr == nil && input.ConvID > 0 {
					convID = input.ConvID
					sess.ConvID = convID
				}
			}
			sess.SetState(service.StateAwaiting)
		}
	}

	// If the frontend sent a conv_id that differs from session, try to recover workflow state
	if req.ConvID != 0 && convID != req.ConvID {
		conv, err := repository.GetConversation(req.ConvID, userID)
		if err != nil {
			sess.Mu.Unlock()
			baseSSEW.SendError("会话不存在或无权访问")
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
		activeWorkflowID = 0
	}
	sess.Mu.Unlock() // Release lock BEFORE LLM calls

	message := strings.TrimSpace(req.Message)

	// Send immediate feedback for non-URL input
	baseSSEW.SendInfo("正在启动分析...")

	if activeWorkflowID == 0 {
		// --- New workflow (no lock held during execution) ---

		// Extract text from URL if needed
		var text, sourceURL string
		if service.IsURL(message) {
			baseSSEW.SendInfo("正在提取链接内容...")
			extracted, err := service.ExtractURL(message)
			if err != nil {
				baseSSEW.SendError("链接提取失败：" + err.Error() + "\n\n请直接粘贴文案内容")
				return
			}
			sourceURL = message
			text = extracted
			baseSSEW.SendInfo(fmt.Sprintf("✅ 已提取 %d 字", len(text)))
		} else {
			text = message
		}

		// Create conversation record for this workflow
		title := extractTitle(text)
		convID := service.EnsureConversation(sess, title)

		// Save user message
		userMsg := &model.Message{
			ConversationID: convID,
			Role:           "user",
			Type:           "text",
			Content:        message,
		}
		repository.CreateMessage(userMsg)

		// Load user style
		userStyle := formatUserStyle(userID)

		// --- 新增：输入类型识别和路由 ---
		classifier := workflow.NewInputClassifier()
		inputType := classifier.Classify(text, sourceURL != "")
		route := workflow.GetRoute(inputType)

		// Send input type info
		baseSSEW.SendInfo(fmt.Sprintf("识别输入类型：%s，从「%s」阶段开始", inputType, route.StartStageID))

		// Build workflow input
		input := workflow.WorkflowInput{
			Text:       text,
			SourceURL:  sourceURL,
			UserStyle:  userStyle,
			UserID:     userID,
			ConvID:     convID,
			InputType:  inputType,
		}

		// Create SSE writer that saves messages to DB
		sseW := workflow.NewMessageSavingSSEWriter(baseSSEW, convID)

		// Create and start engine with route (NO lock held during LLM calls)
		engine := workflow.NewEngine(wfLoader, sseW)
		err := engine.StartWithRoute("viral_script", input, route)

		// --- Phase 2: Update session state (brief lock) ---
		wfID := engine.WorkflowID()
		sess.Mu.Lock()
		if err == workflow.ErrWaitingHuman {
			// Workflow paused for human input
			sess.ActiveWorkflowID = wfID
			sess.SetState(service.StateAwaiting)
		} else if err != nil {
			// Workflow failed — engine already sent SSE error
			sess.ActiveWorkflowID = 0
			sess.SetState(service.StateIdle)
		} else {
			// Workflow completed - update conversation with script_id
			sess.ActiveWorkflowID = 0
			sess.SetState(service.StateComplete)
			// Get script_id from workflow output and update conversation
			if wf, wfErr := repository.GetWorkflow(wfID); wfErr == nil && wf.OutputJSON != "" {
				var output map[string]interface{}
				if jsonErr := json.Unmarshal([]byte(wf.OutputJSON), &output); jsonErr == nil {
					if scriptID, ok := output["script_id"].(float64); ok && scriptID > 0 {
						repository.UpdateConversationMeta(convID, map[string]interface{}{
							"state":     1,
							"script_id": uint(scriptID),
						})
					}
				}
			}
		}
		sess.Mu.Unlock()
	} else {
		// --- Resume paused workflow (no lock held during execution) ---
		convID := sess.ConvID

		// Fallback: get convID from workflow record if session lost it
		if convID == 0 {
			wf, wfErr := repository.GetWorkflow(activeWorkflowID)
			if wfErr == nil {
				if wf.ConvID != nil && *wf.ConvID > 0 {
					convID = *wf.ConvID
					sess.ConvID = convID
				} else {
					// Last fallback: extract from InputJSON
					var input workflow.WorkflowInput
					if jsonErr := json.Unmarshal([]byte(wf.InputJSON), &input); jsonErr == nil && input.ConvID > 0 {
						convID = input.ConvID
						sess.ConvID = convID
					}
				}
			}
		}

		// Save user message
		userMsg := &model.Message{
			ConversationID: convID,
			Role:           "user",
			Type:           "text",
			Content:        message,
		}
		repository.CreateMessage(userMsg)

		// Create SSE writer that saves messages to DB
		sseW := workflow.NewMessageSavingSSEWriter(baseSSEW, convID)

		engine := workflow.NewEngine(wfLoader, sseW)
		err := engine.Resume(activeWorkflowID, message)

		// --- Phase 2: Update session state (brief lock) ---
		sess.Mu.Lock()
		if err == workflow.ErrWaitingHuman {
			// Still paused (e.g. user sent adjustment note, engine re-paused)
			sess.SetState(service.StateAwaiting)
		} else if err != nil {
			// Workflow failed
			sess.ActiveWorkflowID = 0
			sess.SetState(service.StateIdle)
		} else {
			// Workflow completed - update conversation with script_id
			sess.ActiveWorkflowID = 0
			sess.SetState(service.StateComplete)
			// Get script_id from workflow output and update conversation
			if wf, wfErr := repository.GetWorkflow(activeWorkflowID); wfErr == nil && wf.OutputJSON != "" {
				var output map[string]interface{}
				if jsonErr := json.Unmarshal([]byte(wf.OutputJSON), &output); jsonErr == nil {
					if scriptID, ok := output["script_id"].(float64); ok && scriptID > 0 {
						repository.UpdateConversationMeta(convID, map[string]interface{}{
							"state":     1,
							"script_id": uint(scriptID),
						})
					}
				}
			}
		}
		sess.Mu.Unlock()
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

// extractTitle generates a simple title from user input
func extractTitle(input string) string {
	// Remove extra whitespace and newlines
	input = strings.TrimSpace(input)
	input = strings.ReplaceAll(input, "\n", " ")
	input = strings.ReplaceAll(input, "\r", " ")

	// Remove common prefixes
	for _, prefix := range []string{"请帮我", "帮我", "我想", "能不能", "可以", "麻烦"} {
		if strings.HasPrefix(input, prefix) {
			input = strings.TrimPrefix(input, prefix)
			break
		}
	}
	input = strings.TrimSpace(input)

	// Truncate to 25 characters
	runes := []rune(input)
	if len(runes) > 25 {
		return string(runes[:25]) + "..."
	}
	return input
}

// GetProfile returns user profile + style
func GetProfile(c *gin.Context) {
	userID := c.GetUint("userID")
	style, _ := repository.GetStyleByUserID(userID)
	c.JSON(http.StatusOK, gin.H{"style": style})
}

// GetStyleDoc returns user style document for frontend initialization check
func GetStyleDoc(c *gin.Context) {
	userID := c.GetUint("userID")
	style, err := repository.GetStyleByUserID(userID)
	if err != nil || style == nil {
		c.JSON(http.StatusOK, gin.H{
			"is_initialized": false,
		})
		return
	}
	// Consider initialized if user has set language style or emotion tone
	isInitialized := style.LanguageStyle != "" || style.EmotionTone != ""
	c.JSON(http.StatusOK, gin.H{
		"is_initialized": isInitialized,
	})
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

// DeleteConversation deletes a conversation and its messages
func DeleteConversation(c *gin.Context) {
	userID := c.GetUint("userID")
	var params struct {
		ID uint `uri:"id"`
	}
	if err := c.ShouldBindUri(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := repository.DeleteConversation(params.ID, userID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "会话不存在或删除失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "会话已删除"})
}

// handleMockSSE sends a mock SSE stream for E2E testing
func handleMockSSE(sseW workflow.SSEWriter, userID uint) {
	// Step 1: Analyze viral elements
	sseW.SendStep(1, "分析爆款元素")
	sseW.SendInfo("正在分析文案结构...")

	// Stage: parallel analysis
	sseW.SendStageStart("mock-stage-1", "并行分析", "parallel")
	sseW.SendWorkerStart("mock-stage-1", "hook_analyzer", "开头钩子分析")
	sseW.SendWorkerToken("hook_analyzer", "发现强力开头钩子：\"今天给大家分享一个超级实用的学习方法\"")
	sseW.SendWorkerDone("hook_analyzer")

	sseW.SendWorkerStart("mock-stage-1", "emotion_analyzer", "情绪曲线分析")
	sseW.SendWorkerToken("emotion_analyzer", "情绪节奏：先铺垫痛点 → 给出解决方案 → 强力结尾")
	sseW.SendWorkerDone("emotion_analyzer")

	sseW.SendWorkerStart("mock-stage-1", "structure_analyzer", "结构拆解")
	sseW.SendWorkerToken("structure_analyzer", "文案结构：问题引入 → 方法介绍 → 原理解释 → 呼吁行动")
	sseW.SendWorkerDone("structure_analyzer")

	sseW.SendStageDone("mock-stage-1")

	// Synthesis
	sseW.SendSynthStart("mock-stage-1")
	sseW.SendSynthToken("综合分析结果：该文案采用经典的教学分享结构，开头有吸引力的钩子，中间有清晰的逻辑递进。")
	sseW.SendSynthDone("mock-stage-1")

	// Step 2: Generate outline
	sseW.SendStep(2, "生成创作大纲")

	// Send outline data
	outlineData := map[string]any{
		"elements": []string{"开头钩子：提问式引入", "情绪共鸣：痛点场景", "解决方案：简洁步骤", "结尾呼吁：立即行动"},
		"outline": []map[string]string{
			{"part": "开头", "duration": "5-8秒", "content": "提问引入，制造悬念", "emotion": "好奇"},
			{"part": "痛点", "duration": "10-15秒", "content": "描述用户痛点场景", "emotion": "共情"},
			{"part": "方案", "duration": "20-30秒", "content": "给出具体解决方案", "emotion": "期待"},
			{"part": "结尾", "duration": "5-8秒", "content": "呼吁行动，强化记忆", "emotion": "激励"},
		},
		"strategy":           "保持原有结构，替换具体内容，调整语言风格为更口语化",
		"estimated_similarity": "<30%",
	}
	sseW.SendOutline(outlineData)

	// Send action options
	sseW.SendAction("请选择创作方案：", []string{"方案1：保守改写", "方案2：激进改写", "方案3：风格迁移", "方案4：全新创作"})

	// For mock, we directly send complete (simulating user selected option 1)
	sseW.SendStep(3, "撰写终稿")

	// Serial stage for writing
	sseW.SendStageStart("mock-stage-2", "撰写终稿", "serial")
	sseW.SendWorkerStart("mock-stage-2", "writer", "终稿撰写")

	mockScript := `各位朋友，今天要分享一个超实用的效率秘籍！

你是不是经常学习效率很低？明明花了很多时间，却什么都记不住？

告诉你，有个方法叫番茄工作法，25分钟专注+5分钟休息，循环进行。

为什么有效？因为这刚好符合我们大脑的注意力极限！

试一下，你会发现：原来学习可以这么高效！

赶紧行动起来吧！`

	sseW.SendWorkerToken("writer", mockScript)
	sseW.SendWorkerDone("writer")
	sseW.SendStageDone("mock-stage-2")

	// Similarity check
	sseW.SendSimilarity(map[string]any{"score": 18, "status": "通过"})

	// Complete with mock script ID
	sseW.SendComplete(9999)
}
