# Findings & Decisions - 编导小龙虾

## Requirements
### 核心需求（来自Roger的产品文档）

**1. 初始化阶段**
- 用户输入过往3篇口播稿
- 风格建模师Agent分析提取人设风格
- 输出《人设风格说明书》
- 后续结合反馈不断优化

**2. 阶段1：输入处理**
- 接收口播稿文本
- 清理文本（去除特殊字符，统一编码）

**3. 阶段2：两代理并行分析**
- 爆款解构师：分析口播稿DNA（结构、内容、情感、表达）
- 素材补齐师：补充新素材和数据

**4. 阶段3：辩论协调，产出大纲**
- 根据《新素材包》、《人设风格说明书》、《爆款DNA分析报告》
- 检测冲突点
- 多目标优化寻找最佳平衡点
- 生成融合版本《大纲》
- 两种方案可选：
  - Multi-Agent Debate（多智能体辩论）
  - Multi-Agent Consensus（多智能体共识机制）
  - 或者：帕累托最优 + 梯度下降

**5. 阶段4：大纲生成与确认**
- 分析原稿成功要素
- 建议补充新素材
- 展示新大纲
- 等待用户确认
- 如果改动大：改prompt重新走逻辑
- 如果改动小：直接修改并优化prompt

**6. 阶段5：确认后创作**
- 启动创作代理师
- 基于所有输入生成完整初稿

**7. 阶段6：优化审核**
- 相似度检测（<30%）- 用算法
- 事实核查（>90%准确）- 用大模型
- 逻辑检查（>85%连贯）- 用大模型
- 表达优化 - 用大模型
- 不通过则返回修改

**8. 阶段7：最终输出**
- 新的口播稿
- 待确定信息
- 生成总结
- 下一步建议

### Prompt需求
每个Agent都有专门的Prompt：
1. 风格建模师Prompt
2. 爆款解构师Prompt
3. 素材补齐师Prompt
4. 创作代理师Prompt
5. 事实核查Prompt
6. 逻辑检查Prompt
7. 表达优化Prompt

## Research Findings
### 现有系统架构分析

**后端（Go + Gin）**
- 入口：`backend/main.go`
- 核心服务：`backend/internal/service/pipeline.go` - Session状态机
- 聊天处理：`backend/internal/handler/chat_handler.go` - SSE消息处理
- LLM服务：`backend/internal/service/llm_service.go` - Claude API调用
- Prompt构建：`backend/internal/service/prompts.go`

**前端（React + Vite）**
- 入口：`frontend/src/App.tsx`
- 路由：`frontend/src/router.tsx`
- 核心页面：`frontend/src/pages/Dashboard.tsx`
- 状态管理：useReducer
- SSE处理：`frontend/src/lib/sse.ts`

**现有流程**
```
idle → analyzing → awaiting → writing → complete
```

**新流程**
```
初始化 → 输入处理 → 并行分析 → 辩论协调 → 大纲确认 → 创作 → 优化审核 → 最终输出
```

### 技术决策分析

**方案选择：阶段3辩论协调**
1. **Multi-Agent Debate（多智能体辩论）**
   - 优点：更智能，能产生更优解
   - 缺点：实现复杂，响应时间长
   
2. **帕累托最优 + 多目标优化**
   - 优点：确定性结果，实现简单
   - 缺点：可能不如辩论灵活
   
**建议：先用方案2（简单方案），后续迭代再考虑方案1**

## Technical Decisions
| Decision | Rationale |
|----------|-----------|
| 用 Multi-Agent Debate | Roger确认：使用多智能体辩论方案 |
| 用带搜索功能的大模型 | Roger确认：素材补齐师用带搜索功能的大模型搜索 |
| 初始化流程：建议+Plan B | Roger确认：建议用户输入3篇口播稿，没有则提供备选方案 |
| 保持现有SSE架构 | 现有架构稳定，可以在基础上扩展 |
| 新增初始化流程 | 新产品需要风格建模，需要新增初始化阶段 |

## Issues Encountered
| Issue | Resolution |
|-------|------------|
| | |

## Resources
- 项目路径：`/data/code/content_creator_imm`
- 远程仓库：`git@github.com:JD-kriswu/content--creator.git`
- 访问地址：http://next-better-day.online/creator/
- Claude文档：`/data/code/content_creator_imm/CLAUDE.md`
- AI记忆：`/data/code/content_creator_imm/.ai_mem/`

## Visual/Browser Findings
暂无

---
*Update this file after every 2 view/browser/search operations*