package service

import "fmt"

// BuildAnalysisPrompt builds the 5-role analysis + outline prompt.
func BuildAnalysisPrompt(originalText string, style *StyleProfile) string {
	styleSection := "用户暂无风格档案，请使用通用爆款风格。"
	if style != nil && style.LanguageStyle != "" {
		styleSection = fmt.Sprintf(`用户风格档案：
- 语言风格：%s
- 情绪基调：%s
- 典型开场：%s
- 典型结尾：%s
- 标志性元素/口头禅：%s`, style.LanguageStyle, style.EmotionTone, style.OpeningStyle, style.ClosingStyle, style.Catchphrases)
	}

	return fmt.Sprintf(`你是一个专业的短视频口播稿创作系统，由5个专家角色协作完成分析。请严格按照以下格式输出。

%s

---

## 原稿内容

%s

---

请完成以下5个角色的分析，然后输出大纲：

---

### 【角色①：爆款解构师】

分析原稿爆款基因：

**选题分析**
| 项目 | 内容 |
|------|------|
| 选题类型 | [痛点型/干货型/情绪型/反差型] |
| 目标人群 | [具体描述] |
| 核心痛点 | [最打动人的点] |
| 爆款优势 | [为什么这个内容能火] |

**爆款DNA评分**（各维度1-5分）
| 维度 | 评分 | 关键分析 |
|------|------|----------|
| 钩子强度 | X/5 | [分析] |
| 痛点共鸣 | X/5 | [分析] |
| 信息密度 | X/5 | [分析] |
| 节奏把控 | X/5 | [分析] |
| 情绪调动 | X/5 | [分析] |
| 行动引导 | X/5 | [分析] |
| **综合** | **X/30** | |

**必须保留的爆款要素（TOP4）**：
1. [要素1]
2. [要素2]
3. [要素3]
4. [要素4]

---

### 【角色②：风格建模师】

基于用户风格档案，分析风格融合方向：

| 维度 | 特征 | 改写指导 |
|------|------|----------|
| 语言风格 | [分析] | [要求] |
| 情绪基调 | [分析] | [要求] |
| 标志元素 | [分析] | [融入建议] |

---

### 【角色③：素材补齐师】

提出可融入的新素材：

| 类型 | 内容 | 应用位置 |
|------|------|----------|
| 📊 数据 | [数据点] | [段落] |
| ⚡ 反差 | [反差观点] | [段落] |
| 📖 案例 | [案例] | [段落] |
| 💎 金句 | [金句] | 结尾 |

---

### 【角色④：创作代理（预规划）】

初步大纲构思：

| 段落 | 时长 | 内容方向 | 情绪目标 |
|------|------|----------|----------|
| 开场 | Xs | [新钩子] | [情绪] |
| 发展 | Xs | [主体内容] | [情绪] |
| 升华 | Xs | [核心观点] | [情绪] |
| 结尾 | Xs | [引导] | [情绪] |

---

### 【角色⑤：优化代理（预审）】

审查意见：
- [意见1：哪个要素不够强，如何改]
- [意见2：素材是否有事实风险]
- [意见3：风格融合是否自然]

---

### 【辩论决策】

| 分歧点 | 角色④观点 | 角色⑤观点 | 最终决策 |
|--------|-----------|-----------|----------|
| [分歧1] | [观点] | [观点] | [决策] |

融合策略：
- 保留：[什么爆款要素必须保留]
- 替换：[什么用新素材替换]
- 新增：[什么是全新加入的]

---OUTLINE_START---
{
  "elements": ["要素1", "要素2", "要素3", "要素4"],
  "materials": ["素材1（来源）", "素材2（来源）", "素材3（来源）"],
  "outline": [
    {"part": "开场", "duration": "Xs", "content": "[钩子内容]", "emotion": "[情绪]"},
    {"part": "发展", "duration": "Xs", "content": "[主体内容]", "emotion": "[情绪]"},
    {"part": "升华", "duration": "Xs", "content": "[核心观点]", "emotion": "[情绪]"},
    {"part": "结尾", "duration": "Xs", "content": "[引导内容]", "emotion": "[情绪]"}
  ],
  "estimated_similarity": "约XX%%",
  "strategy": "[改写核心策略一句话]"
}
---OUTLINE_END---`, styleSection, originalText)
}

// BuildFinalDraftPrompt builds the prompt for writing the final script.
func BuildFinalDraftPrompt(originalText, outlineJSON, userNote string) string {
	extraNote := ""
	if userNote != "" {
		extraNote = fmt.Sprintf("\n用户额外要求：%s\n", userNote)
	}
	return fmt.Sprintf(`你是专业的短视频口播稿撰写专家。请根据以下大纲，撰写一篇完整的口播稿。

## 参考原稿（仅用于理解内容，不得直接引用）
%s

## 已确认大纲
%s
%s
## 写作要求

1. **字数**：约300-600字（对应1-3分钟视频）
2. **语言**：口语化，适合直接念稿，避免书面语
3. **情绪**：情绪曲线完整，开场吸引，结尾有力
4. **结构**：严格按大纲段落顺序撰写
5. **差异化**：与原稿相似度必须低于30%%，开场钩子必须与原稿完全不同

请直接输出口播稿正文（不需要标注段落名称），然后在最后输出：

---QUALITY_CHECK_START---
事实核查：
- [逐条列出引用的数据/案例，标注是否可信]

逻辑检查：
- [论证链是否完整，是否有矛盾]

口播适配：
- [是否有绕口词，停顿是否自然]
---QUALITY_CHECK_END---`, originalText, outlineJSON, extraNote)
}

// BuildSimilarityCheckPrompt builds the prompt for similarity scoring.
func BuildSimilarityCheckPrompt(original, newScript string) string {
	return fmt.Sprintf(`请对以下两篇文章进行相似度评分。

## 原稿
%s

## 新稿
%s

请从4个维度评估相似度（每个维度0-100%%），严格按JSON格式输出：

{"vocab": 词汇相似度, "sentence": 句式相似度, "structure": 结构相似度, "viewpoint": 观点相似度, "total": 加权总分}

计算公式：total = vocab*0.30 + sentence*0.25 + structure*0.25 + viewpoint*0.20

只输出JSON，不要其他文字。`, original, newScript)
}

// StyleProfile is a simplified version for prompt building
type StyleProfile struct {
	LanguageStyle string
	EmotionTone   string
	OpeningStyle  string
	ClosingStyle  string
	Catchphrases  string
}
