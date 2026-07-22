// ===== 首页逻辑 =====
let config = null;
let scene = 'lan';
let selectedEngineId = 0;
let suggestTimer = null;
let suggestIndex = -1;
let suggestItems = [];

const $ = (s) => document.querySelector(s);
const $$ = (s) => document.querySelectorAll(s);

async function init() {
  try {
    const [cfgRes, sceneRes] = await Promise.all([
      fetch('/api/config').then(r => r.json()),
      fetch('/api/scene').then(r => r.json()),
    ]);
    config = cfgRes;
    applyTheme(config.theme);
    applyBackground(config);
    scene = sceneRes.scene;
    render();
    initScrollProgress();
  } catch (e) {
    $('#loading').textContent = '加载失败: ' + e.message;
  }
}

function getAvailableEngines() {
  return config.engines.filter(e => e.scopes.includes(scene));
}

function getDefaultEngine() {
  const engines = getAvailableEngines();
  if (scene === 'lan') return engines.find(e => e.is_default_lan) || engines[0];
  return engines.find(e => e.is_default_wan) || engines[0];
}

function getVisibleGroups() {
  return config.groups.filter(g => {
    if (g.visible_scenes === 'both') return true;
    return g.visible_scenes === scene;
  });
}

// sceneIcon 返回局域网(wifi)/公网(地球)图标 SVG
function sceneIcon(sc) {
  if (sc === 'lan') {
    return `<svg class="scene-svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M5 12.55a11 11 0 0 1 14.08 0"/><path d="M1.42 9a16 16 0 0 1 21.16 0"/><path d="M8.53 16.11a6 6 0 0 1 6.95 0"/><line x1="12" y1="20" x2="12.01" y2="20"/></svg>`;
  }
  return `<svg class="scene-svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><line x1="2" y1="12" x2="22" y2="12"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>`;
}

function sceneLabel(sc) {
  return sc === 'lan' ? '局域网' : '公网';
}

// applyTheme 将主题标识写到 <html data-theme>，由 CSS 变量驱动换肤
function applyTheme(t) {
  document.documentElement.dataset.theme = t || 'candy';
}

// applyBackground 应用自定义背景图与模糊强度（CSS 变量驱动）
// 注意：必须同时设置 id 和 class，因为 CSS 用 .bg-image-layer 类选择器，JS 用 id 查找
function applyBackground(cfg) {
  const root = document.documentElement;
  // bg_enable 为 false 时不启用自定义背景，显示主题渐变
  const enabled = cfg && cfg.bg_enable !== false;
  const img = enabled ? ((cfg && cfg.bg_image) || '').trim() : '';
  // 渐变层：默认显示，有背景图时隐藏
  let gradLayer = document.getElementById('bg-gradient-layer');
  if (!gradLayer) {
    gradLayer = document.createElement('div');
    gradLayer.id = 'bg-gradient-layer';
    gradLayer.className = 'bg-gradient-layer';
    document.body.insertBefore(gradLayer, document.body.firstChild);
  }
  // 背景图层
  let layer = document.getElementById('bg-image-layer');
  if (!layer) {
    layer = document.createElement('div');
    layer.id = 'bg-image-layer';
    layer.className = 'bg-image-layer';
    document.body.insertBefore(layer, document.body.firstChild);
  }
  layer.style.display = img ? 'block' : 'none';
  gradLayer.style.display = img ? 'none' : 'block';
  document.body.classList.toggle('bg-active', !!img);
  root.style.setProperty('--bg-image', img ? `url("${img.replace(/"/g, '\\"')}")` : 'none');
  root.style.setProperty('--bg-blur', ((cfg && cfg.bg_blur) || 0) + 'px');
  // 背景明暗调节：正=白遮罩增亮，负=黑遮罩增暗，0=标准（无遮罩）
  const bright = (cfg && cfg.bg_brightness != null) ? cfg.bg_brightness : 0;
  if (bright > 0) {
    root.style.setProperty('--bg-bright-color', '255, 255, 255');
    root.style.setProperty('--bg-bright-opacity', bright / 100);
  } else if (bright < 0) {
    root.style.setProperty('--bg-bright-color', '0, 0, 0');
    root.style.setProperty('--bg-bright-opacity', -bright / 100);
  } else {
    root.style.setProperty('--bg-bright-opacity', 0);
  }
  root.style.setProperty('--card-blur', ((cfg && cfg.card_blur != null) ? cfg.card_blur : 14) + 'px');
  // 卡片明暗调节：正=白遮罩增亮，负=黑遮罩增暗，0=标准（无遮罩）
  const cardBright = (cfg && cfg.card_brightness != null) ? cfg.card_brightness : 0;
  if (cardBright > 0) {
    root.style.setProperty('--card-bright-color', '255, 255, 255');
    root.style.setProperty('--card-bright-opacity', cardBright / 100);
  } else if (cardBright < 0) {
    root.style.setProperty('--card-bright-color', '0, 0, 0');
    root.style.setProperty('--card-bright-opacity', -cardBright / 100);
  } else {
    root.style.setProperty('--card-bright-opacity', 0);
  }
  root.style.setProperty('--bar-blur', ((cfg && cfg.bar_blur != null) ? cfg.bar_blur : 14) + 'px');
  // 搜索框明暗调节：正=白遮罩增亮，负=黑遮罩增暗，0=标准（无遮罩）
  const barBright = (cfg && cfg.bar_brightness != null) ? cfg.bar_brightness : 0;
  if (barBright > 0) {
    root.style.setProperty('--bar-bright-color', '255, 255, 255');
    root.style.setProperty('--bar-bright-opacity', barBright / 100);
  } else if (barBright < 0) {
    root.style.setProperty('--bar-bright-color', '0, 0, 0');
    root.style.setProperty('--bar-bright-opacity', -barBright / 100);
  } else {
    root.style.setProperty('--bar-bright-opacity', 0);
  }
  // 边框显示开关：控制搜索框和卡片白色线条
  const cardBorder = !(cfg && cfg.card_border === false);
  root.style.setProperty('--card-border-width', cardBorder ? '1px' : '0px');
  root.style.setProperty('--card-border-color', cardBorder ? 'rgba(255, 255, 255, 0.9)' : 'transparent');
  // 卡片大小：桌面端 / 移动端独立（%，默认100）
  root.style.setProperty('--card-scale-desktop', ((cfg && cfg.card_size != null) ? cfg.card_size : 100) / 100);
  root.style.setProperty('--card-scale-mobile', ((cfg && cfg.card_size_mobile != null) ? cfg.card_size_mobile : 100) / 100);
  // 图标大小：桌面端 / 移动端独立（%，默认100）
  root.style.setProperty('--icon-scale-desktop', ((cfg && cfg.icon_size != null) ? cfg.icon_size : 100) / 100);
  root.style.setProperty('--icon-scale-mobile', ((cfg && cfg.icon_size_mobile != null) ? cfg.icon_size_mobile : 100) / 100);
  // 文字大小：桌面端 / 移动端独立（%，默认100）
  root.style.setProperty('--text-scale-desktop', ((cfg && cfg.text_size != null) ? cfg.text_size : 100) / 100);
  root.style.setProperty('--text-scale-mobile', ((cfg && cfg.text_size_mobile != null) ? cfg.text_size_mobile : 100) / 100);
  // 单行最大数量：桌面端 / 移动端独立（默认6/4）
  root.style.setProperty('--max-cols-desktop', (cfg && cfg.max_cols != null) ? cfg.max_cols : 6);
  root.style.setProperty('--max-cols-mobile', (cfg && cfg.max_cols_mobile != null) ? cfg.max_cols_mobile : 4);
}

function render() {
  const engines = getAvailableEngines();
  const def = getDefaultEngine();
  selectedEngineId = def ? def.id : 0;

  const groups = getVisibleGroups();

  let html = '';
  if (config.show_title) {
    html += `<h1 class="home-title">${esc(config.site_title || '我的导航')}</h1>`;
  }
  html += `
    <div class="search-bar">
      <div class="search-row">
        <div class="engine-picker" id="engine-picker">
          <button type="button" class="engine-trigger" id="engine-trigger" onclick="toggleEngineMenu(event)" title="${esc(getCurrentEngine().name)}">
            <span class="engine-trigger-icon" id="engine-trigger-icon">${engineIconHtml(getCurrentEngine())}</span>
            <svg class="engine-chevron" viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="6 9 12 15 18 9"></polyline></svg>
          </button>
          <div class="engine-menu" id="engine-menu">
            ${engines.map(e => `<button type="button" class="engine-item ${e.id === selectedEngineId ? 'active' : ''}" data-id="${e.id}" onclick="onEnginePick(${e.id})">
              <span class="engine-item-icon">${engineIconHtml(e)}</span>
              <span class="engine-item-name">${esc(e.name)}</span>
              ${e.id === selectedEngineId ? '<svg class="engine-check" viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"></polyline></svg>' : ''}
            </button>`).join('')}
          </div>
        </div>
        <input class="search-input" id="search-input" placeholder="输入关键词搜索..." autocomplete="off"
          oninput="onSearchInput()" onkeydown="onSearchKey(event)" onfocus="onSearchInput()">
        <button class="search-btn" onclick="doSearch()" aria-label="搜索"><svg viewBox="0 0 24 24" width="24" height="24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg></button>
      </div>
      <div class="suggest-list" id="suggest-list"></div>
    </div>
  `;

  if (groups.length === 0) {
    html += `<div class="groups"><div class="loading">暂无站点</div></div>`;
  } else {
    html += `<div class="groups">`;
    for (const g of groups) {
      const sites = config.sites
        .filter(s => s.group_id === g.id)
        .sort((a, b) => a.sort_order - b.sort_order);
      if (sites.length === 0) continue;
      const showGroupName = config.show_group_name !== false; // 默认显示
      const titleHtml = showGroupName ? `<h2>${esc(g.name)}</h2>` : '';
      html += `
        <div class="group-section${showGroupName ? '' : ' no-title'}">
          ${titleHtml}
          <div class="site-grid">
            ${sites.map(s => renderSiteCard(s)).join('')}
          </div>
        </div>
      `;
    }
    html += `</div>`;
  }

  // 管理入口（仅局域网）：WiFi 图标按钮，点击进入管理页
  if (scene === 'lan') {
    const hideClass = config.show_scene ? '' : ' hidden';
    html += `<a class="admin-link${hideClass}" href="/admin.html" title="管理" aria-label="管理">${sceneIcon('lan')}</a>`;
  }

  // 页脚：自定义内容
  let footerContent = '';
  if (config.footer_html) {
    footerContent = `<div class="footer-content">${config.footer_enable_html ? config.footer_html : esc(config.footer_html)}</div>`;
  }
  html += `<footer class="home-footer">${footerContent}</footer>`;

  $('#app').innerHTML = html;
  applyHomeCompact();
}

// 图标过少（内容可在一屏内放下）时，让搜索框与整体内容垂直居中，顶部保留空间
function applyHomeCompact() {
  const cards = $$('.site-card').length;
  const fits = document.documentElement.scrollHeight <= window.innerHeight + 2;
  const compact = fits && cards <= 24;
  document.body.classList.toggle('home-compact', compact);
}

// 自定义滚动进度条：隐藏原生滚动条，改为右缘细进度指示，不抢占内容
function initScrollProgress() {
  if (document.querySelector('.scroll-progress')) return;
  const el = document.createElement('div');
  el.className = 'scroll-progress';
  el.innerHTML = '<div class="scroll-progress__bar"></div>';
  document.body.appendChild(el);
  const bar = el.querySelector('.scroll-progress__bar');
  const update = () => {
    const se = document.scrollingElement || document.documentElement;
    const max = se.scrollHeight - se.clientHeight;
    const pct = max > 0 ? Math.min(100, (se.scrollTop / max) * 100) : 0;
    bar.style.height = pct + '%';
  };
  window.addEventListener('scroll', update, { passive: true });
  window.addEventListener('resize', () => { update(); applyHomeCompact(); });
  update();
}

// 客户端 favicon：直接走公共图标服务，服务端不再代发请求（去除 SSRF / 缓存 DoS 面）
function faviconUrl(rawUrl) {
  try {
    const host = new URL(rawUrl).hostname;
    return 'https://icons.duckduckgo.com/ip3/' + encodeURIComponent(host) + '.ico';
  } catch (_) {
    return '';
  }
}

function renderSiteCard(s) {
  const src = s.logo_url || faviconUrl(s.url);
  const showName = config.show_site_name !== false; // 默认显示
  const nameHtml = showName ? `<div class="site-name">${esc(s.name)}</div>` : '';
  return `
    <a class="site-card${showName ? '' : ' no-name'}" href="${esc(s.url)}" target="_blank" rel="noopener" title="${esc(s.name)}">
      <div class="site-logo-wrap"><img class="site-logo" src="${esc(src)}" alt="${esc(s.name)}" data-name="${esc(s.name)}" onerror="onLogoError(this)"></div>
      ${nameHtml}
    </a>
  `;
}

function onLogoError(img) {
  const name = img.dataset.name || '?';
  const ch = name.charAt(0).toUpperCase();
  const colors = ['#4f46e5', '#0891b2', '#db2777', '#ea580c', '#16a34a', '#7c3aed'];
  const color = colors[name.charCodeAt(0) % colors.length];
  img.parentNode.innerHTML = `<div class="site-logo-fallback" style="background:${color}">${esc(ch)}</div>`;
}

// 引擎图标 HTML：优先用自定义 icon，否则取引擎域名的 favicon，再退化首字母
function engineIconHtml(e) {
  let src = e.icon;
  if (!src) {
    try {
      const origin = new URL(e.url_template.replace('%s', '')).origin;
      src = faviconUrl(origin);
    } catch (_) { src = ''; }
  }
  if (src) return `<img src="${esc(src)}" alt="${esc(e.name)}" onerror="this.style.display='none'">`;
  return `<span class="engine-initial">${esc((e.name || '?').trim().charAt(0))}</span>`;
}

// 点击引擎项切换当前搜索引擎
function onEnginePick(id) {
  selectedEngineId = id;
  const eng = config.engines.find(e => e.id === id);
  // 更新触发器图标
  const trig = $('#engine-trigger-icon');
  if (trig && eng) trig.innerHTML = engineIconHtml(eng);
  const trgBtn = $('#engine-trigger');
  if (trgBtn && eng) trgBtn.title = eng.name;
  // 更新菜单选中态与对勾
  $$('#engine-menu .engine-item').forEach(b => {
    const active = parseInt(b.dataset.id) === id;
    b.classList.toggle('active', active);
    const check = b.querySelector('.engine-check');
    if (active && !check) {
      b.insertAdjacentHTML('beforeend', '<svg class="engine-check" viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"></polyline></svg>');
    } else if (!active && check) {
      check.remove();
    }
  });
  hideEngineMenu();
  const q = $('#search-input').value.trim();
  if (q) { clearTimeout(suggestTimer); fetchSuggest(q); }
}

// 展开/收起引擎下拉
function toggleEngineMenu(e) {
  if (e) e.stopPropagation();
  const picker = $('#engine-picker');
  if (!picker) return;
  picker.classList.toggle('open');
}

// 收起引擎下拉
function hideEngineMenu() {
  const picker = $('#engine-picker');
  if (picker) picker.classList.remove('open');
}

function getCurrentEngine() {
  return config.engines.find(e => e.id === selectedEngineId);
}

function doSearch() {
  const q = $('#search-input').value.trim();
  if (!q) return;
  const engine = getCurrentEngine();
  if (!engine) return;
  const url = engine.url_template.replace('%s', encodeURIComponent(q));
  window.open(url, '_blank');
}

// 搜索建议
function onSearchInput() {
  const q = $('#search-input').value.trim();
  clearTimeout(suggestTimer);
  if (!q) {
    hideSuggest();
    return;
  }
  suggestTimer = setTimeout(() => fetchSuggest(q), 300);
}

async function fetchSuggest(q) {
  const engine = getCurrentEngine();
  if (!engine || !engine.suggestion_url) {
    hideSuggest();
    return;
  }
  try {
    const res = await fetch(`/api/suggest?engine_id=${engine.id}&q=${encodeURIComponent(q)}`);
    const data = await res.json();
    showSuggest(data.suggestions || []);
  } catch {
    hideSuggest();
  }
}

function showSuggest(items) {
  suggestItems = items;
  suggestIndex = -1;
  const el = $('#suggest-list');
  if (!items.length) {
    hideSuggest();
    return;
  }
  el.innerHTML = items.map((s, i) =>
    `<div class="suggest-item" onclick="pickSuggest(${i})">${esc(s)}</div>`
  ).join('');
  el.classList.add('show');
}

function hideSuggest() {
  const el = $('#suggest-list');
  if (el) el.classList.remove('show');
  suggestIndex = -1;
}

function pickSuggest(i) {
  $('#search-input').value = suggestItems[i];
  hideSuggest();
  doSearch();
}

function onSearchKey(e) {
  const el = $('#suggest-list');
  if (!el || !el.classList.contains('show')) {
    if (e.key === 'Enter') doSearch();
    return;
  }
  const items = $$('.suggest-item');
  if (e.key === 'ArrowDown') {
    e.preventDefault();
    suggestIndex = Math.min(suggestIndex + 1, items.length - 1);
    updateSuggestActive();
  } else if (e.key === 'ArrowUp') {
    e.preventDefault();
    suggestIndex = Math.max(suggestIndex - 1, -1);
    updateSuggestActive();
  } else if (e.key === 'Enter') {
    e.preventDefault();
    if (suggestIndex >= 0) {
      pickSuggest(suggestIndex);
    } else {
      hideSuggest();
      doSearch();
    }
  } else if (e.key === 'Escape') {
    hideSuggest();
  }
}

function updateSuggestActive() {
  $$('.suggest-item').forEach((el, i) => {
    el.classList.toggle('active', i === suggestIndex);
  });
}

// 点击外部关闭建议 / 引擎菜单
document.addEventListener('click', (e) => {
  if (!e.target.closest('.search-bar')) hideSuggest();
  if (!e.target.closest('.engine-picker')) hideEngineMenu();
});

function esc(s) {
  const d = document.createElement('div');
  d.textContent = s || '';
  return d.innerHTML;
}

init();
