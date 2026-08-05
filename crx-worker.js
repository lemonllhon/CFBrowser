const DEFAULT_PROD_VERSION = '124.0.0.0'
const CACHE_TTL_SECONDS = 86400
const MAX_SEARCH_RESULTS = 30

export default {
  async fetch(request) {
    try {
      if (request.method === 'OPTIONS') {
        return new Response(null, { headers: corsHeaders() })
      }
      return await handleRequest(request)
    } catch (error) {
      return json({
        ok: false,
        code: 400,
        error: error && error.message ? error.message : String(error),
      }, 400)
    }
  },
}

async function handleRequest(request) {
  const url = new URL(request.url)
  const pathname = normalizePath(url.pathname)

  if (pathname === '/') {
    return json({
      ok: true,
      name: 'Trace Chrome Extension Store Worker',
      domain: url.origin,
      endpoints: [
        'GET /resolve?url=https://chromewebstore.google.com/detail/name/id',
        'POST /chrome/dlink',
        'GET /chrome/crx/:addonId.crx',
        'GET /chrome/search?q=keyword',
        'GET /chrome/search/:keyword',
        'GET /chrome/detail/:addonId',
        'GET /chrome/image?url=https://lh3.googleusercontent.com/...',
      ],
    })
  }

  if (pathname === '/resolve') {
    const target = url.searchParams.get('url') || url.searchParams.get('id') || ''
    return json(buildDlinkResponse(extractChromeAddonId(target), url.origin))
  }

  if (pathname === '/chrome/dlink' && request.method === 'POST') {
    const body = await readRequestBody(request)
    const addonId = extractChromeAddonId(body.addonId || body.id || body.url || body.storeUrl || body.downloadUrl || '')
    return json(buildDlinkResponse(addonId, url.origin))
  }

  if ((pathname === '/chrome/search' || pathname === '/search') && (request.method === 'GET' || request.method === 'POST')) {
    return searchChromeWebStore(request, url, '')
  }

  const searchPathMatch = pathname.match(/^\/(?:chrome\/)?search\/(.+)$/)
  if (searchPathMatch && (request.method === 'GET' || request.method === 'POST')) {
    return searchChromeWebStore(request, url, decodeURIComponent(searchPathMatch[1]))
  }

  const crxMatch = pathname.match(/^\/chrome\/crx\/([a-p]{32})\.crx$/i)
  if (crxMatch && request.method === 'GET') {
    return proxyChromeCrx(crxMatch[1].toLowerCase(), request)
  }

  const detailMatch = pathname.match(/^\/chrome\/detail\/([a-p]{32})$/i)
  if (detailMatch && request.method === 'GET') {
    return fetchChromeDetail(detailMatch[1].toLowerCase(), request, url.origin)
  }

  if (pathname === '/chrome/image' && request.method === 'GET') {
    return proxyChromeImage(url.searchParams.get('url') || '')
  }

  return json({ ok: false, code: 404, error: 'Not found' }, 404)
}

async function searchChromeWebStore(request, requestUrl, pathQuery) {
  const body = request.method === 'POST' ? await readRequestBody(request) : {}
  const query = String(pathQuery || requestUrl.searchParams.get('q') || requestUrl.searchParams.get('query') || body.q || body.query || '').trim()
  if (!query) {
    throw new Error('Search query is required')
  }

  const hl = String(requestUrl.searchParams.get('hl') || body.hl || 'zh-CN').trim() || 'zh-CN'
  const pageSize = clampNumber(Number(requestUrl.searchParams.get('limit') || body.limit || MAX_SEARCH_RESULTS), 1, MAX_SEARCH_RESULTS)
  const storeUrl = `https://chromewebstore.google.com/search/${encodeURIComponent(query)}?hl=${encodeURIComponent(hl)}`
  const upstream = await fetch(storeUrl, {
    headers: chromeStoreHeaders(hl),
    cf: { cacheTtl: CACHE_TTL_SECONDS, cacheEverything: true },
  })
  const html = await upstream.text()
  if (!upstream.ok) {
    return json({
      ok: false,
      code: upstream.status,
      query,
      sourceUrl: storeUrl,
      error: `Chrome Web Store search failed: HTTP ${upstream.status}`,
      detail: html.slice(0, 1000),
    }, upstream.status)
  }

  const cards = parseSearchCards(html, requestUrl.origin)
  const recordItems = parseInitExtensionRecords(html, requestUrl.origin)
  const byId = new Map()
  for (const item of recordItems) {
    byId.set(item.addonId, mergeExtensionItems(byId.get(item.addonId), item))
  }
  const merged = []
  for (const card of cards) {
    const fromData = byId.get(card.addonId)
    merged.push(mergeExtensionItems(fromData, card))
    byId.delete(card.addonId)
  }
  for (const item of byId.values()) {
    merged.push(item)
  }

  const results = uniqueByAddonId(merged).slice(0, pageSize).map(item => finalizeExtensionItem(item, requestUrl.origin))
  return json({
    ok: true,
    code: 200,
    storeType: 'chrome',
    query,
    count: results.length,
    sourceUrl: storeUrl,
    results,
  }, 200, {
    'Cache-Control': `public, max-age=${CACHE_TTL_SECONDS}`,
  })
}

async function fetchChromeDetail(addonId, request, origin) {
  const id = assertAddonId(addonId)
  const hl = new URL(request.url).searchParams.get('hl') || 'zh-CN'
  const pageUrl = `https://chromewebstore.google.com/detail/${id}?hl=${encodeURIComponent(hl)}`
  const upstream = await fetch(pageUrl, {
    headers: chromeStoreHeaders(hl),
    cf: { cacheTtl: CACHE_TTL_SECONDS, cacheEverything: true },
  })
  const html = await upstream.text()
  if (!upstream.ok) {
    return json({
      ok: false,
      code: upstream.status,
      addonId: id,
      sourceUrl: pageUrl,
      error: `Chrome Web Store detail failed: HTTP ${upstream.status}`,
      detail: html.slice(0, 1000),
    }, upstream.status)
  }

  const records = parseInitExtensionRecords(html, origin)
  const fromData = records.find(item => item.addonId === id)
  const fallback = {
    addonId: id,
    name: pickMeta(html, 'og:title') || pickTitle(html),
    description: pickMeta(html, 'og:description') || '',
    icon: pickMeta(html, 'og:image') || '',
    thumbnail: pickMeta(html, 'og:image') || '',
  }
  const item = finalizeExtensionItem(mergeExtensionItems(fallback, fromData), origin)
  item.sourceUrl = pageUrl
  item.screenshots = collectGoogleImageUrls(html)
    .filter(image => image !== item.iconOriginal && image !== item.thumbnailOriginal)
    .slice(0, 12)
    .map(image => ({
      url: buildImageProxyUrl(image, origin),
      proxyUrl: buildImageProxyUrl(image, origin),
      originalUrl: image,
    }))

  return json({
    ok: true,
    code: 200,
    storeType: 'chrome',
    detail: item,
  }, 200, {
    'Cache-Control': `public, max-age=${CACHE_TTL_SECONDS}`,
  })
}

async function proxyChromeCrx(addonId, request) {
  const id = assertAddonId(addonId)
  const upstreamUrl = buildGoogleCrxUrl(id)
  const headers = new Headers()
  headers.set('User-Agent', chromeUserAgent())
  headers.set('Accept', '*/*')
  const range = request.headers.get('Range')
  if (range) headers.set('Range', range)

  const upstream = await fetch(upstreamUrl, {
    method: 'GET',
    headers,
    redirect: 'follow',
    cf: { cacheTtl: CACHE_TTL_SECONDS, cacheEverything: true },
  })

  if (!upstream.ok && upstream.status !== 206) {
    const text = await safeReadText(upstream)
    return json({
      ok: false,
      code: upstream.status,
      error: `Google download failed: HTTP ${upstream.status}`,
      detail: text.slice(0, 1000),
    }, upstream.status)
  }

  const responseHeaders = new Headers(upstream.headers)
  applyCors(responseHeaders)
  responseHeaders.set('Content-Type', 'application/x-chrome-extension')
  responseHeaders.set('Content-Disposition', `attachment; filename="${id}.crx"`)
  responseHeaders.set('Cache-Control', `public, max-age=${CACHE_TTL_SECONDS}`)
  return new Response(upstream.body, {
    status: upstream.status,
    statusText: upstream.statusText,
    headers: responseHeaders,
  })
}

async function proxyChromeImage(rawUrl) {
  const target = normalizeImageUrl(rawUrl)
  const upstream = await fetch(target, {
    headers: {
      'User-Agent': chromeUserAgent(),
      'Accept': 'image/avif,image/webp,image/apng,image/svg+xml,image/*,*/*;q=0.8',
      'Referer': 'https://chromewebstore.google.com/',
    },
    cf: { cacheTtl: CACHE_TTL_SECONDS, cacheEverything: true },
  })
  if (!upstream.ok) {
    return json({
      ok: false,
      code: upstream.status,
      error: `Image proxy failed: HTTP ${upstream.status}`,
    }, upstream.status)
  }
  const headers = new Headers(upstream.headers)
  applyCors(headers)
  headers.set('Cache-Control', `public, max-age=${CACHE_TTL_SECONDS}`)
  return new Response(upstream.body, {
    status: upstream.status,
    headers,
  })
}

function parseSearchCards(html, origin) {
  const marker = '<div class="Cb7Kte"'
  const results = []
  let index = 0
  while (index >= 0 && index < html.length) {
    const start = html.indexOf(marker, index)
    if (start < 0) break
    const next = html.indexOf(marker, start + marker.length)
    const chunk = html.slice(start, next >= 0 ? next : Math.min(html.length, start + 12000))
    index = next >= 0 ? next : html.length

    const addonId = extractAttribute(chunk, 'data-item-id')
    if (!addonId || !/^[a-p]{32}$/i.test(addonId)) continue
    const href = firstMatch(chunk, /<a\b[^>]*href="([^"]+)"/i)
    const image = normalizeGoogleImageUrl(decodeHtml(firstMatch(chunk, /<img\b[^>]*src="([^"]+)"/i)))
    const item = {
      storeType: 'chrome',
      addonId: addonId.toLowerCase(),
      name: textFromHtml(firstMatch(chunk, /<h2\b[^>]*class="[^"]*\bCiI2if\b[^"]*"[^>]*>([\s\S]*?)<\/h2>/i)),
      publisher: textFromHtml(firstMatch(chunk, /<span\b[^>]*class="[^"]*\bcJI8ee\b[^"]*"[^>]*>([\s\S]*?)<\/span>/i)),
      rating: parseOptionalNumber(textFromHtml(firstMatch(chunk, /<span\b[^>]*class="[^"]*\bVq0ZA\b[^"]*"[^>]*>([\s\S]*?)<\/span>/i))),
      description: textFromHtml(firstMatch(chunk, /<p\b[^>]*class="[^"]*\bg3IrHd\b[^"]*"[^>]*>([\s\S]*?)<\/p>/i)),
      verifiedPublisher: /aria-label="(?:由经过验证的发布者发布|Verified publisher)"/i.test(chunk),
      thumbnail: image,
      icon: image,
      chromeStorePath: decodeHtml(href),
    }
    item.storeUrl = absoluteChromeStoreUrl(item.chromeStorePath || `/detail/${item.addonId}`)
    item.detailUrl = `${origin}/chrome/detail/${item.addonId}`
    results.push(item)
  }
  return uniqueByAddonId(results)
}

function parseInitExtensionRecords(html, origin) {
  const records = []
  const callbackRegex = /AF_initDataCallback\(\{[\s\S]*?\}\);<\/script>/g
  let callback
  while ((callback = callbackRegex.exec(html))) {
    const script = callback[0]
    const dataIndex = script.indexOf('data:')
    if (dataIndex < 0) continue
    const arrayStart = script.indexOf('[', dataIndex)
    if (arrayStart < 0) continue
    const arrayText = extractBalancedArray(script, arrayStart)
    if (!arrayText) continue
    try {
      const data = JSON.parse(arrayText)
      collectExtensionRecords(data, records)
    } catch (_) {
      // Chrome occasionally changes this bootstrap payload. Ignore bad chunks.
    }
  }
  return uniqueByAddonId(records.map(record => recordToExtensionItem(record, origin)).filter(Boolean))
}

function collectExtensionRecords(value, out) {
  if (!Array.isArray(value)) return
  if (isExtensionRecord(value)) {
    out.push(value)
  }
  for (const child of value) {
    collectExtensionRecords(child, out)
  }
}

function isExtensionRecord(value) {
  return Array.isArray(value) &&
    typeof value[0] === 'string' &&
    /^[a-p]{32}$/i.test(value[0]) &&
    typeof value[2] === 'string'
}

function recordToExtensionItem(record, origin) {
  const addonId = assertAddonId(record[0])
  const manifestText = findManifestText(record)
  const manifest = parseManifest(manifestText)
  const image = normalizeGoogleImageUrl(stringOrEmpty(record[5] || record[1]))
  const icon = normalizeGoogleImageUrl(stringOrEmpty(record[1] || record[5]))
  const category = findCategory(record)
  return {
    storeType: 'chrome',
    addonId,
    name: stringOrEmpty(record[2] || manifest.name),
    publisher: stringOrEmpty(record[7]),
    rating: parseOptionalNumber(record[3]),
    ratingCount: parseOptionalNumber(record[4]),
    description: stringOrEmpty(record[6] || manifest.description),
    icon,
    thumbnail: image,
    category,
    activeInstallCount: parseOptionalNumber(record[14]),
    lastUpdatedAt: findTimestampIso(record),
    version: stringOrEmpty(manifest.version),
    manifestVersion: parseOptionalNumber(manifest.manifest_version),
    permissions: normalizeStringArray(manifest.permissions),
    hostPermissions: normalizeStringArray(manifest.host_permissions),
    manifestJson: manifestText || '',
    storeUrl: `https://chromewebstore.google.com/detail/${addonId}`,
    detailUrl: `${origin}/chrome/detail/${addonId}`,
  }
}

function finalizeExtensionItem(item, origin) {
  const addonId = assertAddonId(item.addonId)
  const iconOriginal = normalizeGoogleImageUrl(item.iconOriginal || item.icon || item.thumbnailOriginal || item.thumbnail || '')
  const thumbnailOriginal = normalizeGoogleImageUrl(item.thumbnailOriginal || item.thumbnail || item.iconOriginal || item.icon || '')
  const iconProxy = iconOriginal ? buildImageProxyUrl(iconOriginal, origin) : ''
  const thumbnailProxy = thumbnailOriginal ? buildImageProxyUrl(thumbnailOriginal, origin) : ''
  const finalized = {
    ok: true,
    storeType: 'chrome',
    addonId,
    name: stringOrEmpty(item.name),
    publisher: stringOrEmpty(item.publisher),
    verifiedPublisher: Boolean(item.verifiedPublisher),
    rating: item.rating === null || item.rating === undefined ? null : Number(item.rating),
    ratingCount: item.ratingCount === null || item.ratingCount === undefined ? null : Number(item.ratingCount),
    activeInstallCount: item.activeInstallCount === null || item.activeInstallCount === undefined ? null : Number(item.activeInstallCount),
    description: stringOrEmpty(item.description),
    category: item.category || null,
    version: stringOrEmpty(item.version),
    manifestVersion: item.manifestVersion === null || item.manifestVersion === undefined ? null : Number(item.manifestVersion),
    permissions: normalizeStringArray(item.permissions),
    hostPermissions: normalizeStringArray(item.hostPermissions),
    lastUpdatedAt: item.lastUpdatedAt || '',
    icon: iconProxy,
    iconProxy,
    iconOriginal,
    thumbnail: thumbnailProxy,
    thumbnailProxy,
    thumbnailOriginal,
    storeUrl: item.storeUrl || `https://chromewebstore.google.com/detail/${addonId}`,
    detailUrl: item.detailUrl || `${origin}/chrome/detail/${addonId}`,
    downloadUrl: buildGoogleCrxUrl(addonId),
    dlink: `${origin}/chrome/crx/${addonId}.crx`,
  }
  if (item.manifestJson) {
    finalized.manifestJson = item.manifestJson
  }
  return finalized
}

function mergeExtensionItems(base, patch) {
  if (!base) return patch || {}
  if (!patch) return base
  const out = { ...base }
  for (const key of Object.keys(patch)) {
    const value = patch[key]
    if (value === undefined || value === null || value === '') continue
    if (Array.isArray(value) && value.length === 0) continue
    out[key] = value
  }
  return out
}

function buildDlinkResponse(addonId, origin) {
  const id = assertAddonId(addonId)
  const workerUrl = `${origin}/chrome/crx/${id}.crx`
  const googleUrl = buildGoogleCrxUrl(id)
  return {
    ok: true,
    code: 200,
    storeType: 'chrome',
    addonId: id,
    storeUrl: `https://chromewebstore.google.com/detail/${id}`,
    downloadUrl: googleUrl,
    dlink: workerUrl,
    dlinkOffline: [
      { name: 'Cloudflare 离线下载', url: workerUrl },
      { name: 'Google 官方直链', url: googleUrl },
    ],
    dlinkHistory: [],
  }
}

function buildGoogleCrxUrl(addonId) {
  const id = assertAddonId(addonId)
  const x = encodeURIComponent(`id=${id}&uc`)
  return `https://clients2.google.com/service/update2/crx?response=redirect&prodversion=${encodeURIComponent(DEFAULT_PROD_VERSION)}&acceptformat=crx2,crx3&x=${x}`
}

function extractChromeAddonId(input) {
  const value = String(input || '').trim().toLowerCase()
  if (!value) throw new Error('addonId/url is required')
  if (/^[a-p]{32}$/.test(value)) return value
  const decoded = safeDecodeURIComponent(value)
  const patterns = [
    /\/detail\/[^/?#]+\/([a-p]{32})(?:[/?#]|$)/i,
    /\/detail\/([a-p]{32})(?:[/?#]|$)/i,
    /[?&]id=([a-p]{32})(?:[&#]|$)/i,
    /id%3d([a-p]{32})/i,
    /\b([a-p]{32})\b/i,
  ]
  for (const pattern of patterns) {
    const match = decoded.match(pattern)
    if (match && match[1]) return assertAddonId(match[1])
  }
  throw new Error('Invalid Chrome extension URL or ID')
}

function assertAddonId(id) {
  const value = String(id || '').trim().toLowerCase()
  if (!/^[a-p]{32}$/.test(value)) {
    throw new Error('Invalid Chrome extension ID')
  }
  return value
}

async function readRequestBody(request) {
  const contentType = request.headers.get('Content-Type') || ''
  if (contentType.includes('application/json')) {
    return await request.json()
  }
  if (contentType.includes('application/x-www-form-urlencoded') || contentType.includes('multipart/form-data')) {
    const form = await request.formData()
    const out = {}
    for (const entry of form.entries()) {
      out[entry[0]] = String(entry[1])
    }
    return out
  }
  const text = await request.text()
  if (!text.trim()) return {}
  try {
    return JSON.parse(text)
  } catch (_) {
    return { q: text.trim(), addonId: text.trim() }
  }
}

function chromeStoreHeaders(hl) {
  return {
    'User-Agent': chromeUserAgent(),
    'Accept': 'text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8',
    'Accept-Language': `${hl},zh;q=0.9,en;q=0.8`,
    'Referer': 'https://chromewebstore.google.com/',
  }
}

function chromeUserAgent() {
  return `Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/${DEFAULT_PROD_VERSION} Safari/537.36`
}

function json(data, status, extraHeaders) {
  const headers = corsHeaders()
  headers.set('Content-Type', 'application/json; charset=utf-8')
  if (extraHeaders) {
    for (const key of Object.keys(extraHeaders)) {
      headers.set(key, extraHeaders[key])
    }
  }
  return new Response(JSON.stringify(data, null, 2), {
    status: status || 200,
    headers,
  })
}

function corsHeaders() {
  return new Headers({
    'Access-Control-Allow-Origin': '*',
    'Access-Control-Allow-Methods': 'GET,POST,OPTIONS',
    'Access-Control-Allow-Headers': 'Content-Type, Range',
    'Access-Control-Expose-Headers': 'Content-Length, Content-Range, Content-Disposition',
  })
}

function applyCors(headers) {
  for (const entry of corsHeaders().entries()) {
    headers.set(entry[0], entry[1])
  }
}

function normalizePath(pathname) {
  const value = String(pathname || '/').replace(/\/{2,}/g, '/')
  return value.length > 1 && value.endsWith('/') ? value.slice(0, -1) : value
}

function firstMatch(text, regex) {
  const match = text.match(regex)
  return match && match[1] ? match[1] : ''
}

function extractAttribute(html, name) {
  const escaped = name.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  return decodeHtml(firstMatch(html, new RegExp(`${escaped}="([^"]*)"`, 'i')))
}

function textFromHtml(html) {
  return decodeHtml(String(html || '').replace(/<[^>]+>/g, ' ').replace(/\s+/g, ' '))
}

function decodeHtml(value) {
  return String(value || '')
    .replace(/&amp;/g, '&')
    .replace(/&quot;/g, '"')
    .replace(/&#39;/g, "'")
    .replace(/&lt;/g, '<')
    .replace(/&gt;/g, '>')
    .replace(/&#x([0-9a-f]+);/gi, (_, hex) => String.fromCodePoint(parseInt(hex, 16)))
    .replace(/&#(\d+);/g, (_, num) => String.fromCodePoint(parseInt(num, 10)))
    .trim()
}

function safeDecodeURIComponent(value) {
  try {
    return decodeURIComponent(value)
  } catch (_) {
    return value
  }
}

async function safeReadText(response) {
  try {
    return await response.text()
  } catch (_) {
    return ''
  }
}

function parseOptionalNumber(value) {
  if (value === null || value === undefined || value === '') return null
  const number = Number(String(value).replace(/,/g, '').trim())
  return Number.isFinite(number) ? number : null
}

function clampNumber(value, min, max) {
  if (!Number.isFinite(value)) return max
  return Math.max(min, Math.min(max, Math.floor(value)))
}

function stringOrEmpty(value) {
  return typeof value === 'string' ? value.trim() : ''
}

function normalizeStringArray(value) {
  return Array.isArray(value) ? value.map(item => String(item || '').trim()).filter(Boolean) : []
}

function parseManifest(manifestText) {
  if (!manifestText) return {}
  try {
    return JSON.parse(manifestText)
  } catch (_) {
    return {}
  }
}

function findManifestText(value) {
  let found = ''
  walk(value, item => {
    if (!found && typeof item === 'string' && item.includes('"manifest_version"')) {
      found = item
    }
  })
  return found
}

function findCategory(value) {
  let found = null
  walk(value, item => {
    if (!found && Array.isArray(item) && typeof item[0] === 'string' && item[0].includes('/')) {
      found = item[0]
    }
  })
  return found
}

function findTimestampIso(value) {
  let found = ''
  walk(value, item => {
    if (found || !Array.isArray(item) || item.length < 1) return
    const seconds = Number(item[0])
    if (Number.isFinite(seconds) && seconds > 1000000000 && seconds < 2200000000) {
      found = new Date(seconds * 1000).toISOString()
    }
  })
  return found
}

function walk(value, visitor) {
  visitor(value)
  if (Array.isArray(value)) {
    for (const item of value) {
      walk(item, visitor)
    }
  } else if (value && typeof value === 'object') {
    for (const key of Object.keys(value)) {
      walk(value[key], visitor)
    }
  }
}

function extractBalancedArray(text, start) {
  let depth = 0
  let inString = false
  let escape = false
  for (let i = start; i < text.length; i += 1) {
    const ch = text[i]
    if (inString) {
      if (escape) {
        escape = false
      } else if (ch === '\\') {
        escape = true
      } else if (ch === '"') {
        inString = false
      }
      continue
    }
    if (ch === '"') {
      inString = true
      continue
    }
    if (ch === '[') {
      depth += 1
    } else if (ch === ']') {
      depth -= 1
      if (depth === 0) {
        return text.slice(start, i + 1)
      }
    }
  }
  return ''
}

function collectGoogleImageUrls(html) {
  const urls = []
  const regex = /https:\/\/lh3\.googleusercontent\.com\/[^"'\\<>\s]+/g
  let match
  while ((match = regex.exec(html))) {
    urls.push(normalizeGoogleImageUrl(decodeHtml(match[0])))
  }
  return Array.from(new Set(urls)).filter(Boolean)
}

function normalizeGoogleImageUrl(value) {
  const text = String(value || '').trim()
  if (!text) return ''
  return text.startsWith('//') ? `https:${text}` : text
}

function normalizeImageUrl(value) {
  const text = normalizeGoogleImageUrl(decodeHtml(value))
  if (!text) throw new Error('Image URL is required')
  const parsed = new URL(text)
  const host = parsed.hostname.toLowerCase()
  const allowed = host === 'lh3.googleusercontent.com' ||
    host.endsWith('.googleusercontent.com') ||
    host === 'www.gstatic.com' ||
    host.endsWith('.gstatic.com')
  if (!allowed) {
    throw new Error('Image host is not allowed')
  }
  return parsed.toString()
}

function buildImageProxyUrl(imageUrl, origin) {
  if (!imageUrl) return ''
  return `${origin}/chrome/image?url=${encodeURIComponent(imageUrl)}`
}

function absoluteChromeStoreUrl(path) {
  const value = decodeHtml(path || '')
  if (!value) return ''
  try {
    return new URL(value, 'https://chromewebstore.google.com/').toString()
  } catch (_) {
    return ''
  }
}

function uniqueByAddonId(items) {
  const seen = new Set()
  const out = []
  for (const item of items) {
    if (!item || !item.addonId || seen.has(item.addonId)) continue
    seen.add(item.addonId)
    out.push(item)
  }
  return out
}

function pickMeta(html, property) {
  const escaped = property.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  const re1 = new RegExp(`<meta[^>]+property=["']${escaped}["'][^>]+content=["']([^"']*)["']`, 'i')
  const re2 = new RegExp(`<meta[^>]+content=["']([^"']*)["'][^>]+property=["']${escaped}["']`, 'i')
  const match = html.match(re1) || html.match(re2)
  return match ? decodeHtml(match[1]) : ''
}

function pickTitle(html) {
  const match = html.match(/<title[^>]*>([\s\S]*?)<\/title>/i)
  return match ? decodeHtml(match[1]).replace(/\s+-\s+Chrome Web Store.*$/i, '').trim() : ''
}
