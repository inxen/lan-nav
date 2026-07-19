package main

import (
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// Handlers 封装所有 HTTP 处理函数
type Handlers struct {
	store *Store
}

func newHandlers(store *Store) *Handlers {
	return &Handlers{store: store}
}

// validThemes 允许的主题标识，需与前端 CSS 的 [data-theme="..."] 及管理页下拉一致
var validThemes = map[string]bool{
	"candy":    true,
	"mint":     true,
	"sunset":   true,
	"ocean":    true,
	"lavender": true,
	"dark":     true,
	"snow":     true,
}

// ---- 公开接口（无需白名单）----

// GET /api/config — 返回全部配置
func (h *Handlers) getConfig(c *gin.Context) {
	cfg, err := h.store.GetConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, cfg)
}

// GET /api/scene — 根据请求来源 IP 判断当前场景
func (h *Handlers) detectScene(c *gin.Context) {
	ip := clientIP(c.Request)
	lanCIDRs := h.store.GetSetting("lan_cidrs", "")
	scene := "wan"
	if isInLAN(ip, lanCIDRs) {
		scene = "lan"
	}
	c.JSON(http.StatusOK, SceneDetectResponse{Scene: scene})
}

// ---- Engine 管理 ----

func (h *Handlers) createEngine(c *gin.Context) {
	var e Engine
	if err := c.ShouldBindJSON(&e); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	id, err := h.store.CreateEngine(e)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	e.ID = id
	c.JSON(http.StatusOK, e)
}

func (h *Handlers) updateEngine(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var e Engine
	if err := c.ShouldBindJSON(&e); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	e.ID = id
	if err := h.store.UpdateEngine(e); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, e)
}

func (h *Handlers) deleteEngine(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.store.DeleteEngine(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ---- Group 管理 ----

func (h *Handlers) createGroup(c *gin.Context) {
	var g Group
	if err := c.ShouldBindJSON(&g); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	id, err := h.store.CreateGroup(g)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	g.ID = id
	c.JSON(http.StatusOK, g)
}

func (h *Handlers) updateGroup(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var g Group
	if err := c.ShouldBindJSON(&g); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	g.ID = id
	if err := h.store.UpdateGroup(g); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, g)
}

func (h *Handlers) deleteGroup(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.store.DeleteGroup(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ---- Site 管理 ----

func (h *Handlers) createSite(c *gin.Context) {
	var s Site
	if err := c.ShouldBindJSON(&s); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	id, err := h.store.CreateSite(s)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	s.ID = id
	c.JSON(http.StatusOK, s)
}

func (h *Handlers) updateSite(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var s Site
	if err := c.ShouldBindJSON(&s); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	s.ID = id
	if err := h.store.UpdateSite(s); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, s)
}

func (h *Handlers) deleteSite(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.store.DeleteSite(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ---- 其他设置 ----

// PUT /api/admin/settings — 更新站点名称等全局设置
func (h *Handlers) updateSettings(c *gin.Context) {
	var body struct {
		SiteTitle        string `json:"site_title"`
		ShowTitle        bool   `json:"show_title"`
		LanCIDRs         string `json:"lan_cidrs"`
		FooterHTML       string `json:"footer_html"`
		FooterEnableHTML bool   `json:"footer_enable_html"`
		Theme            string `json:"theme"`
		BGEnable         bool   `json:"bg_enable"`
		BGImage          string `json:"bg_image"`
		BGBlur           int    `json:"bg_blur"`
		CardBlur         int    `json:"card_blur"`
		BarBlur          int    `json:"bar_blur"`
		BGOpacity        int    `json:"bg_opacity"`
		CardOpacity      int    `json:"card_opacity"`
		BarOpacity       int    `json:"bar_opacity"`
		ShowSiteName     bool   `json:"show_site_name"`
		ShowGroupName    bool   `json:"show_group_name"`
		ShowScene        bool   `json:"show_scene"`
		CardBorder       bool   `json:"card_border"`
		CardSize         int    `json:"card_size"`
		CardSizeMobile   int    `json:"card_size_mobile"`
		IconSize         int    `json:"icon_size"`
		IconSizeMobile   int    `json:"icon_size_mobile"`
		TextSize         int    `json:"text_size"`
		TextSizeMobile   int    `json:"text_size_mobile"`
		MaxCols          int    `json:"max_cols"`
		MaxColsMobile    int    `json:"max_cols_mobile"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// 校验局域网网段格式
	for _, raw := range strings.Split(body.LanCIDRs, ",") {
		cidr := strings.TrimSpace(raw)
		if cidr == "" {
			continue
		}
		if _, _, err := net.ParseCIDR(cidr); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的网段格式: " + cidr})
			return
		}
	}
	// 校验主题
	if body.Theme != "" && !validThemes[body.Theme] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的主题: " + body.Theme})
		return
	}
	// 校验模糊强度范围
	if body.BGBlur < 0 || body.BGBlur > 40 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "背景模糊需在 0-40 之间"})
		return
	}
	if body.CardBlur < 0 || body.CardBlur > 40 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "卡片模糊需在 0-40 之间"})
		return
	}
	if body.BarBlur < 0 || body.BarBlur > 40 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "搜索框模糊需在 0-40 之间"})
		return
	}
	// 校验透明度范围
	if body.BGOpacity < 0 || body.BGOpacity > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "背景透明度需在 0-100 之间"})
		return
	}
	if body.CardOpacity < 0 || body.CardOpacity > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "卡片透明度需在 0-100 之间"})
		return
	}
	if body.BarOpacity < 0 || body.BarOpacity > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "搜索框透明度需在 0-100 之间"})
		return
	}
	// 校验卡片大小范围（桌面端 / 移动端，% 50-200）
	if body.CardSize < 50 || body.CardSize > 200 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "卡片大小（桌面端）需在 50-200 之间"})
		return
	}
	if body.CardSizeMobile < 50 || body.CardSizeMobile > 200 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "卡片大小（移动端）需在 50-200 之间"})
		return
	}
	// 校验图标大小范围（桌面端 / 移动端，% 50-200）
	if body.IconSize < 50 || body.IconSize > 200 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "图标大小（桌面端）需在 50-200 之间"})
		return
	}
	if body.IconSizeMobile < 50 || body.IconSizeMobile > 200 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "图标大小（移动端）需在 50-200 之间"})
		return
	}
	// 校验文字大小范围（桌面端 / 移动端，% 50-200）
	if body.TextSize < 50 || body.TextSize > 200 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文字大小（桌面端）需在 50-200 之间"})
		return
	}
	if body.TextSizeMobile < 50 || body.TextSizeMobile > 200 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文字大小（移动端）需在 50-200 之间"})
		return
	}
	// 校验单行最大数量范围（桌面端 / 移动端，1-12）
	if body.MaxCols < 1 || body.MaxCols > 12 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "单行最大数量（桌面端）需在 1-12 之间"})
		return
	}
	if body.MaxColsMobile < 1 || body.MaxColsMobile > 12 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "单行最大数量（移动端）需在 1-12 之间"})
		return
	}
	if err := h.store.SetSetting("site_title", body.SiteTitle); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	show := "0"
	if body.ShowTitle {
		show = "1"
	}
	if err := h.store.SetSetting("show_title", show); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := h.store.SetSetting("lan_cidrs", body.LanCIDRs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := h.store.SetSetting("footer_html", body.FooterHTML); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	fhtml := "0"
	if body.FooterEnableHTML {
		fhtml = "1"
	}
	if err := h.store.SetSetting("footer_enable_html", fhtml); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := h.store.SetSetting("theme", body.Theme); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	bgEnable := "0"
	if body.BGEnable {
		bgEnable = "1"
	}
	if err := h.store.SetSetting("bg_enable", bgEnable); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := h.store.SetSetting("bg_image", body.BGImage); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := h.store.SetSetting("bg_blur", strconv.Itoa(body.BGBlur)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := h.store.SetSetting("card_blur", strconv.Itoa(body.CardBlur)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := h.store.SetSetting("bar_blur", strconv.Itoa(body.BarBlur)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := h.store.SetSetting("bg_opacity", strconv.Itoa(body.BGOpacity)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := h.store.SetSetting("card_opacity", strconv.Itoa(body.CardOpacity)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := h.store.SetSetting("bar_opacity", strconv.Itoa(body.BarOpacity)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	showSiteName := "0"
	if body.ShowSiteName {
		showSiteName = "1"
	}
	if err := h.store.SetSetting("show_site_name", showSiteName); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	showGroupName := "0"
	if body.ShowGroupName {
		showGroupName = "1"
	}
	if err := h.store.SetSetting("show_group_name", showGroupName); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	showScene := "0"
	if body.ShowScene {
		showScene = "1"
	}
	if err := h.store.SetSetting("show_scene", showScene); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	cardBorder := "0"
	if body.CardBorder {
		cardBorder = "1"
	}
	if err := h.store.SetSetting("card_border", cardBorder); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := h.store.SetSetting("card_size", strconv.Itoa(body.CardSize)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := h.store.SetSetting("card_size_mobile", strconv.Itoa(body.CardSizeMobile)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := h.store.SetSetting("icon_size", strconv.Itoa(body.IconSize)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := h.store.SetSetting("icon_size_mobile", strconv.Itoa(body.IconSizeMobile)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := h.store.SetSetting("text_size", strconv.Itoa(body.TextSize)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := h.store.SetSetting("text_size_mobile", strconv.Itoa(body.TextSizeMobile)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := h.store.SetSetting("max_cols", strconv.Itoa(body.MaxCols)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := h.store.SetSetting("max_cols_mobile", strconv.Itoa(body.MaxColsMobile)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
