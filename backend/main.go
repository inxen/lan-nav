package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

//go:embed all:web
var webFS embed.FS

func main() {
	port := flag.String("port", "", "监听端口（默认 5725，可被 <data-dir>/config.json 的 port 字段覆盖）")
	dataDir := flag.String("data-dir", "", "用户数据目录（存放数据库与 logos）；默认 <程序所在目录>/data，可用环境变量 DATA_DIR 覆盖")
	dbPath := flag.String("db", "", "数据库文件路径（覆盖 data-dir 推导值，默认 <data-dir>/data.db）")
	logosDir := flag.String("logos-dir", "", "Logo 存储目录（覆盖 data-dir 推导值，默认 <data-dir>/logos）")
	flag.Parse()

	// 解析统一数据目录，并在使用默认目录时做一次旧数据迁移
	finalDataDir, usingDefault := resolveDataDir(*dataDir)
	maybeMigrateLegacy(finalDataDir, usingDefault)

	// 推导 db / logos 路径（允许单独用 --db / --logos-dir 覆盖）
	finalDB := *dbPath
	if finalDB == "" {
		finalDB = filepath.Join(finalDataDir, "data.db")
	}
	finalLogos := *logosDir
	if finalLogos == "" {
		finalLogos = filepath.Join(finalDataDir, "logos")
	}

	// 解析监听端口：--port 显式传入 > <data-dir>/config.json 的 port 字段 > 默认 5725
	finalPort := resolvePort(*port, finalDataDir)

	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(finalDB), 0755); err != nil {
		log.Fatalf("创建数据目录失败: %v", err)
	}
	if err := os.MkdirAll(finalLogos, 0755); err != nil {
		log.Fatalf("创建 logos 目录失败: %v", err)
	}
	logosAbs, _ := filepath.Abs(finalLogos)

	log.Printf("数据目录: %s（数据库=%s，logos=%s）", finalDataDir, finalDB, logosAbs)

	store, err := NewStore(finalDB)
	if err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}
	defer store.Close()

	h := newHandlers(store)

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// 公开 API（无需白名单）
	api := r.Group("/api")
	{
		api.GET("/config", h.getConfig)
		api.GET("/scene", h.detectScene)
		api.GET("/suggest", handleSuggest(store))
	}

	// 管理 API（IP 白名单保护，公网隐藏）
	admin := r.Group("/api/admin", ipWhitelist(store))
	{
		admin.POST("/engines", h.createEngine)
		admin.PUT("/engines/:id", h.updateEngine)
		admin.DELETE("/engines/:id", h.deleteEngine)

		admin.POST("/groups", h.createGroup)
		admin.PUT("/groups/:id", h.updateGroup)
		admin.DELETE("/groups/:id", h.deleteGroup)

		admin.POST("/sites", h.createSite)
		admin.PUT("/sites/:id", h.updateSite)
		admin.DELETE("/sites/:id", h.deleteSite)
		admin.POST("/sites/reorder", h.reorderSites)

		// Logo 上传 & URL 抓取
		admin.POST("/upload-logo", handleUploadLogo(logosAbs))
		admin.POST("/fetch-logo", handleFetchLogo(logosAbs))

		// 背景图上传
		admin.POST("/upload-bg", handleUploadBg(logosAbs))

		// 其他设置
		admin.PUT("/settings", h.updateSettings)
	}

	// Logo 静态文件服务（公开，前端直接 <img src="/logos/xxx.png">）
	r.Static("/logos", logosAbs)

	// 前端静态文件（纯 HTML/JS，无构建步骤）
	serveFrontend(r, store)

	log.Printf("lan-nav 启动于 http://0.0.0.0:%s", finalPort)
	if err := r.Run(":" + finalPort); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}

// serveFrontend 从 embed.FS 提供前端静态文件
func serveFrontend(r *gin.Engine, store *Store) {
	sub, err := fs.Sub(webFS, "web")
	if err != nil {
		log.Fatalf("加载内嵌资源失败: %v", err)
	}
	fileServer := http.FileServer(http.FS(sub))

	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path

		// 管理页 IP 判定：仅局域网放行，公网直接跳回首页（不返回 404）
		if path == "/admin" || path == "/admin.html" || strings.HasPrefix(path, "/admin/") {
			ci := clientIPInfo(c.Request)
			lanCIDRs := store.GetSetting("lan_cidrs", "")
			if !adminAllowed(ci, lanCIDRs) {
				log.Printf("[安全] 拒绝访问后台: %s (路径=%s, lan_cidrs=%q)", ci.IP, path, lanCIDRs)
				c.Redirect(http.StatusFound, "/")
				return
			}
			c.Request.URL.Path = "/admin.html"
			fileServer.ServeHTTP(c.Writer, c.Request)
			return
		}

		// 尝试匹配真实静态文件（JS/CSS/图片等）
		cleanPath := strings.TrimPrefix(path, "/")
		if cleanPath != "" {
			if f, err := sub.Open(cleanPath); err == nil {
				f.Close()
				fileServer.ServeHTTP(c.Writer, c.Request)
				return
			}
		}

		// 其余未知地址（含错误的 API/资源路径）统一回退首页，不出现 404
		c.Request.URL.Path = "/"
		fileServer.ServeHTTP(c.Writer, c.Request)
	})
}

// resolveDataDir 计算统一数据目录
// 优先级：--data-dir > 环境变量 DATA_DIR > 默认（程序目录/data，go run 时回退 cwd/data）
func resolveDataDir(flagVal string) (string, bool) {
	if flagVal != "" {
		// 去掉末尾分隔符，避免 filepath.Join 产生重复斜杠（如 /data/ + data.db）
		return strings.TrimRight(flagVal, `/\`), false
	}
	if env := os.Getenv("DATA_DIR"); env != "" {
		return env, false
	}
	// 默认：程序所在目录下的 data/。
	// go run 会把可执行文件放到临时构建目录，此时用 cwd 更合理。
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		if !strings.Contains(dir, "go-build") && !strings.HasPrefix(dir, os.TempDir()) {
			return filepath.Join(dir, "data"), true
		}
	}
	return "data", true
}

// maybeMigrateLegacy 在使用默认数据目录时，把旧的 cwd 下的 data.db / logos 迁移进统一目录，
// 避免升级后丢失原有配置与上传的 logo。
func maybeMigrateLegacy(dataDir string, usingDefault bool) {
	if !usingDefault {
		return
	}
	targetDB := filepath.Join(dataDir, "data.db")
	if _, err := os.Stat(targetDB); err == nil {
		return // 已迁移
	}

	// 迁移 data.db
	if _, err := os.Stat("data.db"); err == nil {
		if err := os.MkdirAll(dataDir, 0755); err == nil {
			if err := os.Rename("data.db", targetDB); err == nil {
				log.Printf("检测到旧版 data.db，已迁移至 %s", targetDB)
			} else {
				log.Printf("迁移 data.db 失败（可能被其他进程占用）：%v", err)
			}
		}
	}

	// 迁移 logos/
	targetLogos := filepath.Join(dataDir, "logos")
	if fi, err := os.Stat("logos"); err == nil && fi.IsDir() {
		if err := os.MkdirAll(targetLogos, 0755); err == nil {
			entries, _ := os.ReadDir("logos")
			for _, e := range entries {
				src := filepath.Join("logos", e.Name())
				dst := filepath.Join(targetLogos, e.Name())
				if _, derr := os.Stat(dst); derr != nil {
					_ = os.Rename(src, dst)
				}
			}
			if left, _ := os.ReadDir("logos"); len(left) == 0 {
				_ = os.Remove("logos")
			}
			log.Printf("检测到旧版 logos/，已迁移至 %s", targetLogos)
		}
	}
}

// portValue 支持 config.json 中 "port" 写成字符串或数字（二者皆可）。
type portValue struct {
	value string
}

func (p *portValue) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		p.value = s
		return nil
	}
	var n json.Number
	if err := json.Unmarshal(b, &n); err == nil {
		p.value = n.String()
		return nil
	}
	return fmt.Errorf("invalid port value: %s", string(b))
}

// appConfig 对应 <data-dir>/config.json 的可配置项。
// 当前仅端口一项，便于整个 /data 目录随配置一键迁移。
type appConfig struct {
	Port portValue `json:"port"`
}

// loadAppConfig 读取 <data-dir>/config.json；文件缺失或解析失败时返回零值（不影响启动）。
func loadAppConfig(dataDir string) appConfig {
	var cfg appConfig
	b, err := os.ReadFile(filepath.Join(dataDir, "config.json"))
	if err != nil {
		return cfg
	}
	// 解析失败仅记录日志，不阻断启动（回退默认端口）
	if err := json.Unmarshal(b, &cfg); err != nil {
		log.Printf("读取 %s/config.json 失败（将使用默认端口）：%v", dataDir, err)
	}
	return cfg
}

// resolvePort 计算最终监听端口。
// 优先级：--port 显式传入 > <data-dir>/config.json 的 port 字段 > 默认 5725。
func resolvePort(flagVal, dataDir string) string {
	const defaultPort = "5725"
	if flagVal != "" {
		return flagVal
	}
	if cfg := loadAppConfig(dataDir); cfg.Port.value != "" {
		return cfg.Port.value
	}
	return defaultPort
}
