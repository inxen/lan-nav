package main

// Engine 搜索引擎配置
type Engine struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	URLTemplate   string `json:"url_template"`   // 含 %s 占位符，如 https://google.com/search?q=%s
	Icon          string `json:"icon"`            // 自定义图标 URL，空则自动获取 favicon
	Scopes        string `json:"scopes"`          // 可用场景: "lan" / "wan" / "lan,wan"
	IsDefaultLAN  bool   `json:"is_default_lan"`
	IsDefaultWAN  bool   `json:"is_default_wan"`
	SuggestionURL string `json:"suggestion_url"`  // 搜索建议 API，含 %s 占位符，空则不启用补全
	SortOrder     int    `json:"sort_order"`
}

// Group 站点分组
type Group struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	VisibleScenes string `json:"visible_scenes"` // 可见场景: "lan" / "wan" / "both"
	SortOrder     int    `json:"sort_order"`
}

// Site 站点书签
type Site struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	URL       string `json:"url"`
	LogoURL   string `json:"logo_url"` // 自定义 logo，空则自动获取 favicon
	GroupID   int    `json:"group_id"`
	SortOrder int    `json:"sort_order"`
}

// ConfigResponse 前端拉取的完整配置
type ConfigResponse struct {
	Engines          []Engine `json:"engines"`
	Groups           []Group  `json:"groups"`
	Sites            []Site   `json:"sites"`
	SiteTitle        string   `json:"site_title"`        // 首页站点名称（可自定义）
	ShowTitle        bool     `json:"show_title"`        // 是否显示站点名称
	LanCIDRs         string   `json:"lan_cidrs"`         // 局域网网段（逗号分隔 CIDR），空则按私有地址判定
	FooterHTML       string   `json:"footer_html"`       // 页脚自定义内容
	FooterEnableHTML bool     `json:"footer_enable_html"` // 是否将 footer_html 作为 HTML 渲染
	Theme            string   `json:"theme"`             // 主题标识，对应 CSS [data-theme]，空则默认 candy
	BGEnable         bool     `json:"bg_enable"`         // 是否启用自定义背景图；关闭时显示主题渐变
	BGImage          string   `json:"bg_image"`          // 自定义背景图 URL（空则使用主题渐变背景）
	BGBlur           int      `json:"bg_blur"`           // 背景图模糊强度（px，0-40）
	CardBlur         int      `json:"card_blur"`         // 导航卡片模糊强度（px，0-40）
	BarBlur          int      `json:"bar_blur"`          // 搜索框模糊强度（px，0-40），独立于卡片
	BGOpacity        int      `json:"bg_opacity"`        // 背景图透明度（0-100，100=完全不透明）
	CardOpacity      int      `json:"card_opacity"`      // 卡片背景透明度（0-100，默认80，仅影响卡片背景；模糊需半透明才可见）
	BarOpacity       int      `json:"bar_opacity"`       // 搜索框背景透明度（0-100，默认80，独立于卡片）
	ShowSiteName     bool     `json:"show_site_name"`    // 是否在首页卡片下显示站点名称（默认true）
	ShowGroupName    bool     `json:"show_group_name"`   // 是否在首页显示分组名称标题（默认true）
	ShowScene        bool     `json:"show_scene"`        // 是否显示右下角场景图标（局域网WiFi/公网地球）
	CardBorder       bool     `json:"card_border"`       // 是否显示搜索框和卡片的白色边框线条
	AllowedDomains   string   `json:"allowed_domains"`   // 允许调用 /api/suggest 的域名白名单（逗号分隔，可带端口），空则关闭同源校验
	CardSize         int      `json:"card_size"`         // 卡片大小（桌面端，% 默认100，范围50-200）
	CardSizeMobile   int      `json:"card_size_mobile"`  // 卡片大小（移动端，% 默认100，范围50-200）
	IconSize         int    `json:"icon_size"`         // 卡片图标大小（桌面端，% 默认100，范围50-200）
	IconSizeMobile   int    `json:"icon_size_mobile"`  // 卡片图标大小（移动端，% 默认100，范围50-200）
	TextSize         int    `json:"text_size"`         // 卡片文字大小（桌面端，% 默认100，范围50-200）
	TextSizeMobile   int    `json:"text_size_mobile"`  // 卡片文字大小（移动端，% 默认100，范围50-200）
	MaxCols          int    `json:"max_cols"`          // 单行最大卡片数量（桌面端，默认6，范围1-12）
	MaxColsMobile    int    `json:"max_cols_mobile"`   // 单行最大卡片数量（移动端，默认4，范围1-12）
}

// SceneDetectResponse 场景检测结果
type SceneDetectResponse struct {
	Scene string `json:"scene"` // "lan" 或 "wan"
}
