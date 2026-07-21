package main

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// 允许的图片扩展名
var allowedExts = map[string]string{
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".gif":  "image/gif",
	".svg":  "image/svg+xml",
	".ico":  "image/x-icon",
	".webp": "image/webp",
}

// 允许的 Content-Type → 扩展名
var ctToExt = map[string]string{
	"image/png":    ".png",
	"image/jpeg":   ".jpg",
	"image/jpg":    ".jpg",
	"image/gif":    ".gif",
	"image/svg+xml": ".svg",
	"image/x-icon": ".ico",
	"image/webp":   ".webp",
}

// genFilename 生成唯一文件名
func genFilename(ext string) string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b) + ext
}

// handleUploadLogo 处理文件上传
// POST /api/admin/upload-logo  (multipart form, field: "file")
func handleUploadLogo(logosDir string) gin.HandlerFunc {
	return func(c *gin.Context) {
		file, err := c.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "未收到文件"})
			return
		}

		ext := strings.ToLower(filepath.Ext(file.Filename))
		ct, ok := allowedExts[ext]
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的图片格式，允许: png/jpg/jpeg/gif/svg/ico/webp"})
			return
		}

		// 限制文件大小 2MB
		if file.Size > 2*1024*1024 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "文件过大，最大 2MB"})
			return
		}

		filename := genFilename(ext)
		dst := filepath.Join(logosDir, filename)
		if err := c.SaveUploadedFile(file, dst); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "保存文件失败"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"path": "/logos/" + filename,
			"content_type": ct,
			"size":         file.Size,
		})
	}
}

// 管理后台抓取：管理员已受 IP 白名单保护，允许抓取局域网/内网地址（这是本产品的核心用法）。
var adminLogoClient = newHTTPClient(10*time.Second, false)

// handleFetchLogo 从 URL 下载图片到本地
// POST /api/admin/fetch-logo  {"url": "https://..."}
func handleFetchLogo(logosDir string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			URL string `json:"url"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
			return
		}
		if body.URL == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "url 不能为空"})
			return
		}
		if !strings.HasPrefix(body.URL, "http://") && !strings.HasPrefix(body.URL, "https://") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "url 必须以 http:// 或 https:// 开头"})
			return
		}

		u, err := url.Parse(body.URL)
		if err != nil || u.Hostname() == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "url 格式错误"})
			return
		}
		// 注：管理后台接口（/api/admin/fetch-logo）已受 IP 白名单保护，
		// 因此允许抓取局域网/内网地址，不做 SSRF 拦截（详见 adminLogoClient）。

		resp, err := adminLogoClient.Get(body.URL)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "下载失败: " + err.Error()})
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "源返回 " + resp.Status})
			return
		}

		ct := resp.Header.Get("Content-Type")
		// 清理 charset 等后缀
		if idx := strings.Index(ct, ";"); idx > 0 {
			ct = strings.TrimSpace(ct[:idx])
		}
		ext, ok := ctToExt[strings.ToLower(ct)]
		if !ok {
			// 回退：从 URL 路径提取扩展名
			ext = strings.ToLower(filepath.Ext(body.URL))
			if _, ok2 := allowedExts[ext]; !ok2 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "无法识别图片类型: " + ct})
				return
			}
		}

		// 限制 2MB
		data, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024+1))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "读取数据失败"})
			return
		}
		if len(data) > 2*1024*1024 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "文件过大，最大 2MB"})
			return
		}

		filename := genFilename(ext)
		dst := filepath.Join(logosDir, filename)
		if err := os.WriteFile(dst, data, 0644); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "保存文件失败"})
			return
		}

	c.JSON(http.StatusOK, gin.H{
		"path":        "/logos/" + filename,
		"content_type": ct,
		"size":         len(data),
	})
}

}

// handleUploadBg 处理背景图上传（复用图片校验逻辑，限制 8MB）
// POST /api/admin/upload-bg  (multipart form, field: "file")
// 返回 { path: "/logos/xxx.png" }
func handleUploadBg(logosDir string) gin.HandlerFunc {
	return func(c *gin.Context) {
		file, err := c.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "未收到文件"})
			return
		}

		ext := strings.ToLower(filepath.Ext(file.Filename))
		if _, ok := allowedExts[ext]; !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的图片格式，允许: png/jpg/jpeg/gif/svg/ico/webp"})
			return
		}

		// 背景图体积较大，限制 8MB
		if file.Size > 8*1024*1024 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "文件过大，最大 8MB"})
			return
		}

		filename := genFilename(ext)
		dst := filepath.Join(logosDir, filename)
		if err := c.SaveUploadedFile(file, dst); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "保存文件失败"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"path": "/logos/" + filename,
		})
	}
}
