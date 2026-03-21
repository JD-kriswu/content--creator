# 热点雷达爬虫 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 Go 后端服务中实现热点雷达模块，定时抓取微博热搜、抖音热榜、小红书热门话题，结构化存储到 PostgreSQL，供 Skill 查询使用。

**Architecture:** 热点抓取作为独立的 Go 包运行在主服务进程内，使用 `robfig/cron` 调度。每个平台一个独立 Fetcher，实现统一接口，便于扩展。抓取结果写入 `hotspots` 表，超过 24 小时的数据定期清理。

**Tech Stack:** Go 1.22+, robfig/cron v3, colly（HTML 爬虫）, 标准库 net/http

**前置条件:** Plan 1 已完成，`hotspots` 表已存在。

---

## 文件结构

```
server/internal/crawler/
├── crawler.go           # Scheduler：注册所有 Fetcher、启动/停止 cron
├── fetcher.go           # Fetcher 接口定义
├── weibo.go             # 微博热搜 Fetcher
├── douyin.go            # 抖音热榜 Fetcher（基于公开 API）
└── xiaohongshu.go       # 小红书热门话题 Fetcher
server/internal/repository/
└── material_repo.go     # 已有，补充 CleanOldHotspots 方法
```

---

## Task 1: Fetcher 接口 + Scheduler 骨架

**Files:**
- Create: `server/internal/crawler/fetcher.go`
- Create: `server/internal/crawler/crawler.go`

- [ ] **Step 1: 安装依赖**

```bash
cd /data/code/content_creator_imm/server
go get github.com/robfig/cron/v3
go get github.com/gocolly/colly/v2
```

- [ ] **Step 2: 创建 fetcher.go 接口**

```go
// internal/crawler/fetcher.go
package crawler

import "github.com/content-creator-imm/server/internal/model"

// Fetcher 每个平台实现此接口
type Fetcher interface {
    Platform() string
    Fetch() ([]model.Hotspot, error)
}
```

- [ ] **Step 3: 创建 crawler.go（Scheduler）**

```go
// internal/crawler/crawler.go
package crawler

import (
    "log"
    "time"
    "github.com/robfig/cron/v3"
    "github.com/content-creator-imm/server/internal/repository"
)

type Scheduler struct {
    cron     *cron.Cron
    fetchers []Fetcher
    repo     *repository.MaterialRepo
}

func NewScheduler(repo *repository.MaterialRepo) *Scheduler {
    return &Scheduler{
        cron: cron.New(),
        repo: repo,
        fetchers: []Fetcher{
            NewWeiboFetcher(),
            NewDouyinFetcher(),
            NewXiaohongshuFetcher(),
        },
    }
}

func (s *Scheduler) Start() {
    // 启动时立即抓取一次
    s.fetchAll()

    // 每小时抓取
    s.cron.AddFunc("@hourly", s.fetchAll)

    // 每天凌晨 3 点清理 24 小时前数据
    s.cron.AddFunc("0 3 * * *", s.cleanup)

    s.cron.Start()
    log.Println("crawler scheduler started")
}

func (s *Scheduler) Stop() {
    s.cron.Stop()
}

func (s *Scheduler) fetchAll() {
    for _, f := range s.fetchers {
        items, err := f.Fetch()
        if err != nil {
            log.Printf("crawler [%s] error: %v", f.Platform(), err)
            continue
        }
        if len(items) == 0 {
            continue
        }
        if err := s.repo.BulkInsertHotspots(items); err != nil {
            log.Printf("crawler [%s] save error: %v", f.Platform(), err)
        } else {
            log.Printf("crawler [%s] saved %d items", f.Platform(), len(items))
        }
    }
}

func (s *Scheduler) cleanup() {
    cutoff := time.Now().Add(-24 * time.Hour)
    if err := s.repo.CleanOldHotspots(cutoff); err != nil {
        log.Printf("crawler cleanup error: %v", err)
    }
}
```

- [ ] **Step 4: 在 material_repo.go 中补充 CleanOldHotspots**

```go
// 追加到 internal/repository/material_repo.go
func (r *MaterialRepo) CleanOldHotspots(before time.Time) error {
    return r.db.Where("fetched_at < ?", before).Delete(&model.Hotspot{}).Error
}
```

- [ ] **Step 5: 编译验证（fetcher 实现还未写，先用空实现）**

```bash
cd /data/code/content_creator_imm/server && go build ./...
```

- [ ] **Step 6: Commit**

```bash
cd /data/code/content_creator_imm
git add server/
git commit -m "feat: add crawler scheduler and fetcher interface"
```

---

## Task 2: 微博热搜 Fetcher

**Files:**
- Create: `server/internal/crawler/weibo.go`

- [ ] **Step 1: 创建 weibo.go**

微博热搜使用公开接口 `https://weibo.com/ajax/side/hotSearch`（JSON 格式，无需登录）。

```go
// internal/crawler/weibo.go
package crawler

import (
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"
    "github.com/content-creator-imm/server/internal/model"
)

type WeiboFetcher struct{}

func NewWeiboFetcher() *WeiboFetcher { return &WeiboFetcher{} }

func (f *WeiboFetcher) Platform() string { return "weibo" }

func (f *WeiboFetcher) Fetch() ([]model.Hotspot, error) {
    client := &http.Client{Timeout: 10 * time.Second}
    req, _ := http.NewRequest("GET", "https://weibo.com/ajax/side/hotSearch", nil)
    req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; bot/1.0)")
    req.Header.Set("Referer", "https://weibo.com")

    resp, err := client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("weibo fetch: %w", err)
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }

    var result struct {
        Data struct {
            Realtime []struct {
                Word     string `json:"word"`
                Num      int64  `json:"num"`
                Realpos  int    `json:"realpos"`
                Scheme   string `json:"scheme"`
            } `json:"realtime"`
        } `json:"data"`
    }
    if err := json.Unmarshal(body, &result); err != nil {
        return nil, fmt.Errorf("weibo parse: %w", err)
    }

    now := time.Now()
    var items []model.Hotspot
    for _, item := range result.Data.Realtime {
        if item.Word == "" {
            continue
        }
        items = append(items, model.Hotspot{
            Platform:  "weibo",
            Title:     item.Word,
            Rank:      item.Realpos,
            HeatScore: item.Num,
            URL:       "https://s.weibo.com" + item.Scheme,
            FetchedAt: now,
        })
    }
    return items, nil
}
```

- [ ] **Step 2: 编译验证**

```bash
cd /data/code/content_creator_imm/server && go build ./...
```

- [ ] **Step 3: Commit**

```bash
cd /data/code/content_creator_imm
git add server/
git commit -m "feat: add weibo hot search fetcher"
```

---

## Task 3: 抖音热榜 Fetcher

**Files:**
- Create: `server/internal/crawler/douyin.go`

- [ ] **Step 1: 创建 douyin.go**

抖音使用公开热榜接口（无需登录）。

```go
// internal/crawler/douyin.go
package crawler

import (
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"
    "github.com/content-creator-imm/server/internal/model"
)

type DouyinFetcher struct{}

func NewDouyinFetcher() *DouyinFetcher { return &DouyinFetcher{} }

func (f *DouyinFetcher) Platform() string { return "douyin" }

func (f *DouyinFetcher) Fetch() ([]model.Hotspot, error) {
    client := &http.Client{Timeout: 10 * time.Second}
    url := "https://www.iesdouyin.com/web/api/v2/hotsearch/billboard/word/"
    req, _ := http.NewRequest("GET", url, nil)
    req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; bot/1.0)")

    resp, err := client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("douyin fetch: %w", err)
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }

    var result struct {
        WordList []struct {
            Word      string `json:"word"`
            HotValue  int64  `json:"hot_value"`
            Position  int    `json:"position"`
        } `json:"word_list"`
    }
    if err := json.Unmarshal(body, &result); err != nil {
        return nil, fmt.Errorf("douyin parse: %w", err)
    }

    now := time.Now()
    var items []model.Hotspot
    for _, item := range result.WordList {
        items = append(items, model.Hotspot{
            Platform:  "douyin",
            Title:     item.Word,
            Rank:      item.Position,
            HeatScore: item.HotValue,
            URL:       fmt.Sprintf("https://www.douyin.com/search/%s", item.Word),
            FetchedAt: now,
        })
    }
    return items, nil
}
```

- [ ] **Step 2: 编译验证**

```bash
cd /data/code/content_creator_imm/server && go build ./...
```

- [ ] **Step 3: Commit**

```bash
cd /data/code/content_creator_imm
git add server/
git commit -m "feat: add douyin hot search fetcher"
```

---

## Task 4: 小红书热门话题 Fetcher

**Files:**
- Create: `server/internal/crawler/xiaohongshu.go`

- [ ] **Step 1: 创建 xiaohongshu.go**

小红书无公开热榜 API，使用 colly 抓取热门话题页面。

```go
// internal/crawler/xiaohongshu.go
package crawler

import (
    "fmt"
    "strconv"
    "strings"
    "time"
    "github.com/gocolly/colly/v2"
    "github.com/content-creator-imm/server/internal/model"
)

type XiaohongshuFetcher struct{}

func NewXiaohongshuFetcher() *XiaohongshuFetcher { return &XiaohongshuFetcher{} }

func (f *XiaohongshuFetcher) Platform() string { return "xiaohongshu" }

func (f *XiaohongshuFetcher) Fetch() ([]model.Hotspot, error) {
    var items []model.Hotspot
    now := time.Now()
    rank := 1

    c := colly.NewCollector(
        colly.UserAgent("Mozilla/5.0 (compatible; bot/1.0)"),
    )
    c.SetRequestTimeout(10 * time.Second)

    // 解析热门话题列表（selector 需根据实际页面结构调整）
    c.OnHTML(".topic-item", func(e *colly.HTMLElement) {
        title := strings.TrimSpace(e.ChildText(".topic-title"))
        heatStr := strings.TrimSpace(e.ChildText(".topic-heat"))
        heatStr = strings.ReplaceAll(heatStr, "万", "0000")
        heat, _ := strconv.ParseInt(heatStr, 10, 64)
        if title == "" {
            return
        }
        items = append(items, model.Hotspot{
            Platform:  "xiaohongshu",
            Title:     title,
            Rank:      rank,
            HeatScore: heat,
            URL:       fmt.Sprintf("https://www.xiaohongshu.com/search_result?keyword=%s", title),
            FetchedAt: now,
        })
        rank++
    })

    err := c.Visit("https://www.xiaohongshu.com/explore")
    if err != nil {
        // 小红书反爬较强，失败时返回空列表而非报错，避免影响其他平台
        return []model.Hotspot{}, nil
    }
    return items, nil
}
```

- [ ] **Step 2: 编译验证**

```bash
cd /data/code/content_creator_imm/server && go build ./...
```

- [ ] **Step 3: Commit**

```bash
cd /data/code/content_creator_imm
git add server/
git commit -m "feat: add xiaohongshu hot topic fetcher"
```

---

## Task 5: 集成到主服务 + 验收测试

**Files:**
- Modify: `server/main.go`

- [ ] **Step 1: 在 main.go 中启动 Scheduler**

```go
// main.go main() 函数中，r.Run() 之前添加
crawlerScheduler := crawler.NewScheduler(materialRepo)
crawlerScheduler.Start()
defer crawlerScheduler.Stop()
```

- [ ] **Step 2: 完整编译验证**

```bash
cd /data/code/content_creator_imm/server && go build ./...
```

Expected: 无报错

- [ ] **Step 3: 运行服务验证爬虫启动**

```bash
cd /data/code/content_creator_imm/server
docker compose up -d db
sleep 3
go run . &
sleep 5
# 检查热点数据是否写入
curl "http://localhost:8080/api/v1/hotspot?platform=weibo" \
  -H "Authorization: Bearer <token>"
kill %1
docker compose down
```

Expected: 返回微博热搜列表（至少有数据，即使部分平台抓取失败也不影响服务启动）

- [ ] **Step 4: Commit**

```bash
cd /data/code/content_creator_imm
git add server/
git commit -m "feat: integrate crawler scheduler into main service - Plan 2 complete"
```

---

## 验收标准

- [ ] 服务启动时自动执行一次全平台抓取
- [ ] `GET /api/v1/hotspot?platform=weibo` 返回热搜数据
- [ ] `GET /api/v1/hotspot?platform=douyin` 返回热榜数据
- [ ] 某平台抓取失败不影响服务整体运行（降级为空列表）
- [ ] 24 小时前的数据自动清理

---

## 注意事项

- 各平台接口/页面结构可能变化，selector 和 URL 需定期维护
- 小红书反爬较强，抓取失败时静默降级，不报错
- 生产环境建议配置代理池（在 Fetcher 的 http.Client 中设置）
- 热点数据每小时刷新，Skill 本地缓存 TTL 设为 1 小时与之对应

---

## 下一步

- **Plan 3**: Admin Backend（React + Ant Design Pro 前端管理界面）
- **Plan 4**: Skill 改造（本地缓存层 + 远端 API 接入）
