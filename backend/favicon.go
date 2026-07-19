package main

import (
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"net/url"
)

// faviconCache 内存缓存站点图标，避免重复请求
type faviconCache struct {
	mu    sync.RWMutex
	store map[string][]byte
}

func newFaviconCache() *faviconCache {
	return &faviconCache{store: make(map[string][]byte)}
}

func (fc *faviconCache) get(key string) ([]byte, bool) {
	fc.mu.RLock()
	defer fc.mu.RUnlock()
	v, ok := fc.store[key]
	return v, ok
}

func (fc *faviconCache) set(key string, data []byte) {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	fc.store[key] = data
}

var safeFavClient = safeHTTPClient(6 * time.Second)

// handleFavicon 代理获取站点 favicon 并缓存
// GET /api/favicon?url=https://github.com
func handleFavicon(fc *faviconCache) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := c.Query("url")
		if raw == "" {
			c.Status(http.StatusBadRequest)
			return
		}
		// 仅当完全无 scheme（裸主机名）时补 https://；带其它 scheme（ftp/file/...）的保持原样，
		// 后续 scheme 校验会直接拒绝，避免被误拼成 https:// 绕过。
		if !strings.Contains(raw, "://") {
			raw = "https://" + raw
		}
		u, err := url.Parse(raw)
		if err != nil || u.Host == "" {
			c.Status(http.StatusBadRequest)
			return
		}
		if u.Scheme != "http" && u.Scheme != "https" {
			c.Status(http.StatusBadRequest)
			return
		}
		if err := hostAllowed(u.Hostname()); err != nil {
			c.Status(http.StatusForbidden)
			return
		}
		origin := u.Scheme + "://" + u.Host

		// 查缓存
		if data, ok := fc.get(origin); ok {
			c.Data(http.StatusOK, "image/x-icon", data)
			return
		}

		// 请求站点 favicon.ico
		favURL := origin + "/favicon.ico"
		resp, err := safeFavClient.Get(favURL)
		if err == nil && resp.StatusCode == 200 {
			data, rerr := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
			resp.Body.Close()
			if rerr == nil && len(data) > 0 {
				fc.set(origin, data)
				ct := resp.Header.Get("Content-Type")
				if ct == "" {
					ct = "image/x-icon"
				}
				c.Data(http.StatusOK, ct, data)
				return
			}
		}
		if resp != nil {
			resp.Body.Close()
		}

		// 回退：尝试 DuckDuckGo 图标服务
		ddgURL := "https://icons.duckduckgo.com/ip3/" + u.Host + ".ico"
		resp2, err2 := safeFavClient.Get(ddgURL)
		if err2 == nil && resp2.StatusCode == 200 {
			data, rerr := io.ReadAll(io.LimitReader(resp2.Body, 512*1024))
			resp2.Body.Close()
			if rerr == nil && len(data) > 0 {
				fc.set(origin, data)
				c.Data(http.StatusOK, "image/x-icon", data)
				return
			}
		}
		if resp2 != nil {
			resp2.Body.Close()
		}

		// 全部失败：返回 204，前端用文字占位
		c.Status(http.StatusNoContent)
	}
}
