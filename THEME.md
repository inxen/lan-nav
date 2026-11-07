# 主题开发文档（lan-nav）

> 目标：让导航站支持多套外观主题，用户可在「管理页 → 其他设置 → 主题外观」自由切换；开发者可低成本新增自定义主题。

---

## 1. 主题机制

采用 **「CSS 变量 + `data-theme` 属性」** 的换肤方案，**无需重新编译前端 JS**。

- 所有颜色/渐变/圆角等视觉参数都抽成 CSS 变量，集中在 `backend/web/css/style.css` 顶部的 `:root` 与 `[data-theme="xxx"]` 块中。
- 前端在拿到配置后，把主题标识写到根元素：
  ```js
  document.documentElement.dataset.theme = config.theme || 'candy';
  ```
- 浏览器自动用当前主题块里的变量值渲染全部组件。**切换主题即时生效，不需要刷新页面以外操作**。

数据流：

```
管理页保存 theme  ──PUT /api/admin/settings──▶  settings 表 (key="theme")
                                                  │
首页/管理页加载  ──GET /api/config──▶  config.theme  ──▶  <html data-theme="...">  ──▶  CSS 变量生效
```

---

## 2. 内置主题

| 标识 (theme) | 中文名 | 风格 |
|---|---|---|
| `candy`    | 糖果   | 粉紫（默认） |
| `mint`     | 薄荷   | 清新绿蓝 |
| `sunset`   | 晚霞   | 暖橙粉 |
| `ocean`    | 海洋   | 蓝绿 |
| `lavender` | 薰衣草 | 紫调 |
| `dark`     | 暗夜   | 奶油黑（深色） |

`dark` 是深色主题，覆盖了背景、卡片、输入框等「表面」变量；其余为浅色主题，仅改主色/渐变/光斑等。

---

## 3. 完整变量清单

定义在 `:root`（= candy 默认值），各主题块只需覆盖要变化的部分。

### 颜色
| 变量 | 含义 |
|---|---|
| `--primary` | 主色（按钮/激活态/图标） |
| `--primary-strong` | 深主色（用于浅色背景上的文字/链接，保证对比度） |
| `--primary-hover` | 主色悬停态 |
| `--accent` | 点缀色 |
| `--primary-grad` | 主渐变（按钮、激活 Tab、标题装饰圆点） |
| `--title-grad` | 标题文字渐变 |
| `--text` | 主文字色 |
| `--text-secondary` | 次要文字 |
| `--text-tertiary` | 占位符/弱化文字 |

### 边框与卡片
| 变量 | 含义 |
|---|---|
| `--border` | 默认边框 |
| `--border-hover` | 悬停边框 |
| `--card-bg` | 站点卡片背景（**有实际底色**，卡片靠它形成可见表面，不再是纯透明） |
| `--card-border` | 卡片/搜索框/表单的描边色（暗色主题请用低透明度白） |
| `--card-shadow` | 卡片阴影（保留兼容，组件已统一使用 `--surface-shadow`） |
| `--primary-light` | 主色浅色（输入框聚焦光环、选中底） |
| `--on-primary` | 主色/主渐变背景上的文字色（按钮、激活 Tab、首字母图标） |

### 阴影（组件统一使用，按需覆盖）
| 变量 | 含义 |
|---|---|
| `--shadow-tint` | 阴影基色 RGB 分量，如 `48, 38, 72`；深色主题设为 `0, 0, 0` |
| `--surface-shadow` | 卡片/搜索栏默认阴影 |
| `--surface-shadow-hover` | 悬停阴影 |
| `--surface-shadow-lg` | 浮层阴影（下拉菜单、弹窗） |

> 阴影不要写死 `rgba(199,150,220,…)` 这类主题色值——换主题后会残留粉紫影。统一用 `rgba(var(--shadow-tint), α)` 推导，或直接用上面三个阴影变量。


### 表面（深色主题重点覆盖）
| 变量 | 含义 |
|---|---|
| `--bar-bg` | 搜索栏背景 |
| `--panel-bg` | 表单/列表项背景 |
| `--input-bg` | 输入框背景 |
| `--suggest-bg` | 搜索补全下拉背景 |

### 背景与装饰
| 变量 | 含义 |
|---|---|
| `--bg-gradient` | 页面背景渐变（配合 `bgShift` 动画流动） |
| `--blob1` / `--blob2` | 两个漂浮光斑颜色 |
| `--logo-bg` | 站点 logo 底色 |

### 形状（通常无需改）
`--radius`(20px) / `--radius-sm`(12px) / `--radius-pill`(999px) / `--transition`

`--icon-basis`(68px) 是卡内图标的基准边长，实际尺寸 = `--card-scale` × `--icon-scale` × 该值；移动端媒体查询内会覆写为 60px。图标不随卡片容器查询单位缩放，改卡片大小后无需再调图标比例。

---

## 4. 如何新增一套主题

以新增 `forest`（森林绿）为例：

### 步骤 1：在 `style.css` 加主题块

在 `[data-theme="dark"] { ... }` 之后追加（只写要覆盖的变量）：

```css
/* 森林 forest — 深绿 */
[data-theme="forest"] {
  --primary: #4ade80;
  --primary-strong: #16a34a;
  --primary-hover: #22c55e;
  --accent: #2dd4bf;
  --primary-grad: linear-gradient(135deg, #86efac 0%, #5eead4 100%);
  --title-grad: linear-gradient(135deg, #4ade80, #2dd4bf);
  --text: #3f5a4a;
  --text-secondary: #6b8576;
  --text-tertiary: #a3c2ad;
  --border: #cfe9d8;
  --border-hover: #a7d8bd;
  --card-bg: rgba(255, 255, 255, 0.8);
  --card-border: rgba(255, 255, 255, 0.9);
  --card-shadow: 0 8px 28px rgba(74, 222, 128, 0.16);
  --primary-light: rgba(74, 222, 128, 0.14);
  --bar-bg: rgba(255, 255, 255, 0.92);
  --panel-bg: rgba(255, 255, 255, 0.92);
  --input-bg: #fff;
  --suggest-bg: rgba(255, 255, 255, 0.97);
  --bg-gradient: linear-gradient(135deg, #e6fff0 0%, #e0f7ee 50%, #eafaf2 100%);
  --blob1: #a7f3cf;
  --blob2: #99f6e0;
  --logo-bg: linear-gradient(135deg, #e6fff0, #e0f7f2);
}
```

> 深色主题务必同时覆盖 `--bar-bg` / `--panel-bg` / `--input-bg` / `--suggest-bg` / `--text*` / `--border*`，否则文字会与深色背景撞色。

### 步骤 2：后端白名单放行

编辑 `backend/handlers.go` 的 `validThemes`，加入新标识，否则保存会被 400 拒绝：

```go
var validThemes = map[string]bool{
    "candy": true, "mint": true, "sunset": true,
    "ocean": true, "lavender": true, "dark": true,
    "forest": true, // 新增
}
```

### 步骤 3：管理页下拉加选项

编辑 `backend/web/admin.html`，在「主题外观」的 `<select id="setting-theme">` 内加一项：

```html
<option value="forest">森林 · 深绿</option>
```

### 步骤 4：重新编译

CSS 与 HTML 通过 `//go:embed all:web` 编译进二进制，**改完必须重新 build**：

```bash
cd backend
export PATH="$PATH:/c/Users/cinho/.workbuddy/binaries/go/go/bin"
export GOPATH="C:/Users/cinho/.workbuddy/go"
export GOPROXY="goproxy.cn"
go build -o lan-nav.exe .
```

重启服务即可在「其他设置」看到并切换到新主题。

---

## 5. 关键文件索引

| 文件 | 作用 |
|---|---|
| `backend/web/css/style.css` | 主题变量定义 + 各主题块 + 组件样式 |
| `backend/web/js/app.js` | `applyTheme()` 将 `config.theme` 写到 `<html data-theme>` |
| `backend/web/js/admin.js` | 设置页读取/保存 `theme` |
| `backend/web/admin.html` | 「主题外观」下拉选项 |
| `backend/models.go` | `ConfigResponse.Theme` 字段 |
| `backend/store.go` | `GetConfig()` 读取 `theme`（默认 `candy`） |
| `backend/handlers.go` | `validThemes` 白名单 + `updateSettings` 校验与保存 |

---

## 6. 设计约定

- **默认主题**：`candy`。`settings` 表中 `theme` 为空时回退到它。
- **对比度**：浅色背景上的重点文字用 `--primary-strong`（深主色），不要直接用 `--primary`，避免看不清。
- **渐变上的文字**：主渐变偏浅，白字对比度不足（约 2:1）。按钮、激活 Tab、首字母图标统一用 `--on-primary`（深墨色，默认 `#2a1f36`），不要写死 `#fff`。
- **卡片必须有表面**：`--card-bg` 的不透明度建议 ≥ 0.55。卡片现在是"有色表面 + 描边 + 阴影"三层结构，若把 `--card-bg` 设为 `transparent`，卡片会完全融进背景（这正是旧版的主要视觉问题）。
- **新增主题向后兼容**：只增不改既有变量名；若需新增变量，记得在 `:root` 给默认值，否则旧主题块会缺变量。
- **动画**：`bgShift` / `float1` / `float2` 是背景流动与光斑漂浮动画，主题切换不影响它们（只换颜色）。
- **无障碍**：全站已有 `:focus-visible` 焦点环与 `prefers-reduced-motion` 降级，新增组件请沿用，不要用 `outline: none` 直接抹掉焦点。
