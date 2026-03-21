# content-creator-pro 新 Skill 构建 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 从零构建 `content-creator-pro` skill，实现 5 角色并行分析 + 辩论决策 + 大纲确认 + 终稿生成的完整口播稿创作流程，通过 DAL 层接入远端 API 和本地缓存，支持多用户风格档案和离线降级。

**Architecture:** 新 skill 完全独立于原 `content-creator-imm`，包含：SKILL.md（流程编排）+ lib/（DAL、缓存、API客户端、离线队列）+ utils/（评分标准）+ templates/（钩子库）。5 角色以 Claude prompt section 形式实现，辩论机制通过结构化中间输出实现，DAL 通过 Bash 工具调用 Node.js 脚本。

**Tech Stack:** Claude Code Skill（Markdown），Node.js（DAL 层），标准库 fs / http / https / crypto

**前置条件:** Plan 1 Go 后端服务已部署，用户有账号 token。

---

## 文件结构

```
~/.claude/skills/content-creator-pro/
├── SKILL.md                    # 主 Skill：完整流程编排（触发 → 9步 → 输出）
├── lib/
│   ├── dal.js                  # 数据访问层（统一读写入口）
│   ├── cache.js                # 本地文件缓存（TTL 管理）
│   ├── api-client.js           # 远端 API HTTP 客户端
│   └── sync-queue.js           # 离线队列（写入 + 批量上传）
├── utils/
│   ├── dna-scoring.md          # 爆款 DNA 6维度评分标准
│   ├── viral-scoring.md        # 新稿爆款潜力 5维度评分标准
│   └── similarity-check.md     # 相似度检测方法与降重技巧
└── templates/
    ├── hooks-library.md        # 开场钩子库（50+ 类型）
    └── rewrite-patterns.md     # 改写模式库（按选题类型分类）
```

---

## Task 1: 目录结构 + DAL 基础模块

**Files:**
- Create: `~/.claude/skills/content-creator-pro/lib/cache.js`
- Create: `~/.claude/skills/content-creator-pro/lib/api-client.js`
- Create: `~/.claude/skills/content-creator-pro/lib/sync-queue.js`

- [ ] **Step 1: 创建目录结构**

```bash
mkdir -p ~/.claude/skills/content-creator-pro/lib
mkdir -p ~/.claude/skills/content-creator-pro/utils
mkdir -p ~/.claude/skills/content-creator-pro/templates
```

- [ ] **Step 2: 创建 cache.js**

```javascript
// ~/.claude/skills/content-creator-pro/lib/cache.js
const fs = require('fs');
const path = require('path');
const os = require('os');

const BASE_DIR = path.join(os.homedir(), '.skill-pro');

const TTL = {
  hotspot:   60 * 60 * 1000,          // 1小时
  materials: 24 * 60 * 60 * 1000,     // 24小时
  style:     7  * 24 * 60 * 60 * 1000, // 7天
  profile:   7  * 24 * 60 * 60 * 1000,
};

function init() {
  ['', 'materials', 'scripts'].forEach(sub =>
    fs.mkdirSync(path.join(BASE_DIR, sub), { recursive: true })
  );
  const queueFile = path.join(BASE_DIR, 'sync_queue.json');
  const indexFile = path.join(BASE_DIR, 'scripts', 'index.json');
  if (!fs.existsSync(queueFile)) fs.writeFileSync(queueFile, '[]');
  if (!fs.existsSync(indexFile)) fs.writeFileSync(indexFile, '[]');
}

function read(key, sub) {
  const file = sub
    ? path.join(BASE_DIR, key, `${sub}.json`)
    : path.join(BASE_DIR, `${key}.json`);
  if (!fs.existsSync(file)) return null;
  try { return JSON.parse(fs.readFileSync(file, 'utf8')); } catch { return null; }
}

function write(key, data, sub) {
  const file = sub
    ? path.join(BASE_DIR, key, `${sub}.json`)
    : path.join(BASE_DIR, `${key}.json`);
  fs.writeFileSync(file, JSON.stringify({ data, cachedAt: Date.now() }));
}

function fresh(cached, ttlKey) {
  return cached && (Date.now() - cached.cachedAt < TTL[ttlKey]);
}

function loadConfig() {
  const f = path.join(BASE_DIR, 'config.json');
  if (!fs.existsSync(f)) return null;
  return JSON.parse(fs.readFileSync(f, 'utf8'));
}

function saveConfig(cfg) {
  fs.writeFileSync(path.join(BASE_DIR, 'config.json'), JSON.stringify(cfg, null, 2));
}

function saveScript(script) {
  const date = new Date().toISOString().split('T')[0];
  const filename = `${date}-${(script.id || Date.now())}.md`;
  fs.writeFileSync(
    path.join(BASE_DIR, 'scripts', filename),
    `# ${script.title || '口播稿'}\n\n${script.content}`
  );
  const idxFile = path.join(BASE_DIR, 'scripts', 'index.json');
  const idx = JSON.parse(fs.readFileSync(idxFile, 'utf8'));
  idx.unshift({ id: script.id, title: script.title, file: filename, createdAt: Date.now() });
  fs.writeFileSync(idxFile, JSON.stringify(idx.slice(0, 500)));
}

module.exports = { init, read, write, fresh, loadConfig, saveConfig, saveScript, BASE_DIR };
```

- [ ] **Step 3: 创建 api-client.js**

```javascript
// ~/.claude/skills/content-creator-pro/lib/api-client.js
const https = require('https');
const http = require('http');

function req(base, token, method, urlPath, body, ms = 5000) {
  return new Promise((resolve, reject) => {
    const url = new URL(base + urlPath);
    const lib = url.protocol === 'https:' ? https : http;
    const r = lib.request(
      { hostname: url.hostname, port: url.port || (url.protocol === 'https:' ? 443 : 80),
        path: url.pathname + url.search, method,
        headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${token}` },
        timeout: ms },
      (res) => {
        let d = '';
        res.on('data', c => d += c);
        res.on('end', () => {
          try { resolve({ status: res.statusCode, data: JSON.parse(d) }); }
          catch { resolve({ status: res.statusCode, data: d }); }
        });
      }
    );
    r.on('timeout', () => { r.destroy(); reject(new Error('timeout')); });
    r.on('error', reject);
    if (body) r.write(JSON.stringify(body));
    r.end();
  });
}

class ApiClient {
  constructor({ api_base, token, timeout_ms = 5000 }) {
    this.base = api_base;
    this.token = token;
    this.ms = timeout_ms;
  }
  ping()                    { return req(this.base, this.token, 'GET', '/ping', null, 2000).then(r => r.status === 200).catch(() => false); }
  getProfile()              { return req(this.base, this.token, 'GET', '/user/profile', null, this.ms); }
  updateStyle(style)        { return req(this.base, this.token, 'PUT', '/user/style', style, this.ms); }
  getHotspot(platform = '') { return req(this.base, this.token, 'GET', `/hotspot${platform ? '?platform=' + platform : ''}`, null, this.ms); }
  getMaterials(topic = '', limit = 10) { return req(this.base, this.token, 'GET', `/materials?topic=${encodeURIComponent(topic)}&limit=${limit}`, null, this.ms); }
  saveScript(script)        { return req(this.base, this.token, 'POST', '/scripts', script, this.ms); }
  sync(items)               { return req(this.base, this.token, 'POST', '/sync', items, this.ms); }
}

module.exports = ApiClient;
```

- [ ] **Step 4: 创建 sync-queue.js**

```javascript
// ~/.claude/skills/content-creator-pro/lib/sync-queue.js
const fs = require('fs');
const path = require('path');
const os = require('os');
const { randomUUID } = require('crypto');

const FILE = path.join(os.homedir(), '.skill-pro', 'sync_queue.json');

const read  = () => { try { return JSON.parse(fs.readFileSync(FILE, 'utf8')); } catch { return []; } };
const write = q  => fs.writeFileSync(FILE, JSON.stringify(q));

function enqueue(opType, payload) {
  const q = read();
  const op_id = randomUUID();
  q.push({ op_id, op_type: opType, payload, enqueuedAt: Date.now() });
  write(q);
  return op_id;
}

async function flush(apiClient) {
  const q = read();
  if (!q.length) return { success: 0, failed: [] };
  try {
    const res = await apiClient.sync(q);
    if (res.status === 200) {
      const failSet = new Set((res.data.failed || []).map(f => f.split(':')[0]));
      write(q.filter(i => failSet.has(i.op_id)));
      return { success: res.data.success || 0, failed: res.data.failed || [] };
    }
  } catch (e) {
    return { success: 0, failed: [e.message] };
  }
  return { success: 0, failed: ['unknown error'] };
}

module.exports = { enqueue, flush, length: () => read().length };
```

- [ ] **Step 5: 验证模块可正常加载**

```bash
node -e "
require(process.env.HOME + '/.claude/skills/content-creator-pro/lib/cache.js').init();
console.log('✅ cache ok');
new (require(process.env.HOME + '/.claude/skills/content-creator-pro/lib/api-client.js'))({api_base:'http://x',token:'x'});
console.log('✅ api-client ok');
require(process.env.HOME + '/.claude/skills/content-creator-pro/lib/sync-queue.js');
console.log('✅ sync-queue ok');
"
```

Expected: 3 行 ✅

- [ ] **Step 6: Commit**

```bash
cd /data/code/content_creator_imm
git add docs/ && git status
```

---

## Task 2: DAL 主模块

**Files:**
- Create: `~/.claude/skills/content-creator-pro/lib/dal.js`

- [ ] **Step 1: 创建 dal.js**

```javascript
// ~/.claude/skills/content-creator-pro/lib/dal.js
const cache     = require('./cache');
const ApiClient = require('./api-client');
const queue     = require('./sync-queue');

let client  = null;
let online  = false;
let degraded = false;

async function init() {
  cache.init();

  const cfg = cache.loadConfig();
  if (!cfg || !cfg.api_base || !cfg.token) {
    console.log([
      '',
      '⚠️  【未配置】请先完成初始化：',
      `  mkdir -p ~/.skill-pro`,
      `  cat > ~/.skill-pro/config.json << 'EOF'`,
      `  {"api_base":"https://your-server/api/v1","token":"your-jwt-token","timeout_ms":5000}`,
      `  EOF`,
      '',
      '  token 请从管理后台登录后获取',
      '',
    ].join('\n'));
    degraded = true;
    return { online: false, degraded: true, reason: 'no-config' };
  }

  client = new ApiClient(cfg);
  online = await client.ping();
  degraded = !online;

  if (degraded) {
    console.log('⚠️  【降级模式】远端不可达，本次使用本地缓存，稿件将在恢复后自动同步');
  } else {
    const pending = queue.length();
    if (pending > 0) {
      process.stdout.write(`📤 发现 ${pending} 条待同步数据，正在上传...`);
      const r = await queue.flush(client);
      console.log(` 完成（成功 ${r.success} 条）`);
    }
  }

  return { online, degraded, reason: online ? 'ok' : 'unreachable' };
}

async function getStyle() {
  const cached = cache.read('style');
  if (online && !cache.fresh(cached, 'style')) {
    try {
      const r = await client.getProfile();
      if (r.status === 200 && r.data.style) {
        cache.write('style', r.data.style);
        return r.data.style;
      }
    } catch {}
  }
  if (cached?.data) {
    if (degraded) console.log('  📁 使用本地缓存风格档案');
    return cached.data;
  }
  return null;
}

async function getHotspot(platform = '') {
  const cached = cache.read('hotspot');
  if (online && !cache.fresh(cached, 'hotspot')) {
    try {
      const r = await client.getHotspot(platform);
      if (r.status === 200) { cache.write('hotspot', r.data.data || []); return r.data.data || []; }
    } catch {}
  }
  if (degraded && cached?.data) console.log('  📁 使用本地缓存热点数据');
  return cached?.data || [];
}

async function getMaterials(topic = '') {
  const sub = topic || 'general';
  const cached = cache.read('materials', sub);
  if (online && !cache.fresh(cached, 'materials')) {
    try {
      const r = await client.getMaterials(topic);
      if (r.status === 200) { cache.write('materials', r.data.data || [], sub); return r.data.data || []; }
    } catch {}
  }
  if (degraded && cached?.data) console.log('  📁 使用本地缓存素材数据');
  return cached?.data || [];
}

async function saveScript(script) {
  cache.saveScript(script);                                   // 始终写本地副本

  if (online) {
    try {
      const r = await client.saveScript(script);
      if (r.status === 201) {
        console.log(`✅ 稿件已保存到远端 (ID: ${r.data.id})`);
        return { ...script, id: r.data.id };
      }
    } catch {}
  }

  const opId = queue.enqueue('create_script', script);
  console.log(`📋 稿件已保存到本地，恢复网络后自动同步 (opId: ${opId})`);
  return script;
}

async function updateStyle(style) {
  if (online) {
    try { await client.updateStyle(style); return; } catch {}
  }
  queue.enqueue('update_style', style);
}

module.exports = { init, getStyle, getHotspot, getMaterials, saveScript, updateStyle };
```

- [ ] **Step 2: 端到端验证 DAL**

```bash
# 前提: Plan 1 服务在 localhost:8080 运行，且已登录获取 token
TOKEN="<替换为真实token>"
mkdir -p ~/.skill-pro
cat > ~/.skill-pro/config.json << EOF
{"api_base":"http://localhost:8080/api/v1","token":"$TOKEN","timeout_ms":5000}
EOF

node -e "
const dal = require(process.env.HOME + '/.claude/skills/content-creator-pro/lib/dal.js');
dal.init().then(s => {
  console.log('状态:', JSON.stringify(s));
  return dal.getHotspot('weibo');
}).then(h => {
  console.log('热点数量:', h.length);
  process.exit(0);
});
"
```

Expected:
```
状态: {"online":true,"degraded":false,"reason":"ok"}
热点数量: N
```

- [ ] **Step 3: 验证降级模式**

```bash
# 临时改为不可达地址
cat > ~/.skill-pro/config.json << 'EOF'
{"api_base":"http://127.0.0.1:19999/api/v1","token":"fake","timeout_ms":5000}
EOF

node -e "
const dal = require(process.env.HOME + '/.claude/skills/content-creator-pro/lib/dal.js');
dal.init().then(s => console.log('降级状态:', JSON.stringify(s)));
"
```

Expected: `降级状态: {"online":false,"degraded":true,"reason":"unreachable"}`

---

## Task 3: utils/ 和 templates/ 内容文件

**Files:**
- Create: `~/.claude/skills/content-creator-pro/utils/dna-scoring.md`
- Create: `~/.claude/skills/content-creator-pro/utils/viral-scoring.md`
- Create: `~/.claude/skills/content-creator-pro/utils/similarity-check.md`
- Create: `~/.claude/skills/content-creator-pro/templates/hooks-library.md`
- Create: `~/.claude/skills/content-creator-pro/templates/rewrite-patterns.md`

- [ ] **Step 1: 创建 dna-scoring.md**

```bash
cat > ~/.claude/skills/content-creator-pro/utils/dna-scoring.md << 'EOF'
# 爆款 DNA 评分标准

## 6 维度评分（各 5 分，满分 30 分）

| 维度 | 1分 | 3分 | 5分 |
|------|-----|-----|-----|
| **钩子强度** | 平淡开场 | 有悬念或问题 | 极强反差/颠覆认知/数字冲击 |
| **痛点共鸣** | 泛泛而谈 | 触及特定人群痛点 | 精准戳中普遍深层痛点 |
| **信息密度** | 废话多，干货少 | 有价值信息但有冗余 | 每句话都有信息量，零废话 |
| **节奏把控** | 平铺直叙 | 有起伏但不流畅 | 紧凑、有节奏感、停顿恰当 |
| **情绪调动** | 情绪平淡 | 有情绪波动 | 情绪曲线完整，高潮明确 |
| **行动引导** | 无引导 | 有引导但不强 | 强烈的点赞/关注/评论驱动 |

## 综合评分换算

| 总分 | 等级 | 预测表现 |
|------|------|----------|
| 26-30 | ★★★★★ | 极大概率爆款 |
| 21-25 | ★★★★☆ | 较高爆款概率 |
| 16-20 | ★★★☆☆ | 中等，需优化 |
| ≤15   | ★★☆☆☆ | 较难破圈 |
EOF
```

- [ ] **Step 2: 创建 viral-scoring.md**

```bash
cat > ~/.claude/skills/content-creator-pro/utils/viral-scoring.md << 'EOF'
# 新稿爆款潜力评分

## 5 维度评分（各 10 分，满分 50 分）

| 维度 | 评分标准 |
|------|----------|
| **原创度** | 与原稿相似度越低分越高；< 15% → 10分，15-25% → 7分，25-30% → 4分 |
| **病毒传播力** | 是否有引发讨论/转发的爆点；强 → 10，中 → 6，弱 → 2 |
| **情绪穿透力** | 情绪钩子强度 × 共鸣深度；强共鸣 → 10，一般 → 5，无感 → 1 |
| **实用价值** | 观众看完是否有收获/改变；明确干货 → 10，模糊价值 → 5，纯娱乐 → 3 |
| **完播预期** | 基于节奏和吸引力预测完播率；预测 > 70% → 10，50-70% → 7，< 50% → 3 |

## 综合评分

总分 / 5 = 综合评分（满分 10 分）

| 评分 | 等级 |
|------|------|
| 9-10 | S 级：强烈推荐发布 |
| 7-8  | A 级：推荐发布 |
| 5-6  | B 级：优化后发布 |
| ≤4   | C 级：建议重写 |
EOF
```

- [ ] **Step 3: 创建 similarity-check.md**

```bash
cat > ~/.claude/skills/content-creator-pro/utils/similarity-check.md << 'EOF'
# 相似度检测方法

## 4 维度加权（总相似度 ≥ 30% 需重写）

| 维度 | 权重 | 检测方法 |
|------|------|----------|
| 词汇相似度 | 30% | 去停用词后共同词汇占比 |
| 句式相似度 | 25% | 主谓宾结构对比 |
| 结构相似度 | 25% | 段落功能序列对比 |
| 观点相似度 | 20% | 核心观点语义重合度 |

## 降重核心动作

1. **更换钩子角度**：原稿数字冲击 → 改用反问/场景代入
2. **替换案例素材**：原稿用 A 案例 → 换 B 案例（同类不同例）
3. **改变句式**：陈述句 → 疑问句/排比句
4. **调整段落顺序**：痛点前置 → 钩子前置
5. **新增反差观点**：原观点正向 → 加入反向视角

## 自检清单

- [ ] 70% 以上原词已替换
- [ ] 主要句式结构已改变
- [ ] 开场钩子与原稿完全不同
- [ ] 至少 1 个新素材/案例加入
- [ ] 核心观点有差异化角度
EOF
```

- [ ] **Step 4: 创建 hooks-library.md**

```bash
cat > ~/.claude/skills/content-creator-pro/templates/hooks-library.md << 'EOF'
# 开场钩子库

## 数字冲击型
- 「XX年，我靠一件事，从XX变成XX」
- 「90%的人都不知道，XX其实是……」
- 「3个字，让你少走XX年弯路」

## 反问引导型
- 「你有没有想过，为什么XX总是……」
- 「如果我告诉你XX，你信吗？」
- 「为什么同样努力，结果差这么多？」

## 场景代入型
- 「上班路上，你是不是也在想……」
- 「每次XX的时候，我都会……」
- 「昨天有个粉丝问我……」

## 颠覆认知型
- 「我们从小被教育的XX，其实是错的」
- 「XX不是你想的那样」
- 「专家说的XX，我亲测无效」

## 悬念制造型
- 「今天要说的事，可能会得罪很多人」
- 「这个秘密，我憋了很久了」
- 「说一件真实发生在我身上的事」
EOF
```

- [ ] **Step 5: 创建 rewrite-patterns.md**

```bash
cat > ~/.claude/skills/content-creator-pro/templates/rewrite-patterns.md << 'EOF'
# 改写模式库

## 痛点型选题改写模式
结构：钩子（戳痛点）→ 放大痛点 → 根因分析 → 解决方案 → 行动引导
改写策略：更换具体场景，保留痛点本质，用新案例佐证

## 干货型选题改写模式
结构：价值承诺（你能学到什么）→ 核心内容（3-5点）→ 总结金句 → 引导关注
改写策略：调整内容顺序，替换数据来源，增加反差观点

## 情绪型选题改写模式
结构：情绪触发（共鸣场景）→ 情绪升温 → 价值观表达 → 情感收尾
改写策略：换场景保情绪，用新金句替换原句，保留情绪曲线结构

## 反差型选题改写模式
结构：常识陈述 → 颠覆（"但其实……"）→ 证据 → 新认知 → 引导思考
改写策略：换颠覆角度，替换佐证案例，保留反差结构
EOF
```

- [ ] **Step 6: Commit**

```bash
cd /data/code/content_creator_imm
git add docs/ && git commit -m "docs: note that content-creator-pro skill files being created"
```

---

## Task 4: 主 SKILL.md（流程编排核心）

**Files:**
- Create: `~/.claude/skills/content-creator-pro/SKILL.md`

这是最核心的文件，实现 5 角色并行分析 + 辩论机制 + 完整创作流程。

- [ ] **Step 1: 创建 SKILL.md**

```bash
cat > ~/.claude/skills/content-creator-pro/SKILL.md << 'SKILLEOF'
---
name: content-creator-pro
description: 多用户口播稿改写专业版 - 5角色并行分析+辩论决策+风格档案+热点素材，从链接到终稿一站式完成，结果自动同步到远端服务
---

# 口播稿专业创作系统

## 触发条件

- 用户提供爆款视频/文案链接或直接粘贴文案
- 用户说"帮我改写这个"、"参考这个写一个"
- 用户说"开始创作"（已有上下文）

---

## Step 0: 初始化

创作开始，首先运行初始化脚本：

```bash
node -e "
const dal = require(process.env.HOME + '/.claude/skills/content-creator-pro/lib/dal.js');
dal.init().then(s => console.log(JSON.stringify(s)));
" 2>&1
```

根据输出：
- `"reason":"ok"` → 正常模式，继续
- `"reason":"unreachable"` → 降级模式，提示用户后继续（使用本地缓存）
- `"reason":"no-config"` → **停止**，引导用户完成配置后重新触发

---

## Step 1: 原稿获取与纠错

### 1.1 获取原文

- 若提供链接：使用 WebFetch 工具提取正文
- 若直接粘贴：直接使用

### 1.2 AI 纠错

对原文进行纠错（语音识别错误、别字、标点），输出：

```
【原始文本】
[原文]

【纠错文本】
[纠正后全文]

【修改说明】
- [修改点1]
- [修改点2]
```

---

## Step 2: 拉取用户数据

并行执行（使用 Bash 工具）：

```bash
node -e "
const dal = require(process.env.HOME + '/.claude/skills/content-creator-pro/lib/dal.js');
Promise.all([
  dal.getStyle(),
  dal.getHotspot('douyin'),
  dal.getMaterials('')   // topic 在分析后填入，此处先拿通用素材
]).then(([style, hotspots, materials]) => {
  console.log(JSON.stringify({ style, hotspots: hotspots.slice(0,10), materials }));
});
" 2>&1
```

将返回的 `style`、`hotspots`、`materials` 保存到上下文，供后续步骤使用。

---

## Step 3: 5 角色并行分析

以下 5 个角色**同时**对原稿进行分析，在**一次回复**中完成全部输出：

---

### 【角色①：爆款解构师】

分析原稿的爆款基因：

**选题分析**
| 项目 | 内容 |
|------|------|
| 选题类型 | [痛点型/干货型/情绪型/反差型] |
| 目标人群 | [具体描述] |
| 核心痛点 | [最打动人的点] |
| 爆款优势 | [为什么这个内容能火] |

**爆款 DNA 评分**（参考 `utils/dna-scoring.md`）
| 维度 | 评分 | 关键分析 |
|------|------|----------|
| 钩子强度 | X/5 | [分析] |
| 痛点共鸣 | X/5 | [分析] |
| 信息密度 | X/5 | [分析] |
| 节奏把控 | X/5 | [分析] |
| 情绪调动 | X/5 | [分析] |
| 行动引导 | X/5 | [分析] |
| **综合** | **X/30** | |

**段落结构**
| 段落 | 时长 | 功能 | 核心内容 | 情绪目标 |
|------|------|------|----------|----------|
| 开场 | Xs | 钩子 | [内容] | [情绪] |
| [段落2] | Xs | [功能] | [内容] | [情绪] |
| 结尾 | Xs | 引导 | [内容] | [情绪] |

**必须保留的爆款要素**（TOP 4）：
1. [要素1]
2. [要素2]
3. [要素3]
4. [要素4]

---

### 【角色②：风格建模师】

基于 Step 2 拉取的风格档案 `style` 输出：

**用户风格画像**
| 维度 | 特征 | 改写指导 |
|------|------|----------|
| 语言风格 | [口语化/专业/接地气] | [具体要求] |
| 情绪基调 | [理性/感性/幽默] | [具体要求] |
| 典型开场 | [特征] | [模仿方向] |
| 典型结尾 | [特征] | [模仿方向] |
| 标志元素 | [口头禅/固定句式] | [融入建议] |

> 若无风格档案（style 为 null），输出：「使用通用爆款风格，建议创作完成后在管理后台完善风格档案」

---

### 【角色③：素材补齐师】

基于 Step 2 拉取的热点和素材，结合 WebSearch 补充：

**当前热点关联**（来自远端热点雷达）
| 热点 | 关联度 | 融入建议 |
|------|--------|----------|
| [热点1] | 高/中/低 | [如何融入] |
| [热点2] | 高/中/低 | [如何融入] |

**新素材库**
| 类型 | 内容 | 来源 | 应用位置 |
|------|------|------|----------|
| 📊 数据 | [数据点] | [来源] | [段落] |
| ⚡ 反差 | [反差观点] | [来源] | [段落] |
| 📖 案例 | [案例] | [来源] | [段落] |
| 💎 金句 | [金句] | [来源] | 结尾 |

**素材应用建议**：
- [建议1]
- [建议2]

---

### 【角色④：创作代理（预规划）】

基于前三角色分析，提出大纲设想：

**初步大纲构思**
| 段落 | 时长 | 来源策略 | 情绪目标 |
|------|------|----------|----------|
| 开场 | Xs | [新钩子方向] | [情绪] |
| [段落2] | Xs | [原稿+新素材] | [情绪] |
| 结尾 | Xs | [风格化引导] | [情绪] |

**与原稿差异化点**：
1. [差异1]
2. [差异2]

---

### 【角色⑤：优化代理（预审）】

对以上分析提出质疑和优化建议：

**审查意见**：
- [意见1：哪个要素不够强，如何改]
- [意见2：素材是否有事实风险]
- [意见3：风格融合是否自然]

---

## Step 4: 辩论决策

针对角色间的分歧点（角色④大纲 vs 角色⑤审查意见），输出融合方案：

**分歧点与决策**
| 分歧 | 角色④观点 | 角色⑤观点 | 决策 | 理由 |
|------|-----------|-----------|------|------|
| [分歧1] | [观点] | [观点] | [决策] | [理由] |
| [分歧2] | [观点] | [观点] | [决策] | [理由] |

**融合策略**：
- 保留：[什么爆款要素必须保留]
- 替换：[什么用新素材替换]
- 新增：[什么是全新加入的]
- 风格：[如何落实用户风格]

---

## Step 5: 大纲确认（⚠️ 等待用户确认）

输出最终大纲，**必须等待用户回复后才进入终稿撰写**：

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📋 口播稿大纲（请确认后开始撰写）
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

【原稿成功要素】
① [要素1]
② [要素2]
③ [要素3]
④ [要素4]

【融合新素材】
① [素材1]（来源: [来源]）
② [素材2]（来源: [来源]）
③ [素材3]（来源: [来源]）

【新口播稿大纲】
1）[开场钩子]（约Xs）
2）[第二部分]（约Xs）
3）[第三部分]（约Xs）
4）[结尾引导]（约Xs）
预计总时长：约Xs | 预计相似度：<30%

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
[1] ✅ 确认，开始撰写终稿
[2] 🔄 调整大纲（请说明方向）
[3] 🔄 更换素材方向
[4] 🔙 重新分析（说明原因）
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

---

## Step 6: 撰写终稿

用户选择 [1] 确认后，按大纲撰写完整口播稿：

**写作要求**：
1. 字数：按时长估算（约 200 字/分钟）
2. 风格：严格按风格建模师画像执行
3. 语言：口语化，适合直接念稿，无书面句式
4. 情绪：情绪曲线符合辩论决策中的设计
5. 素材：自然融入，不生硬

---

## Step 7: 质量检测

### 7.1 事实核查

- 检查引用数据是否可信
- 检查案例描述是否准确
- 标记不确定信息为 ⚠️

### 7.2 逻辑检查

- 论证链是否完整
- 观点间是否矛盾

### 7.3 口播适配检查

- 是否有绕口/难念的词
- 停顿点是否自然
- 书面句式是否已口语化

---

## Step 8: 相似度检测

按 `utils/similarity-check.md` 4维度加权计算：

```
总相似度 = 词汇×30% + 句式×25% + 结构×25% + 观点×20%
```

| 维度 | 相似度 |
|------|--------|
| 词汇 | XX% |
| 句式 | XX% |
| 结构 | XX% |
| 观点 | XX% |
| **综合** | **XX%** |

- **< 30%**：✅ 通过，进入 Step 9
- **≥ 30%**：❌ 按 `utils/similarity-check.md` 降重策略重写，最多 3 次

---

## Step 9: 保存 + 最终输出

### 9.1 保存稿件

```bash
node -e "
const dal = require(process.env.HOME + '/.claude/skills/content-creator-pro/lib/dal.js');
const script = {
  title: '[从终稿提取标题]',
  platform: '[目标平台]',
  content: \`[终稿全文]\`,
  similarity_score: [相似度小数如0.18],
  viral_score: [爆款综合评分如8.2],
  source_url: '[原稿来源]',
  quality_report: JSON.stringify({ similarity: [对象], viral: [对象] })
};
dal.init().then(() => dal.saveScript(script)).then(s => console.log('saved:', s.id || 'local'));
" 2>&1

# 静默更新风格档案
node -e "
const dal = require(process.env.HOME + '/.claude/skills/content-creator-pro/lib/dal.js');
dal.init().then(() => dal.updateStyle({
  language_style: '[本次风格]',
  emotion_tone: '[情绪基调]'
})).catch(() => {});
" 2>&1 &
```

### 9.2 最终输出格式

```markdown
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📝 口播稿终稿
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

[完整终稿正文]

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📊 质量报告

相似度：XX%（✅ < 30%）
爆款评分：X.X/10（[等级]）

核心优势：
① [优势1]
② [优势2]
③ [优势3]

⚠️ 待确认信息（如有）：
- [待核实的数据/说法]

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📱 发布建议

标题：
① [推荐标题1]
② [推荐标题2]

标签：#[tag1] #[tag2] #[tag3] #[tag4] #[tag5]

发布时段：[建议时间]
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

---

## 错误处理

| 情况 | 处理方式 |
|------|----------|
| 链接无法提取 | 请用户直接粘贴文案 |
| 风格档案为空 | 使用通用爆款风格，提示用户完善档案 |
| 热点/素材接口失败 | 使用本地缓存或 WebSearch 补充 |
| 相似度 3 次仍 ≥ 30% | 提示用户调整方向，列出 4 个调整选项 |
| 保存失败 | 已写本地副本，提示恢复网络后自动同步 |

---

## 相关文件

- `utils/dna-scoring.md` — 爆款 DNA 6维度评分标准
- `utils/viral-scoring.md` — 新稿爆款潜力评分标准
- `utils/similarity-check.md` — 相似度检测与降重技巧
- `templates/hooks-library.md` — 开场钩子库
- `templates/rewrite-patterns.md` — 按选题类型的改写模式
SKILLEOF
```

- [ ] **Step 2: 验证文件写入正确**

```bash
head -5 ~/.claude/skills/content-creator-pro/SKILL.md
wc -l ~/.claude/skills/content-creator-pro/SKILL.md
```

Expected: 首行为 `---`，总行数 > 200

- [ ] **Step 3: Commit**

```bash
cd /data/code/content_creator_imm
git add docs/ && git commit -m "docs: plan4 updated - content-creator-pro new skill from scratch"
```

---

## Task 5: 注册 Skill + 端到端验证

**Files:**
- Modify: Superpowers skills 注册配置（通过 update-config skill 更新）

- [ ] **Step 1: 确认 skill 目录结构完整**

```bash
find ~/.claude/skills/content-creator-pro -type f | sort
```

Expected 输出：
```
~/.claude/skills/content-creator-pro/SKILL.md
~/.claude/skills/content-creator-pro/lib/api-client.js
~/.claude/skills/content-creator-pro/lib/cache.js
~/.claude/skills/content-creator-pro/lib/dal.js
~/.claude/skills/content-creator-pro/lib/sync-queue.js
~/.claude/skills/content-creator-pro/templates/hooks-library.md
~/.claude/skills/content-creator-pro/templates/rewrite-patterns.md
~/.claude/skills/content-creator-pro/utils/dna-scoring.md
~/.claude/skills/content-creator-pro/utils/similarity-check.md
~/.claude/skills/content-creator-pro/utils/viral-scoring.md
```

- [ ] **Step 2: 检查 SKILL.md frontmatter 格式正确**

```bash
head -5 ~/.claude/skills/content-creator-pro/SKILL.md
```

Expected:
```yaml
---
name: content-creator-pro
description: 多用户口播稿改写专业版 - ...
---
```

- [ ] **Step 3: 模拟 DAL 全流程**

```bash
# 确保服务运行且 config.json 配置正确
node -e "
async function test() {
  const dal = require(process.env.HOME + '/.claude/skills/content-creator-pro/lib/dal.js');
  const s = await dal.init();
  console.log('init:', s.reason);

  const style = await dal.getStyle();
  console.log('style:', style ? '有风格档案' : '无风格档案（使用默认）');

  const hotspots = await dal.getHotspot('weibo');
  console.log('hotspots:', hotspots.length, '条');

  const materials = await dal.getMaterials('财经');
  console.log('materials:', materials.length, '条');

  const saved = await dal.saveScript({
    title: '测试稿件',
    content: '这是一段测试口播稿内容',
    platform: 'douyin',
    similarity_score: 0.18,
    viral_score: 7.5
  });
  console.log('saved:', saved.id ? '远端' : '本地');
}
test().catch(console.error);
" 2>&1
```

Expected（有网时）:
```
init: ok
style: 无风格档案（使用默认）
hotspots: N 条
materials: 0 条
saved: 远端
```

- [ ] **Step 4: 验证降级模式下稿件写入本地队列**

```bash
# 临时断网模拟
node -e "
process.env.FORCE_OFFLINE = '1';
// 改用不可达地址测试
const origConfig = require(process.env.HOME + '/.skill-pro/config.json');
" 2>&1
# 简化：直接查看 sync_queue.json 是否有内容
cat ~/.skill-pro/sync_queue.json
```

- [ ] **Step 5: 最终 Commit**

```bash
cd /data/code/content_creator_imm
git add docs/
git commit -m "$(cat <<'EOF'
docs: complete Plan 4 - content-creator-pro new skill build plan

Full skill from scratch with:
- 5-agent parallel analysis + debate mechanism
- DAL layer (cache/api-client/sync-queue)
- Local file cache with TTL + offline degradation
- Remote sync on recovery

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## 验收标准

- [ ] `~/.claude/skills/content-creator-pro/` 目录结构完整（10个文件）
- [ ] DAL `init()` 正常模式返回 `reason: ok`
- [ ] DAL `init()` 降级模式返回 `reason: unreachable` 且不崩溃
- [ ] `getStyle/getHotspot/getMaterials` 有缓存时不请求远端
- [ ] `saveScript` 正常模式写远端 + 本地，降级模式写本地 + 队列
- [ ] SKILL.md 在 Claude Code 中可通过 `/content-creator-pro` 触发
- [ ] 5 角色分析在 Step 3 中**一次回复**完成全部输出
- [ ] Step 5 大纲确认**阻断**后续流程，等待用户回复

---

## 4 个 Plan 依赖关系

```
Plan 1 (Go后端) ─────┬──→ Plan 2 (热点爬虫)
                      ├──→ Plan 3 (Admin前端)
                      └──→ Plan 4 (新Skill)  ← 当前计划
```

Plan 1 完成后，Plan 2/3/4 可并行执行。
