package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"golang.org/x/text/encoding/simplifiedchinese"
)

var suggestClient = &http.Client{Timeout: 5 * time.Second}

// decodeBody 检测响应编码并转为 UTF-8
// 部分搜索引擎（如百度）返回 GBK 编码
func decodeBody(body []byte, contentType string) []byte {
	// 已经是合法 UTF-8 则无需转换
	if utf8.Valid(body) {
		return body
	}

	// 检查 Content-Type 中的 charset
	ct := strings.ToLower(contentType)
	needGBK := false
	if strings.Contains(ct, "gbk") || strings.Contains(ct, "gb2312") || strings.Contains(ct, "gb18030") {
		needGBK = true
	}

	// 非 UTF-8 且未声明 charset，尝试 GBK（百度常见情况）
	if !needGBK && !utf8.Valid(body) {
		needGBK = true
	}

	if needGBK {
		if decoded, err := simplifiedchinese.GBK.NewDecoder().Bytes(body); err == nil {
			return decoded
		}
	}

	return body
}

// handleSuggest 代理搜索引擎建议 API
// GET /api/suggest?engine_id=1&q=keyword
func handleSuggest(store *Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		engineIDStr := c.Query("engine_id")
		q := c.Query("q")
		if q == "" || strings.TrimSpace(q) == "" {
			c.JSON(http.StatusOK, gin.H{"suggestions": []string{}})
			return
		}

		engineID := 0
		for _, ch := range engineIDStr {
			if ch < '0' || ch > '9' {
				engineID = -1
				break
			}
			engineID = engineID*10 + int(ch-'0')
		}
		if engineID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid engine_id"})
			return
		}

		engine, err := store.GetEngineByID(engineID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "engine not found"})
			return
		}
		if engine.SuggestionURL == "" {
			c.JSON(http.StatusOK, gin.H{"suggestions": []string{}})
			return
		}

		// 替换占位符（对查询关键词做 URL 编码，支持中文等非 ASCII 字符）
		suggestURL := strings.Replace(engine.SuggestionURL, "%s", url.QueryEscape(q), 1)

		req, err := http.NewRequest("GET", suggestURL, nil)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"suggestions": []string{}})
			return
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
		req.Header.Set("Accept", "application/json, text/plain, */*")

		resp, err := suggestClient.Do(req)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"suggestions": []string{}})
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			c.JSON(http.StatusOK, gin.H{"suggestions": []string{}})
			return
		}

		body, err := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"suggestions": []string{}})
			return
		}

		// 编码处理：部分搜索引擎（如百度）返回 GBK 编码，需转 UTF-8
		body = decodeBody(body, resp.Header.Get("Content-Type"))

		suggestions := parseSuggestions(body)
		c.JSON(http.StatusOK, gin.H{"suggestions": suggestions})
	}
}

// parseSuggestions 解析各种搜索引擎的建议 API 响应
// 支持: Google/Baidu [query, [s1,s2,...]], DuckDuckGo [{phrase:"s1"},...], Bing {AS:{Results:[{Suggests:[{Txt:"s1"},...]}]}}
func parseSuggestions(body []byte) []string {
	var raw interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return []string{}
	}

	// 尝试 Google/Baidu 格式: ["query", ["s1", "s2", ...]]
	if arr, ok := raw.([]interface{}); ok && len(arr) >= 2 {
		if suggestions, ok := arr[1].([]interface{}); ok {
			result := make([]string, 0, len(suggestions))
			for _, s := range suggestions {
				if str, ok := s.(string); ok && str != "" {
					result = append(result, str)
				}
			}
			if len(result) > 0 {
				return result
			}
		}
	}

	// 尝试 DuckDuckGo 格式: [{"phrase":"s1"}, {"phrase":"s2"}, ...]
	if arr, ok := raw.([]interface{}); ok {
		result := make([]string, 0, len(arr))
		for _, item := range arr {
			if m, ok := item.(map[string]interface{}); ok {
				if phrase, ok := m["phrase"].(string); ok && phrase != "" {
					result = append(result, phrase)
				}
			}
		}
		if len(result) > 0 {
			return result
		}
	}

	// 尝试 Bing 格式: {"AS":{"Results":[{"Suggests":[{"Txt":"s1"},...]}]}}
	if m, ok := raw.(map[string]interface{}); ok {
		if as, ok := m["AS"].(map[string]interface{}); ok {
			if results, ok := as["Results"].([]interface{}); ok {
				suggestions := make([]string, 0)
				for _, r := range results {
					if rm, ok := r.(map[string]interface{}); ok {
						if suggests, ok := rm["Suggests"].([]interface{}); ok {
							for _, s := range suggests {
								if sm, ok := s.(map[string]interface{}); ok {
									if txt, ok := sm["Txt"].(string); ok && txt != "" {
										suggestions = append(suggestions, txt)
									}
								}
							}
						}
					}
				}
				if len(suggestions) > 0 {
					return suggestions
				}
			}
		}
	}

	return []string{}
}
