package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
	"strings"
)

// WebSearchConfig 联网搜索配置
type WebSearchConfig struct {
	Provider string // "serper" / "baidu"
	APIKey   string
}

// WebSearchService 联网搜索服务
type WebSearchService struct {
	config WebSearchConfig
	client *http.Client
}

// SearchResult 搜索结果
type SearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

// NewWebSearchService 创建联网搜索服务
func NewWebSearchService(config WebSearchConfig) *WebSearchService {
	return &WebSearchService{
		config: config,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Search 执行搜索
func (s *WebSearchService) Search(query string) ([]SearchResult, error) {
	switch s.config.Provider {
	case "serper":
		return s.searchSerper(query)
	case "baidu":
		return s.searchBaidu(query)
	default:
		return nil, fmt.Errorf("unsupported web search provider: %s", s.config.Provider)
	}
}

// searchSerper 使用 Serper (Google Search API) 搜索
func (s *WebSearchService) searchSerper(query string) ([]SearchResult, error) {
	url := "https://google.serper.dev/search"

	payload := map[string]string{
		"q": query,
	}
	jsonPayload, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("X-API-KEY", s.config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("serper API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var serperResp struct {
		Organic []struct {
			Title   string `json:"title"`
			Link    string `json:"link"`
			Snippet string `json:"snippet"`
		} `json:"organic"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&serperResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	// 转换为 SearchResult
	results := make([]SearchResult, 0, len(serperResp.Organic))
	for _, item := range serperResp.Organic {
		results = append(results, SearchResult{
			Title:   item.Title,
			URL:     item.Link,
			Snippet: item.Snippet,
		})
	}

	return results, nil
}

// searchBaidu 使用百度搜索 API 搜索
func (s *WebSearchService) searchBaidu(query string) ([]SearchResult, error) {
	// TODO: 实现百度搜索 API 集成
	// 参考: https://developer.baidu.com/search
	return nil, fmt.Errorf("baidu search not implemented yet")
}

// FormatSearchResults 格式化搜索结果为 Markdown 文本
func FormatSearchResults(results []SearchResult, maxResults int) string {
	if len(results) == 0 {
		return "未找到相关搜索结果"
	}

	if maxResults > 0 && len(results) > maxResults {
		results = results[:maxResults]
	}

	var sb strings.Builder
	sb.WriteString("## 联网搜索结果\n\n")

	for i, r := range results {
		sb.WriteString(fmt.Sprintf("### %d. %s\n", i+1, r.Title))
		sb.WriteString(fmt.Sprintf("来源: %s\n\n", r.URL))
		sb.WriteString(fmt.Sprintf("摘要: %s\n\n", r.Snippet))
		sb.WriteString("---\n\n")
	}

	return sb.String()
}

// Global web search service instance
var webSearchService *WebSearchService

// InitWebSearchService 初始化联网搜索服务
func InitWebSearchService(config WebSearchConfig) {
	webSearchService = NewWebSearchService(config)
}

// GetWebSearchService 获取联网搜索服务实例
func GetWebSearchService() *WebSearchService {
	return webSearchService
}