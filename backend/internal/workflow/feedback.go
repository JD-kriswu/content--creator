package workflow

import (
	"strings"
)

// FeedbackType 定义反馈类型
type FeedbackType string

const (
	FeedbackTypeSingle FeedbackType = "single" // 单篇反馈：重跑特定模块
	FeedbackTypeSystem FeedbackType = "system" // 系统反馈：写入风格规则
)

// FeedbackClassifier 反馈类型识别器
type FeedbackClassifier struct{}

// NewFeedbackClassifier 创建反馈类型识别器
func NewFeedbackClassifier() *FeedbackClassifier {
	return &FeedbackClassifier{}
}

// Classify 根据用户输入判断反馈类型
func (c *FeedbackClassifier) Classify(userInput string) FeedbackType {
	lower := strings.ToLower(userInput)

	// 系统反馈关键词：涉及长期、规则、风格
	systemKeywords := []string{
		"以后",
		"长期",
		"永远",
		"规则",
		"风格",
		"每次",
		"所有",
		"全部",
		"总是",
	}
	for _, kw := range systemKeywords {
		if strings.Contains(lower, kw) {
			return FeedbackTypeSystem
		}
	}

	// 默认为单篇反馈
	return FeedbackTypeSingle
}

// ParseFeedbackIntent 解析用户反馈意图，返回目标阶段和约束内容
// 返回值：(目标阶段ID, 是否需要重跑, 约束内容)
func ParseFeedbackIntent(userInput string, currentStageID string) (targetStageID string, shouldRerun bool, constraint string) {
	lower := strings.ToLower(userInput)

	// 卖点相关反馈
	if strings.Contains(lower, "卖点") {
		return "selling_points", true, "卖点调整：" + userInput
	}

	// 篇幅相关反馈
	if strings.Contains(lower, "太长") || strings.Contains(lower, "太短") || strings.Contains(lower, "篇幅") {
		return "write", true, "篇幅调整：" + userInput
	}

	// 风格相关反馈（单篇）
	if strings.Contains(lower, "太口语") || strings.Contains(lower, "太正式") || strings.Contains(lower, "语气") {
		return "write", true, "风格调整：" + userInput
	}

	// 内容质量反馈
	if strings.Contains(lower, "不真实") || strings.Contains(lower, "数据不对") || strings.Contains(lower, "案例") {
		return "write", true, "内容调整：" + userInput
	}

	// 默认：根据当前阶段返回
	return "", false, ""
}

// IsSystemFeedbackRequest 检查用户是否在请求系统级规则更新
func IsSystemFeedbackRequest(userInput string) bool {
	lower := strings.ToLower(userInput)

	// 检查是否包含确认关键词
	confirmKeywords := []string{
		"保存规则",
		"确认规则",
		"添加规则",
		"记住",
	}
	for _, kw := range confirmKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}

	return false
}

// ExtractSuggestedRule 从用户反馈中提取建议规则
func ExtractSuggestedRule(userInput string) string {
	// 简单实现：返回用户输入作为建议规则
	// 实际场景可能需要 LLM 辅助提取
	return userInput
}