# Trace Browser Cloudflare 静态官网

这个目录是可直接部署到 Cloudflare Pages 的自包含静态项目。

## 目录内容

- `index.html`：官网页面
- `styles.css`：页面样式
- `script.js`：导航、动效和轮播逻辑
- `download-worker.js`：Cloudflare 下载反代 Worker，按最新 Release 资产提供官网下载
- `assets/`：logo、favicon 和官网截图资源
- `_headers`：Cloudflare Pages 响应头配置

## Cloudflare Pages 直接上传

1. 进入 Cloudflare Dashboard。
2. 创建 Pages 项目。
3. 选择直接上传。
4. 上传整个 `website/CFBrowser` 目录内的文件。

## 连接 Git 仓库部署

如果 Cloudflare 连接的是整个仓库：

- Build command：留空
- Build output directory：`website/cloudflare`
- Root directory：仓库根目录

这个项目不需要安装依赖，也不需要构建命令。

## 下载反代 Worker

官网的下载按钮使用以下稳定入口，由 Cloudflare Worker 解析最新 Release 资产：

- `GET /download/trace-browser/{windows|macos|linux}/{amd64|arm64}/{installer|portable}`
- `GET /download/chromium/{windows|macos|linux}/{amd64|arm64}/{installer|portable}`
- `GET /api/trace-browser/latest`
- `GET /api/chromium/latest`

将 `download-worker.js` 部署为绑定在 `browser.lemon.vin` 上的 Worker，并让未匹配的请求继续回源到 Pages。Worker 只允许访问 `lemon-casino/trace-browser-release` 与 `lemon-casino/chromium`，不会把站点变成开放代理。

注意：普通 Cloudflare Pages“直接上传静态文件”不会自动执行 Worker。必须选择 Pages Advanced Mode / Workers 部署 `_worker.js`，或者单独部署 `download-worker.js` 并将 `browser.lemon.vin/*` 路由绑定到它；否则 `/download/...` 会被 Pages 当成普通路径回退到 `index.html`，浏览器就会看到官网 HTML 而不是文件下载。

如果使用 Cloudflare Pages Advanced Mode，可以将该文件作为根目录的 `_worker.js`，并配置 `ASSETS` 静态资源绑定；如果使用独立 Worker，则保留现有 Pages 作为源站，让 Worker 只处理 `/download/*` 与 `/api/*` 路径。`/api/trace-browser/latest` 会同时公开发布资产和 SHA-256 校验资产，桌面端更新器会通过官网同源读取校验文件。

## 代理健康检查

Worker 同时提供 `GET /api/proxy-health`（也支持 `HEAD`），用于 Trace Browser 通过待测代理获取真实公网出口信息。接口直接读取 Cloudflare 边缘注入的访问者 IP 和 `request.cf` 元数据，不访问上游服务，并强制返回 `Cache-Control: no-store`。响应包含 IPv4/IPv6、国家、地区、城市、ASN、运营商组织、Cloudflare 数据中心和请求时间等字段。

该接口是节点公网可达性和出口信息的主检测来源。IPPure 只在桌面端缺少住宅/机房、风险分数和纯净度补充数据时，由独立单并发队列调用；IPPure 失败不会改变主健康结果。

可选地在 Worker 中配置 `GITHUB_TOKEN` Secret，降低 GitHub API 的匿名请求限制。官网不展示 Chromium 内核下载入口，但软件后续的内核列表可以调用 `/api/chromium/latest`，下载时调用对应 `/download/chromium/...` 地址。
