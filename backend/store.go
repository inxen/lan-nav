package main

import (
	"database/sql"
	"fmt"
	"strconv"

	_ "modernc.org/sqlite"
)

// Store 封装 SQLite 数据访问
type Store struct {
	db *sql.DB
}

// NewStore 打开数据库并初始化表结构与种子数据
func NewStore(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	db.SetMaxOpenConns(1) // SQLite 写并发安全
	s := &Store{db: db}
	if err := s.init(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) init() error {
	schema := `
	CREATE TABLE IF NOT EXISTS engines (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		url_template TEXT NOT NULL,
		icon TEXT DEFAULT '',
		scopes TEXT DEFAULT 'lan,wan',
		is_default_lan INTEGER DEFAULT 0,
		is_default_wan INTEGER DEFAULT 0,
		sort_order INTEGER DEFAULT 0
	);
	CREATE TABLE IF NOT EXISTS groups (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		visible_scenes TEXT DEFAULT 'both',
		sort_order INTEGER DEFAULT 0
	);
	CREATE TABLE IF NOT EXISTS sites (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		url TEXT NOT NULL,
		logo_url TEXT DEFAULT '',
		group_id INTEGER DEFAULT 0,
		sort_order INTEGER DEFAULT 0
	);
	CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT
	);
	`
	if _, err := s.db.Exec(schema); err != nil {
		return fmt.Errorf("init schema: %w", err)
	}
	// 迁移：为已存在的 engines 表添加 suggestion_url 列
	s.db.Exec("ALTER TABLE engines ADD COLUMN suggestion_url TEXT DEFAULT ''")
	// 迁移：为已有引擎填充默认 suggestion_url（旧库升级时列虽存在但值为空）
	s.migrateSuggestionURLs()
	return s.seed()
}

// migrateSuggestionURLs 按引擎名称补填默认 suggestion_url
func (s *Store) migrateSuggestionURLs() {
	defaults := map[string]string{
		"Google": "https://suggestqueries.google.com/complete/search?client=firefox&q=%s",
		"Bing":   "https://api.bing.com/qsonhs.aspx?type=cb&q=%s",
		"百度":    "https://suggestion.baidu.com/su?wd=%s&action=opensearch",
	}
	for name, su := range defaults {
		s.db.Exec("UPDATE engines SET suggestion_url=? WHERE name=? AND (suggestion_url='' OR suggestion_url IS NULL)", su, name)
	}
}

// seed 写入默认数据（仅首次）
func (s *Store) seed() error {
	var n int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM engines").Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	engines := []Engine{
		{Name: "Google", URLTemplate: "https://www.google.com/search?q=%s", Scopes: "lan,wan", IsDefaultLAN: true, SuggestionURL: "https://suggestqueries.google.com/complete/search?client=firefox&q=%s", SortOrder: 0},
		{Name: "Bing", URLTemplate: "https://www.bing.com/search?q=%s", Scopes: "lan,wan", IsDefaultWAN: true, SuggestionURL: "https://api.bing.com/qsonhs.aspx?type=cb&q=%s", SortOrder: 1},
		{Name: "百度", URLTemplate: "https://www.baidu.com/s?wd=%s", Scopes: "lan,wan", SuggestionURL: "https://suggestion.baidu.com/su?wd=%s&action=opensearch", SortOrder: 2},
	}
	for _, e := range engines {
		if _, err := s.CreateEngine(e); err != nil {
			return err
		}
	}
	groups := []Group{
		{Name: "常用", VisibleScenes: "both", SortOrder: 0},
		{Name: "内网服务", VisibleScenes: "lan", SortOrder: 1},
	}
	for _, g := range groups {
		if _, err := s.CreateGroup(g); err != nil {
			return err
		}
	}
	sites := []Site{
		{Name: "GitHub", URL: "https://github.com", GroupID: 1, SortOrder: 0},
		{Name: "Google", URL: "https://google.com", GroupID: 1, SortOrder: 1},
		{Name: "YouTube", URL: "https://youtube.com", GroupID: 1, SortOrder: 2},
	}
	for _, st := range sites {
		if _, err := s.CreateSite(st); err != nil {
			return err
		}
	}
	return nil
}

// ---------------- Engine CRUD ----------------

func (s *Store) GetEngines() ([]Engine, error) {
	rows, err := s.db.Query("SELECT id, name, url_template, icon, scopes, is_default_lan, is_default_wan, suggestion_url, sort_order FROM engines ORDER BY sort_order, id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Engine
	for rows.Next() {
		var e Engine
		var dLAN, dWAN int
		if err := rows.Scan(&e.ID, &e.Name, &e.URLTemplate, &e.Icon, &e.Scopes, &dLAN, &dWAN, &e.SuggestionURL, &e.SortOrder); err != nil {
			return nil, err
		}
		e.IsDefaultLAN = dLAN == 1
		e.IsDefaultWAN = dWAN == 1
		list = append(list, e)
	}
	return list, nil
}

func (s *Store) CreateEngine(e Engine) (int, error) {
	dLAN, dWAN := 0, 0
	if e.IsDefaultLAN {
		dLAN = 1
	}
	if e.IsDefaultWAN {
		dWAN = 1
	}
	if e.Scopes == "" {
		e.Scopes = "lan,wan"
	}
	res, err := s.db.Exec("INSERT INTO engines (name, url_template, icon, scopes, is_default_lan, is_default_wan, suggestion_url, sort_order) VALUES (?,?,?,?,?,?,?,?)",
		e.Name, e.URLTemplate, e.Icon, e.Scopes, dLAN, dWAN, e.SuggestionURL, e.SortOrder)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	// 设为默认时清除其他默认
	if e.IsDefaultLAN {
		s.db.Exec("UPDATE engines SET is_default_lan=0 WHERE id!=?", id)
	}
	if e.IsDefaultWAN {
		s.db.Exec("UPDATE engines SET is_default_wan=0 WHERE id!=?", id)
	}
	return int(id), nil
}

func (s *Store) UpdateEngine(e Engine) error {
	dLAN, dWAN := 0, 0
	if e.IsDefaultLAN {
		dLAN = 1
	}
	if e.IsDefaultWAN {
		dWAN = 1
	}
	_, err := s.db.Exec("UPDATE engines SET name=?, url_template=?, icon=?, scopes=?, is_default_lan=?, is_default_wan=?, suggestion_url=?, sort_order=? WHERE id=?",
		e.Name, e.URLTemplate, e.Icon, e.Scopes, dLAN, dWAN, e.SuggestionURL, e.SortOrder, e.ID)
	if err != nil {
		return err
	}
	if e.IsDefaultLAN {
		s.db.Exec("UPDATE engines SET is_default_lan=0 WHERE id!=?", e.ID)
	}
	if e.IsDefaultWAN {
		s.db.Exec("UPDATE engines SET is_default_wan=0 WHERE id!=?", e.ID)
	}
	return nil
}

func (s *Store) DeleteEngine(id int) error {
	_, err := s.db.Exec("DELETE FROM engines WHERE id=?", id)
	return err
}

func (s *Store) GetEngineByID(id int) (*Engine, error) {
	var e Engine
	var dLAN, dWAN int
	err := s.db.QueryRow("SELECT id, name, url_template, icon, scopes, is_default_lan, is_default_wan, suggestion_url, sort_order FROM engines WHERE id=?", id).
		Scan(&e.ID, &e.Name, &e.URLTemplate, &e.Icon, &e.Scopes, &dLAN, &dWAN, &e.SuggestionURL, &e.SortOrder)
	if err != nil {
		return nil, err
	}
	e.IsDefaultLAN = dLAN == 1
	e.IsDefaultWAN = dWAN == 1
	return &e, nil
}

// ---------------- Group CRUD ----------------

func (s *Store) GetGroups() ([]Group, error) {
	rows, err := s.db.Query("SELECT id, name, visible_scenes, sort_order FROM groups ORDER BY sort_order, id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Group
	for rows.Next() {
		var g Group
		if err := rows.Scan(&g.ID, &g.Name, &g.VisibleScenes, &g.SortOrder); err != nil {
			return nil, err
		}
		list = append(list, g)
	}
	return list, nil
}

func (s *Store) CreateGroup(g Group) (int, error) {
	if g.VisibleScenes == "" {
		g.VisibleScenes = "both"
	}
	res, err := s.db.Exec("INSERT INTO groups (name, visible_scenes, sort_order) VALUES (?,?,?)",
		g.Name, g.VisibleScenes, g.SortOrder)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	return int(id), nil
}

func (s *Store) UpdateGroup(g Group) error {
	_, err := s.db.Exec("UPDATE groups SET name=?, visible_scenes=?, sort_order=? WHERE id=?",
		g.Name, g.VisibleScenes, g.SortOrder, g.ID)
	return err
}

func (s *Store) DeleteGroup(id int) error {
	_, err := s.db.Exec("DELETE FROM groups WHERE id=?", id)
	return err
}

// ---------------- Site CRUD ----------------

func (s *Store) GetSites() ([]Site, error) {
	rows, err := s.db.Query("SELECT id, name, url, logo_url, group_id, sort_order FROM sites ORDER BY group_id, sort_order, id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Site
	for rows.Next() {
		var st Site
		if err := rows.Scan(&st.ID, &st.Name, &st.URL, &st.LogoURL, &st.GroupID, &st.SortOrder); err != nil {
			return nil, err
		}
		list = append(list, st)
	}
	return list, nil
}

func (s *Store) CreateSite(st Site) (int, error) {
	res, err := s.db.Exec("INSERT INTO sites (name, url, logo_url, group_id, sort_order) VALUES (?,?,?,?,?)",
		st.Name, st.URL, st.LogoURL, st.GroupID, st.SortOrder)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	return int(id), nil
}

func (s *Store) UpdateSite(st Site) error {
	_, err := s.db.Exec("UPDATE sites SET name=?, url=?, logo_url=?, group_id=?, sort_order=? WHERE id=?",
		st.Name, st.URL, st.LogoURL, st.GroupID, st.SortOrder, st.ID)
	return err
}

func (s *Store) DeleteSite(id int) error {
	_, err := s.db.Exec("DELETE FROM sites WHERE id=?", id)
	return err
}

// GetConfig 一次性返回全部配置
func (s *Store) GetConfig() (*ConfigResponse, error) {
	engines, err := s.GetEngines()
	if err != nil {
		return nil, err
	}
	groups, err := s.GetGroups()
	if err != nil {
		return nil, err
	}
	sites, err := s.GetSites()
	if err != nil {
		return nil, err
	}
	siteTitle := s.GetSetting("site_title", "我的导航")
	showTitle := s.GetSetting("show_title", "1") == "1"
	lanCIDRs := s.GetSetting("lan_cidrs", "")
	footerHTML := s.GetSetting("footer_html", "")
	footerEnableHTML := s.GetSetting("footer_enable_html", "0") == "1"
	theme := s.GetSetting("theme", "candy")
	bgEnable := s.GetSetting("bg_enable", "1") == "1" // 默认启用（向后兼容）
	bgImage := s.GetSetting("bg_image", "")
	bgBlur := atoiDefault(s.GetSetting("bg_blur", "0"), 0)
	cardBlur := atoiDefault(s.GetSetting("card_blur", "14"), 14)
	bgBrightness := atoiDefault(s.GetSetting("bg_brightness", "0"), 0)
	cardBrightness := atoiDefault(s.GetSetting("card_brightness", "0"), 0)
	// 搜索框独立参数：未设置时回退到卡片值（向后兼容旧库）
	barBlur := atoiDefault(s.GetSetting("bar_blur", ""), cardBlur)
	barBrightness := atoiDefault(s.GetSetting("bar_brightness", "0"), 0)
	showSiteName := s.GetSetting("show_site_name", "1") == "1"
	showGroupName := s.GetSetting("show_group_name", "1") == "1"
	showScene := s.GetSetting("show_scene", "1") == "1"
	cardBorder := s.GetSetting("card_border", "1") == "1"
	allowedDomains := s.GetSetting("allowed_domains", "")
	cardSize := atoiDefault(s.GetSetting("card_size", "100"), 100)
	cardSizeMobile := atoiDefault(s.GetSetting("card_size_mobile", "100"), 100)
	iconSize := atoiDefault(s.GetSetting("icon_size", "100"), 100)
	iconSizeMobile := atoiDefault(s.GetSetting("icon_size_mobile", "100"), 100)
	textSize := atoiDefault(s.GetSetting("text_size", "100"), 100)
	textSizeMobile := atoiDefault(s.GetSetting("text_size_mobile", "100"), 100)
	maxCols := atoiDefault(s.GetSetting("max_cols", "6"), 6)
	maxColsMobile := atoiDefault(s.GetSetting("max_cols_mobile", "4"), 4)
	return &ConfigResponse{
		Engines:          engines,
		Groups:           groups,
		Sites:            sites,
		SiteTitle:        siteTitle,
		ShowTitle:        showTitle,
		LanCIDRs:         lanCIDRs,
		FooterHTML:       footerHTML,
		FooterEnableHTML: footerEnableHTML,
		Theme:            theme,
		BGEnable:         bgEnable,
		BGImage:          bgImage,
		BGBlur:           bgBlur,
		CardBlur:         cardBlur,
		BarBlur:          barBlur,
		BGBrightness:     bgBrightness,
		CardBrightness:   cardBrightness,
		BarBrightness:    barBrightness,
		ShowSiteName:     showSiteName,
		ShowGroupName:    showGroupName,
		ShowScene:        showScene,
		CardBorder:       cardBorder,
		AllowedDomains:   allowedDomains,
		CardSize:         cardSize,
		CardSizeMobile:   cardSizeMobile,
		IconSize:         iconSize,
		IconSizeMobile:   iconSizeMobile,
		TextSize:         textSize,
		TextSizeMobile:   textSizeMobile,
		MaxCols:          maxCols,
		MaxColsMobile:    maxColsMobile,
	}, nil
}

// GetSetting 读取设置项，不存在时返回默认值
func (s *Store) GetSetting(key, def string) string {
	var v string
	err := s.db.QueryRow("SELECT value FROM settings WHERE key=?", key).Scan(&v)
	if err != nil {
		return def
	}
	return v
}

// SetSetting 写入设置项
func (s *Store) SetSetting(key, value string) error {
	_, err := s.db.Exec("INSERT INTO settings(key, value) VALUES(?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value", key, value)
	return err
}

// Close 关闭数据库
func (s *Store) Close() error {
	return s.db.Close()
}

// atoiDefault 安全解析整数，失败时返回默认值
func atoiDefault(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}
