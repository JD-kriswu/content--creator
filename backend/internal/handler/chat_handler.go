package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"content-creator-imm/internal/model"
	"content-creator-imm/internal/repository"
	"content-creator-imm/internal/service"

	"github.com/gin-gonic/gin"
)

// sseWriter wraps gin context to send SSE events
type sseWriter struct {
	c *gin.Context
}

func (w *sseWriter) send(eventType string, data interface{}) {
	b, _ := json.Marshal(data)
	fmt.Fprintf(w.c.Writer, "data: %s\n\n", b)
	w.c.Writer.Flush()
}

func (w *sseWriter) sendToken(token string) {
	w.send("msg", map[string]string{"type": "token", "content": token})
}

func (w *sseWriter) sendStep(step int, name string) {
	w.send("msg", map[string]interface{}{"type": "step", "step": step, "name": name})
}

func (w *sseWriter) sendOutline(data *service.OutlineData) {
	w.send("msg", map[string]interface{}{"type": "outline", "data": data})
}

func (w *sseWriter) sendAction(options []string) {
	w.send("msg", map[string]interface{}{"type": "action", "options": options})
}

func (w *sseWriter) sendError(msg string) {
	w.send("msg", map[string]string{"type": "error", "message": msg})
}

func (w *sseWriter) sendComplete(scriptID uint) {
	w.send("msg", map[string]interface{}{"type": "complete", "scriptId": scriptID})
}

func (w *sseWriter) sendInfo(msg string) {
	w.send("msg", map[string]string{"type": "info", "content": msg})
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
	service.ResetSession(userID)
	c.JSON(http.StatusOK, gin.H{"message": "session reset"})
}

// SendMessage handles user messages with SSE streaming response
func SendMessage(c *gin.Context) {
	userID := c.GetUint("userID")

	var req struct {
		Message string `json:"message" binding:"required"`
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

	w := &sseWriter{c}
	sess := service.GetOrCreateSession(userID)
	sess.Mu.Lock()
	defer sess.Mu.Unlock()

	switch sess.State {
	case service.StateIdle:
		handleIdle(w, sess, userID, req.Message)

	case service.StateAwaiting:
		handleAwaiting(w, sess, userID, req.Message)

	default:
		w.sendError("正在处理中，请稍候...")
	}
}

func handleIdle(w *sseWriter, sess *service.ChatSession, userID uint, input string) {
	sess.State = service.StateAnalyzing
	input = strings.TrimSpace(input)

	// Step 1: Get original text
	w.sendStep(1, "获取原稿内容")
	var originalText string
	if service.IsURL(input) {
		w.sendInfo("正在提取链接内容...")
		text, err := service.ExtractURL(input)
		if err != nil {
			w.sendError("链接提取失败：" + err.Error() + "\n\n请直接粘贴文案内容")
			sess.State = service.StateIdle
			return
		}
		sess.SourceURL = input
		originalText = text
		w.sendInfo(fmt.Sprintf("✅ 已提取 %d 字", len(text)))
	} else {
		originalText = input
	}
	sess.OriginalText = originalText

	// Step 2: Get user style
	w.sendStep(2, "读取风格档案")
	var styleProfile *service.StyleProfile
	style, err := repository.GetStyleByUserID(userID)
	if err == nil && style != nil {
		styleProfile = &service.StyleProfile{
			LanguageStyle: style.LanguageStyle,
			EmotionTone:   style.EmotionTone,
			OpeningStyle:  style.OpeningStyle,
			ClosingStyle:  style.ClosingStyle,
			Catchphrases:  style.Catchphrases,
		}
		w.sendInfo("✅ 已加载个人风格档案")
	} else {
		w.sendInfo("⚠️ 暂无风格档案，使用通用爆款风格")
	}

	// Step 3-4: 5-role analysis + debate (single streaming call)
	w.sendStep(3, "5角色并行分析（含辩论决策）")
	prompt := service.BuildAnalysisPrompt(originalText, styleProfile)

	var fullAnalysis strings.Builder
	_, err = service.StreamClaude("你是专业的短视频创作分析系统。", prompt, func(token string) bool {
		fullAnalysis.WriteString(token)
		w.sendToken(token)
		return true
	})
	if err != nil {
		w.sendError("分析失败：" + err.Error())
		sess.State = service.StateIdle
		return
	}

	sess.AnalysisFull = fullAnalysis.String()

	// Step 5: Parse outline and show for confirmation
	w.sendStep(5, "大纲生成完成，等待确认")
	outlineData, outlineJSON := service.ParseOutlineFromAnalysis(sess.AnalysisFull)
	sess.OutlineJSON = outlineJSON
	sess.OutlineData = outlineData

	if outlineData != nil {
		w.sendOutline(outlineData)
	}
	w.sendAction([]string{"1. ✅ 确认，开始撰写终稿", "2. 🔄 调整大纲（请说明方向）", "3. 🔄 更换素材方向", "4. 🔙 重新分析"})
	sess.State = service.StateAwaiting
}

func handleAwaiting(w *sseWriter, sess *service.ChatSession, userID uint, input string) {
	input = strings.TrimSpace(input)
	choice := strings.ToLower(input)

	switch {
	case choice == "1" || strings.HasPrefix(choice, "确认") || strings.HasPrefix(choice, "1."):
		writeFinalDraft(w, sess, userID, "")

	case choice == "4" || strings.HasPrefix(choice, "重新"):
		sess.State = service.StateIdle
		w.sendInfo("已重置，请重新输入原稿或链接。")

	default:
		// User wants to adjust - treat their message as a note and re-analyze
		if strings.HasPrefix(choice, "2") || strings.HasPrefix(input, "调整") {
			sess.UserNote = input
			w.sendInfo("✅ 已记录调整意见，将在撰写时参考。直接输入 \"1\" 或 \"确认\" 开始撰写，或继续说明要求。")
		} else if strings.HasPrefix(choice, "3") || strings.HasPrefix(input, "更换") {
			sess.State = service.StateIdle
			w.sendInfo("请重新输入原稿，并说明希望的素材方向。")
		} else {
			// Treat as additional note
			sess.UserNote = input
			w.sendInfo("已记录您的要求。输入 \"1\" 开始按此方向撰写终稿，或继续调整。")
		}
	}
}

func writeFinalDraft(w *sseWriter, sess *service.ChatSession, userID uint, _ string) {
	sess.State = service.StateWriting

	// Step 6: Write final draft
	w.sendStep(6, "撰写终稿")
	prompt := service.BuildFinalDraftPrompt(sess.OriginalText, sess.OutlineJSON, sess.UserNote)

	var draftBuilder strings.Builder
	_, err := service.StreamClaude("你是专业的短视频口播稿撰写专家，擅长写出口语化、高传播力的内容。", prompt, func(token string) bool {
		draftBuilder.WriteString(token)
		w.sendToken(token)
		return true
	})
	if err != nil {
		w.sendError("终稿撰写失败：" + err.Error())
		sess.State = service.StateAwaiting
		return
	}
	sess.FinalDraft = draftBuilder.String()

	// Step 7-8: Similarity check (non-streaming)
	w.sendStep(8, "相似度检测")
	simPrompt := service.BuildSimilarityCheckPrompt(sess.OriginalText, sess.FinalDraft)
	simResult, err := service.CallClaude("", simPrompt)

	var simScore float64 = 0
	var viralScore float64 = 7.5

	if err == nil {
		// Try to parse JSON
		var scores struct {
			Vocab     float64 `json:"vocab"`
			Sentence  float64 `json:"sentence"`
			Structure float64 `json:"structure"`
			Viewpoint float64 `json:"viewpoint"`
			Total     float64 `json:"total"`
		}
		// Find JSON in response
		start := strings.Index(simResult, "{")
		end := strings.LastIndex(simResult, "}")
		if start >= 0 && end > start {
			if json.Unmarshal([]byte(simResult[start:end+1]), &scores) == nil {
				simScore = scores.Total / 100.0
				w.send("msg", map[string]interface{}{
					"type": "similarity",
					"data": scores,
				})
			}
		}
	}

	// Step 9: Save
	w.sendStep(9, "保存稿件")
	script, err := service.SaveScript(userID, sess, simScore, viralScore)
	if err != nil {
		w.sendError("保存失败：" + err.Error())
		// Still complete - user has the content
		w.sendComplete(0)
	} else {
		w.sendComplete(script.ID)
	}

	sess.State = service.StateComplete
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
