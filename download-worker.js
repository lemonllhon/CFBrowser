/**
 * Trace Browser Cloudflare download proxy.
 *
 * Deploy this Worker in front of the Pages site (or use it as the Pages
 * `_worker.js` entrypoint with an ASSETS binding). The public website never
 * exposes GitHub asset URLs: stable Cloudflare paths resolve the newest
 * allowed release asset on every request.
 */

const SOURCES = {
  'trace-browser': {
    repository: 'lemon-casino/trace-browser-release',
    label: 'Trace Browser',
  },
  chromium: {
    repository: 'lemon-casino/chromium',
    label: 'Chromium',
  },
}

const PLATFORM_NAMES = {
  windows: 'Windows',
  macos: 'macOS',
  linux: 'Linux',
}

const ARCHITECTURE_NAMES = {
  amd64: 'amd64 / x64',
  arm64: 'arm64 / ARM64',
}

const PACKAGE_NAMES = {
  installer: '安装版',
  portable: '便携版',
}

const SUPPORTED_DOWNLOADS = new Set(['installer', 'portable'])
const RELEASE_CACHE_SECONDS = 300

function corsHeaders() {
  return {
    'Access-Control-Allow-Origin': '*',
    'Access-Control-Allow-Methods': 'GET, HEAD, OPTIONS',
    'Access-Control-Allow-Headers': 'Content-Type, Range, If-None-Match',
    'Access-Control-Expose-Headers': 'Content-Length, Content-Range, Content-Disposition, ETag, Last-Modified',
  }
}

function jsonResponse(payload, status = 200, extraHeaders = {}) {
  return new Response(JSON.stringify(payload, null, 2), {
    status,
    headers: {
      'Content-Type': 'application/json; charset=utf-8',
      'Cache-Control': status >= 400 ? 'no-store' : `public, max-age=0, s-maxage=${RELEASE_CACHE_SECONDS}, stale-while-revalidate=60`,
      ...corsHeaders(),
      ...extraHeaders,
    },
  })
}

function errorResponse(message, status = 404, detail = '') {
  return jsonResponse({ ok: false, error: message, detail }, status)
}

function githubHeaders(env, accept = 'application/vnd.github+json') {
  const headers = new Headers({
    Accept: accept,
    'User-Agent': 'Trace-Browser-Cloudflare-Download-Proxy',
    'X-GitHub-Api-Version': '2022-11-28',
  })
  if (env.GITHUB_TOKEN) {
    headers.set('Authorization', `Bearer ${env.GITHUB_TOKEN}`)
  }
  return headers
}

async function fetchLatestRelease(source, env) {
  const config = SOURCES[source]
  const endpoint = `https://api.github.com/repos/${config.repository}/releases/latest`
  const response = await fetch(endpoint, {
    headers: githubHeaders(env),
    cf: { cacheTtl: RELEASE_CACHE_SECONDS, cacheEverything: true },
  })
  if (!response.ok) {
    const body = await response.text()
    throw new Error(`${config.label} 最新版本接口返回 HTTP ${response.status}: ${body.slice(0, 240)}`)
  }
  return response.json()
}

function normalizedArchitecture(value) {
  const text = String(value || '').toLowerCase()
  if (text === 'arm64' || text === 'aarch64' || text === 'armv8') return 'arm64'
  if (text === 'amd64' || text === 'x64' || text === 'x86_64') return 'amd64'
  return ''
}

function isArchitectureMatch(name, architecture) {
  const text = String(name || '').toLowerCase()
  if (architecture === 'arm64') return /(arm64|aarch64|armv8)/i.test(text)
  if (architecture === 'amd64') return /(amd64|x64|x86_64)/i.test(text)
  return false
}

function isPlatformMatch(name, platform) {
  const text = String(name || '').toLowerCase()
  if (platform === 'windows') return /(windows|win32|win64|win)/i.test(text)
  if (platform === 'macos') return /(macos|darwin|osx)/i.test(text)
  if (platform === 'linux') return /linux|\.deb$|\.tar\.gz$/i.test(text)
  return false
}

function isChecksumOrSource(name) {
  return /sha256|checksum|source code|patchset/i.test(String(name || ''))
}

function appAssetMatches(asset, platform, architecture, packageKind) {
  const name = String(asset.name || '')
  if (isChecksumOrSource(name)) return false
  if (platform === 'windows') {
    if (packageKind === 'installer') return architecture === 'amd64' && /^TraceBrowser-Setup-.*\.exe$/i.test(name)
    return isPlatformMatch(name, platform) && isArchitectureMatch(name, architecture) && /^TraceBrowser-Portable-.*\.zip$/i.test(name)
  }
  if (!isPlatformMatch(name, platform) || !isArchitectureMatch(name, architecture)) return false
  if (platform === 'macos') return packageKind === 'installer' && /^TraceBrowser-.*-macos-(amd64|arm64)\.dmg$/i.test(name)
  if (platform === 'linux') {
    if (packageKind === 'installer') return /^trace-browser_.*_(amd64|arm64)\.deb$/i.test(name)
    return /^TraceBrowser-.*-linux-(amd64|arm64)\.tar\.gz$/i.test(name)
  }
  return false
}

function chromiumAssetMatches(asset, platform, architecture, packageKind) {
  const name = String(asset.name || '')
  if (isChecksumOrSource(name) || !isPlatformMatch(name, platform) || !isArchitectureMatch(name, architecture)) return false
  if (platform === 'windows') {
    if (packageKind === 'installer') return /installer.*\.exe$/i.test(name)
    return /windows.*\.zip$/i.test(name)
  }
  if (platform === 'macos') return packageKind === 'installer' && /macos.*\.dmg$/i.test(name)
  if (platform === 'linux') {
    if (packageKind === 'installer') return /\.appimage$/i.test(name)
    return /linux.*\.tar\.xz$/i.test(name)
  }
  return false
}

function pickAsset(release, source, platform, architecture, packageKind) {
  const assets = Array.isArray(release.assets) ? release.assets : []
  const matches = assets.filter(asset => source === 'trace-browser'
    ? appAssetMatches(asset, platform, architecture, packageKind)
    : chromiumAssetMatches(asset, platform, architecture, packageKind))
  return matches[0] || null
}

function publicAssetURL(request, source, assetName) {
  const url = new URL(request.url)
  return `${url.origin}/download/${source}/asset/${encodeURIComponent(assetName)}`
}

function publicReleasePayload(request, source, release) {
  const config = SOURCES[source]
  const assets = (Array.isArray(release.assets) ? release.assets : [])
    .filter(asset => asset && asset.name && !isChecksumOrSource(asset.name))
    .map(asset => ({
      name: asset.name,
      size: Number(asset.size || 0),
      contentType: asset.content_type || 'application/octet-stream',
      updatedAt: asset.updated_at || release.published_at || '',
      downloadUrl: publicAssetURL(request, source, asset.name),
    }))

  return {
    ok: true,
    source,
    product: config.label,
    repository: config.repository,
    version: String(release.tag_name || release.name || '').replace(/^v/i, ''),
    tagName: release.tag_name || '',
    name: release.name || release.tag_name || '',
    publishedAt: release.published_at || '',
    notes: release.body || '',
    releaseUrl: `${new URL(request.url).origin}/#downloads`,
    assets,
  }
}

async function proxyAsset(request, env, source, asset) {
  const config = SOURCES[source]
  const upstreamURL = asset.browser_download_url
  if (!upstreamURL) return errorResponse('发布资产没有下载地址', 502)

  const headers = githubHeaders(env, 'application/octet-stream')
  for (const headerName of ['Range', 'If-None-Match', 'If-Modified-Since']) {
    const value = request.headers.get(headerName)
    if (value) headers.set(headerName, value)
  }
  const upstream = await fetch(upstreamURL, {
    method: request.method,
    headers,
    redirect: 'follow',
  })
  if (!upstream.ok && upstream.status !== 206 && upstream.status !== 304) {
    return errorResponse(`${config.label} 下载源返回 HTTP ${upstream.status}`, 502)
  }

  const responseHeaders = new Headers(corsHeaders())
  for (const headerName of ['Content-Type', 'Content-Length', 'Content-Range', 'Content-Disposition', 'ETag', 'Last-Modified', 'Accept-Ranges']) {
    const value = upstream.headers.get(headerName)
    if (value) responseHeaders.set(headerName, value)
  }
  responseHeaders.set('Cache-Control', 'public, max-age=0, s-maxage=300, stale-while-revalidate=60')
  responseHeaders.set('X-Trace-Download-Source', 'cloudflare-release-proxy')
  return new Response(upstream.body, { status: upstream.status, headers: responseHeaders })
}

async function handleDownload(request, env, source, parts) {
  const platform = parts[0]
  const architecture = normalizedArchitecture(parts[1])
  const packageKind = parts[2]
  if (!PLATFORM_NAMES[platform] || !architecture || !SUPPORTED_DOWNLOADS.has(packageKind)) {
    return errorResponse('不支持的平台、架构或安装包类型', 400)
  }

  const release = await fetchLatestRelease(source, env)
  const asset = pickAsset(release, source, platform, architecture, packageKind)
  if (!asset) {
    return errorResponse(`当前最新版本没有找到 ${PLATFORM_NAMES[platform]} ${ARCHITECTURE_NAMES[architecture]} ${PACKAGE_NAMES[packageKind]}`, 404)
  }
  return proxyAsset(request, env, source, asset)
}

async function handleNamedAsset(request, env, source, encodedAssetName) {
  let assetName
  try {
    assetName = decodeURIComponent(encodedAssetName)
  } catch {
    return errorResponse('资产名称编码无效', 400)
  }
  if (!assetName || assetName.includes('/') || assetName.includes('\\') || isChecksumOrSource(assetName)) {
    return errorResponse('不允许下载该资产', 403)
  }

  const release = await fetchLatestRelease(source, env)
  const asset = (Array.isArray(release.assets) ? release.assets : []).find(item => item.name === assetName)
  if (!asset) return errorResponse('当前最新版本中没有找到该资产', 404)
  return proxyAsset(request, env, source, asset)
}

async function handleAPI(request, env, source) {
  const release = await fetchLatestRelease(source, env)
  return jsonResponse(publicReleasePayload(request, source, release))
}

async function serveSite(request, env) {
  if (env.ASSETS && typeof env.ASSETS.fetch === 'function') return env.ASSETS.fetch(request)
  return fetch(request)
}

export default {
  async fetch(request, env) {
    if (request.method === 'OPTIONS') return new Response(null, { status: 204, headers: corsHeaders() })
    if (!['GET', 'HEAD'].includes(request.method)) return errorResponse('只支持 GET、HEAD 和 OPTIONS', 405)

    const url = new URL(request.url)
    const apiMatch = url.pathname.match(/^\/api\/(trace-browser|chromium)\/latest\/?$/)
    if (apiMatch) {
      try {
        return await handleAPI(request, env, apiMatch[1])
      } catch (error) {
        return errorResponse('获取最新版本失败', 502, error instanceof Error ? error.message : String(error))
      }
    }

    const stableDownloadMatch = url.pathname.match(/^\/download\/(trace-browser|chromium)\/(windows|macos|linux)\/(amd64|arm64)\/(installer|portable)\/?$/)
    if (stableDownloadMatch) {
      try {
        return await handleDownload(request, env, stableDownloadMatch[1], stableDownloadMatch.slice(2))
      } catch (error) {
        return errorResponse('下载代理失败', 502, error instanceof Error ? error.message : String(error))
      }
    }

    const namedAssetMatch = url.pathname.match(/^\/download\/(trace-browser|chromium)\/asset\/([^/]+)\/?$/)
    if (namedAssetMatch) {
      try {
        return await handleNamedAsset(request, env, namedAssetMatch[1], namedAssetMatch[2])
      } catch (error) {
        return errorResponse('下载代理失败', 502, error instanceof Error ? error.message : String(error))
      }
    }

    return serveSite(request, env)
  },
}
