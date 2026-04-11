package feishu

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"content-creator-imm/internal/model"
	"content-creator-imm/internal/repository"
	"content-creator-imm/internal/service"
	"content-creator-imm/internal/workflow"
)

// Router handles WebSocket events and routes them to workflow engine.
type Router struct {
	loader *workflow.Loader
}

// NewRouter creates a new Router with the given workflow loader.
func NewRouter(loader *workflow.Loader) *Router {
	return &Router{loader: loader}
}

// HandleEvent processes incoming WebSocket events.
// It routes events based on type: im.message.receive_v1 or card.action.trigger.
func (r *Router) HandleEvent(event WSEvent) {
	// Get bot info from repository
	bot, err := repository.GetFeishuBotByAppID(event.AppID)
	if err != nil {
		log.Printf("[FeishuRouter] bot not found for app_id=%s: %v", event.AppID, err)
		return
	}

	switch event.Type {
	case "im.message.receive_v1":
		r.handleMessage(event, bot)
	case "card.action.trigger":
		r.handleCardAction(event, bot)
	default:
		log.Printf("[FeishuRouter] unknown event type: %s", event.Type)
	}
}

// handleMessage processes im.message.receive_v1 events.
func (r *Router) handleMessage(wsEvent WSEvent, bot *model.FeishuBot) {
	var msgEvent MessageEvent
	if err := json.Unmarshal(wsEvent.Event, &msgEvent); err != nil {
		log.Printf("[FeishuRouter] failed to parse message event: %v", err)
		return
	}

	chatID := msgEvent.Message.ChatID
	openID := msgEvent.Sender.OpenID
	content := msgEvent.Message.Content

	// Check if session is busy (analyzing/writing)
	sessMgr := service.GetFeishuSessionMgr()
	if sessMgr.IsBusy(chatID) {
		log.Printf("[FeishuRouter] session busy, ignoring message from %s", chatID)
		// Send a quick response that we're busy
		r.sendBusyMessage(chatID, bot)
		return
	}

	// Get or create feishu user
	feishuUser, err := repository.GetOrCreateFeishuUserByOpenID(openID)
	if err != nil {
		log.Printf("[FeishuRouter] failed to get/create feishu user: %v", err)
		return
	}

	// Get or create session
	sess := sessMgr.GetOrCreate(chatID, bot.ID, feishuUser.UserID, feishuUser.ID)

	// Route based on session state
	switch sess.State {
	case service.FeishuIdle:
		r.handleIdle(chatID, bot, feishuUser, content, sess)
	case service.FeishuAwaiting:
		r.handleAwaiting(chatID, bot, content, sess)
	default:
		log.Printf("[FeishuRouter] unexpected state %s for chat %s", sess.State, chatID)
	}
}

// handleIdle processes messages when session is idle (start new workflow).
func (r *Router) handleIdle(chatID string, bot *model.FeishuBot, feishuUser *model.FeishuUser, content string, sess *service.FeishuSession) {
	// Parse message content (飞书消息内容是 JSON string)
	text := parseFeishuMessageContent(content)
	if text == "" {
		log.Printf("[FeishuRouter] empty message content from %s", chatID)
		return
	}

	// Get or create feishu conversation mapping
	_, convID, err := repository.GetOrCreateFeishuConv(bot.ID, chatID, feishuUser.UserID)
	if err != nil {
		log.Printf("[FeishuRouter] failed to get/create feishu conv: %v", err)
		return
	}

	// Update session with conv ID
	sessMgr := service.GetFeishuSessionMgr()
	sessMgr.SetConvID(chatID, convID)

	// Create SSE writer for Feishu card updates
	sseW := NewFeishuSSEWriter(chatID, bot.AppID, bot.AppSecret, 500) // 500ms throttle
	if err := sseW.Init(); err != nil {
		log.Printf("[FeishuRouter] failed to init SSE writer: %v", err)
		return
	}

	// Extract text from URL if needed
	var sourceText, sourceURL string
	if service.IsURL(text) {
		sseW.SendInfo("正在提取链接内容...")
		extracted, err := service.ExtractURL(text)
		if err != nil {
			sseW.SendError("链接提取失败：" + err.Error())
			return
		}
		sourceURL = text
		sourceText = extracted
		sseW.SendInfo(fmt.Sprintf("已提取 %d 字", len(sourceText)))
	} else {
		sourceText = text
	}

	// Classify input type
	classifier := workflow.NewInputClassifier()
	inputType := classifier.Classify(sourceText, sourceURL != "")
	route := workflow.GetRoute(inputType)

	sseW.SendInfo(fmt.Sprintf("识别输入类型：%s，从「%s」阶段开始", inputType, route.StartStageID))

	// Build workflow input
	input := workflow.WorkflowInput{
		Text:      sourceText,
		SourceURL: sourceURL,
		UserStyle: formatFeishuUserStyle(feishuUser.UserID),
		UserID:    feishuUser.UserID,
		ConvID:    convID,
		InputType: inputType,
	}

	// Update session state to analyzing
	sessMgr.SetState(chatID, service.FeishuAnalyzing)

	// Create and start engine
	engine := workflow.NewEngine(r.loader, sseW)
	err = engine.StartWithRoute("viral_script", input, route)

	// Update session state after workflow completes
	sessMgr.SetWorkflowID(chatID, engine.WorkflowID())
	if err == workflow.ErrWaitingHuman {
		sessMgr.SetState(chatID, service.FeishuAwaiting)
	} else if err != nil {
		sessMgr.SetState(chatID, service.FeishuIdle)
	} else {
		sessMgr.SetState(chatID, service.FeishuIdle)
	}
}

// handleAwaiting processes messages when session is awaiting user confirmation.
func (r *Router) handleAwaiting(chatID string, bot *model.FeishuBot, content string, sess *service.FeishuSession) {
	// Parse message content
	text := parseFeishuMessageContent(content)
	if text == "" {
		return
	}

	// Check if user wants to cancel/start new
	if strings.Contains(text, "取消") || strings.Contains(text, "重新") {
		// Clear session and let user start fresh
		service.GetFeishuSessionMgr().Clear(chatID)
		return
	}

	// Get workflow ID from session
	if sess.WorkflowID == 0 {
		log.Printf("[FeishuRouter] no workflow ID in awaiting state for %s", chatID)
		service.GetFeishuSessionMgr().SetState(chatID, service.FeishuIdle)
		return
	}

	// Create SSE writer for card updates
	sseW := NewFeishuSSEWriter(chatID, bot.AppID, bot.AppSecret, 500)
	if err := sseW.Init(); err != nil {
		log.Printf("[FeishuRouter] failed to init SSE writer: %v", err)
		return
	}

	// Update session state to writing
	sessMgr := service.GetFeishuSessionMgr()
	sessMgr.SetState(chatID, service.FeishuWriting)

	// Resume workflow with user input
	engine := workflow.NewEngine(r.loader, sseW)
	err := engine.Resume(sess.WorkflowID, text)

	// Update session state after workflow completes
	if err == workflow.ErrWaitingHuman {
		sessMgr.SetState(chatID, service.FeishuAwaiting)
	} else if err != nil {
		sessMgr.SetState(chatID, service.FeishuIdle)
	} else {
		sessMgr.SetState(chatID, service.FeishuIdle)
	}
}

// handleCardAction processes card.action.trigger events (button clicks).
func (r *Router) handleCardAction(wsEvent WSEvent, bot *model.FeishuBot) {
	var cardEvent CardActionEvent
	if err := json.Unmarshal(wsEvent.Event, &cardEvent); err != nil {
		log.Printf("[FeishuRouter] failed to parse card action event: %v", err)
		return
	}

	chatID := cardEvent.ChatID

	// Get session
	sessMgr := service.GetFeishuSessionMgr()
	sess := sessMgr.Get(chatID)
	if sess == nil || sess.State != service.FeishuAwaiting {
		log.Printf("[FeishuRouter] ignoring card action for non-awaiting session %s", chatID)
		return
	}

	// Extract action value
	actionValue := cardEvent.Action.Value
	var userChoice string
	if v, ok := actionValue["choice"]; ok {
		userChoice = v
	} else if v, ok := actionValue["action"]; ok {
		userChoice = v
	} else {
		// Default: use first value if available
		for _, v := range actionValue {
			userChoice = v
			break
		}
	}

	if userChoice == "" {
		log.Printf("[FeishuRouter] no action value in card event for %s", chatID)
		return
	}

	// Treat as resume input
	r.handleAwaiting(chatID, bot, userChoice, sess)
}

// parseFeishuMessageContent extracts text from Feishu message content JSON.
// Feishu message content format: {"text": "actual message text"} for text messages.
func parseFeishuMessageContent(content string) string {
	// Try to parse as JSON
	var textContent struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal([]byte(content), &textContent); err == nil {
		return strings.TrimSpace(textContent.Text)
	}
	// Fallback: return raw content
	return strings.TrimSpace(content)
}

// formatFeishuUserStyle formats user style for workflow input.
func formatFeishuUserStyle(userID uint) string {
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

// sendBusyMessage sends a quick card message when session is busy.
func (r *Router) sendBusyMessage(chatID string, bot *model.FeishuBot) {
	api := service.NewFeishuAPI(bot.AppID, bot.AppSecret)
	card := Card{
		Config: CardConfig{WideScreenMode: true},
		Header: CardHeader{
			Title:    CardText{Tag: "plain_text", Content: "请稍等"},
			Template: "blue",
		},
		Elements: []CardElement{
			{
				Tag:  "div",
				Text: &CardText{Tag: "lark_md", Content: "正在处理上一个任务，请稍后再发消息..."},
			},
		},
	}
	cardJSON, _ := json.Marshal(card)
	_, err := api.CreateCard(chatID, string(cardJSON))
	if err != nil {
		log.Printf("[FeishuRouter] failed to send busy message: %v", err)
	}
}