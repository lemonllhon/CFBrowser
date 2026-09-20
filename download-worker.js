/**
 * Trace Browser Cloudflare download proxy.
 *
 * Deploy this Worker in front of the Pages site (or use it as the Pages
 * `_worker.js` entrypoint with an ASSETS binding). The public website never
 * exposes GitHub asset URLs: stable Cloudflare paths resolve the newest
 * allowed release asset on every request, while release-specific paths keep
 * desktop clients on the version the user selected.
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

// Release metadata and package URLs exposed to desktop clients must always
// stay on the official domain, even when the Worker is reached through an
// alternate hostname during deployment or testing.
const PUBLIC_ORIGIN = 'https://browser.lemon.vin'
const SUPPORTED_DOWNLOADS = new Set(['installer', 'portable'])
const RELEASE_CACHE_SECONDS = 30
const DOWNLOAD_CACHE_SECONDS = 300
const RELEASE_STALE_WHILE_REVALIDATE_SECONDS = 15
const PROXY_HEALTH_PATH = /^\/api\/proxy-health\/?$/

// Each package becomes available independently after its own checksum uploads.
const RELEASE_LOOKUP_PAGE_SIZE = 100
const RELEASE_LOOKUP_MAX_PAGES = 5

function traceBrowserPackageDefinitions(version) {
  return ['amd64', 'arm64'].flatMap(architecture => {
    const windowsSuffix = architecture === 'amd64' ? 'win-x64' : 'win-arm64'
    const definitions = [
      ['windows', 'installer', `TraceBrowser-Setup-${version}-${windowsSuffix}.exe`],
      ['windows', 'portable', `TraceBrowser-Portable-${version}-${windowsSuffix}.zip`],
      ['windows', 'selfupdate', `TraceBrowser-SelfUpdate-${version}-windows-${architecture}.zip`],
      ['macos', 'installer', `TraceBrowser-${version}-macos-${architecture}.dmg`],
      ['linux', 'installer', `trace-browser_${version}_${architecture}.deb`],
      ['linux', 'portable', `TraceBrowser-${version}-linux-${architecture}.tar.gz`],
    ]
    if (architecture === 'amd64') definitions.push(['windows', 'installer', `TraceBrowser-Setup-${version}.exe`])
    return definitions.map(([platform, packageKind, name]) => ({
      platform, architecture, packageKind, name,
      checksumNames: [
        `TraceBrowser-${version}-${platform}-${architecture}.sha256.txt`,
        ...(platform === 'windows' ? ['SHA256SUMS'] : []),
      ],
    }))
  })
}

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
      'Cache-Control': status >= 400
        ? 'no-store'
        : `public, max-age=0, s-maxage=${RELEASE_CACHE_SECONDS}, stale-while-revalidate=${RELEASE_STALE_WHILE_REVALIDATE_SECONDS}`,
      ...corsHeaders(),
      ...extraHeaders,
    },
  })
}

function errorResponse(message, status = 404, detail = '') {
  return jsonResponse({ ok: false, error: message, detail }, status)
}

function countryNameFromCode(countryCode) {
  if (!countryCode) return ''
  try {
    return new Intl.DisplayNames(['en'], { type: 'region' }).of(countryCode) || countryCode
  } catch {
    return countryCode
  }
}

function proxyHealthResponse(request) {
  const cf = request.cf || {}
  const ip = String(request.headers.get('CF-Connecting-IP') || '').trim()
  if (!ip) {
    return new Response(JSON.stringify({ ok: false, error: 'Cloudflare 未提供访问者出口 IP' }), {
      status: 503,
      headers: {
        'Content-Type': 'application/json; charset=utf-8',
        'Cache-Control': 'no-store, no-cache, must-revalidate',
        ...corsHeaders(),
      },
    })
  }

  const checkedAt = new Date().toISOString()
  const countryCode = String(cf.country || request.headers.get('CF-IPCountry') || '').trim().toUpperCase()
  const payload = {
    ok: true,
    source: 'trace-cloudflare',
    ip,
    ipVersion: ip.includes(':') ? 6 : 4,
    country: countryNameFromCode(countryCode),
    countryCode,
    region: String(cf.region || ''),
    regionCode: String(cf.regionCode || ''),
    city: String(cf.city || ''),
    continent: String(cf.continent || ''),
    postalCode: String(cf.postalCode || ''),
    timezone: String(cf.timezone || ''),
    latitude: String(cf.latitude || ''),
    longitude: String(cf.longitude || ''),
    asn: Number(cf.asn || 0),
    asOrganization: String(cf.asOrganization || ''),
    colo: String(cf.colo || ''),
    httpProtocol: String(cf.httpProtocol || ''),
    tlsVersion: String(cf.tlsVersion || ''),
    requestId: String(request.headers.get('CF-Ray') || ''),
    checkedAt,
  }
  const body = request.method === 'HEAD' ? null : JSON.stringify(payload)
  return new Response(body, {
    status: 200,
    headers: {
      'Content-Type': 'application/json; charset=utf-8',
      'Cache-Control': 'no-store, no-cache, must-revalidate, proxy-revalidate',
      Pragma: 'no-cache',
      Expires: '0',
      'X-Trace-Health-Source': 'cloudflare-edge',
      'X-Trace-Health-Checked-At': checkedAt,
      ...corsHeaders(),
    },
  })
}

function identityETag(identity, prefix) {
  let hash = 2166136261
  for (let index = 0; index < identity.length; index += 1) {
    hash ^= identity.charCodeAt(index)
    hash = Math.imul(hash, 16777619)
  }
  return `"${prefix}-${(hash >>> 0).toString(16)}"`
}

function releaseETag(release, source, target = null) {
  const assets = downloadableReleaseAssets(source, release, target)
  return identityETag(JSON.stringify([
    release.id, release.tag_name, release.name, release.body, release.updated_at, target,
    assets.map(asset => [asset.id, asset.name, asset.size, asset.updated_at, asset.browser_download_url]),
  ]), 'release')
}

function notModifiedResponse(etag) {
  return new Response(null, {
    status: 304,
    headers: {
      'Cache-Control': `public, max-age=0, s-maxage=${RELEASE_CACHE_SECONDS}, stale-while-revalidate=${RELEASE_STALE_WHILE_REVALIDATE_SECONDS}`,
      ETag: etag,
      ...corsHeaders(),
    },
  })
}

function requestMatchesETag(request, etag) {
  const value = request.headers.get('If-None-Match')
  if (!value) return false
  const normalize = candidate => candidate.trim().replace(/^W\//i, '')
  const normalizedETag = normalize(etag)
  return value.split(',').some(candidate => candidate.trim() === '*' || normalize(candidate) === normalizedETag)
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

async function fetchLatestRelease(source, env, target = null) {
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
  const latest = await response.json()
  if (source !== 'trace-browser' || traceBrowserReleaseAvailable(latest, target)) return latest

  // Only this target falls back when its build has not uploaded yet. Other
  // platforms/architectures can already use the new release.
  const available = await fetchAvailableTraceBrowserReleases(env, 1, target)
  return available[0] || null
}

async function fetchReleaseList(source, env, limit = 100, page = 1) {
  const config = SOURCES[source]
  const safeLimit = Math.max(1, Math.min(100, Number(limit) || 100))
  const endpoint = `https://api.github.com/repos/${config.repository}/releases?per_page=${safeLimit}&page=${page}`
  const response = await fetch(endpoint, {
    headers: githubHeaders(env),
    cf: { cacheTtl: RELEASE_CACHE_SECONDS, cacheEverything: true },
  })
  if (!response.ok) {
    const body = await response.text()
    throw new Error(`${config.label} 版本列表接口返回 HTTP ${response.status}: ${body.slice(0, 240)}`)
  }
  const payload = await response.json()
  return Array.isArray(payload) ? payload : []
}

async function fetchAvailableTraceBrowserReleases(env, limit, target = null) {
  const available = []
  for (let page = 1; page <= RELEASE_LOOKUP_MAX_PAGES; page += 1) {
    const releases = await fetchReleaseList('trace-browser', env, RELEASE_LOOKUP_PAGE_SIZE, page)
    for (const release of releases) {
      if (traceBrowserReleaseAvailable(release, target)) available.push(release)
      if (available.length >= limit) return available
    }
    if (releases.length < RELEASE_LOOKUP_PAGE_SIZE) break
  }
  return available
}

async function fetchReleaseById(source, env, releaseId) {
  const config = SOURCES[source]
  const endpoint = `https://api.github.com/repos/${config.repository}/releases/${releaseId}`
  const response = await fetch(endpoint, {
    headers: githubHeaders(env),
    cf: { cacheTtl: RELEASE_CACHE_SECONDS, cacheEverything: true },
  })
  if (!response.ok) {
    const body = await response.text()
    throw new Error(`${config.label} 指定版本接口返回 HTTP ${response.status}: ${body.slice(0, 240)}`)
  }
  return response.json()
}

function traceBrowserReleaseVersion(release) {
  const version = String(release?.tag_name || '').trim().replace(/^(?:(?:mac|linux)-)?v/i, '')
  return /^\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$/.test(version) ? version : ''
}

function isUploadedAsset(asset) {
  return asset?.state === 'uploaded' && Number.isFinite(asset.size) && asset.size > 0 &&
    typeof asset.browser_download_url === 'string' && asset.browser_download_url.trim() !== ''
}

function readyTraceBrowserPackages(release) {
  const version = traceBrowserReleaseVersion(release)
  if (!version || release.draft || release.prerelease || !Number.isSafeInteger(release.id) ||
      release.id <= 0 || !Number.isFinite(Date.parse(release.published_at))) return []
  const assets = new Map((Array.isArray(release.assets) ? release.assets : [])
    .filter(isUploadedAsset).map(asset => [asset.name, asset]))
  return traceBrowserPackageDefinitions(version).flatMap(definition => {
    const asset = assets.get(definition.name)
    const checksum = definition.checksumNames.map(name => assets.get(name)).find(Boolean)
    return asset && checksum ? [{ ...definition, asset, checksum }] : []
  })
}

function packageMatchesTarget(pkg, target) {
  if (!target || target.assetName) return true
  return pkg.platform === target.platform && pkg.architecture === target.architecture &&
    (pkg.packageKind === target.packageKind || (target.packageKind === 'selfupdate' && pkg.packageKind === 'portable'))
}

function downloadableReleaseAssets(source, release, target = null) {
  const assets = Array.isArray(release?.assets) ? release.assets : []
  if (source !== 'trace-browser') return assets
  const names = new Set(readyTraceBrowserPackages(release).filter(pkg => packageMatchesTarget(pkg, target))
    .flatMap(pkg => [pkg.asset.name, pkg.checksum.name]))
  return assets.filter(asset => isUploadedAsset(asset) && names.has(asset.name))
}

function traceBrowserReleaseAvailable(release, target = null) {
  if (target?.assetName) return downloadableReleaseAssets('trace-browser', release).some(asset => asset.name === target.assetName)
  return readyTraceBrowserPackages(release).some(pkg => packageMatchesTarget(pkg, target))
}

function traceBrowserRequestTarget(request) {
  const query = new URL(request.url).searchParams
  if (!['platform', 'arch', 'package'].some(key => query.has(key))) return null
  const rawPlatform = String(query.get('platform') || '').toLowerCase()
  const platform = rawPlatform === 'darwin' ? 'macos' : rawPlatform
  const architecture = normalizedArchitecture(query.get('arch'))
  const packageKind = query.get('package') || (platform === 'linux' ? 'portable' : 'installer')
  if (!PLATFORM_NAMES[platform] || !architecture || !['installer', 'portable', 'selfupdate'].includes(packageKind) ||
      (platform === 'macos' && packageKind !== 'installer') || (platform !== 'windows' && packageKind === 'selfupdate')) {
    return { error: '平台、架构或安装包类型无效' }
  }
  return { platform, architecture, packageKind }
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

function isChecksumAsset(name) {
  return /sha256|checksum/i.test(String(name || '')) || /\.(sha256|sha512|sums|checksum|checksums|sha256\.txt|sha512\.txt)$/i.test(String(name || ''))
}

function isSourceAsset(name) {
  return /source code|patchset/i.test(String(name || ''))
}

function isChecksumOrSource(name) {
  return isChecksumAsset(name) || isSourceAsset(name)
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
  if (source === 'trace-browser') {
    const packages = readyTraceBrowserPackages(release).filter(pkg => packageMatchesTarget(pkg, { platform, architecture, packageKind }))
    return (packages.find(pkg => pkg.packageKind === packageKind) || packages[0])?.asset || null
  }
  const assets = downloadableReleaseAssets(source, release)
  const matches = assets.filter(asset => chromiumAssetMatches(asset, platform, architecture, packageKind))
  return matches[0] || null
}

function publicAssetURL(request, source, assetName, releaseId = 0) {
  const releasePart = Number.isSafeInteger(Number(releaseId)) && Number(releaseId) > 0
    ? `/release/${Number(releaseId)}`
    : ''
  return `${PUBLIC_ORIGIN}/download/${source}${releasePart}/asset/${encodeURIComponent(assetName)}`
}

function publicReleasePayload(request, source, release, options = {}) {
  const config = SOURCES[source]
  const releaseId = Number(release.id || 0)
  const assets = downloadableReleaseAssets(source, release, options.target)
    // Keep checksum assets in the public manifest so desktop clients can
    // verify packages after downloading through the same Cloudflare source.
    // Source archives remain hidden because they are not installable assets.
    .filter(asset => asset && asset.name && !isSourceAsset(asset.name))
    .map(asset => ({
      name: asset.name,
      size: Number(asset.size || 0),
      contentType: asset.content_type || 'application/octet-stream',
      updatedAt: asset.updated_at || release.published_at || '',
      releaseId,
      downloadUrl: publicAssetURL(request, source, asset.name, releaseId),
    }))

  return {
    ok: true,
    ready: true,
    source,
    product: config.label,
    repository: config.repository,
    releaseId,
    version: source === 'trace-browser' ? traceBrowserReleaseVersion(release) : String(release.tag_name || release.name || '').replace(/^v/i, ''),
    tagName: release.tag_name || '',
    name: release.name || release.tag_name || '',
    publishedAt: release.published_at || '',
    notes: options.includeNotes === false ? '' : release.body || '',
    releaseUrl: `${PUBLIC_ORIGIN}/#downloads`,
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
  responseHeaders.set('Cache-Control', `public, max-age=0, s-maxage=${DOWNLOAD_CACHE_SECONDS}, stale-while-revalidate=60`)
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

  const release = await fetchLatestRelease(source, env, { platform, architecture, packageKind })
  if (!release) return errorResponse('该平台和架构暂无可下载的正式版本', 404)
  const asset = pickAsset(release, source, platform, architecture, packageKind)
  if (!asset) {
    return errorResponse(`当前最新版本没有找到 ${PLATFORM_NAMES[platform]} ${ARCHITECTURE_NAMES[architecture]} ${PACKAGE_NAMES[packageKind]}`, 404)
  }
  return proxyAsset(request, env, source, asset)
}

async function handleNamedAsset(request, env, source, encodedAssetName) {
  return handleNamedAssetForRelease(request, env, source, 0, encodedAssetName)
}

async function handleNamedAssetForRelease(request, env, source, releaseId, encodedAssetName) {
  let assetName
  try {
    assetName = decodeURIComponent(encodedAssetName)
  } catch {
    return errorResponse('资产名称编码无效', 400)
  }
  if (!assetName || assetName.includes('/') || assetName.includes('\\') || isSourceAsset(assetName)) {
    return errorResponse('不允许下载该资产', 403)
  }

  const release = releaseId > 0
    ? await fetchReleaseById(source, env, releaseId)
    : await fetchLatestRelease(source, env, { assetName })
  if (!release) return errorResponse('没有找到已上传的指定资产', 404)
  const asset = downloadableReleaseAssets(source, release).find(item => item.name === assetName)
  if (!asset) return errorResponse(releaseId > 0 ? '指定版本中没有找到该资产' : '当前最新版本中没有找到该资产', 404)
  return proxyAsset(request, env, source, asset)
}

async function handleAPI(request, env, source) {
  const target = source === 'trace-browser' ? traceBrowserRequestTarget(request) : null
  if (target?.error) return errorResponse(target.error, 400)
  const release = await fetchLatestRelease(source, env, target)
  if (!release) return jsonResponse({ ok: true, ready: false, source, assets: [] }, 200, { 'Cache-Control': 'no-store' })
  const etag = releaseETag(release, source, target)
  if (requestMatchesETag(request, etag)) return notModifiedResponse(etag)
  return jsonResponse(publicReleasePayload(request, source, release, { target }), 200, { ETag: etag })
}

function releaseListETag(releases, source, target) {
  return identityETag(releases.map(release => releaseETag(release, source, target)).join('|'), 'releases')
}

async function handleReleasesAPI(request, env, source) {
  const url = new URL(request.url)
  const target = source === 'trace-browser' ? traceBrowserRequestTarget(request) : null
  if (target?.error) return errorResponse(target.error, 400)
  const limit = Math.max(1, Math.min(100, Math.floor(Number(url.searchParams.get('limit')) || 100)))
  const releases = source === 'trace-browser'
    ? await fetchAvailableTraceBrowserReleases(env, limit, target)
    : await fetchReleaseList(source, env, limit)
  const etag = releaseListETag(releases, source, target)
  if (requestMatchesETag(request, etag)) return notModifiedResponse(etag)
  const payload = releases.map(release => publicReleasePayload(request, source, release, { includeNotes: false, target }))
  const first = payload[0] || {}
  return jsonResponse({
    ok: true,
    source,
    product: first.product || SOURCES[source].label,
    repository: first.repository || SOURCES[source].repository,
    count: payload.length,
    releases: payload,
  }, 200, { ETag: etag })
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
    if (PROXY_HEALTH_PATH.test(url.pathname)) {
      return proxyHealthResponse(request)
    }
    const apiMatch = url.pathname.match(/^\/api\/(trace-browser|chromium)\/latest\/?$/)
    if (apiMatch) {
      try {
        return await handleAPI(request, env, apiMatch[1])
      } catch (error) {
        return errorResponse('获取最新版本失败', 502, error instanceof Error ? error.message : String(error))
      }
    }

    const releasesAPIMatch = url.pathname.match(/^\/api\/(trace-browser|chromium)\/releases\/?$/)
    if (releasesAPIMatch) {
      try {
        return await handleReleasesAPI(request, env, releasesAPIMatch[1])
      } catch (error) {
        return errorResponse('获取版本列表失败', 502, error instanceof Error ? error.message : String(error))
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

    const namedReleaseAssetMatch = url.pathname.match(/^\/download\/(trace-browser|chromium)\/release\/(\d+)\/asset\/([^/]+)\/?$/)
    if (namedReleaseAssetMatch) {
      try {
        return await handleNamedAssetForRelease(
          request,
          env,
          namedReleaseAssetMatch[1],
          Number(namedReleaseAssetMatch[2]),
          namedReleaseAssetMatch[3],
        )
      } catch (error) {
        return errorResponse('下载代理失败', 502, error instanceof Error ? error.message : String(error))
      }
    }

    return serveSite(request, env)
  },
}
