# lan-nav

个人首页导航。分局域网（LAN）/ 公网（WAN）双场景：顶部搜索栏按访问场景自动切换搜索引擎；后台管理仅对私网 / 本机访客开放，公网访客看不到局域网分组与后台入口。

## 特性

- **双场景自动识别**：依据客户端 IP 自动判定 LAN / WAN，搜索栏与站点分组随之切换。
- **站点拖动排序**：管理页「站点书签」点「排序」进入拖动模式，调整完点「保存」才落库——拖动过程不发任何请求，保存时一次性提交并在服务端单事务写入，失败不会留下改到一半的顺序。
- **搜索引擎 logo**：支持本地上传与从网络抓取（复用站点书签的 logo 设置方式）。
- **管理后台访问控制**：默认仅私网 IP 与本机可访问 `/admin.html` 与 `/api/admin/*`；公网访客访问后台会被 302 跳回首页。
- **统一数据目录**：数据库（`data.db`）与上传资源（`logos/`、`背景图`）集中存放在 `data/`，可整目录打包迁移。
- **端口可配置**：默认 `5725`，支持 `config.json` 或 `--port` 覆盖。
- **Docker 友好**：单二进制（Go embed 冻结前端 + modernc.org/sqlite 无 cgo），多阶段构建纯静态镜像。

## 快速开始（Docker）

镜像由 GitHub Actions 自动构建并推送到 `ghcr.io`，**多架构（amd64 / arm64 / arm/v7 / arm/v6 / ppc64le / s390x）**，会自动匹配你的平台：

```bash
docker run -d \
  --name lan-nav \
  -p 5725:5725 \
  -v "$PWD/data:/data" \
  --restart unless-stopped \
  ghcr.io/inxen/lan-nav:latest
```

或使用 `docker-compose.yml`（仓库已附）：

```bash
docker compose up -d
```

- 访问首页：<http://localhost:5725>
- 管理后台：<http://localhost:5725/admin.html>（仅私网 / 本机可进）

## 从源码运行

```bash
cd backend
go run .          # 开发模式，首次运行自动创建 data/ 并写入默认 config.json
# 或编译后运行：
go build -a -o lan-nav . && ./lan-nav
```

> 修改前端（`backend/web/`）后必须重新 `go build -a`，因为前端资源通过 `//go:embed all:web` 冻结进二进制。

## 配置

### 端口

默认 `5725`。三种方式（优先级从高到低）：

1. 启动参数：`--port 9000`
2. 数据目录下的 `config.json`：`{ "port": 9000 }`（位于 `data/config.json`，随数据目录迁移）
3. 不指定则回退默认 `5725`

> `config.json` 的 `port` 既支持数字 `5725` 也支持字符串 `"5725"`。

### 数据目录

所有运行数据都在 `data/`（可通过 `--data-dir /path/to/data` 或环境变量 `DATA_DIR` 指定任意位置）：

```
data/
├── data.db        # SQLite：站点、搜索引擎、全局设置
├── config.json    # 端口等运行时配置
└── logos/         # 上传的站点 / 引擎 logo、背景图
```

整目录复制即可完成迁移；旧版散落在程序目录下的 `data.db` / `logos/` 会在首次使用默认目录时自动迁入。

### 管理后台访问控制

后台（含 `/admin.html` 与 `/api/admin/*`）的访问判定：

- **本机 / 私网 IP**：放行（拥有者管理自己的导航）。
- **公网 IP**：拒绝（302 跳首页）。

反向代理（nginx 等）部署时，后端会信任**私网对端**转发的真实客户端 IP（读取 `X-Real-IP` / `X-Forwarded-For` 末值），因此公网访客仍会被正确识别为公网。若代理位于公网 CDN 边缘节点，可通过环境变量 `TRUSTED_PROXY=1.2.3.4,5.6.7.0/24` 显式声明可信代理。

## 反向代理示例（nginx）

参见 `deploy/nginx.conf.example`，关键点：

```nginx
location / {
    proxy_pass http://127.0.0.1:5725;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
}
```

## 构建镜像（本地）

```bash
docker build -t lan-nav .
docker run -d -p 5725:5725 -v "$PWD/data:/data" lan-nav
```

## 许可证

[MIT](LICENSE) © inxen
