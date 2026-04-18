package workflow

import (
	"strings"
)

// RouteConfig 定义路由配置：输入类型对应的起始阶段和跳过阶段
type RouteConfig struct {
	StartStageID string   // 起始阶段 ID
	SkipStages   []string // 需跳过的阶段 ID 列表
	JumpAfter    string   // 特殊：在此阶段完成后跳到 write（用于 script_with_outline）
}

// RouteTable 路由表：输入类型 → 路由配置
var RouteTable = map[InputType]RouteConfig{
	InputTypeOriginalScript: {
		StartStageID: "research",
		SkipStages:   nil,
	},
	InputTypeIdea: {
		StartStageID: "create",
		SkipStages:   []string{"research", "material_check", "material_curator"},
	},
	InputTypeOutline: {
		StartStageID: "write",
		SkipStages:   []string{"research", "material_check", "material_curator", "create", "optimize", "confirm_outline"},
	},
	InputTypeDraft: {
		StartStageID: "write",
		SkipStages:   []string{"research", "material_check", "material_curator", "create", "optimize", "confirm_outline"},
	},
	InputTypeScriptWithMaterial: {
		StartStageID: "research",
		SkipStages:   nil, // material_curator 由 skip_if 条件跳过
	},
	InputTypeScriptWithOutline: {
		StartStageID: "research",
		SkipStages:   []string{"create", "optimize", "confirm_outline"},
		JumpAfter:    "research", // research 完成后直接跳到 write
	},
}

// InputClassifier 输入类型识别器
type InputClassifier struct{}

// NewInputClassifier 创建输入类型识别器
func NewInputClassifier() *InputClassifier {
	return &InputClassifier{}
}

// Classify 根据用户输入判断输入类型
func (c *InputClassifier) Classify(text string, hasURL bool) InputType {
	text = strings.TrimSpace(text)
	lowerText := strings.ToLower(text)

	// 1. 明确标注优先（用户主动声明类型）
	if hasExplicitOutlineMarker(lowerText) {
		return InputTypeOutline
	}
	if hasExplicitDraftMarker(lowerText) {
		return InputTypeDraft
	}
	if hasExplicitIdeaMarker(lowerText) {
		return InputTypeIdea
	}
	if hasExplicitMaterialMarker(lowerText) {
		return InputTypeScriptWithMaterial
	}
	if hasExplicitOutlineWithScriptMarker(lowerText) {
		return InputTypeScriptWithOutline
	}

	// 2. URL 直接判定为原稿
	if hasURL {
		return InputTypeOriginalScript
	}

	// 3. 结构特征判断（大纲特征：有明确的分段标记）
	if hasOutlineStructure(text) {
		return InputTypeOutline
	}

	// 4. 长度判断（短文本且无结构 → 想法）
	if len(text) < 100 && !hasOutlineStructure(text) {
		return InputTypeIdea
	}

	// 5. 默认为原稿
	return InputTypeOriginalScript
}

// hasExplicitOutlineMarker 检测明确的大纲标注
func hasExplicitOutlineMarker(text string) bool {
	markers := []string{
		"大纲:",
		"这是大纲",
		"我的大纲",
		"已提供大纲",
		"大纲如下",
	}
	for _, m := range markers {
		if strings.Contains(text, m) {
			return true
		}
	}
	return false
}

// hasExplicitDraftMarker 检测明确的草稿标注
func hasExplicitDraftMarker(text string) bool {
	markers := []string{
		"草稿",
		"未完成",
		"初稿",
		"半成品",
		"待完善",
	}
	for _, m := range markers {
		if strings.Contains(text, m) {
			return true
		}
	}
	return false
}

// hasExplicitIdeaMarker 检测明确的想法标注
func hasExplicitIdeaMarker(text string) bool {
	markers := []string{
		"我的想法",
		"我想说",
		"我的观点",
		"我想讲",
		"一个想法",
		"有个观点",
	}
	for _, m := range markers {
		if strings.Contains(text, m) {
			return true
		}
	}
	return false
}

// hasExplicitMaterialMarker 检测明确的素材标注
func hasExplicitMaterialMarker(text string) bool {
	markers := []string{
		"已有素材",
		"自带素材",
		"我有素材",
		"素材已准备",
	}
	for _, m := range markers {
		if strings.Contains(text, m) {
			return true
		}
	}
	return false
}

// hasExplicitOutlineWithScriptMarker 检测「原稿+大纲」标注
func hasExplicitOutlineWithScriptMarker(text string) bool {
	// 同时包含「原稿」和「大纲」关键词
	hasScript := strings.Contains(text, "原稿") || strings.Contains(text, "爆款")
	hasOutline := strings.Contains(text, "大纲") && !strings.Contains(text, "生成大纲")
	return hasScript && hasOutline
}

// hasOutlineStructure 检测大纲结构特征
func hasOutlineStructure(text string) bool {
	// 大纲特征：包含明确的结构分段标记
	markers := []string{
		"开头",
		"正文",
		"结尾",
		"第一段",
		"第二段",
		"第三段",
		"CTA",
		"钩子",
	}
	count := 0
	for _, m := range markers {
		if strings.Contains(text, m) {
			count++
		}
	}
	// 至少包含 2 个结构标记才判定为大纲
	return count >= 2
}

// GetRoute 根据输入类型获取路由配置
func GetRoute(inputType InputType) RouteConfig {
	if route, ok := RouteTable[inputType]; ok {
		return route
	}
	// 默认返回完整流程
	return RouteConfig{
		StartStageID: "research",
		SkipStages:   nil,
	}
}