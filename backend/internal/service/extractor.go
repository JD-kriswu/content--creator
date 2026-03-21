package service

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// ExtractURL fetches the URL and extracts readable text content.
func ExtractURL(rawURL string) (string, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; ContentBot/1.0)")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // max 1MB
	if err != nil {
		return "", err
	}

	text := extractText(string(body))
	if len(text) < 100 {
		return "", fmt.Errorf("提取的文本过短，请直接粘贴文案内容")
	}
	if len(text) > 5000 {
		text = text[:5000]
	}
	return text, nil
}

func extractText(htmlContent string) string {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return htmlContent
	}

	var sb strings.Builder
	var skip bool
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			tag := strings.ToLower(n.Data)
			if tag == "script" || tag == "style" || tag == "nav" ||
				tag == "footer" || tag == "header" || tag == "aside" {
				skip = true
				defer func() { skip = false }()
			}
		}
		if !skip && n.Type == html.TextNode {
			t := strings.TrimSpace(n.Data)
			if len(t) > 10 {
				sb.WriteString(t)
				sb.WriteString("\n")
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return sb.String()
}

// IsURL checks if input looks like a URL
func IsURL(s string) bool {
	s = strings.TrimSpace(s)
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}
