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

## 按平台、架构独立检查更新

每个平台、架构独立发布：例如 Windows amd64 的安装包和对应校验文件上传完成，Windows amd64 安装版即可发现更新；Windows ARM64 或其他系统的构建失败不会阻塞它。预检不再要求同一 Release 的所有平台文件齐全。

桌面端应携带目标参数，例如：

```text
/api/trace-browser/latest?platform=windows&arch=amd64&package=installer
/api/trace-browser/latest?platform=windows&arch=arm64&package=selfupdate
/api/trace-browser/latest?platform=macos&arch=arm64&package=installer
/api/trace-browser/latest?platform=linux&arch=amd64&package=portable
```

`platform` 支持 `windows`、`macos`（兼容 `darwin`）、`linux`，`arch` 支持 `amd64`、`arm64`（兼容 `x64`、`x86_64`、`aarch64`）。`package` 支持 `installer`、`portable`，Windows 便携客户端更新使用 `selfupdate`，并兼容退回完整便携 ZIP。`/api/trace-browser/releases` 支持相同的目标参数。

某个包可用的条件仅为：正式发布、包属于该版本、包及其对应校验文件均为 `uploaded` 状态、大小大于零且有下载地址。校验文件分别为：

- Windows：`TraceBrowser-{version}-windows-{arch}.sha256.txt`，兼容旧的 `SHA256SUMS`。
- macOS：`TraceBrowser-{version}-macos-{arch}.sha256.txt`。
- Linux：`TraceBrowser-{version}-linux-{arch}.sha256.txt`。

当前目标尚未准备好时，仅该目标回退到上一可用版本；其他准备好的目标直接使用新版本。历史查找每页 100 条，最多 5 页，版本列表先按目标筛选再应用 `limit`。没有可用目标版本时，接口正常返回 `200`、`ok: true`、`ready: false`、空 `assets`，客户端应视为暂无更新。返回的包地址固定 release ID，下载过程中不会串到其他版本。

未携带目标参数的旧请求继续兼容，返回有任意可用包的最新正式版本及已可用资产；精确的按平台/架构回退需要桌面端发送上述参数。稳定下载入口根据路径中的平台、架构和包类型独立选择版本。ETag 包含资产状态对应的清单，同一版本后续上传其他架构时会更新缓存标识。

Windows 发布工作流也需配合：各架构在自己的构建任务中上传发布包，使用独立的架构校验文件，不能再通过 `needs: build` 等待整个矩阵成功，也不能并行覆盖共享的 `SHA256SUMS`。Chromium 保留独立的发布规则。

本地回归测试（Node.js 22 或更新版本，无需安装依赖）：

```sh
node --test download-worker.test.mjs
```

测试覆盖各系统与架构单独完成上传、其他目标失败时仍可更新、目标回退、暂无更新、正确选包与下载、独立校验文件、旧接口兼容、分页与 ETag 刷新。测试文件通过 `.assetsignore` 排除出静态资源上传。

## 代理健康检查

Worker 同时提供 `GET /api/proxy-health`（也支持 `HEAD`），用于 Trace Browser 通过待测代理获取真实公网出口信息。接口直接读取 Cloudflare 边缘注入的访问者 IP 和 `request.cf` 元数据，不访问上游服务，并强制返回 `Cache-Control: no-store`。响应包含 IPv4/IPv6、国家、地区、城市、ASN、运营商组织、Cloudflare 数据中心和请求时间等字段。

该接口是节点公网可达性和出口信息的主检测来源。IPPure 只在桌面端缺少住宅/机房、风险分数和纯净度补充数据时，由独立单并发队列调用；IPPure 失败不会改变主健康结果。

可选地在 Worker 中配置 `GITHUB_TOKEN` Secret，降低 GitHub API 的匿名请求限制。软件内核下载列表调用 `/api/chromium/releases` 获取多个版本，并使用响应中带 release ID 的 `/download/chromium/release/{releaseId}/asset/{assetName}` 地址下载用户选择的版本；旧的 `/api/chromium/latest` 和 `/download/chromium/asset/{assetName}` 入口仍然兼容。
