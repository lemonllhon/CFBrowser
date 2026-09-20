# Trace Browser Cloudflare 静态官网

这个目录是可直接部署到 Cloudflare Pages 的自包含静态项目。

## 目录内容

- `index.html`：官网页面
- `styles.css`：页面样式
- `script.js`：导航、动效和轮播逻辑
- `download-worker.js`：Cloudflare 下载反代 Worker，提供最新版本和可按 Release 选择的官网下载
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
- `GET /api/trace-browser/releases?limit=100`
- `GET /api/chromium/releases?limit=100`

将 `download-worker.js` 部署为绑定在 `browser.lemon.vin` 上的 Worker，并让未匹配的请求继续回源到 Pages。Worker 只允许访问 `lemon-casino/trace-browser-release` 与 `lemon-casino/chromium`，不会把站点变成开放代理。

注意：普通 Cloudflare Pages“直接上传静态文件”不会自动执行 Worker。必须选择 Pages Advanced Mode / Workers 部署 `_worker.js`，或者单独部署 `download-worker.js` 并将 `browser.lemon.vin/*` 路由绑定到它；否则 `/download/...` 会被 Pages 当成普通路径回退到 `index.html`，浏览器就会看到官网 HTML 而不是文件下载。

如果使用 Cloudflare Pages Advanced Mode，可以将该文件作为根目录的 `_worker.js`，并配置 `ASSETS` 静态资源绑定；如果使用独立 Worker，则保留现有 Pages 作为源站，让 Worker 只处理 `/download/*` 与 `/api/*` 路径。`/api/trace-browser/latest` 会同时公开发布资产和 SHA-256 校验资产，桌面端更新器会通过官网同源读取校验文件。`/api/*/releases` 返回最多 100 个已发布版本，返回的资产地址带有 release ID，因此下载旧版本时不会被重新解析为最新版本。公开给桌面端的版本和下载地址固定使用 `https://browser.lemon.vin`。

## 完整发布后再提示更新

`/api/trace-browser/latest`、Trace Browser 版本列表和稳定下载入口统一只公开完整的正式版本。判定条件为同一版本的以下 **17 个资产**全部存在，GitHub 上传状态为 `uploaded`，文件大小大于零且具有下载地址：

- Windows amd64 / arm64：安装包、便携包、自更新包，共 6 个文件。
- macOS amd64 / arm64：DMG，共 2 个文件。
- Linux amd64 / arm64：DEB、tar.gz，共 4 个文件。
- Windows `SHA256SUMS`，以及 macOS / Linux 各架构的 `.sha256.txt`，共 5 个校验文件。

三个发布工作流可继续独立构建与上传。最新发布尚未齐全时，接口会向前查找上一完整正式版本，桌面端继续按该版本检查更新，官网下载也使用同一版本。最后一个资产上传完成后，接口在缓存刷新后自动切换到新版本。草稿、预发布、旧版本残留文件、零字节文件和上传中资产不能满足完整性条件。Chromium 保持原有独立发布规则。

历史查找每页 100 条，最多 5 页；版本列表先筛选完整版本，再应用 `limit`。如果查找范围内没有任何完整正式版本，最新版本接口返回不缓存的 503，不会公开半成品版本号。版本指定下载仍固定 release ID，不会在新版本发布时跳到其他版本。

本次修复在 Cloudflare Worker 生效：将此目录的更新部署到 Cloudflare 后，现有桌面端即可使用，无需为此重新编译桌面程序。发布资产命名或支持的平台发生变化时，需同步修改 `download-worker.js` 中的完整性规则及测试。

本地回归测试（Node.js 22 或更新版本，无需安装依赖）：

```sh
node --test download-worker.test.mjs
```

测试覆盖分批上传、全部资产门槛、上传状态和版本一致性、上一完整版本回退、分页、ETag 切换、版本列表过滤及下载版本固定。测试文件已通过 `.assetsignore` 排除出静态资源上传。

## 代理健康检查

Worker 同时提供 `GET /api/proxy-health`（也支持 `HEAD`），用于 Trace Browser 通过待测代理获取真实公网出口信息。接口直接读取 Cloudflare 边缘注入的访问者 IP 和 `request.cf` 元数据，不访问上游服务，并强制返回 `Cache-Control: no-store`。响应包含 IPv4/IPv6、国家、地区、城市、ASN、运营商组织、Cloudflare 数据中心和请求时间等字段。

该接口是节点公网可达性和出口信息的主检测来源。IPPure 只在桌面端缺少住宅/机房、风险分数和纯净度补充数据时，由独立单并发队列调用；IPPure 失败不会改变主健康结果。

可选地在 Worker 中配置 `GITHUB_TOKEN` Secret，降低 GitHub API 的匿名请求限制。软件内核下载列表调用 `/api/chromium/releases` 获取多个版本，并使用响应中带 release ID 的 `/download/chromium/release/{releaseId}/asset/{assetName}` 地址下载用户选择的版本；旧的 `/api/chromium/latest` 和 `/download/chromium/asset/{assetName}` 入口仍然兼容。
