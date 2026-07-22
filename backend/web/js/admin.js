// ===== 管理页逻辑 =====
let config = null;

const $ = (s) => document.querySelector(s);
const $$ = (s) => document.querySelectorAll(s);
const esc = (s) => { const d = document.createElement('div'); d.textContent = s || ''; return d.innerHTML; };

// 站点图标地址：优先 logo_url，否则用客户端 favicon（公共图标服务，服务端不代发请求）
function logoSrc(s) {
  if (s.logo_url) return s.logo_url;
  try {
    const u = new URL(s.url);
    return 'https://icons.duckduckgo.com/ip3/' + encodeURIComponent(u.hostname) + '.ico';
  } catch (e) {
    return '';
  }
}

// applyTheme 将主题标识写到 <html data-theme>，由 CSS 变量驱动换肤
function applyTheme(t) {
  document.documentElement.dataset.theme = t || 'candy';
}

// applyBackground 应用自定义背景图与模糊强度（CSS 变量驱动）
// 注意：必须同时设置 id 和 class，因为 CSS 用 .bg-image-layer 类选择器，JS 用 id 查找
function applyBackground(cfg) {
  const root = document.documentElement;
  // bg_enable 为 false 时不启用自定义背景（预览对象默认开启）
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

// ===== 初始化 =====
async function init() {
  // 防御：公网直接跳回首页（服务端已对 /admin 做 302，此处二次兜底）
  try {
    const sr = await fetch('/api/scene');
    if (sr.ok) {
      const sd = await sr.json();
      if (sd.scene === 'wan') { location.href = '/'; return; }
    }
  } catch (e) { /* 网络异常交由下方加载失败处理 */ }

  // Tab 切换
  $$('.tab').forEach(t => {
    t.onclick = () => switchTab(t.dataset.tab);
  });

  // Logo 文件上传
  $('#logo-file').onchange = (e) => handleFileUpload('logo', e);
  $('#elogo-file').onchange = (e) => handleFileUpload('elogo', e);

  try {
    await reload();
    initScrollProgress();
  } catch (e) {
    if (confirm('加载失败，可能不在局域网环境。返回首页？')) location.href = '/';
  }
}

async function reload() {
  const res = await fetch('/api/config');
  if (!res.ok) throw new Error('加载失败');
  config = await res.json();
  applyTheme(config.theme);
  applyBackground(config);
  renderAll();
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
  window.addEventListener('resize', update);
  update();
}

function switchTab(name) {
  $$('.tab').forEach(t => t.classList.toggle('active', t.dataset.tab === name));
  $$('.panel').forEach(p => p.classList.toggle('show', p.id === 'panel-' + name));
}

function renderAll() {
  renderSites();
  renderGroups();
  renderEngines();
  renderGroupSelect();
  renderSettings();
}

// ===== 其他设置 =====
function renderSettings() {
  if (!config) return;
  $('#setting-title').value = config.site_title || '';
  $('#setting-show-title').checked = !!config.show_title;
  $('#setting-lan-cidrs').value = config.lan_cidrs || '';
  $('#setting-footer-html').value = config.footer_html || '';
  $('#setting-footer-html-enable').checked = !!config.footer_enable_html;
  $('#setting-theme').value = config.theme || 'candy';
  $('#setting-bg-enable').checked = !!config.bg_enable;
  $('#setting-bg-image').value = config.bg_image || '';
  const bgBlur = config.bg_blur || 0;
  const cardBlur = (config.card_blur != null) ? config.card_blur : 14;
  const barBlur = (config.bar_blur != null) ? config.bar_blur : 14;
  $('#setting-bg-blur').value = bgBlur;
  $('#bg-blur-val').textContent = bgBlur;
  const bgBrightness = (config.bg_brightness != null) ? config.bg_brightness : 0;
  const cardBrightness = (config.card_brightness != null) ? config.card_brightness : 0;
  const barBrightness = (config.bar_brightness != null) ? config.bar_brightness : 0;
  $('#setting-bg-brightness').value = bgBrightness;
  $('#bg-bright-val').textContent = bgBrightness === 0 ? '标准' : (bgBrightness > 0 ? '增亮 ' + bgBrightness + '%' : '增暗 ' + (-bgBrightness) + '%');
  $('#setting-card-brightness').value = cardBrightness;
  $('#card-bright-val').textContent = cardBrightness === 0 ? '标准' : (cardBrightness > 0 ? '增亮 ' + cardBrightness + '%' : '增暗 ' + (-cardBrightness) + '%');
  $('#setting-card-blur').value = cardBlur;
  $('#card-blur-val').textContent = cardBlur;
  $('#setting-bar-blur').value = barBlur;
  $('#bar-blur-val').textContent = barBlur;
  $('#setting-bar-brightness').value = barBrightness;
  $('#bar-bright-val').textContent = barBrightness === 0 ? '标准' : (barBrightness > 0 ? '增亮 ' + barBrightness + '%' : '增暗 ' + (-barBrightness) + '%');
  const cardSize = (config.card_size != null) ? config.card_size : 100;
  const cardSizeMobile = (config.card_size_mobile != null) ? config.card_size_mobile : 100;
  $('#setting-card-size').value = cardSize;
  $('#card-size-val').textContent = cardSize + '%';
  $('#setting-card-size-mobile').value = cardSizeMobile;
  $('#card-size-mobile-val').textContent = cardSizeMobile + '%';
  const iconSize = (config.icon_size != null) ? config.icon_size : 100;
  const iconSizeMobile = (config.icon_size_mobile != null) ? config.icon_size_mobile : 100;
  $('#setting-icon-size').value = iconSize;
  $('#icon-size-val').textContent = iconSize + '%';
  $('#setting-icon-size-mobile').value = iconSizeMobile;
  $('#icon-size-mobile-val').textContent = iconSizeMobile + '%';
  const textSize = (config.text_size != null) ? config.text_size : 100;
  const textSizeMobile = (config.text_size_mobile != null) ? config.text_size_mobile : 100;
  $('#setting-text-size').value = textSize;
  $('#text-size-val').textContent = textSize + '%';
  $('#setting-text-size-mobile').value = textSizeMobile;
  $('#text-size-mobile-val').textContent = textSizeMobile + '%';
  const maxCols = (config.max_cols != null) ? config.max_cols : 6;
  const maxColsMobile = (config.max_cols_mobile != null) ? config.max_cols_mobile : 4;
  $('#setting-max-cols').value = maxCols;
  $('#max-cols-val').textContent = maxCols;
  $('#setting-max-cols-mobile').value = maxColsMobile;
  $('#max-cols-mobile-val').textContent = maxColsMobile;
  $('#setting-show-group-name').checked = config.show_group_name !== false;
  $('#setting-show-site-name').checked = config.show_site_name !== false;
  $('#setting-show-scene').checked = !!config.show_scene;
  $('#setting-card-border').checked = config.card_border !== false;
  $('#setting-allowed-domains').value = config.allowed_domains || '';
}

async function saveSettings() {
  const body = {
    site_title: $('#setting-title').value.trim() || '我的导航',
    show_title: $('#setting-show-title').checked,
    lan_cidrs: $('#setting-lan-cidrs').value.trim(),
    footer_html: $('#setting-footer-html').value,
    footer_enable_html: $('#setting-footer-html-enable').checked,
    theme: $('#setting-theme').value,
    bg_enable: $('#setting-bg-enable').checked,
    bg_image: $('#setting-bg-image').value.trim(),
    bg_blur: parseInt($('#setting-bg-blur').value) || 0,
    bg_brightness: parseInt($('#setting-bg-brightness').value) || 0,
    card_blur: parseInt($('#setting-card-blur').value) || 0,
    card_brightness: parseInt($('#setting-card-brightness').value) || 0,
    card_size: parseInt($('#setting-card-size').value) || 100,
    card_size_mobile: parseInt($('#setting-card-size-mobile').value) || 100,
    icon_size: parseInt($('#setting-icon-size').value) || 100,
    icon_size_mobile: parseInt($('#setting-icon-size-mobile').value) || 100,
    text_size: parseInt($('#setting-text-size').value) || 100,
    text_size_mobile: parseInt($('#setting-text-size-mobile').value) || 100,
    max_cols: parseInt($('#setting-max-cols').value) || 6,
    max_cols_mobile: parseInt($('#setting-max-cols-mobile').value) || 4,
    bar_blur: parseInt($('#setting-bar-blur').value) || 0,
    bar_brightness: parseInt($('#setting-bar-brightness').value) || 0,
    show_group_name: $('#setting-show-group-name').checked,
    show_site_name: $('#setting-show-site-name').checked,
    show_scene: $('#setting-show-scene').checked,
    card_border: $('#setting-card-border').checked,
    allowed_domains: $('#setting-allowed-domains').value.trim(),
  };
  try {
    await api('PUT', '/api/admin/settings', body);
    await reload();
    alert('设置已保存');
  } catch (e) {
    alert('保存失败: ' + e.message);
  }
}

// 背景明暗调节滑钮实时预览（仅在有背景图时可见效果）
function onBgBrightnessInput(slider) {
  const v = parseInt(slider.value) || 0;
  const label = document.getElementById('bg-bright-val');
  label.textContent = v === 0 ? '标准' : (v > 0 ? '增亮 ' + v + '%' : '增暗 ' + (-v) + '%');
  const img = $('#setting-bg-image').value.trim();
  if (img) {
    applyBackground({
      bg_enable: $('#setting-bg-enable').checked,
      bg_image: img,
      bg_blur: parseInt($('#setting-bg-blur').value) || 0,
      bg_brightness: v,
      card_blur: parseInt($('#setting-card-blur').value) || 0,
      bar_blur: parseInt($('#setting-bar-blur').value) || 0,
    });
  }
}

// 卡片/搜索框明暗调节滑块标签（admin 页无卡片/搜索框实例，仅更新标签；实时效果在首页）
function setBrightLabel(slider, labelId) {
  const v = parseInt(slider.value) || 0;
  document.getElementById(labelId).textContent = v === 0 ? '标准' : (v > 0 ? '增亮 ' + v + '%' : '增暗 ' + (-v) + '%');
}

// 背景图上传：上传到服务端后回填 URL 输入框，并实时预览
async function uploadBg() {
  const fileInput = $('#setting-bg-file');
  const file = fileInput.files[0];
  if (!file) { alert('请先选择一张图片'); return; }
  const btn = fileInput.parentElement.querySelector('button');
  const oldText = btn.textContent;
  btn.textContent = '上传中…';
  btn.disabled = true;
  try {
    const fd = new FormData();
    fd.append('file', file);
    const res = await fetch('/api/admin/upload-bg', { method: 'POST', body: fd });
    const data = await res.json();
    if (!res.ok) throw new Error(data.error || '上传失败');
    $('#setting-bg-image').value = data.path;
    $('#setting-bg-enable').checked = true; // 上传后自动启用
    applyBackground({
      bg_enable: true,
      bg_image: data.path,
      bg_blur: parseInt($('#setting-bg-blur').value) || 0,
      bg_brightness: parseInt($('#setting-bg-brightness').value) || 0,
      card_blur: parseInt($('#setting-card-blur').value) || 0,
      bar_blur: parseInt($('#setting-bar-blur').value) || 0,
    });
    alert('上传成功，点击「保存设置」后全站生效');
  } catch (e) {
    alert('上传失败: ' + e.message);
  } finally {
    btn.textContent = oldText;
    btn.disabled = false;
    fileInput.value = '';
  }
}

// ===== 站点书签（按分组 + 紧凑芯片 + 点图标弹菜单）=====
function renderSites() {
  closeSiteMenu();
  const list = $('#site-list');
  const groups = [...config.groups].sort((a, b) => a.sort_order - b.sort_order);
  const sites = config.sites || [];
  let html = '';
  // 有站点的分组才渲染，空分组不占空间
  for (const g of groups) {
    const gsites = sites.filter(s => s.group_id === g.id).sort((a, b) => a.sort_order - b.sort_order);
    if (!gsites.length) continue;
    html += `
      <div class="sg-section">
        <div class="sg-header">
          <span class="sg-name">${esc(g.name)}</span>
          <span class="sg-count">${gsites.length}</span>
        </div>
        <div class="sg-grid">${gsites.map(chipHTML).join('')}</div>
      </div>`;
  }
  // 未分组
  const groupedIds = new Set(groups.map(g => g.id));
  const ungrouped = sites.filter(s => !groupedIds.has(s.group_id)).sort((a, b) => a.sort_order - b.sort_order);
  if (ungrouped.length) {
    html += `
      <div class="sg-section">
        <div class="sg-header">
          <span class="sg-name">未分组</span>
          <span class="sg-count">${ungrouped.length}</span>
        </div>
        <div class="sg-grid">${ungrouped.map(chipHTML).join('')}</div>
      </div>`;
  }
  list.innerHTML = html
    ? `<div class="sites-by-group">${html}</div>`
    : '<div class="list-item"><span class="list-meta">暂无站点</span></div>';
}

// 单个站点芯片：图标（点击弹菜单）+ 名称
function chipHTML(s) {
  const src = logoSrc(s);
  const initial = (s.name || '?').trim().charAt(0).toUpperCase();
  return `
    <div class="site-chip" title="${esc(s.url)}">
      <button class="chip-icon" onclick="openSiteMenu(event, ${s.id})" aria-label="操作菜单" title="点击图标进行编辑或删除">
        <img src="${esc(src)}" alt="" onerror="this.style.display='none';this.nextElementSibling.style.display='flex'">
        <span class="chip-fallback" style="display:none">${esc(initial)}</span>
      </button>
      <span class="chip-name">${esc(s.name)}</span>
    </div>`;
}

// ===== 小弹窗菜单（点击图标触发，编辑/删除）=====
let menuEl = null;
function ensureMenu() {
  if (menuEl) return;
  menuEl = document.createElement('div');
  menuEl.className = 'site-menu';
  menuEl.innerHTML = `
    <button data-act="edit">编辑</button>
    <button data-act="delete" class="danger">删除</button>`;
  menuEl.addEventListener('click', (e) => {
    const act = e.target.dataset.act;
    if (!act) return;
    const id = parseInt(menuEl.dataset.id, 10);
    closeSiteMenu();
    if (act === 'edit') editSite(id);
    else if (act === 'delete') removeSite(id);
  });
  // 点击菜单/图标之外的区域关闭
  document.addEventListener('click', (e) => {
    if (menuEl.style.display === 'block' && !menuEl.contains(e.target) && !e.target.closest('.chip-icon')) {
      closeSiteMenu();
    }
  }, true);
  document.body.appendChild(menuEl);
}

function openSiteMenu(e, id) {
  e.stopPropagation();
  ensureMenu();
  menuEl.dataset.id = id;
  menuEl.style.display = 'block';
  // 定位到图标附近（fixed 定位，用视口坐标）
  const r = e.currentTarget.getBoundingClientRect();
  const mw = menuEl.offsetWidth || 124;
  const mh = menuEl.offsetHeight || 76;
  let left = r.left;
  let top = r.bottom + 6;
  if (left + mw > window.innerWidth) left = window.innerWidth - mw - 8;
  if (top + mh > window.innerHeight) top = r.top - mh - 6;
  menuEl.style.left = left + 'px';
  menuEl.style.top = top + 'px';
}

function closeSiteMenu() {
  if (menuEl) menuEl.style.display = 'none';
}

function renderGroupSelect() {
  const sel = $('#site-group');
  sel.innerHTML = '<option value="0">未分组</option>' +
    config.groups.map(g => `<option value="${g.id}">${esc(g.name)}</option>`).join('');
}

function resetSite() {
  $('#site-id').value = '';
  $('#site-name').value = '';
  $('#site-url').value = '';
  $('#site-group').value = '0';
  $('#site-sort').value = '0';
  $('#site-form-title').textContent = '添加站点';
  $('.btn-primary[onclick="saveSite()"]').textContent = '添加';
  $('#site-cancel').style.display = 'none';
  clearLogo('logo');
}

function editSite(id) {
  const s = config.sites.find(s => s.id === id);
  if (!s) return;
  $('#site-id').value = s.id;
  $('#site-name').value = s.name;
  $('#site-url').value = s.url;
  $('#site-group').value = s.group_id;
  $('#site-sort').value = s.sort_order;
  $('#site-form-title').textContent = '编辑站点';
  $('.btn-primary[onclick="saveSite()"]').textContent = '更新';
  $('#site-cancel').style.display = '';
  setLogoPreview('logo', s.logo_url);
  $('html,body').scrollTop = 0;
}

async function saveSite() {
  const name = $('#site-name').value.trim();
  const url = $('#site-url').value.trim();
  if (!name || !url) return;
  const id = $('#site-id').value;
  const body = {
    name, url,
    logo_url: getLogoPath('logo'),
    group_id: parseInt($('#site-group').value) || 0,
    sort_order: parseInt($('#site-sort').value) || 0,
  };
  try {
    if (id) {
      await api('PUT', `/api/admin/sites/${id}`, body);
    } else {
      await api('POST', '/api/admin/sites', body);
    }
    resetSite();
    await reload();
  } catch (e) {
    alert('保存失败: ' + e.message);
  }
}

async function removeSite(id) {
  if (!confirm('确定删除该站点？')) return;
  try {
    await api('DELETE', `/api/admin/sites/${id}`);
    await reload();
  } catch (e) {
    alert('删除失败: ' + e.message);
  }
}

// ===== Logo 上传/抓取 =====
function setLogoPreview(prefix, path) {
  const box = $('#' + prefix + '-preview-box');
  const pathEl = $('#' + prefix + '-path');
  const clearBtn = $('#' + prefix + '-clear-btn');
  if (path) {
    box.innerHTML = `<img src="${esc(path)}" alt="logo" onerror="this.parentNode.innerHTML='<span class=\\'logo-placeholder\\'>加载失败</span>'">`;
    pathEl.textContent = '当前: ' + path;
    pathEl.style.display = '';
    clearBtn.style.display = '';
  } else {
    box.innerHTML = '<span class="logo-placeholder">无</span>';
    pathEl.style.display = 'none';
    clearBtn.style.display = 'none';
  }
  $('#' + prefix + '-error').style.display = 'none';
}

function getLogoPath(prefix) {
  const pathEl = $('#' + prefix + '-path');
  return pathEl.style.display !== 'none' ? pathEl.textContent.replace('当前: ', '') : '';
}

function clearLogo(prefix) {
  setLogoPreview(prefix, '');
  $('#' + prefix + '-fetch-url').value = '';
}

async function handleFileUpload(prefix, e) {
  const file = e.target.files[0];
  if (!file) return;
  $('#' + prefix + '-upload-text').textContent = '上传中...';
  $('#' + prefix + '-error').style.display = 'none';
  const fd = new FormData();
  fd.append('file', file);
  try {
    const res = await fetch('/api/admin/upload-logo', { method: 'POST', body: fd });
    const data = await res.json();
    if (!res.ok) throw new Error(data.error || '上传失败');
    setLogoPreview(prefix, data.path);
  } catch (err) {
    const el = $('#' + prefix + '-error');
    el.textContent = err.message;
    el.style.display = '';
  } finally {
    $('#' + prefix + '-upload-text').textContent = '本地上传';
    e.target.value = '';
  }
}

async function fetchLogo(prefix) {
  const url = $('#' + prefix + '-fetch-url').value.trim();
  if (!url) return;
  const btn = $('#' + prefix + '-fetch-btn');
  btn.textContent = '抓取中...';
  btn.disabled = true;
  $('#' + prefix + '-error').style.display = 'none';
  try {
    const res = await fetch('/api/admin/fetch-logo', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ url }),
    });
    const data = await res.json();
    if (!res.ok) throw new Error(data.error || '抓取失败');
    setLogoPreview(prefix, data.path);
    $('#' + prefix + '-fetch-url').value = '';
  } catch (err) {
    const el = $('#' + prefix + '-error');
    el.textContent = err.message;
    el.style.display = '';
  } finally {
    btn.textContent = '抓取';
    btn.disabled = false;
  }
}

// ===== 站点分组 =====
function renderGroups() {
  const list = $('#group-list');
  const groups = [...config.groups].sort((a, b) => a.sort_order - b.sort_order);
  list.innerHTML = groups.map(g => `
    <div class="list-item">
      <div class="list-info">
        <span class="list-name">${esc(g.name)}</span>
        <span class="list-meta">可见: ${esc(g.visible_scenes)}</span>
      </div>
      <div class="list-actions">
        <button onclick="editGroup(${g.id})">编辑</button>
        <button class="danger" onclick="removeGroup(${g.id})">删除</button>
      </div>
    </div>
  `).join('') || '<div class="list-item"><span class="list-meta">暂无分组</span></div>';
}

function resetGroup() {
  $('#group-id').value = '';
  $('#group-name').value = '';
  $('#group-scenes').value = 'both';
  $('#group-sort').value = '0';
  $('#group-form-title').textContent = '添加分组';
  $('.btn-primary[onclick="saveGroup()"]').textContent = '添加';
  $('#group-cancel').style.display = 'none';
}

function editGroup(id) {
  const g = config.groups.find(g => g.id === id);
  if (!g) return;
  $('#group-id').value = g.id;
  $('#group-name').value = g.name;
  $('#group-scenes').value = g.visible_scenes;
  $('#group-sort').value = g.sort_order;
  $('#group-form-title').textContent = '编辑分组';
  $('.btn-primary[onclick="saveGroup()"]').textContent = '更新';
  $('#group-cancel').style.display = '';
  $('html,body').scrollTop = 0;
}

async function saveGroup() {
  const name = $('#group-name').value.trim();
  if (!name) return;
  const id = $('#group-id').value;
  const body = {
    name,
    visible_scenes: $('#group-scenes').value,
    sort_order: parseInt($('#group-sort').value) || 0,
  };
  try {
    if (id) await api('PUT', `/api/admin/groups/${id}`, body);
    else await api('POST', '/api/admin/groups', body);
    resetGroup();
    await reload();
  } catch (e) {
    alert('保存失败: ' + e.message);
  }
}

async function removeGroup(id) {
  if (!confirm('确定删除该分组？')) return;
  try {
    await api('DELETE', `/api/admin/groups/${id}`);
    await reload();
  } catch (e) {
    alert('删除失败: ' + e.message);
  }
}

// ===== 搜索引擎 =====
function renderEngines() {
  const list = $('#engine-list');
  const engines = [...config.engines].sort((a, b) => a.sort_order - b.sort_order);
  list.innerHTML = engines.map(e => {
    let meta = e.scopes;
    if (e.is_default_lan) meta += ' · 局域网默认';
    if (e.is_default_wan) meta += ' · 公网默认';
    return `
      <div class="list-item">
        <div class="list-info">
          <span class="list-name">${esc(e.name)}</span>
          <span class="list-meta">${esc(meta)}</span>
        </div>
        <div class="list-actions">
          <button onclick="editEngine(${e.id})">编辑</button>
          <button class="danger" onclick="removeEngine(${e.id})">删除</button>
        </div>
      </div>
    `;
  }).join('') || '<div class="list-item"><span class="list-meta">暂无引擎</span></div>';
}

function resetEngine() {
  $('#engine-id').value = '';
  $('#engine-name').value = '';
  $('#engine-url').value = '';
  setLogoPreview('elogo', '');
  $('#engine-suggest').value = '';
  $('#engine-scopes').value = 'lan,wan';
  $('#engine-default-lan').checked = false;
  $('#engine-default-wan').checked = false;
  $('#engine-sort').value = '0';
  $('#engine-form-title').textContent = '添加引擎';
  $('.btn-primary[onclick="saveEngine()"]').textContent = '添加';
  $('#engine-cancel').style.display = 'none';
}

function editEngine(id) {
  const e = config.engines.find(e => e.id === id);
  if (!e) return;
  $('#engine-id').value = e.id;
  $('#engine-name').value = e.name;
  $('#engine-url').value = e.url_template;
  setLogoPreview('elogo', e.icon);
  $('#engine-suggest').value = e.suggestion_url || '';
  $('#engine-scopes').value = e.scopes;
  $('#engine-default-lan').checked = e.is_default_lan;
  $('#engine-default-wan').checked = e.is_default_wan;
  $('#engine-sort').value = e.sort_order;
  $('#engine-form-title').textContent = '编辑引擎';
  $('.btn-primary[onclick="saveEngine()"]').textContent = '更新';
  $('#engine-cancel').style.display = '';
  $('html,body').scrollTop = 0;
}

async function saveEngine() {
  const name = $('#engine-name').value.trim();
  const url_template = $('#engine-url').value.trim();
  if (!name || !url_template) return;
  const id = $('#engine-id').value;
  const body = {
    name, url_template,
    icon: getLogoPath('elogo'),
    suggestion_url: $('#engine-suggest').value.trim(),
    scopes: $('#engine-scopes').value,
    is_default_lan: $('#engine-default-lan').checked,
    is_default_wan: $('#engine-default-wan').checked,
    sort_order: parseInt($('#engine-sort').value) || 0,
  };
  try {
    if (id) await api('PUT', `/api/admin/engines/${id}`, body);
    else await api('POST', '/api/admin/engines', body);
    resetEngine();
    await reload();
  } catch (e) {
    alert('保存失败: ' + e.message);
  }
}

async function removeEngine(id) {
  if (!confirm('确定删除该引擎？')) return;
  try {
    await api('DELETE', `/api/admin/engines/${id}`);
    await reload();
  } catch (e) {
    alert('删除失败: ' + e.message);
  }
}

// ===== 工具函数 =====
async function api(method, url, body) {
  const opts = { method, headers: {} };
  if (body) {
    opts.headers['Content-Type'] = 'application/json';
    opts.body = JSON.stringify(body);
  }
  const res = await fetch(url, opts);
  if (!res.ok) {
    const data = await res.json().catch(() => ({}));
    throw new Error(data.error || `HTTP ${res.status}`);
  }
  return res.json();
}

init();
