import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Sliders } from 'lucide-react'
import { Button, Card, ConfirmModal, FormItem, Input, Modal, Select, Switch, Table, Textarea, toast } from '../../../shared/components'
import type { SortOrder, TableColumn } from '../../../shared/components/Table'
import type { BrowserProxy, ProxyIPHealthResult } from '../types'
import { fetchBrowserProxies, fetchBrowserProxyGroups, saveBrowserProxies, browserProxyTestSpeed, browserProxyBatchTestSpeed, browserProxyPreviewBatchTestSpeed, browserProxyCheckIPHealth, browserProxyBatchCheckIPHealth, browserProxyPreviewBatchCheckIPHealth, fetchClashImportFromURL } from '../api'
import { EventsOn } from '../../../wailsjs/runtime/runtime'
import yaml from 'js-yaml'

// 内置代理 ID，不可删除、不可编辑
const BUILTIN_PROXY_IDS = new Set(['__direct__', '__local__'])
const PROXY_LATENCY_CACHE_KEY = 'browser:proxyPool:latencyMap:v1'
const PROXY_IP_HEALTH_CACHE_KEY = 'browser:proxyPool:ipHealthMap:v1'
const PROXY_SOURCE_IGNORED_NAMES_KEY = 'browser:proxyPool:sourceIgnoredProxyNames:v1'
const PROXY_GLOBAL_AUTO_REFRESH_KEY = 'browser:proxyPool:globalAutoRefreshEnabled:v1'
const PROXY_GLOBAL_REFRESH_INTERVAL_KEY = 'browser:proxyPool:globalRefreshIntervalM:v1'
const PROXY_LATENCY_CACHE_TTL_MS = 12 * 60 * 60 * 1000
const PROXY_IP_HEALTH_CACHE_TTL_MS = 12 * 60 * 60 * 1000
const PROXY_COLUMNS_STORAGE_KEY = 'browser:proxyPoolColumns:v1'
const DEFAULT_PROXY_COLUMN_KEYS = ['checkbox', 'proxyName', 'type', 'server', 'port', 'latency', 'ipHealth', 'actions']
type ColumnOption = {
  key: string
  label: string
  locked?: boolean
}

const PROXY_COLUMN_OPTIONS: ColumnOption[] = [
  { key: 'checkbox', label: '选择', locked: true },
  { key: 'proxyName', label: '代理名称' },
  { key: 'groupName', label: '分组' },
  { key: 'source', label: '来源' },
  { key: 'type', label: '类型' },
  { key: 'server', label: '服务器' },
  { key: 'port', label: '端口' },
  { key: 'latency', label: '延迟' },
  { key: 'ipHealth', label: 'IP健康' },
  { key: 'actions', label: '操作', locked: true },
]

function readStoredProxyColumnKeys() {
  try {
    const parsed = JSON.parse(localStorage.getItem(PROXY_COLUMNS_STORAGE_KEY) || '[]')
    if (Array.isArray(parsed)) {
      const allowed = PROXY_COLUMN_OPTIONS.map(item => item.key)
      const valid = parsed.filter((key): key is string => typeof key === 'string' && allowed.includes(key))
      if (valid.length > 0) return valid
    }
  } catch { /* ignore */ }
  return DEFAULT_PROXY_COLUMN_KEYS
}

const BUILTIN_PROXIES: BrowserProxy[] = [
  { proxyId: '__direct__', proxyName: '直连（不走代理）', proxyConfig: 'direct://' },
  { proxyId: '__local__', proxyName: '本地代理', proxyConfig: 'http://127.0.0.1:7890' },
]

function isBuiltinProxy(proxy: Pick<BrowserProxy, 'proxyId' | 'proxyConfig'>): boolean {
  return BUILTIN_PROXY_IDS.has(proxy.proxyId) || proxy.proxyConfig.trim() === 'direct://'
}

function ensureBuiltinProxies(proxies: BrowserProxy[]): BrowserProxy[] {
  const result = [...proxies]
  for (const builtin of BUILTIN_PROXIES) {
    if (!result.find(p => p.proxyId === builtin.proxyId)) {
      result.unshift(builtin)
    }
  }
  return result
}

interface ClashProxy {
  name: string
  type: string
  server: string
  port: number
  [key: string]: any
}

type ProxyImportMode = 'clash' | 'direct'
type ProxyResourceView = 'proxies' | 'sources'
type PreviewLatencyFilter = 'all' | 'untested' | 'testing' | 'ok' | 'fast' | 'slow' | 'timeout' | 'unsupported'
type PreviewHealthFilter = 'all' | 'untested' | 'ok' | 'failed' | 'highRisk' | 'residential' | 'datacenter'

const PREVIEW_LATENCY_FILTER_OPTIONS: { value: PreviewLatencyFilter; label: string }[] = [
  { value: 'all', label: '全部延迟' },
  { value: 'untested', label: '未测速' },
  { value: 'testing', label: '测速中' },
  { value: 'ok', label: '可用' },
  { value: 'fast', label: '低延迟' },
  { value: 'slow', label: '高延迟' },
  { value: 'timeout', label: '超时' },
  { value: 'unsupported', label: '不支持' },
]

const PREVIEW_HEALTH_FILTER_OPTIONS: { value: PreviewHealthFilter; label: string }[] = [
  { value: 'all', label: '全部健康' },
  { value: 'untested', label: '未检测' },
  { value: 'ok', label: '检测通过' },
  { value: 'failed', label: '检测失败' },
  { value: 'highRisk', label: '高风险' },
  { value: 'residential', label: '住宅IP' },
  { value: 'datacenter', label: '机房IP' },
]

interface DirectImportForm {
  proxyName: string
  protocol: 'http' | 'https' | 'socks5'
  server: string
  port: string
  username: string
  password: string
}

interface DirectImportLineResult {
  name: string
  config: string
}

const DIRECT_PROXY_PROTOCOL_OPTIONS = [
  { value: 'http', label: 'HTTP' },
  { value: 'https', label: 'HTTPS' },
  { value: 'socks5', label: 'SOCKS5' },
] as const

const INITIAL_DIRECT_IMPORT_FORM: DirectImportForm = {
  proxyName: '',
  protocol: 'http',
  server: '',
  port: '',
  username: '',
  password: '',
}

interface ImportCandidate {
  proxyName: string
  proxyConfig: string
}

interface ProxyDisplayInfo {
  proxyId: string
  proxyName: string
  proxyConfig: string
  groupName: string
  sourceId: string
  sourceUrl: string
  sourceAutoRefresh: boolean
  sourceRefreshIntervalM: number
  sourceLastRefreshAt: string
  type: string
  server: string
  port: number
  latencyMs?: number
}

interface URLImportSourceMeta {
  sourceId: string
  sourceUrl: string
  sourceNamePrefix: string
  sourceGroupName: string
  sourceDnsServers: string
  sourceAutoRefresh: boolean
  sourceRefreshIntervalM: number
  sourceLastRefreshAt: string
  proxyCount: number
}

function parseProxyInfo(proxyConfig: string): { type: string; server: string; port: number } {
  const cfg = proxyConfig.trim()
  if (cfg === 'direct://') return { type: 'direct', server: '-', port: 0 }
  const urlMatch = cfg.match(/^([a-zA-Z0-9+\-]+):\/\//)
  if (urlMatch) {
    const scheme = urlMatch[1].toLowerCase()
    try {
      const u = new URL(cfg)
      return { type: scheme, server: u.hostname, port: parseInt(u.port) || 0 }
    } catch {
      return { type: scheme, server: '-', port: 0 }
    }
  }
  try {
    const parsed = yaml.load(cfg) as ClashProxy[] | ClashProxy
    const proxy = Array.isArray(parsed) ? parsed[0] : parsed
    return { type: proxy?.type || '-', server: proxy?.server || '-', port: proxy?.port || 0 }
  } catch {
    return { type: '-', server: '-', port: 0 }
  }
}

function toDisplayList(proxies: BrowserProxy[]): ProxyDisplayInfo[] {
  return proxies.map(p => {
    const info = parseProxyInfo(p.proxyConfig)
    return {
      proxyId: p.proxyId,
      proxyName: p.proxyName,
      proxyConfig: p.proxyConfig,
      groupName: p.groupName || '',
      sourceId: p.sourceId || '',
      sourceUrl: p.sourceUrl || '',
      sourceAutoRefresh: !!p.sourceAutoRefresh,
      sourceRefreshIntervalM: Math.max(0, Number(p.sourceRefreshIntervalM || 0)),
      sourceLastRefreshAt: p.sourceLastRefreshAt || '',
      ...info,
    }
  })
}

function proxyToYaml(proxy: ClashProxy): string {
  return yaml.dump([proxy], { flowLevel: -1, lineWidth: -1 }).trim()
}

function quoteYamlScalar(value: string): string {
  const v = value.trim()
  if (!v) return "''"
  return `'${v.replace(/'/g, "''")}'`
}

function normalizeImportedProxyArray(payload: unknown): ClashProxy[] | null {
  const asArray = (input: unknown): ClashProxy[] => {
    if (!Array.isArray(input)) return []
    return input.filter((item): item is ClashProxy => !!item && typeof item === 'object')
  }

  if (Array.isArray(payload)) {
    return asArray(payload)
  }
  if (!payload || typeof payload !== 'object') {
    return null
  }

  const record = payload as Record<string, unknown>
  if (Array.isArray(record.proxies)) {
    return asArray(record.proxies)
  }
  if (Array.isArray(record.proxy)) {
    return asArray(record.proxy)
  }
  if (Array.isArray(record.Proxy)) {
    return asArray(record.Proxy)
  }
  return null
}

function safeDecodeURIComponent(value: string): string {
  try {
    return decodeURIComponent(value)
  } catch {
    return value
  }
}

function decodeBase64ImportText(raw: string): string | null {
  const candidate = raw.trim().replace(/\s+/g, '')
  if (!candidate || !/^[A-Za-z0-9+/_-]+={0,2}$/.test(candidate)) return null

  const variants = new Set<string>()
  variants.add(candidate)
  const mod = candidate.length % 4
  if (mod !== 0) {
    variants.add(candidate + '='.repeat(4 - mod))
  }

  for (const variant of variants) {
    const normalized = variant.replace(/-/g, '+').replace(/_/g, '/')
    try {
      const binary = atob(normalized)
      const bytes = Uint8Array.from(binary, ch => ch.charCodeAt(0))
      const decoded = new TextDecoder().decode(bytes).replace(/\r\n/g, '\n').trim()
      if (decoded) return decoded
    } catch {
      // try next variant
    }
  }
  return null
}

function collectClashImportTextAttempts(input: string): string[] {
  const attempts: string[] = []
  const seen = new Set<string>()
  const append = (value: string | null | undefined) => {
    const text = (value || '').replace(/\uFEFF/g, '').replace(/\r\n/g, '\n').trim()
    if (!text || seen.has(text)) return
    seen.add(text)
    attempts.push(text)
  }

  append(input)
  append(safeDecodeURIComponent(input))

  for (const text of [...attempts]) {
    append(decodeBase64ImportText(text))
  }
  return attempts
}

function stringValue(value: unknown): string {
  if (value === null || value === undefined) return ''
  return String(value).trim()
}

function numberValue(value: unknown): number {
  const n = Number(stringValue(value))
  if (!Number.isFinite(n)) return 0
  return Math.round(n)
}

function firstText(...values: Array<string | null | undefined>): string {
  for (const value of values) {
    const text = (value || '').trim()
    if (text) return text
  }
  return ''
}

function setStringField(target: ClashProxy, key: string, value: string | null | undefined) {
  const text = (value || '').trim()
  if (text) target[key] = text
}

function isTruthyParam(value: string | null): boolean {
  const text = (value || '').trim().toLowerCase()
  return text === '1' || text === 'true' || text === 'yes'
}

function shareURLName(url: URL, index: number): string {
  const raw = url.hash ? url.hash.replace(/^#/, '') : ''
  return safeDecodeURIComponent(raw).trim() || `导入代理 ${index + 1}`
}

function applyTransportParams(proxy: ClashProxy, networkValue: string, path = '', host = '', serviceName = '') {
  const network = networkValue.trim().toLowerCase()
  if (!network) return
  proxy.network = network

  if (network === 'ws') {
    const wsOpts: Record<string, unknown> = {}
    if (path.trim()) wsOpts.path = path.trim()
    if (host.trim()) wsOpts.headers = { Host: host.trim() }
    if (Object.keys(wsOpts).length > 0) proxy['ws-opts'] = wsOpts
  }

  if (network === 'grpc') {
    const grpcOpts: Record<string, unknown> = {}
    if (serviceName.trim()) grpcOpts['grpc-service-name'] = serviceName.trim()
    if (Object.keys(grpcOpts).length > 0) proxy['grpc-opts'] = grpcOpts
  }
}

function parseVmessShareURI(raw: string, index: number): ClashProxy | null {
  const decoded = decodeBase64ImportText(raw.slice('vmess://'.length))
  if (!decoded) return null

  let payload: Record<string, unknown>
  try {
    payload = JSON.parse(decoded) as Record<string, unknown>
  } catch {
    return null
  }

  const server = stringValue(payload.add)
  const port = numberValue(payload.port)
  const uuid = stringValue(payload.id)
  if (!server || !port || !uuid) return null

  const proxy: ClashProxy = {
    name: firstText(stringValue(payload.ps), `导入代理 ${index + 1}`),
    type: 'vmess',
    server,
    port,
    uuid,
    alterId: numberValue(payload.aid),
    cipher: firstText(stringValue(payload.scy), stringValue(payload.cipher), 'auto'),
  }

  const tls = stringValue(payload.tls).toLowerCase()
  if (tls && tls !== 'none') {
    proxy.tls = true
    setStringField(proxy, 'servername', firstText(stringValue(payload.sni), stringValue(payload.host)))
  }
  setStringField(proxy, 'client-fingerprint', stringValue(payload.fp))
  applyTransportParams(proxy, firstText(stringValue(payload.net), stringValue(payload.network)), stringValue(payload.path), stringValue(payload.host), stringValue(payload.serviceName))
  return proxy
}

function parseUserInfoShareURI(raw: string, index: number, protocol: 'vless' | 'trojan'): ClashProxy | null {
  let parsed: URL
  try {
    parsed = new URL(raw)
  } catch {
    return null
  }

  const server = parsed.hostname
  const port = Number(parsed.port)
  const user = safeDecodeURIComponent(parsed.username)
  if (!server || !Number.isFinite(port) || port <= 0 || !user) return null

  const q = parsed.searchParams
  const proxy: ClashProxy = {
    name: shareURLName(parsed, index),
    type: protocol,
    server,
    port,
  }
  if (protocol === 'vless') {
    proxy.uuid = user
    setStringField(proxy, 'flow', q.get('flow'))
  } else {
    proxy.password = user
  }

  const security = (q.get('security') || '').toLowerCase()
  if (security === 'tls' || security === 'reality') {
    proxy.tls = true
    setStringField(proxy, 'servername', firstText(q.get('sni'), q.get('peer'), q.get('servername')))
    if (security === 'reality') {
      const realityOpts: Record<string, string> = {}
      if (q.get('pbk')) realityOpts['public-key'] = q.get('pbk') || ''
      if (q.get('sid')) realityOpts['short-id'] = q.get('sid') || ''
      if (Object.keys(realityOpts).length > 0) proxy['reality-opts'] = realityOpts
    }
  }
  setStringField(proxy, 'client-fingerprint', q.get('fp'))
  if (isTruthyParam(q.get('allowInsecure')) || isTruthyParam(q.get('skip-cert-verify'))) {
    proxy['skip-cert-verify'] = true
  }

  applyTransportParams(proxy, firstText(q.get('type'), q.get('network')), q.get('path') || '', q.get('host') || '', q.get('serviceName') || '')
  return proxy
}

function parseSSHostPort(hostPart: string): { server: string; port: number } | null {
  try {
    const parsed = new URL(`ss://${hostPart}`)
    const port = Number(parsed.port)
    if (!parsed.hostname || !Number.isFinite(port) || port <= 0) return null
    return { server: parsed.hostname, port }
  } catch {
    return null
  }
}

function parseSSShareURI(raw: string, index: number): ClashProxy | null {
  let body = raw.slice('ss://'.length).trim()
  let fragment = ''
  const hashIndex = body.indexOf('#')
  if (hashIndex >= 0) {
    fragment = body.slice(hashIndex + 1)
    body = body.slice(0, hashIndex)
  }
  const queryIndex = body.indexOf('?')
  if (queryIndex >= 0) {
    body = body.slice(0, queryIndex)
  }

  let method = ''
  let password = ''
  let endpoint: { server: string; port: number } | null = null

  if (body.includes('@')) {
    const at = body.lastIndexOf('@')
    let userPart = body.slice(0, at)
    const hostPart = body.slice(at + 1)
    userPart = decodeBase64ImportText(userPart) || safeDecodeURIComponent(userPart)
    const parts = userPart.split(/:(.*)/s).filter(Boolean)
    if (parts.length < 2) return null
    method = parts[0].trim()
    password = parts[1].trim()
    endpoint = parseSSHostPort(hostPart)
  } else {
    const decoded = decodeBase64ImportText(body)
    if (!decoded) return null
    const at = decoded.lastIndexOf('@')
    if (at < 0) return null
    const userPart = decoded.slice(0, at)
    const hostPart = decoded.slice(at + 1)
    const parts = userPart.split(/:(.*)/s).filter(Boolean)
    if (parts.length < 2) return null
    method = parts[0].trim()
    password = parts[1].trim()
    endpoint = parseSSHostPort(hostPart)
  }

  if (!method || !password || !endpoint) return null
  return {
    name: safeDecodeURIComponent(fragment).trim() || `导入代理 ${index + 1}`,
    type: 'ss',
    server: endpoint.server,
    port: endpoint.port,
    cipher: method,
    password,
  }
}

function parseShareURIToClashProxy(raw: string, index: number): ClashProxy | null {
  const value = raw.trim()
  const lower = value.toLowerCase()
  if (lower.startsWith('vmess://')) return parseVmessShareURI(value, index)
  if (lower.startsWith('vless://')) return parseUserInfoShareURI(value, index, 'vless')
  if (lower.startsWith('trojan://')) return parseUserInfoShareURI(value, index, 'trojan')
  if (lower.startsWith('ss://')) return parseSSShareURI(value, index)
  return null
}

function parseShareSubscriptionText(raw: string): ClashProxy[] | null {
  const proxies: ClashProxy[] = []
  raw.replace(/\r\n/g, '\n').split('\n').forEach(line => {
    const value = line.trim()
    if (!value || value.startsWith('#')) return
    const proxy = parseShareURIToClashProxy(value, proxies.length)
    if (proxy) proxies.push(proxy)
  })
  return proxies.length > 0 ? proxies : null
}

function normalizeLooseClashImportText(raw: string): string {
  const normalizedNewline = raw.replace(/\uFEFF/g, '').replace(/\r\n/g, '\n').trim()
  if (!normalizedNewline) return normalizedNewline

  const lines = normalizedNewline.split('\n')
  const fixedLines = lines.map(line => {
    // 容错: -节点名, type: vless, server: ... => - { name: '节点名', type: vless, server: ... }
    const m = line.match(/^(\s*)-\s*([^,{][^,]*?)\s*,\s*(type\s*:.*)$/i)
    if (!m) return line
    const indent = m[1] || ''
    const name = m[2] || ''
    const tail = m[3] || ''
    return `${indent}- { name: ${quoteYamlScalar(name)}, ${tail.trim()} }`
  })

  const hasProxiesRoot = fixedLines.some(line => /^\s*proxies\s*:/.test(line))
  if (hasProxiesRoot) {
    return fixedLines.join('\n')
  }

  const looksLikeProxyList = fixedLines.some(line => /^\s*-\s*/.test(line))
  if (!looksLikeProxyList) {
    return fixedLines.join('\n')
  }

  const indented = fixedLines.map(line => {
    if (!line.trim()) return line
    return `  ${line}`
  })
  return `proxies:\n${indented.join('\n')}`
}

function parseClashImportText(raw: string): ClashProxy[] {
  const input = raw.trim()
  if (!input) {
    throw new Error('请输入 YAML 或订阅内容')
  }

  let lastError: unknown = null
  for (const sourceText of collectClashImportTextAttempts(input)) {
    const attempts = [sourceText]
    const normalized = normalizeLooseClashImportText(sourceText)
    if (normalized && normalized !== sourceText) {
      attempts.push(normalized)
    }

    const shareProxies = parseShareSubscriptionText(sourceText)
    if (shareProxies) {
      return shareProxies
    }

    for (const text of attempts) {
      try {
        const parsed = yaml.load(text)
        const proxies = normalizeImportedProxyArray(parsed)
        if (proxies) {
          return proxies
        }
      } catch (error) {
        lastError = error
      }
    }
  }

  if (lastError && typeof lastError === 'object' && lastError !== null && 'message' in lastError) {
    throw new Error(String((lastError as { message?: string }).message || '解析失败'))
  }
  throw new Error('无效的订阅格式，需要包含 proxies 数组，或 Base64/分享链接节点')
}

function normalizeDirectProxyConfig(raw: string): string {
  const trimmed = raw.trim()
  if (!trimmed) return ''
  if (/^socket:\/\//i.test(trimmed)) {
    return trimmed.replace(/^socket:\/\//i, 'socks5://')
  }
  if (/^socks:\/\//i.test(trimmed)) {
    return trimmed.replace(/^socks:\/\//i, 'socks5://')
  }
  return trimmed
}

function buildDirectProxyConfigFromParts(protocol: DirectImportForm['protocol'], host: string, portText: string, username = '', password = ''): string {
  const cleanHost = host.trim()
  const cleanPort = portText.trim()
  if (!cleanHost || !cleanPort) {
    throw new Error('缺少主机或端口')
  }
  if (!/^\d+$/.test(cleanPort)) {
    throw new Error('端口必须为数字')
  }
  const port = Number(cleanPort)
  if (port < 1 || port > 65535) {
    throw new Error('端口必须在 1-65535 之间')
  }

  const auth = username.trim()
    ? `${encodeURIComponent(username.trim())}${password ? `:${encodeURIComponent(password)}` : ''}@`
    : ''
  return `${protocol}://${auth}${formatDirectProxyHost(cleanHost)}:${port}`
}

function parseDirectProxyLine(line: string, index: number, defaultProtocol: DirectImportForm['protocol']): DirectImportLineResult {
  const raw = line.trim()
  if (!raw) {
    throw new Error('代理内容为空')
  }

  const parts = raw.split(/\s+/)
  const first = normalizeDirectProxyConfig(parts[0] || '')
  const explicitName = parts.slice(1).join(' ').trim()
  let candidate = first
  if (!/^[a-zA-Z][a-zA-Z0-9+.-]*:\/\//.test(candidate)) {
    const colonParts = first.split(':')
    if (colonParts.length >= 2 && colonParts.length <= 4) {
      const [host, port, username = '', password = ''] = colonParts
      try {
        candidate = buildDirectProxyConfigFromParts(defaultProtocol, host, port, username, password)
      } catch (error: any) {
        throw new Error(`第 ${index + 1} 行${error?.message ? `：${error.message}` : '格式无效'}`)
      }
    } else {
      candidate = `${defaultProtocol}://${first}`
    }
  }
  let parsedURL: URL
  try {
    parsedURL = new URL(candidate)
  } catch {
    throw new Error(`第 ${index + 1} 行不是有效代理地址`)
  }

  const scheme = parsedURL.protocol.replace(':', '').toLowerCase()
  if (!['http', 'https', 'socks5'].includes(scheme)) {
    throw new Error(`第 ${index + 1} 行协议不支持，仅支持 HTTP / HTTPS / SOCKS5`)
  }
  const port = Number(parsedURL.port || 0)
  if (!parsedURL.hostname || !port || port < 1 || port > 65535) {
    throw new Error(`第 ${index + 1} 行缺少有效主机或端口`)
  }

  const normalizedConfig = parsedURL.toString().replace(/\/$/, '')
  const host = parsedURL.hostname.replace(/^\[(.*)\]$/, '$1')
  return {
    name: explicitName || resolveDirectProxyName('', scheme, host, port, index, ''),
    config: normalizedConfig,
  }
}

function parseDirectProxyBatchText(raw: string, defaultProtocol: DirectImportForm['protocol']): ImportCandidate[] {
  const lines = raw
    .split(/\r?\n/)
    .map(line => line.trim())
    .filter(line => line && !line.startsWith('#'))

  if (lines.length === 0) return []

  return lines.map((line, index) => {
    const result = parseDirectProxyLine(line, index, defaultProtocol)
    return {
      proxyName: result.name,
      proxyConfig: result.config,
    }
  })
}

function resolveDirectProxyName(rawName: string, scheme: string, server: string, port: number, index: number, prefix: string): string {
  const name = rawName.trim()
  const fallbackName = server
    ? `${scheme.toUpperCase()}-${server}${port > 0 ? `:${port}` : ''}`
    : `导入代理 ${index + 1}`
  const finalName = name || fallbackName
  return prefix ? `${prefix}-${finalName}` : finalName
}

function formatDirectProxyHost(raw: string): string {
  const host = raw.trim()
  if (!host) return ''
  if (host.startsWith('[') && host.endsWith(']')) {
    return host
  }
  return host.includes(':') ? `[${host}]` : host
}

function buildDirectImportCandidate(form: DirectImportForm): ImportCandidate {
  const serverInput = form.server.trim()
  if (!serverInput) {
    throw new Error('请输入代理地址')
  }
  if (/^[a-zA-Z][a-zA-Z0-9+.-]*:\/\//.test(serverInput)) {
    throw new Error('代理地址只需要填写主机名或 IP，不需要协议头')
  }

  const portInput = form.port.trim()
  if (!portInput) {
    throw new Error('请输入代理端口')
  }
  if (!/^\d+$/.test(portInput)) {
    throw new Error('代理端口必须为数字')
  }

  const port = Number(portInput)
  if (port < 1 || port > 65535) {
    throw new Error('代理端口必须在 1-65535 之间')
  }

  const username = form.username.trim()
  const password = form.password
  if (password && !username) {
    throw new Error('填写密码时请同时填写账号')
  }

  const auth = username
    ? `${encodeURIComponent(username)}${password ? `:${encodeURIComponent(password)}` : ''}@`
    : ''
  const rawConfig = `${form.protocol}://${auth}${formatDirectProxyHost(serverInput)}:${port}`

  let parsedURL: URL
  try {
    parsedURL = new URL(rawConfig)
  } catch {
    throw new Error('请输入有效的代理地址')
  }

  if (!parsedURL.hostname) {
    throw new Error('请输入有效的代理地址')
  }

  const normalizedConfig = normalizeDirectProxyConfig(parsedURL.toString()).replace(/\/$/, '')
  const normalizedServer = parsedURL.hostname.replace(/^\[(.*)\]$/, '$1')

  return {
    proxyName: resolveDirectProxyName(form.proxyName, form.protocol, normalizedServer, port, 0, ''),
    proxyConfig: normalizedConfig,
  }
}

function buildImportCandidatesFromClash(parsedProxies: ClashProxy[], prefix: string): ImportCandidate[] {
  return parsedProxies.map((proxy, index) => ({
    proxyName: resolveImportedProxyName(proxy, index, prefix),
    proxyConfig: proxyToYaml(proxy),
  }))
}

function buildImportPreview(candidates: ImportCandidate[], groupName: string): ProxyDisplayInfo[] {
  return candidates.map((candidate, index) => {
    const info = parseProxyInfo(candidate.proxyConfig)
    return {
      proxyId: `preview-${index}`,
      proxyName: candidate.proxyName,
      proxyConfig: candidate.proxyConfig,
      groupName,
      sourceId: '',
      sourceUrl: '',
      sourceAutoRefresh: false,
      sourceRefreshIntervalM: 0,
      sourceLastRefreshAt: '',
      type: info.type || '-',
      server: info.server || '-',
      port: info.port || 0,
    }
  })
}

function parseTimestampMs(value: string): number {
  const v = (value || '').trim()
  if (!v) return 0
  const t = Date.parse(v)
  return Number.isFinite(t) ? t : 0
}

function normalizeRefreshIntervalM(value: number): number {
  if (!Number.isFinite(value)) return 0
  if (value <= 0) return 0
  if (value < 5) return 5
  if (value > 24 * 60) return 24 * 60
  return Math.round(value)
}

function sourceHostLabel(sourceURL: string): string {
  const raw = (sourceURL || '').trim()
  if (!raw) return ''
  try {
    const u = new URL(raw)
    return u.host || raw
  } catch {
    return raw
  }
}

function normalizeSourceURL(sourceURL: string): string {
  const raw = (sourceURL || '').trim()
  if (!raw) return ''
  try {
    const parsed = new URL(raw)
    parsed.hash = ''
    return parsed.toString()
  } catch {
    return raw
  }
}

function buildStableSourceID(sourceURL: string, sourceNamePrefix: string): string {
  const key = `${normalizeSourceURL(sourceURL)}|||${sourceNamePrefix.trim()}`
  // djb2 变体，输出稳定且实现简单。
  let hash = 5381
  for (let i = 0; i < key.length; i += 1) {
    hash = ((hash << 5) + hash) ^ key.charCodeAt(i)
  }
  const unsigned = hash >>> 0
  return `src-${unsigned.toString(36)}`
}

function resolveImportSourceID(list: BrowserProxy[], sourceURL: string, sourceNamePrefix: string): string {
  const normalizedURL = normalizeSourceURL(sourceURL)
  const normalizedPrefix = sourceNamePrefix.trim()
  const existing = list.find(item =>
    normalizeSourceURL(item.sourceUrl || '') === normalizedURL &&
    (item.sourceNamePrefix || '').trim() === normalizedPrefix &&
    (item.sourceId || '').trim() !== ''
  )
  if (existing?.sourceId?.trim()) {
    return existing.sourceId.trim()
  }
  return buildStableSourceID(sourceURL, sourceNamePrefix)
}

function collectURLImportSources(list: BrowserProxy[]): URLImportSourceMeta[] {
  const sourceMap = new Map<string, URLImportSourceMeta>()
  for (const item of list) {
    const sourceId = (item.sourceId || '').trim()
    const sourceUrl = (item.sourceUrl || '').trim()
    if (!sourceId || !sourceUrl) continue

    const last = sourceMap.get(sourceId)
    const currentLastRefreshAt = item.sourceLastRefreshAt || ''
    if (!last) {
      sourceMap.set(sourceId, {
        sourceId,
        sourceUrl,
        sourceNamePrefix: (item.sourceNamePrefix || '').trim(),
        sourceGroupName: (item.groupName || '').trim(),
        sourceDnsServers: (item.dnsServers || '').trim(),
        sourceAutoRefresh: !!item.sourceAutoRefresh,
        sourceRefreshIntervalM: normalizeRefreshIntervalM(Number(item.sourceRefreshIntervalM || 0)),
        sourceLastRefreshAt: currentLastRefreshAt,
        proxyCount: 1,
      })
      continue
    }

    last.proxyCount += 1
    if (
      parseTimestampMs(currentLastRefreshAt) > parseTimestampMs(last.sourceLastRefreshAt) &&
      currentLastRefreshAt.trim()
    ) {
      last.sourceLastRefreshAt = currentLastRefreshAt
    }
  }
  return Array.from(sourceMap.values())
}

function nextProxyID(): string {
  return `proxy-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
}

function resolveImportedProxyName(proxy: ClashProxy, index: number, prefix: string): string {
  const rawName = (proxy.name || '').trim() || `导入代理 ${index + 1}`
  return prefix ? `${prefix}-${rawName}` : rawName
}

function createExistingProxyIDPicker(oldSourceProxies: BrowserProxy[]) {
  const exactMap = new Map<string, BrowserProxy[]>()
  const nameMap = new Map<string, BrowserProxy[]>()
  oldSourceProxies.forEach(item => {
    const exactKey = `${item.proxyName}|||${item.proxyConfig}`
    const exactList = exactMap.get(exactKey) || []
    exactList.push(item)
    exactMap.set(exactKey, exactList)

    const nameKey = item.proxyName
    const nameList = nameMap.get(nameKey) || []
    nameList.push(item)
    nameMap.set(nameKey, nameList)
  })

  return (name: string, configText: string): string | null => {
    const exactKey = `${name}|||${configText}`
    const exactList = exactMap.get(exactKey)
    if (exactList && exactList.length > 0) {
      const item = exactList.shift()
      if (item?.proxyId) return item.proxyId
    }

    const nameList = nameMap.get(name)
    if (nameList && nameList.length > 0) {
      const item = nameList.shift()
      if (item?.proxyId) return item.proxyId
    }
    return null
  }
}

function buildRefreshedSourceProxies(
  parsedProxies: ClashProxy[],
  oldSourceProxies: BrowserProxy[],
  meta: URLImportSourceMeta,
  refreshedAt: string
): BrowserProxy[] {
  const pickExisting = createExistingProxyIDPicker(oldSourceProxies)

  const prefix = meta.sourceNamePrefix.trim()
  const sourceGroupName = meta.sourceGroupName.trim()
  const sourceDnsServers = meta.sourceDnsServers.trim()
  const refreshed: BrowserProxy[] = []

  parsedProxies.forEach((proxy, idx) => {
    const proxyName = resolveImportedProxyName(proxy, idx, prefix)
    const proxyConfig = proxyToYaml(proxy)
    const proxyId = pickExisting(proxyName, proxyConfig) || nextProxyID()

    refreshed.push({
      proxyId,
      proxyName,
      proxyConfig,
      dnsServers: sourceDnsServers || undefined,
      groupName: sourceGroupName || undefined,
      sourceId: meta.sourceId,
      sourceUrl: meta.sourceUrl,
      sourceNamePrefix: prefix || undefined,
      sourceAutoRefresh: meta.sourceAutoRefresh,
      sourceRefreshIntervalM: meta.sourceRefreshIntervalM,
      sourceLastRefreshAt: refreshedAt,
    })
  })

  return refreshed
}

function renameSourceProxyName(proxyName: string, oldPrefix: string, newPrefix: string): string {
  const currentName = proxyName.trim()
  const old = oldPrefix.trim()
  const next = newPrefix.trim()
  const baseName = old && currentName.startsWith(`${old}-`)
    ? currentName.slice(old.length + 1)
    : currentName
  return next ? `${next}-${baseName}` : baseName
}

function readSourceIgnoredProxyNames(): Record<string, string[]> {
  try {
    const raw = localStorage.getItem(PROXY_SOURCE_IGNORED_NAMES_KEY)
    if (!raw) return {}
    const parsed = JSON.parse(raw)
    if (!parsed || typeof parsed !== 'object') return {}
    const cleaned: Record<string, string[]> = {}
    Object.entries(parsed as Record<string, unknown>).forEach(([sourceId, value]) => {
      if (!sourceId.trim() || !Array.isArray(value)) return
      const names = value
        .map(item => (typeof item === 'string' ? item.trim() : ''))
        .filter(Boolean)
      if (names.length > 0) {
        cleaned[sourceId] = names
      }
    })
    return cleaned
  } catch {
    return {}
  }
}

function writeSourceIgnoredProxyNames(data: Record<string, string[]>) {
  try {
    const cleaned: Record<string, string[]> = {}
    Object.entries(data).forEach(([sourceId, names]) => {
      const key = sourceId.trim()
      if (!key || !Array.isArray(names)) return
      const validNames = names.map(name => (name || '').trim()).filter(Boolean)
      if (validNames.length > 0) {
        cleaned[key] = validNames
      }
    })
    localStorage.setItem(PROXY_SOURCE_IGNORED_NAMES_KEY, JSON.stringify(cleaned))
  } catch {
    // ignore write failures
  }
}

function appendSourceIgnoredProxyNames(sourceId: string, names: string[]) {
  const sourceKey = sourceId.trim()
  if (!sourceKey || names.length === 0) return
  const cleaned = names.map(name => name.trim()).filter(Boolean)
  if (cleaned.length === 0) return

  const existing = readSourceIgnoredProxyNames()
  existing[sourceKey] = [...(existing[sourceKey] || []), ...cleaned]
  writeSourceIgnoredProxyNames(existing)
}

function applyIgnoredProxyNamesForSource(
  parsedProxies: ClashProxy[],
  sourceNamePrefix: string,
  ignoredProxyNames: string[]
): ClashProxy[] {
  if (ignoredProxyNames.length === 0) return parsedProxies
  const ignoredCounter = new Map<string, number>()
  ignoredProxyNames.forEach(name => {
    const key = name.trim()
    if (!key) return
    ignoredCounter.set(key, (ignoredCounter.get(key) || 0) + 1)
  })
  if (ignoredCounter.size === 0) return parsedProxies

  return parsedProxies.filter((proxy, idx) => {
    const proxyName = resolveImportedProxyName(proxy, idx, sourceNamePrefix)
    const count = ignoredCounter.get(proxyName) || 0
    if (count <= 0) return true
    if (count === 1) {
      ignoredCounter.delete(proxyName)
    } else {
      ignoredCounter.set(proxyName, count - 1)
    }
    return false
  })
}

function readGlobalRefreshConfig(): { enabled: boolean; intervalM: number } {
  try {
    const rawEnabled = localStorage.getItem(PROXY_GLOBAL_AUTO_REFRESH_KEY)
    const rawInterval = localStorage.getItem(PROXY_GLOBAL_REFRESH_INTERVAL_KEY)
    const enabled = rawEnabled === '1'
    const interval = normalizeRefreshIntervalM(Number(rawInterval || 0))
    return {
      enabled,
      intervalM: interval > 0 ? interval : 60,
    }
  } catch {
    return { enabled: false, intervalM: 60 }
  }
}

function writeGlobalRefreshConfig(enabled: boolean, intervalM: number) {
  try {
    localStorage.setItem(PROXY_GLOBAL_AUTO_REFRESH_KEY, enabled ? '1' : '0')
    localStorage.setItem(PROXY_GLOBAL_REFRESH_INTERVAL_KEY, String(intervalM))
  } catch {
    // ignore write failures
  }
}

function toLatencyValue(ok: boolean, latencyMs: number, error?: string): number {
  if (ok) return latencyMs
  return error?.includes('不支持') ? -3 : -2
}

function readLatencyCache(): Record<string, number> {
  try {
    const raw = localStorage.getItem(PROXY_LATENCY_CACHE_KEY)
    if (!raw) return {}
    const parsed = JSON.parse(raw) as { timestamp?: number; data?: Record<string, number> }
    if (!parsed?.timestamp || !parsed?.data) return {}
    if (Date.now() - parsed.timestamp > PROXY_LATENCY_CACHE_TTL_MS) return {}
    const cleaned: Record<string, number> = {}
    Object.entries(parsed.data).forEach(([proxyId, latency]) => {
      if (typeof latency === 'number' && Number.isFinite(latency) && latency !== -1) {
        cleaned[proxyId] = latency
      }
    })
    return cleaned
  } catch {
    return {}
  }
}

function writeLatencyCache(data: Record<string, number>) {
  try {
    const cleaned: Record<string, number> = {}
    Object.entries(data).forEach(([proxyId, latency]) => {
      if (typeof latency === 'number' && Number.isFinite(latency) && latency !== -1) {
        cleaned[proxyId] = latency
      }
    })
    localStorage.setItem(PROXY_LATENCY_CACHE_KEY, JSON.stringify({
      timestamp: Date.now(),
      data: cleaned,
    }))
  } catch {
    // ignore write failures
  }
}

function readIPHealthCache(): Record<string, ProxyIPHealthResult> {
  try {
    const raw = localStorage.getItem(PROXY_IP_HEALTH_CACHE_KEY)
    if (!raw) return {}
    const parsed = JSON.parse(raw) as { timestamp?: number; data?: Record<string, ProxyIPHealthResult> }
    if (!parsed?.timestamp || !parsed?.data) return {}
    if (Date.now() - parsed.timestamp > PROXY_IP_HEALTH_CACHE_TTL_MS) return {}
    const cleaned: Record<string, ProxyIPHealthResult> = {}
    Object.entries(parsed.data).forEach(([proxyId, item]) => {
      if (item && typeof item === 'object') {
        cleaned[proxyId] = item
      }
    })
    return cleaned
  } catch {
    return {}
  }
}

function writeIPHealthCache(data: Record<string, ProxyIPHealthResult>) {
  try {
    localStorage.setItem(PROXY_IP_HEALTH_CACHE_KEY, JSON.stringify({
      timestamp: Date.now(),
      data,
    }))
  } catch {
    // ignore write failures
  }
}

function normalizePreviewSearchText(value: unknown): string {
  return String(value || '').trim().toLowerCase()
}

function previewLatencyMatchesFilter(latency: number | undefined, filter: PreviewLatencyFilter): boolean {
  if (filter === 'all') return true
  if (filter === 'untested') return latency === undefined
  if (filter === 'testing') return latency === -1
  if (filter === 'timeout') return latency === -2
  if (filter === 'unsupported') return latency === -3
  if (filter === 'ok') return typeof latency === 'number' && latency >= 0
  if (filter === 'fast') return typeof latency === 'number' && latency >= 0 && latency < 200
  if (filter === 'slow') return typeof latency === 'number' && latency >= 500
  return true
}

function previewHealthMatchesFilter(result: ProxyIPHealthResult | undefined, checking: boolean, filter: PreviewHealthFilter): boolean {
  if (filter === 'all') return true
  if (filter === 'untested') return !checking && !result
  if (filter === 'ok') return !!result?.ok
  if (filter === 'failed') return !!result && !result.ok
  if (filter === 'highRisk') return !!result?.ok && (result.fraudScore >= 70 || result.isBroadcast)
  if (filter === 'residential') return !!result?.ok && result.isResidential
  if (filter === 'datacenter') return !!result?.ok && !result.isResidential
  return true
}

export function ProxyPoolPage() {
  const [proxies, setProxies] = useState<BrowserProxy[]>([])
  const [displayList, setDisplayList] = useState<ProxyDisplayInfo[]>([])
  const [loading, setLoading] = useState(true)
  const [groups, setGroups] = useState<string[]>([])

  const [filterProtocol, setFilterProtocol] = useState<string>('all')
  const [filterKeyword, setFilterKeyword] = useState('')
  const [filterGroup, setFilterGroup] = useState<string>('all')
  const [visibleColumnKeys, setVisibleColumnKeys] = useState<string[]>(readStoredProxyColumnKeys)
  const [resourceView, setResourceView] = useState<ProxyResourceView>('proxies')
  const [sortColumn, setSortColumn] = useState<string>('') // 默认不排序
  const [sortOrder, setSortOrder] = useState<SortOrder>(undefined)

  const [latencyMap, setLatencyMap] = useState<Record<string, number>>({})
  const [testingAll, setTestingAll] = useState(false)
  const [ipHealthMap, setIPHealthMap] = useState<Record<string, ProxyIPHealthResult>>({})
  const [checkingIPHealthIds, setCheckingIPHealthIds] = useState<Set<string>>(new Set())
  const [checkingAllIPHealth, setCheckingAllIPHealth] = useState(false)

  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set())
  const [batchDeleteConfirmOpen, setBatchDeleteConfirmOpen] = useState(false)
  const [deleteTimeoutConfirmOpen, setDeleteTimeoutConfirmOpen] = useState(false)

  const [importModalOpen, setImportModalOpen] = useState(false)
  const [importMode, setImportMode] = useState<ProxyImportMode>('clash')
  const [importUrl, setImportUrl] = useState('')
  const [importResolvedUrl, setImportResolvedUrl] = useState('')
  const [importText, setImportText] = useState('')
  const [importDnsServers, setImportDnsServers] = useState('')
  const [importNamePrefix, setImportNamePrefix] = useState('')
  const [importGroupName, setImportGroupName] = useState('')
  const [directImportForm, setDirectImportForm] = useState<DirectImportForm>(() => ({ ...INITIAL_DIRECT_IMPORT_FORM }))
  const [directImportText, setDirectImportText] = useState('')
  const [previewModalOpen, setPreviewModalOpen] = useState(false)
  const [previewList, setPreviewList] = useState<ProxyDisplayInfo[]>([])
  const [previewSelectedIds, setPreviewSelectedIds] = useState<Set<string>>(new Set())
  const [previewKeyword, setPreviewKeyword] = useState('')
  const [previewLatencyFilter, setPreviewLatencyFilter] = useState<PreviewLatencyFilter>('all')
  const [previewHealthFilter, setPreviewHealthFilter] = useState<PreviewHealthFilter>('all')
  const [previewCountryFilter, setPreviewCountryFilter] = useState('all')
  const [previewLatencyMap, setPreviewLatencyMap] = useState<Record<string, number>>({})
  const [previewIPHealthMap, setPreviewIPHealthMap] = useState<Record<string, ProxyIPHealthResult>>({})
  const [previewCheckingIPHealthIds, setPreviewCheckingIPHealthIds] = useState<Set<string>>(new Set())
  const [previewTestingAll, setPreviewTestingAll] = useState(false)
  const [previewCheckingAllIPHealth, setPreviewCheckingAllIPHealth] = useState(false)
  const [removedPreviewProxyNames, setRemovedPreviewProxyNames] = useState<string[]>([])
  const [importing, setImporting] = useState(false)
  const [fetchingImportUrl, setFetchingImportUrl] = useState(false)
  const [refreshingAllSources, setRefreshingAllSources] = useState(false)
  const [refreshingSourceIds, setRefreshingSourceIds] = useState<Set<string>>(new Set())
  const [globalAutoRefreshEnabled, setGlobalAutoRefreshEnabled] = useState(false)
  const [globalRefreshIntervalM, setGlobalRefreshIntervalM] = useState('60')
  const [sourceEditModalOpen, setSourceEditModalOpen] = useState(false)
  const [editingSource, setEditingSource] = useState<URLImportSourceMeta | null>(null)
  const [sourceEditForm, setSourceEditForm] = useState({ sourceUrl: '', groupName: '', namePrefix: '', dnsServers: '' })
  const [sourceDeleteConfirmOpen, setSourceDeleteConfirmOpen] = useState(false)
  const [deletingSource, setDeletingSource] = useState<URLImportSourceMeta | null>(null)

  const [editModalOpen, setEditModalOpen] = useState(false)
  const [editingProxy, setEditingProxy] = useState<BrowserProxy | null>(null)
  const [editForm, setEditForm] = useState({ proxyName: '', proxyConfig: '', dnsServers: '', groupName: '' })
  const [saving, setSaving] = useState(false)

  const [deleteConfirmOpen, setDeleteConfirmOpen] = useState(false)
  const [deletingId, setDeletingId] = useState<string | null>(null)
  const [ipHealthDetailOpen, setIPHealthDetailOpen] = useState(false)
  const [currentIPHealthDetail, setCurrentIPHealthDetail] = useState<ProxyIPHealthResult | null>(null)
  const proxiesRef = useRef<BrowserProxy[]>([])
  const refreshingSourceIdsRef = useRef<Set<string>>(new Set())
  const autoRefreshRunningRef = useRef(false)
  const globalRefreshInterval = useMemo(() => {
    const interval = normalizeRefreshIntervalM(Number(globalRefreshIntervalM || 0))
    return interval > 0 ? interval : 60
  }, [globalRefreshIntervalM])

  useEffect(() => {
    const cfg = readGlobalRefreshConfig()
    setGlobalAutoRefreshEnabled(cfg.enabled)
    setGlobalRefreshIntervalM(String(cfg.intervalM))
    setLatencyMap(readLatencyCache())
    setIPHealthMap(readIPHealthCache())
    loadProxies()
  }, [])

  useEffect(() => {
    writeLatencyCache(latencyMap)
  }, [latencyMap])

  useEffect(() => {
    writeIPHealthCache(ipHealthMap)
  }, [ipHealthMap])

  useEffect(() => {
    const lockedKeys = PROXY_COLUMN_OPTIONS.filter(item => item.locked).map(item => item.key)
    localStorage.setItem(PROXY_COLUMNS_STORAGE_KEY, JSON.stringify(Array.from(new Set([...lockedKeys, ...visibleColumnKeys]))))
  }, [visibleColumnKeys])

  useEffect(() => {
    writeGlobalRefreshConfig(globalAutoRefreshEnabled, globalRefreshInterval)
  }, [globalAutoRefreshEnabled, globalRefreshInterval])

  useEffect(() => {
    proxiesRef.current = proxies
  }, [proxies])

  useEffect(() => {
    refreshingSourceIdsRef.current = refreshingSourceIds
  }, [refreshingSourceIds])

  useEffect(() => {
    if (!proxies.length) return
    const validIds = new Set(proxies.map(p => p.proxyId))
    setLatencyMap(prev => {
      let changed = false
      const next: Record<string, number> = {}
      Object.entries(prev).forEach(([proxyId, latency]) => {
        if (validIds.has(proxyId)) {
          next[proxyId] = latency
        } else {
          changed = true
        }
      })
      return changed ? next : prev
    })

    setIPHealthMap(prev => {
      let changed = false
      const next: Record<string, ProxyIPHealthResult> = {}
      Object.entries(prev).forEach(([proxyId, health]) => {
        if (validIds.has(proxyId)) {
          next[proxyId] = health
        } else {
          changed = true
        }
      })
      return changed ? next : prev
    })
  }, [proxies])

  const loadProxies = async () => {
    setLoading(true)
    try {
      const raw = await fetchBrowserProxies()
      const proxyList = ensureBuiltinProxies(raw)
      const persistedLatency: Record<string, number> = {}
      const persistedIPHealth: Record<string, ProxyIPHealthResult> = {}
      proxyList.forEach(proxy => {
        if (proxy.lastTestedAt) {
          persistedLatency[proxy.proxyId] = (proxy.lastTestOk ?? false)
            ? (proxy.lastLatencyMs ?? -2)
            : -2
        }
        if (proxy.lastIPHealthJson) {
          try {
            const parsed = JSON.parse(proxy.lastIPHealthJson) as ProxyIPHealthResult
            if (parsed && typeof parsed === 'object' && parsed.proxyId) {
              persistedIPHealth[proxy.proxyId] = parsed
            }
          } catch {
            // ignore bad historical json
          }
        }
      })

      setProxies(proxyList)
      setDisplayList(toDisplayList(proxyList))
      setLatencyMap(prev => ({ ...persistedLatency, ...prev }))
      setIPHealthMap(prev => ({ ...persistedIPHealth, ...prev }))
      const grps = await fetchBrowserProxyGroups()
      setGroups(grps)
    } finally {
      setLoading(false)
    }
  }

  // 直接保存完整列表，内置代理保护由后端负责
  const saveProxies = useCallback(async (list: BrowserProxy[]) => {
    await saveBrowserProxies(list)
    setProxies(list)
    setDisplayList(toDisplayList(list))
    // 刷新分组列表（可能有新分组加入）
    const grps = await fetchBrowserProxyGroups()
    setGroups(grps)
  }, [])

  const sourceMetas = useMemo(() => collectURLImportSources(proxies), [proxies])
  const hasURLImportSources = sourceMetas.length > 0

  const refreshSingleSource = useCallback(async (sourceId: string, silent: boolean) => {
    const currentList = proxiesRef.current
    const metas = collectURLImportSources(currentList)
    const meta = metas.find(item => item.sourceId === sourceId)
    if (!meta) return false

    if (refreshingSourceIdsRef.current.has(sourceId)) return false
    setRefreshingSourceIds(prev => {
      const next = new Set(prev)
      next.add(sourceId)
      return next
    })

    try {
      const result = await fetchClashImportFromURL(meta.sourceUrl)
      const parsed = parseClashImportText(result.content || '')
      if (!parsed.length) {
        throw new Error('订阅内容未解析到可用代理')
      }
      const ignoredNameMap = readSourceIgnoredProxyNames()
      const sourceIgnoredNames = ignoredNameMap[sourceId] || []
      const filteredParsed = applyIgnoredProxyNamesForSource(parsed, meta.sourceNamePrefix, sourceIgnoredNames)

      const latest = proxiesRef.current
      const oldSourceProxies = latest.filter(item => (item.sourceId || '').trim() === sourceId)
      const refreshedAt = new Date().toISOString()
      const effectiveMeta: URLImportSourceMeta = {
        ...meta,
        sourceAutoRefresh: globalAutoRefreshEnabled,
        sourceRefreshIntervalM: globalRefreshInterval,
        proxyCount: meta.proxyCount,
      }
      const refreshedSourceProxies = buildRefreshedSourceProxies(filteredParsed, oldSourceProxies, effectiveMeta, refreshedAt)

      const merged = latest
        .filter(item => (item.sourceId || '').trim() !== sourceId)
        .concat(refreshedSourceProxies)

      await saveProxies(merged)
      if (!silent) {
        toast.success(`订阅刷新成功：${meta.sourceUrl}（${refreshedSourceProxies.length} 条）`)
      }
      return true
    } catch (error: any) {
      if (!silent) {
        toast.error(error?.message || '订阅刷新失败')
      }
      return false
    } finally {
      setRefreshingSourceIds(prev => {
        const next = new Set(prev)
        next.delete(sourceId)
        return next
      })
    }
  }, [globalAutoRefreshEnabled, globalRefreshInterval, saveProxies])

  const handleRefreshAllSources = useCallback(async (silent = false) => {
    const metas = collectURLImportSources(proxiesRef.current)
    if (metas.length === 0) {
      if (!silent) {
        toast.info('当前没有 URL 导入订阅')
      }
      return
    }

    setRefreshingAllSources(true)
    let successCount = 0
    for (const meta of metas) {
      // 串行刷新，避免并发保存导致覆盖
      // eslint-disable-next-line no-await-in-loop
      const ok = await refreshSingleSource(meta.sourceId, true)
      if (ok) successCount += 1
    }
    setRefreshingAllSources(false)

    if (!silent) {
      if (successCount === metas.length) {
        toast.success(`订阅刷新完成：${successCount}/${metas.length}`)
      } else {
        toast.warning(`订阅刷新完成：成功 ${successCount}/${metas.length}`)
      }
    }
  }, [refreshSingleSource])

  useEffect(() => {
    const runAutoRefresh = async () => {
      if (autoRefreshRunningRef.current || refreshingAllSources) {
        return
      }
      if (!globalAutoRefreshEnabled) {
        return
      }
      const intervalMs = globalRefreshInterval * 60 * 1000
      const metas = collectURLImportSources(proxiesRef.current).filter(meta => {
        if (!meta.sourceUrl.trim()) return false
        const last = parseTimestampMs(meta.sourceLastRefreshAt)
        return last <= 0 || Date.now() - last >= intervalMs
      })
      if (metas.length === 0) {
        return
      }

      autoRefreshRunningRef.current = true
      try {
        for (const meta of metas) {
          // eslint-disable-next-line no-await-in-loop
          await refreshSingleSource(meta.sourceId, true)
        }
      } finally {
        autoRefreshRunningRef.current = false
      }
    }

    void runAutoRefresh()
    const timer = window.setInterval(() => {
      void runAutoRefresh()
    }, 60 * 1000)

    return () => {
      window.clearInterval(timer)
    }
  }, [globalAutoRefreshEnabled, globalRefreshInterval, refreshingAllSources, refreshSingleSource])

  const protocolOptions = useMemo(
    () => ['all', ...Array.from(new Set(displayList.map(p => p.type).filter(t => t !== '-')))],
    [displayList]
  )

  const getLatencySortTuple = (proxyId: string): [number, number] => {
    const v = latencyMap[proxyId]
    if (v === undefined) return [4, Number.MAX_SAFE_INTEGER]
    if (v === -1) return [1, Number.MAX_SAFE_INTEGER] // 测速中
    if (v === -2) return [2, Number.MAX_SAFE_INTEGER] // 超时
    if (v === -3) return [3, Number.MAX_SAFE_INTEGER] // 不支持
    return [0, v] // 正常延迟
  }

  const compareText = (a: string, b: string) => a.localeCompare(b, 'zh-CN')

  const compareByColumn = (a: ProxyDisplayInfo, b: ProxyDisplayInfo, column: string) => {
    switch (column) {
      case 'proxyName':
        return compareText(a.proxyName || '', b.proxyName || '')
      case 'groupName':
        return compareText(a.groupName || '', b.groupName || '')
      case 'type':
        return compareText(a.type || '', b.type || '')
      case 'server':
        return compareText(a.server || '', b.server || '')
      case 'port':
        return (a.port || 0) - (b.port || 0)
      case 'latency': {
        const [rankA, valA] = getLatencySortTuple(a.proxyId)
        const [rankB, valB] = getLatencySortTuple(b.proxyId)
        if (rankA !== rankB) return rankA - rankB
        if (valA !== valB) return valA - valB
        return compareText(a.proxyName || '', b.proxyName || '')
      }
      default:
        return 0
    }
  }

  const filteredList = useMemo(() => {
    const filtered = displayList.filter(p => {
      const matchProtocol = filterProtocol === 'all' || p.type === filterProtocol
      const matchKeyword = !filterKeyword || p.proxyName.toLowerCase().includes(filterKeyword.toLowerCase()) || p.server.toLowerCase().includes(filterKeyword.toLowerCase())
      const matchGroup = filterGroup === 'all' || p.groupName === filterGroup
      return matchProtocol && matchKeyword && matchGroup
    })

    if (!sortColumn || !sortOrder) return filtered

    return [...filtered].sort((a, b) => {
      const cmp = compareByColumn(a, b, sortColumn)
      return sortOrder === 'asc' ? cmp : -cmp
    })
  }, [displayList, filterProtocol, filterKeyword, filterGroup, sortColumn, sortOrder, latencyMap])

  const allFilteredSelected = filteredList.length > 0 && filteredList.every(p => selectedIds.has(p.proxyId))
  const someFilteredSelected = filteredList.some(p => selectedIds.has(p.proxyId))
  const timeoutProxyIds = useMemo(() => {
    return proxies
      .filter(p => {
        if (isBuiltinProxy(p)) return false
        const cachedLatency = latencyMap[p.proxyId]
        if (cachedLatency === -2) return true
        return !!p.lastTestedAt && p.lastTestOk === false
      })
      .map(p => p.proxyId)
  }, [proxies, latencyMap])

  const previewCountryOptions = useMemo(() => {
    const countries = new Set<string>()
    Object.values(previewIPHealthMap).forEach(result => {
      const country = (result?.country || '').trim()
      if (result?.ok && country) countries.add(country)
    })
    return [
      { value: 'all', label: '全部地区' },
      ...Array.from(countries).sort((a, b) => a.localeCompare(b)).map(country => ({ value: country, label: country })),
    ]
  }, [previewIPHealthMap])

  const filteredPreviewList = useMemo(() => {
    const keyword = normalizePreviewSearchText(previewKeyword)
    return previewList.filter(item => {
      const latency = previewLatencyMap[item.proxyId]
      if (!previewLatencyMatchesFilter(latency, previewLatencyFilter)) return false

      const health = previewIPHealthMap[item.proxyId]
      const checking = previewCheckingIPHealthIds.has(item.proxyId)
      if (!previewHealthMatchesFilter(health, checking, previewHealthFilter)) return false

      if (previewCountryFilter !== 'all' && (health?.country || '') !== previewCountryFilter) return false

      if (!keyword) return true
      const searchText = [
        item.proxyName,
        item.groupName,
        item.type,
        item.server,
        item.port,
        health?.ip,
        health?.country,
        health?.region,
        health?.city,
        health?.asOrganization,
        health?.fraudScore,
        health?.isResidential ? '住宅 residential' : '机房 datacenter',
      ].map(normalizePreviewSearchText).join(' ')
      return searchText.includes(keyword)
    })
  }, [
    previewList,
    previewKeyword,
    previewLatencyFilter,
    previewHealthFilter,
    previewCountryFilter,
    previewLatencyMap,
    previewIPHealthMap,
    previewCheckingIPHealthIds,
  ])

  const previewSelectedCount = previewSelectedIds.size
  const previewAllFilteredSelected = filteredPreviewList.length > 0 && filteredPreviewList.every(p => previewSelectedIds.has(p.proxyId))
  const previewSomeFilteredSelected = filteredPreviewList.some(p => previewSelectedIds.has(p.proxyId))
  const previewHasActiveFilter = !!previewKeyword.trim() || previewLatencyFilter !== 'all' || previewHealthFilter !== 'all' || previewCountryFilter !== 'all'
  const previewTestableList = filteredPreviewList.filter(p => p.proxyConfig !== 'direct://')

  const resetPreviewDetectionState = () => {
    setPreviewSelectedIds(new Set())
    setPreviewKeyword('')
    setPreviewLatencyFilter('all')
    setPreviewHealthFilter('all')
    setPreviewCountryFilter('all')
    setPreviewLatencyMap({})
    setPreviewIPHealthMap({})
    setPreviewCheckingIPHealthIds(new Set())
    setPreviewTestingAll(false)
    setPreviewCheckingAllIPHealth(false)
  }

  const handleToggleAll = () => {
    if (allFilteredSelected) {
      setSelectedIds(prev => {
        const next = new Set(prev)
        filteredList.forEach(p => next.delete(p.proxyId))
        return next
      })
    } else {
      setSelectedIds(prev => {
        const next = new Set(prev)
        filteredList.filter(p => !BUILTIN_PROXY_IDS.has(p.proxyId)).forEach(p => next.add(p.proxyId))
        return next
      })
    }
  }

  const handleToggleOne = (proxyId: string) => {
    if (BUILTIN_PROXY_IDS.has(proxyId)) return
    setSelectedIds(prev => {
      const next = new Set(prev)
      next.has(proxyId) ? next.delete(proxyId) : next.add(proxyId)
      return next
    })
  }

  const handleBatchDeleteConfirm = async () => {
    try {
      const newProxies = proxies.filter(p => !selectedIds.has(p.proxyId))
      await saveProxies(newProxies)
      toast.success(`已删除 ${selectedIds.size} 个代理`)
      setSelectedIds(new Set())
    } catch (error: any) {
      toast.error(error?.message || '删除失败')
    }
  }

  const handleDeleteTimeoutConfirm = async () => {
    const deleteIds = new Set(timeoutProxyIds)
    if (deleteIds.size === 0) {
      setDeleteTimeoutConfirmOpen(false)
      toast.info('没有可删除的测试超时节点')
      return
    }
    try {
      const newProxies = proxies.filter(p => !deleteIds.has(p.proxyId))
      await saveProxies(newProxies)
      setLatencyMap(prev => {
        const next = { ...prev }
        deleteIds.forEach(id => { delete next[id] })
        return next
      })
      setIPHealthMap(prev => {
        const next = { ...prev }
        deleteIds.forEach(id => { delete next[id] })
        return next
      })
      setSelectedIds(prev => {
        const next = new Set(prev)
        deleteIds.forEach(id => next.delete(id))
        return next
      })
      toast.success(`已删除 ${deleteIds.size} 个测试超时节点`)
    } catch (error: any) {
      toast.error(error?.message || '删除失败')
    } finally {
      setDeleteTimeoutConfirmOpen(false)
    }
  }

  const handleTestOne = async (record: ProxyDisplayInfo) => {
    if (record.proxyConfig === 'direct://') {
      toast.info('直连模式无需测速')
      return
    }
    setLatencyMap(prev => ({ ...prev, [record.proxyId]: -1 }))
    const result = await browserProxyTestSpeed(record.proxyId)
    const val = toLatencyValue(result.ok, result.latencyMs, result.error)
    setLatencyMap(prev => ({ ...prev, [record.proxyId]: val }))
  }

  const handleTestAll = async () => {
    const testable = filteredList.filter(p => p.proxyConfig !== 'direct://')
    if (testable.length === 0) return
    setTestingAll(true)
    const init: Record<string, number> = {}
    testable.forEach(p => { init[p.proxyId] = -1 })
    setLatencyMap(prev => ({ ...prev, ...init }))

    // 监听后端实时推送的单个测速结果
    const off = EventsOn('proxy:speed:result', (data: { proxyId: string; ok: boolean; latencyMs: number; error: string }) => {
      const val = toLatencyValue(data.ok, data.latencyMs, data.error)
      setLatencyMap(prev => ({ ...prev, [data.proxyId]: val }))
    })

    try {
      const proxyIds = testable.map(p => p.proxyId)
      const results = await browserProxyBatchTestSpeed(proxyIds, 20)
      setLatencyMap(prev => {
        const next = { ...prev }
        results.forEach(result => {
          next[result.proxyId] = toLatencyValue(result.ok, result.latencyMs, result.error)
        })
        return next
      })
    } finally {
      off()
      setTestingAll(false)
    }
  }

  const handleCheckOneIPHealth = async (record: ProxyDisplayInfo) => {
    if (record.proxyConfig === 'direct://') {
      toast.info('直连模式无需检测')
      return
    }
    if (checkingIPHealthIds.has(record.proxyId)) return

    setCheckingIPHealthIds(prev => new Set(prev).add(record.proxyId))
    try {
      const result = await browserProxyCheckIPHealth(record.proxyId)
      setIPHealthMap(prev => ({ ...prev, [record.proxyId]: result }))
      if (!result.ok) {
        toast.error(result.error || `${record.proxyName} 检测失败`)
      }
    } finally {
      setCheckingIPHealthIds(prev => {
        const next = new Set(prev)
        next.delete(record.proxyId)
        return next
      })
    }
  }

  const handleCheckAllIPHealth = async () => {
    const testable = filteredList.filter(p => p.proxyConfig !== 'direct://')
    if (testable.length === 0) return
    setCheckingAllIPHealth(true)

    const ids = testable.map(p => p.proxyId)
    const idSet = new Set(ids)
    setCheckingIPHealthIds(prev => new Set([...Array.from(prev), ...ids]))

    const off = EventsOn('proxy:iphealth:result', (data: ProxyIPHealthResult) => {
      if (!data?.proxyId || !idSet.has(data.proxyId)) return
      setIPHealthMap(prev => ({ ...prev, [data.proxyId]: data }))
      setCheckingIPHealthIds(prev => {
        const next = new Set(prev)
        next.delete(data.proxyId)
        return next
      })
    })

    try {
      const results = await browserProxyBatchCheckIPHealth(ids, 10)
      setIPHealthMap(prev => {
        const next = { ...prev }
        results.forEach(result => {
          if (result?.proxyId && idSet.has(result.proxyId)) {
            next[result.proxyId] = result
          }
        })
        return next
      })
      const failed = results.filter(r => !r.ok).length
      if (failed > 0) {
        toast.info(`IP 健康检测完成：成功 ${results.length - failed}，失败 ${failed}`)
      } else {
        toast.success(`IP 健康检测完成：共 ${results.length} 条`)
      }
    } finally {
      off()
      setCheckingIPHealthIds(prev => {
        const next = new Set(prev)
        ids.forEach(id => next.delete(id))
        return next
      })
      setCheckingAllIPHealth(false)
    }
  }

  const handleToggleAllPreview = () => {
    if (previewAllFilteredSelected) {
      setPreviewSelectedIds(prev => {
        const next = new Set(prev)
        filteredPreviewList.forEach(p => next.delete(p.proxyId))
        return next
      })
    } else {
      setPreviewSelectedIds(prev => {
        const next = new Set(prev)
        filteredPreviewList.forEach(p => next.add(p.proxyId))
        return next
      })
    }
  }

  const handleTogglePreviewOne = (proxyId: string) => {
    setPreviewSelectedIds(prev => {
      const next = new Set(prev)
      next.has(proxyId) ? next.delete(proxyId) : next.add(proxyId)
      return next
    })
  }

  const handleSelectOnlyFilteredPreview = () => {
    if (filteredPreviewList.length === 0) {
      toast.info('当前筛选没有可选择的代理')
      return
    }
    setPreviewSelectedIds(new Set(filteredPreviewList.map(item => item.proxyId)))
  }

  const removePreviewItems = (removeIds: Set<string>) => {
    if (removeIds.size === 0) return
    const removedNames = previewList
      .filter(item => removeIds.has(item.proxyId))
      .map(item => item.proxyName)
    setPreviewList(prev => prev.filter(item => !removeIds.has(item.proxyId)))
    setPreviewSelectedIds(prev => {
      const next = new Set(prev)
      removeIds.forEach(id => next.delete(id))
      return next
    })
    setPreviewLatencyMap(prev => {
      const next = { ...prev }
      removeIds.forEach(id => { delete next[id] })
      return next
    })
    setPreviewIPHealthMap(prev => {
      const next = { ...prev }
      removeIds.forEach(id => { delete next[id] })
      return next
    })
    setPreviewCheckingIPHealthIds(prev => {
      const next = new Set(prev)
      removeIds.forEach(id => next.delete(id))
      return next
    })
    setRemovedPreviewProxyNames(prev => [...prev, ...removedNames])
  }

  const handleRemoveFilteredPreview = () => {
    if (filteredPreviewList.length === 0) {
      toast.info('当前筛选没有可删除的代理')
      return
    }
    removePreviewItems(new Set(filteredPreviewList.map(item => item.proxyId)))
  }

  const handleKeepFilteredPreview = () => {
    if (filteredPreviewList.length === 0) {
      toast.info('当前筛选没有可保留的代理')
      return
    }
    const keepIds = new Set(filteredPreviewList.map(item => item.proxyId))
    const removeIds = new Set(previewList.filter(item => !keepIds.has(item.proxyId)).map(item => item.proxyId))
    removePreviewItems(removeIds)
    setPreviewSelectedIds(keepIds)
  }

  const handlePreviewTestAll = async () => {
    const testable = previewTestableList
    if (testable.length === 0) {
      toast.info('当前筛选没有可测速的代理')
      return
    }
    setPreviewTestingAll(true)
    const init: Record<string, number> = {}
    testable.forEach(p => { init[p.proxyId] = -1 })
    setPreviewLatencyMap(prev => ({ ...prev, ...init }))

    const idSet = new Set(testable.map(p => p.proxyId))
    const off = EventsOn('proxy:preview:speed:result', (data: { proxyId: string; ok: boolean; latencyMs: number; error: string }) => {
      if (!data?.proxyId || !idSet.has(data.proxyId)) return
      setPreviewLatencyMap(prev => ({ ...prev, [data.proxyId]: toLatencyValue(data.ok, data.latencyMs, data.error) }))
    })

    try {
      const results = await browserProxyPreviewBatchTestSpeed(
        testable.map(p => ({ proxyId: p.proxyId, proxyConfig: p.proxyConfig })),
        20
      )
      setPreviewLatencyMap(prev => {
        const next = { ...prev }
        results.forEach(result => {
          if (result?.proxyId && idSet.has(result.proxyId)) {
            next[result.proxyId] = toLatencyValue(result.ok, result.latencyMs, result.error)
          }
        })
        return next
      })
      const failed = results.filter(result => !result.ok).length
      if (failed > 0) {
        toast.info(`预览测速完成：可用 ${results.length - failed}，异常 ${failed}`)
      } else {
        toast.success(`预览测速完成：共 ${results.length} 条`)
      }
    } finally {
      off()
      setPreviewTestingAll(false)
    }
  }

  const handlePreviewCheckIPHealth = async () => {
    const testable = previewTestableList
    if (testable.length === 0) {
      toast.info('当前筛选没有可检测的代理')
      return
    }
    setPreviewCheckingAllIPHealth(true)
    const ids = testable.map(p => p.proxyId)
    const idSet = new Set(ids)
    setPreviewCheckingIPHealthIds(prev => new Set([...Array.from(prev), ...ids]))

    const off = EventsOn('proxy:preview:iphealth:result', (data: ProxyIPHealthResult) => {
      if (!data?.proxyId || !idSet.has(data.proxyId)) return
      setPreviewIPHealthMap(prev => ({ ...prev, [data.proxyId]: data }))
      setPreviewCheckingIPHealthIds(prev => {
        const next = new Set(prev)
        next.delete(data.proxyId)
        return next
      })
    })

    try {
      const results = await browserProxyPreviewBatchCheckIPHealth(
        testable.map(p => ({ proxyId: p.proxyId, proxyConfig: p.proxyConfig })),
        10
      )
      setPreviewIPHealthMap(prev => {
        const next = { ...prev }
        results.forEach(result => {
          if (result?.proxyId && idSet.has(result.proxyId)) {
            next[result.proxyId] = result
          }
        })
        return next
      })
      const failed = results.filter(result => !result.ok).length
      if (failed > 0) {
        toast.info(`预览 IP 健康检测完成：成功 ${results.length - failed}，失败 ${failed}`)
      } else {
        toast.success(`预览 IP 健康检测完成：共 ${results.length} 条`)
      }
    } finally {
      off()
      setPreviewCheckingIPHealthIds(prev => {
        const next = new Set(prev)
        ids.forEach(id => next.delete(id))
        return next
      })
      setPreviewCheckingAllIPHealth(false)
    }
  }

  const renderLatency = (record: ProxyDisplayInfo) => {
    if (record.proxyConfig === 'direct://') {
      return <span className="text-[var(--color-text-muted)] text-xs">不适用</span>
    }
    const val = latencyMap[record.proxyId]
    if (val === undefined) return <span className="text-[var(--color-text-muted)] text-xs">-</span>
    if (val === -1) return <span className="text-[var(--color-text-muted)] text-xs animate-pulse">测速中...</span>
    if (val === -2) return <span className="text-red-500 text-xs">超时</span>
    if (val === -3) return <span className="text-gray-400 text-xs">不支持</span>
    const color = val < 200 ? 'text-green-500' : val < 500 ? 'text-yellow-500' : 'text-red-500'
    return <span className={`text-xs font-medium ${color}`}>{val} ms</span>
  }

  const openIPHealthDetail = (proxyId: string) => {
    const result = ipHealthMap[proxyId]
    if (!result) return
    setCurrentIPHealthDetail(result)
    setIPHealthDetailOpen(true)
  }

  const renderIPHealth = (record: ProxyDisplayInfo) => {
    if (record.proxyConfig === 'direct://') {
      return <span className="text-[var(--color-text-muted)] text-xs">不适用</span>
    }
    if (checkingIPHealthIds.has(record.proxyId)) {
      return <span className="text-[var(--color-text-muted)] text-xs animate-pulse">检测中...</span>
    }

    const result = ipHealthMap[record.proxyId]
    if (!result) return <span className="text-[var(--color-text-muted)] text-xs">-</span>
    if (!result.ok) {
      return (
        <div className="flex items-center gap-2">
          <span className="text-xs text-red-500 truncate max-w-[120px]" title={result.error || '检测失败'}>失败</span>
          <Button size="sm" variant="ghost" onClick={(e) => { e.stopPropagation(); openIPHealthDetail(record.proxyId) }}>原始</Button>
        </div>
      )
    }

    const location = [result.country, result.region, result.city].filter(Boolean).join(' / ')
    return (
      <div className="flex items-center gap-2 min-w-0">
        <div className="min-w-0">
          <div className="text-xs text-[var(--color-text-primary)] truncate">{result.ip || '-'}</div>
          <div className="text-[11px] text-[var(--color-text-muted)] truncate">
            {`fraud ${result.fraudScore} | ${result.isResidential ? '住宅' : '机房'}${location ? ` | ${location}` : ''}`}
          </div>
        </div>
        <Button size="sm" variant="ghost" onClick={(e) => { e.stopPropagation(); openIPHealthDetail(record.proxyId) }}>原始</Button>
      </div>
    )
  }

  const renderPreviewLatency = (record: ProxyDisplayInfo) => {
    if (record.proxyConfig === 'direct://') {
      return <span className="text-[var(--color-text-muted)] text-xs">不适用</span>
    }
    const val = previewLatencyMap[record.proxyId]
    if (val === undefined) return <span className="text-[var(--color-text-muted)] text-xs">-</span>
    if (val === -1) return <span className="text-[var(--color-text-muted)] text-xs animate-pulse">测速中...</span>
    if (val === -2) return <span className="text-red-500 text-xs">超时</span>
    if (val === -3) return <span className="text-gray-400 text-xs">不支持</span>
    const color = val < 200 ? 'text-green-500' : val < 500 ? 'text-yellow-500' : 'text-red-500'
    return <span className={`text-xs font-medium ${color}`}>{val} ms</span>
  }

  const openPreviewIPHealthDetail = (proxyId: string) => {
    const result = previewIPHealthMap[proxyId]
    if (!result) return
    setCurrentIPHealthDetail(result)
    setIPHealthDetailOpen(true)
  }

  const renderPreviewIPHealth = (record: ProxyDisplayInfo) => {
    if (record.proxyConfig === 'direct://') {
      return <span className="text-[var(--color-text-muted)] text-xs">不适用</span>
    }
    if (previewCheckingIPHealthIds.has(record.proxyId)) {
      return <span className="text-[var(--color-text-muted)] text-xs animate-pulse">检测中...</span>
    }

    const result = previewIPHealthMap[record.proxyId]
    if (!result) return <span className="text-[var(--color-text-muted)] text-xs">-</span>
    if (!result.ok) {
      return (
        <div className="flex items-center gap-2">
          <span className="text-xs text-red-500 truncate max-w-[120px]" title={result.error || '检测失败'}>失败</span>
          <Button size="sm" variant="ghost" onClick={(e) => { e.stopPropagation(); openPreviewIPHealthDetail(record.proxyId) }}>原始</Button>
        </div>
      )
    }

    const location = [result.country, result.region, result.city].filter(Boolean).join(' / ')
    return (
      <div className="flex items-center gap-2 min-w-0">
        <div className="min-w-0">
          <div className="text-xs text-[var(--color-text-primary)] truncate">{result.ip || '-'}</div>
          <div className="text-[11px] text-[var(--color-text-muted)] truncate">
            {`fraud ${result.fraudScore} | ${result.isResidential ? '住宅' : '机房'}${location ? ` | ${location}` : ''}`}
          </div>
        </div>
        <Button size="sm" variant="ghost" onClick={(e) => { e.stopPropagation(); openPreviewIPHealthDetail(record.proxyId) }}>原始</Button>
      </div>
    )
  }

  const toggleVisibleColumn = (key: string) => {
    const option = PROXY_COLUMN_OPTIONS.find(item => item.key === key)
    if (option?.locked) return
    setVisibleColumnKeys(prev => {
      const next = prev.includes(key) ? prev.filter(item => item !== key) : [...prev, key]
      const lockedKeys = PROXY_COLUMN_OPTIONS.filter(item => item.locked).map(item => item.key)
      return Array.from(new Set([...lockedKeys, ...next]))
    })
  }

  const allColumns: TableColumn<ProxyDisplayInfo>[] = [
    {
      key: 'checkbox',
      title: '',
      width: '40px',
      render: (_, record) => (
        <input
          type="checkbox"
          checked={selectedIds.has(record.proxyId)}
          disabled={BUILTIN_PROXY_IDS.has(record.proxyId)}
          onChange={() => handleToggleOne(record.proxyId)}
          onClick={e => e.stopPropagation()}
          className="w-4 h-4 rounded border-[var(--color-border)] accent-[var(--color-primary)] cursor-pointer disabled:opacity-30 disabled:cursor-not-allowed"
        />
      ),
    },
    { key: 'proxyName', title: '代理名称', width: '180px', sortable: true },
    { key: 'groupName', title: '分组', width: '100px', sortable: true, render: (val) => val ? <span className="px-1.5 py-0.5 text-xs rounded bg-[var(--color-accent)]/10 text-[var(--color-accent)]">{val}</span> : '-' },
    {
      key: 'source',
      title: '来源',
      width: '180px',
      render: (_, record) => {
        if (!record.sourceUrl) return '-'
        const host = sourceHostLabel(record.sourceUrl)
        return (
          <div className="text-xs leading-5">
            <div className="text-[var(--color-text-primary)] truncate" title={record.sourceUrl}>{host}</div>
            <div className="text-[var(--color-text-muted)]">
              {globalAutoRefreshEnabled ? `自动刷新 ${globalRefreshInterval} 分钟（全局）` : '手动刷新'}
            </div>
          </div>
        )
      },
    },
    { key: 'type', title: '类型', width: '90px', sortable: true },
    { key: 'server', title: '服务器', width: '180px', sortable: true },
    { key: 'port', title: '端口', width: '80px', sortable: true, render: (val) => val || '-' },
    {
      key: 'latency',
      title: '延迟',
      width: '90px',
      sortable: true,
      render: (_, record) => renderLatency(record),
    },
    {
      key: 'ipHealth',
      title: 'IP健康',
      width: '280px',
      render: (_, record) => renderIPHealth(record),
    },
    {
      key: 'actions',
      title: '操作',
      width: '320px',
      render: (_, record) => {
        const isBuiltin = BUILTIN_PROXY_IDS.has(record.proxyId)
        const hasSource = !!record.sourceId && !!record.sourceUrl
        return (
          <div className="flex gap-2">
            {hasSource && (
              <Button
                size="sm"
                variant="secondary"
                onClick={(e) => { e.stopPropagation(); void refreshSingleSource(record.sourceId, false) }}
                loading={refreshingSourceIds.has(record.sourceId)}
              >
                刷新订阅
              </Button>
            )}
            <Button
              size="sm" variant="ghost"
              onClick={(e) => { e.stopPropagation(); handleTestOne(record) }}
              loading={latencyMap[record.proxyId] === -1}
              disabled={record.proxyConfig === 'direct://'}
            >测速</Button>
            <Button
              size="sm" variant="ghost"
              onClick={(e) => { e.stopPropagation(); handleCheckOneIPHealth(record) }}
              loading={checkingIPHealthIds.has(record.proxyId)}
              disabled={record.proxyConfig === 'direct://'}
            >IP健康</Button>
            <Button
              size="sm" variant="ghost"
              disabled={isBuiltin}
              title={isBuiltin ? '内置代理不可编辑' : undefined}
              onClick={(e) => { e.stopPropagation(); if (!isBuiltin) handleEdit(record) }}
            >编辑</Button>
            <Button
              size="sm" variant="danger"
              disabled={isBuiltin}
              title={isBuiltin ? '内置代理不可删除' : undefined}
              onClick={(e) => { e.stopPropagation(); if (!isBuiltin) handleDeleteClick(record.proxyId) }}
            >删除</Button>
          </div>
        )
      },
    },
  ]

  const handleRemovePreviewProxy = (proxyId: string) => {
    removePreviewItems(new Set([proxyId]))
  }

  const previewColumns: TableColumn<ProxyDisplayInfo>[] = [
    {
      key: 'checkbox',
      title: (
        <input
          type="checkbox"
          checked={previewAllFilteredSelected}
          ref={el => { if (el) el.indeterminate = previewSomeFilteredSelected && !previewAllFilteredSelected }}
          onChange={handleToggleAllPreview}
          onClick={e => e.stopPropagation()}
          className="w-4 h-4 rounded border-[var(--color-border)] accent-[var(--color-primary)] cursor-pointer"
          title="选择当前筛选结果"
        />
      ),
      width: '44px',
      render: (_, record) => (
        <input
          type="checkbox"
          checked={previewSelectedIds.has(record.proxyId)}
          onChange={() => handleTogglePreviewOne(record.proxyId)}
          onClick={e => e.stopPropagation()}
          className="w-4 h-4 rounded border-[var(--color-border)] accent-[var(--color-primary)] cursor-pointer"
        />
      ),
    },
    {
      key: 'proxyName',
      title: '代理名称',
      width: '220px',
      render: (_, record) => (
        <div className="min-w-0">
          <div className="truncate text-[var(--color-text-primary)]" title={record.proxyName}>{record.proxyName}</div>
          {record.groupName && <div className="text-[11px] text-[var(--color-text-muted)] truncate">{record.groupName}</div>}
        </div>
      ),
    },
    { key: 'type', title: '类型', width: '80px' },
    { key: 'server', title: '服务器', width: '170px', render: (val) => <span className="truncate block max-w-[170px]" title={String(val || '-')}>{val || '-'}</span> },
    { key: 'port', title: '端口', width: '70px', render: (val) => val || '-' },
    {
      key: 'latency',
      title: '延迟',
      width: '90px',
      render: (_, record) => renderPreviewLatency(record),
    },
    {
      key: 'ipHealth',
      title: 'IP健康',
      width: '280px',
      render: (_, record) => renderPreviewIPHealth(record),
    },
    {
      key: 'actions',
      title: '操作',
      width: '96px',
      render: (_, record) => (
        <Button
          size="sm"
          variant="danger"
          onClick={() => handleRemovePreviewProxy(record.proxyId)}
        >
          删除
        </Button>
      ),
    },
  ]

  const handleEdit = (record: ProxyDisplayInfo) => {
    const proxy = proxies.find(p => p.proxyId === record.proxyId)
    if (proxy) {
      setEditingProxy(proxy)
      setEditForm({ proxyName: proxy.proxyName, proxyConfig: proxy.proxyConfig, dnsServers: proxy.dnsServers || '', groupName: proxy.groupName || '' })
      setEditModalOpen(true)
    }
  }

  const handleSaveProxy = async () => {
    if (!editForm.proxyName.trim()) { toast.error('请输入代理名称'); return }
    if (!editingProxy) return
    setSaving(true)
    try {
      const newProxies = proxies.map(p =>
        p.proxyId === editingProxy.proxyId
          ? { ...p, proxyName: editForm.proxyName, proxyConfig: editForm.proxyConfig, dnsServers: editForm.dnsServers, groupName: editForm.groupName }
          : p
      )
      await saveProxies(newProxies)
      setEditModalOpen(false)
      toast.success('代理已更新')
    } catch (error: any) {
      toast.error(error?.message || '保存失败')
    } finally {
      setSaving(false)
    }
  }

  const handleDeleteClick = (proxyId: string) => {
    setDeletingId(proxyId)
    setDeleteConfirmOpen(true)
  }

  const handleDeleteConfirm = async () => {
    if (!deletingId) return
    try {
      const newProxies = proxies.filter(p => p.proxyId !== deletingId)
      await saveProxies(newProxies)
      setSelectedIds(prev => { const next = new Set(prev); next.delete(deletingId); return next })
      toast.success('代理已删除')
    } catch (error: any) {
      toast.error(error?.message || '删除失败')
    }
    setDeletingId(null)
  }

  const handleImportModeChange = (nextMode: ProxyImportMode) => {
    setImportMode(nextMode)
    setImportResolvedUrl('')
    if (nextMode !== 'clash') {
      setImportUrl('')
      setImportDnsServers('')
    }
  }

  const handleOpenImportCenter = (mode: ProxyImportMode = 'clash') => {
    setImportMode(mode)
    setImportModalOpen(true)
  }

  const handleEditSource = (source: URLImportSourceMeta) => {
    setEditingSource(source)
    setSourceEditForm({
      sourceUrl: source.sourceUrl,
      groupName: source.sourceGroupName,
      namePrefix: source.sourceNamePrefix,
      dnsServers: source.sourceDnsServers,
    })
    setSourceEditModalOpen(true)
  }

  const handleSaveSource = async () => {
    if (!editingSource) return
    const nextURL = sourceEditForm.sourceUrl.trim()
    if (!nextURL) {
      toast.error('订阅 URL 不能为空')
      return
    }
    try {
      const parsed = new URL(nextURL)
      if (!['http:', 'https:'].includes(parsed.protocol)) {
        toast.error('订阅 URL 仅支持 HTTP / HTTPS')
        return
      }
    } catch {
      toast.error('订阅 URL 格式无效')
      return
    }

    const nextGroup = sourceEditForm.groupName.trim()
    const nextPrefix = sourceEditForm.namePrefix.trim()
    const nextDNS = sourceEditForm.dnsServers.trim()
    const updated = proxies.map(item => {
      if ((item.sourceId || '').trim() !== editingSource.sourceId) return item
      return {
        ...item,
        proxyName: renameSourceProxyName(item.proxyName, editingSource.sourceNamePrefix, nextPrefix),
        groupName: nextGroup || undefined,
        dnsServers: nextDNS || undefined,
        sourceUrl: nextURL,
        sourceNamePrefix: nextPrefix || undefined,
      }
    })

    try {
      await saveProxies(updated)
      setSourceEditModalOpen(false)
      setEditingSource(null)
      toast.success('订阅已更新')
    } catch (error: any) {
      toast.error(error?.message || '订阅保存失败')
    }
  }

  const handleDeleteSourceClick = (source: URLImportSourceMeta) => {
    setDeletingSource(source)
    setSourceDeleteConfirmOpen(true)
  }

  const handleDeleteSourceConfirm = async () => {
    if (!deletingSource) return
    try {
      const updated = proxies.filter(item => (item.sourceId || '').trim() !== deletingSource.sourceId)
      await saveProxies(updated)
      setDeletingSource(null)
      toast.success('订阅已删除')
    } catch (error: any) {
      toast.error(error?.message || '订阅删除失败')
    }
  }

  const handleFetchImportURL = async () => {
    const targetURL = importUrl.trim()
    if (!targetURL) {
      toast.error('请输入订阅 URL')
      return
    }

    setFetchingImportUrl(true)
    try {
      const result = await fetchClashImportFromURL(targetURL)
      const content = (result?.content || '').trim()
      if (!content) {
        throw new Error('订阅内容为空')
      }

      setImportResolvedUrl((result?.url || targetURL).trim())
      setImportText(content)

      if (!importDnsServers.trim() && typeof result?.dnsServers === 'string' && result.dnsServers.trim()) {
        setImportDnsServers(result.dnsServers.trim())
      }
      if (!importGroupName.trim() && typeof result?.suggestedGroup === 'string' && result.suggestedGroup.trim()) {
        setImportGroupName(result.suggestedGroup.trim())
      }

      toast.success(`URL 获取成功，检测到 ${Math.max(0, Number(result?.proxyCount || 0))} 个代理`)
    } catch (error: any) {
      setImportResolvedUrl('')
      toast.error(error?.message || 'URL 获取失败')
    } finally {
      setFetchingImportUrl(false)
    }
  }

  const handleParseImport = () => {
    try {
      const prefix = importNamePrefix.trim()
      const candidates = importMode === 'clash'
        ? buildImportCandidatesFromClash(parseClashImportText(importText), prefix)
        : [
            ...parseDirectProxyBatchText(directImportText, directImportForm.protocol),
            ...(directImportForm.server.trim() || directImportForm.port.trim()
              ? [buildDirectImportCandidate(directImportForm)]
              : []),
          ]
      if (!candidates.length) {
        toast.error('未解析到可导入代理')
        return
      }
      const preview = buildImportPreview(candidates, importGroupName.trim())
      resetPreviewDetectionState()
      setRemovedPreviewProxyNames([])
      setPreviewList(preview)
      setPreviewSelectedIds(new Set(preview.map(item => item.proxyId)))
      setImportModalOpen(false)
      setPreviewModalOpen(true)
    } catch (error: any) {
      toast.error(`解析失败: ${error?.message || '未知错误'}`)
    }
  }

  const handleConfirmImport = async () => {
    const selectedPreviewList = previewList.filter(item => previewSelectedIds.has(item.proxyId))
    if (selectedPreviewList.length === 0) {
      toast.error('请至少选择 1 个代理后再导入')
      return
    }
    setImporting(true)
    try {
      const sourceURL = importMode === 'clash' ? importResolvedUrl.trim() : ''
      const isURLImport = !!sourceURL
      const sourceNamePrefix = importMode === 'clash' ? importNamePrefix.trim() : ''
      const sourceID = isURLImport ? resolveImportSourceID(proxies, sourceURL, sourceNamePrefix) : ''
      const sourceAutoRefresh = isURLImport ? globalAutoRefreshEnabled : false
      const sourceRefreshIntervalM = sourceAutoRefresh ? globalRefreshInterval : 0
      const sourceLastRefreshAt = isURLImport ? new Date().toISOString() : ''
      const oldSourceProxies = isURLImport
        ? proxies.filter(item => (item.sourceId || '').trim() === sourceID)
        : []
      const pickExistingID = createExistingProxyIDPicker(oldSourceProxies)

      const newProxies: BrowserProxy[] = selectedPreviewList.map((p) => ({
        proxyId: pickExistingID(p.proxyName, p.proxyConfig) || nextProxyID(),
        proxyName: p.proxyName,
        proxyConfig: p.proxyConfig,
        dnsServers: importMode === 'clash' ? importDnsServers.trim() || undefined : undefined,
        groupName: importGroupName.trim() || undefined,
        sourceId: sourceID || undefined,
        sourceUrl: sourceURL || undefined,
        sourceNamePrefix: sourceNamePrefix || undefined,
        sourceAutoRefresh,
        sourceRefreshIntervalM,
        sourceLastRefreshAt: sourceLastRefreshAt || undefined,
      }))
      const allProxies = isURLImport
        ? proxies.filter(item => (item.sourceId || '').trim() !== sourceID).concat(newProxies)
        : [...proxies, ...newProxies]
      await saveProxies(allProxies)
      const unselectedPreviewProxyNames = previewList
        .filter(item => !previewSelectedIds.has(item.proxyId))
        .map(item => item.proxyName)
      const ignoredProxyNames = [...removedPreviewProxyNames, ...unselectedPreviewProxyNames]
      if (isURLImport && ignoredProxyNames.length > 0) {
        appendSourceIgnoredProxyNames(sourceID, ignoredProxyNames)
      }
      setPreviewModalOpen(false)
      setImportUrl('')
      setImportResolvedUrl('')
      setImportText('')
      setImportDnsServers('')
      setImportNamePrefix('')
      setImportGroupName('')
      setDirectImportForm({ ...INITIAL_DIRECT_IMPORT_FORM })
      setDirectImportText('')
      setPreviewList([])
      resetPreviewDetectionState()
      setRemovedPreviewProxyNames([])
      toast.success(`成功导入 ${newProxies.length} 个代理`)
    } catch (error: any) {
      toast.error(error?.message || '导入失败')
    } finally {
      setImporting(false)
    }
  }

  const selectedCount = selectedIds.size
  const canParseImport = importMode === 'clash'
    ? !!importText.trim()
    : !!directImportText.trim() || (!!directImportForm.server.trim() && !!directImportForm.port.trim())

  const sourceColumns: TableColumn<URLImportSourceMeta>[] = [
    {
      key: 'sourceUrl',
      title: '订阅',
      width: '300px',
      render: (_, record) => (
        <div className="text-xs leading-5 min-w-0 max-w-[280px] overflow-hidden">
          <div className="text-[var(--color-text-primary)] truncate" title={record.sourceUrl}>{sourceHostLabel(record.sourceUrl)}</div>
          <div className="text-[var(--color-text-muted)] truncate" title={record.sourceUrl}>{record.sourceUrl}</div>
        </div>
      ),
    },
    { key: 'proxyCount', title: '节点数', width: '80px', render: (val) => val || 0 },
    { key: 'sourceGroupName', title: '分组', width: '120px', render: (val) => val ? <span className="px-1.5 py-0.5 text-xs rounded bg-[var(--color-accent)]/10 text-[var(--color-accent)]">{val}</span> : '-' },
    { key: 'sourceNamePrefix', title: '名称前缀', width: '120px', render: (val) => val || '-' },
    {
      key: 'sourceRefreshIntervalM',
      title: '刷新策略',
      width: '150px',
      render: () => globalAutoRefreshEnabled ? `全局 ${globalRefreshInterval} 分钟` : '手动刷新',
    },
    {
      key: 'sourceLastRefreshAt',
      title: '最近刷新',
      width: '180px',
      render: (val) => val ? new Date(String(val)).toLocaleString() : '-',
    },
    {
      key: 'actions',
      title: '操作',
      width: '320px',
      render: (_, record) => (
        <div className="flex gap-2">
          <Button
            size="sm"
            variant="secondary"
            onClick={() => void refreshSingleSource(record.sourceId, false)}
            loading={refreshingSourceIds.has(record.sourceId)}
          >
            刷新
          </Button>
          <Button
            size="sm"
            variant="ghost"
            onClick={() => {
              setResourceView('proxies')
              setFilterProtocol('all')
              setFilterKeyword('')
              setFilterGroup(record.sourceGroupName || 'all')
            }}
          >
            查看节点
          </Button>
          <Button
            size="sm"
            variant="ghost"
            onClick={() => handleEditSource(record)}
          >
            编辑
          </Button>
          <Button
            size="sm"
            variant="danger"
            onClick={() => handleDeleteSourceClick(record)}
          >
            删除
          </Button>
        </div>
      ),
    },
  ]

  const columns = allColumns.filter(column => visibleColumnKeys.includes(column.key))

  return (
    <div className="space-y-5 animate-fade-in">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold text-[var(--color-text-primary)]">代理资源中心</h1>
          <p className="text-sm text-[var(--color-text-muted)] mt-1">统一管理订阅、YAML 节点与 HTTP / HTTPS / SOCKS5 批量导入</p>
        </div>
        <div className="flex gap-2">
          <Button
            size="sm"
            variant="secondary"
            onClick={() => void handleRefreshAllSources(false)}
            loading={refreshingAllSources}
            disabled={!hasURLImportSources}
          >
            刷新订阅
          </Button>
          <Button size="sm" variant="secondary" onClick={handleCheckAllIPHealth} loading={checkingAllIPHealth} disabled={filteredList.length === 0}>检测IP健康</Button>
          <Button size="sm" variant="secondary" onClick={handleTestAll} loading={testingAll} disabled={filteredList.length === 0}>测试全部</Button>
          <Button
            size="sm"
            variant="danger"
            onClick={() => setDeleteTimeoutConfirmOpen(true)}
            disabled={timeoutProxyIds.length === 0}
            title="删除除直连和本地代理之外，最近测速结果为超时的节点"
          >
            删除超时节点{timeoutProxyIds.length > 0 ? ` (${timeoutProxyIds.length})` : ''}
          </Button>
          <details className="relative">
            <summary className="list-none inline-flex items-center justify-center h-8 w-8 rounded-md border border-[var(--color-border-default)] text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-bg-secondary)] cursor-pointer" title="选择显示列">
              <Sliders className="w-4 h-4" />
            </summary>
            <div className="absolute right-0 top-9 z-20 w-56 rounded-lg border border-[var(--color-border-default)] bg-[var(--color-bg-surface)] shadow-lg p-2">
              <div className="text-xs font-medium text-[var(--color-text-muted)] px-2 py-1">显示列</div>
              {PROXY_COLUMN_OPTIONS.map(option => (
                <label key={option.key} className="flex items-center gap-2 px-2 py-1.5 text-sm text-[var(--color-text-primary)] rounded hover:bg-[var(--color-bg-secondary)] cursor-pointer">
                  <input
                    type="checkbox"
                    className="w-4 h-4 accent-[var(--color-primary)]"
                    checked={visibleColumnKeys.includes(option.key)}
                    disabled={option.locked}
                    onChange={() => toggleVisibleColumn(option.key)}
                  />
                  <span>{option.label}</span>
                </label>
              ))}
            </div>
          </details>
        </div>
      </div>

      <Card>
        <div className="rounded-lg border border-[var(--color-border-default)] bg-[var(--color-bg-secondary)] p-4">
          <p className="text-sm font-medium text-[var(--color-text-primary)]">自动切换说明</p>
          <p className="text-xs text-[var(--color-text-muted)] mt-1">
            这里维护的分组可在实例「新建配置 / 编辑配置」里选择为自动切换代理池。实例启动后浏览器连接本地固定中转端口，真实出口 IP 会在中转层按分钟轮询切换。
          </p>
        </div>
      </Card>

      <Card>
        <div className="flex items-center gap-2 mb-4">
          <Button
            size="sm"
            variant={resourceView === 'proxies' ? undefined : 'secondary'}
            onClick={() => setResourceView('proxies')}
          >
            代理节点
          </Button>
          <Button
            size="sm"
            variant={resourceView === 'sources' ? undefined : 'secondary'}
            onClick={() => setResourceView('sources')}
          >
            订阅管理{sourceMetas.length > 0 ? ` (${sourceMetas.length})` : ''}
          </Button>
          <Button
            size="sm"
            variant="ghost"
            onClick={() => handleOpenImportCenter('clash')}
          >
            添加资源
          </Button>
        </div>
        {resourceView === 'sources' && (
          <Table
            columns={sourceColumns}
            data={sourceMetas}
            rowKey="sourceId"
            loading={loading}
            emptyText="暂无订阅来源，点击「添加资源」添加 Clash 订阅 URL"
            tableLayout="fixed"
            tableMinWidth="1280px"
          />
        )}
        {resourceView === 'proxies' && (
          <>
        <div className="flex items-center gap-3 mb-4">
          <Input
            value={filterKeyword}
            onChange={e => setFilterKeyword(e.target.value)}
            placeholder="搜索名称或服务器..."
            style={{ width: '220px' }}
          />
          <select
            value={filterProtocol}
            onChange={e => setFilterProtocol(e.target.value)}
            className="px-3 py-1.5 text-sm rounded-md border border-[var(--color-border)] bg-[var(--color-bg-secondary)] text-[var(--color-text-primary)] focus:outline-none focus:ring-1 focus:ring-[var(--color-primary)]"
          >
            {protocolOptions.map(p => (
              <option key={p} value={p}>{p === 'all' ? '全部协议' : p.toUpperCase()}</option>
            ))}
          </select>
          <select
            value={filterGroup}
            onChange={e => setFilterGroup(e.target.value)}
            className="px-3 py-1.5 text-sm rounded-md border border-[var(--color-border)] bg-[var(--color-bg-secondary)] text-[var(--color-text-primary)] focus:outline-none focus:ring-1 focus:ring-[var(--color-primary)]"
          >
            <option value="all">全部分组</option>
            {groups.map(g => <option key={g} value={g}>{g}</option>)}
          </select>
          {(filterProtocol !== 'all' || filterKeyword || filterGroup !== 'all') && (
            <Button size="sm" variant="ghost" onClick={() => { setFilterProtocol('all'); setFilterKeyword(''); setFilterGroup('all') }}>清除筛选</Button>
          )}
          <div className="flex items-center gap-2 rounded-md border border-[var(--color-border)] bg-[var(--color-bg-secondary)] px-2 py-1.5">
            <span className="text-xs text-[var(--color-text-muted)]">全局自动刷新</span>
            <Switch
              checked={globalAutoRefreshEnabled}
              onChange={(checked) => setGlobalAutoRefreshEnabled(checked)}
            />
            <Input
              type="number"
              min={5}
              max={1440}
              value={globalRefreshIntervalM}
              onChange={e => setGlobalRefreshIntervalM(e.target.value)}
              className="w-24"
              disabled={!globalAutoRefreshEnabled}
            />
            <span className="text-xs text-[var(--color-text-muted)]">分钟</span>
          </div>
          <div className="flex-1" />
          {filteredList.length > 0 && (
            <label className="flex items-center gap-1.5 text-sm text-[var(--color-text-muted)] cursor-pointer select-none">
              <input
                type="checkbox"
                checked={allFilteredSelected}
                ref={el => { if (el) el.indeterminate = someFilteredSelected && !allFilteredSelected }}
                onChange={handleToggleAll}
                className="w-4 h-4 rounded border-[var(--color-border)] accent-[var(--color-primary)] cursor-pointer"
              />
              全选
            </label>
          )}
          {selectedCount > 0 && (
            <Button size="sm" variant="danger" onClick={() => setBatchDeleteConfirmOpen(true)}>
              删除所选 ({selectedCount})
            </Button>
          )}
        </div>
        <Table
          columns={columns}
          data={filteredList}
          rowKey="proxyId"
          loading={loading}
          emptyText="暂无代理配置，点击上方按钮添加或导入"
          sortColumn={sortColumn}
          sortOrder={sortOrder}
          onSort={({ column, order }) => {
            setSortColumn(column)
            setSortOrder(order)
          }}
        />
          </>
        )}
      </Card>

      <Modal open={importModalOpen} onClose={() => setImportModalOpen(false)} title="订阅与代理导入" width="640px"
        footer={
          <>
            <Button variant="secondary" onClick={() => setImportModalOpen(false)} disabled={fetchingImportUrl}>取消</Button>
            <Button onClick={handleParseImport} disabled={fetchingImportUrl || !canParseImport}>解析</Button>
          </>
        }>
        <div className="space-y-4">
          <div className="grid grid-cols-2 gap-2">
            <Button
              variant={importMode === 'clash' ? undefined : 'secondary'}
              onClick={() => handleImportModeChange('clash')}
            >
              <span className="flex flex-col items-center leading-tight">
                <span>订阅 / YAML</span>
                <span className="text-[11px] opacity-80">Clash、Base64、分享链接</span>
              </span>
            </Button>
            <Button
              variant={importMode === 'direct' ? undefined : 'secondary'}
              onClick={() => handleImportModeChange('direct')}
            >
              HTTP / HTTPS / SOCKS5
            </Button>
          </div>
          <p className="text-sm text-[var(--color-text-muted)]">
            {importMode === 'clash'
              ? '支持管理 Clash 订阅 URL、直接粘贴 YAML，也支持 v2rayN Base64 订阅/分享链接；通过 URL 导入后会进入订阅管理，可统一刷新'
              : '支持批量粘贴 HTTP / HTTPS / SOCKS5 代理，也可以用表单补充单条带认证代理'}
          </p>
          {importMode === 'clash' && (
            <>
              <FormItem label="订阅 URL（可选）">
                <div className="flex gap-2">
                  <Input
                    value={importUrl}
                    onChange={e => {
                      const next = e.target.value
                      setImportUrl(next)
                      if (importResolvedUrl.trim() && next.trim() !== importResolvedUrl.trim()) {
                        setImportResolvedUrl('')
                      }
                    }}
                    placeholder="https://example.com/clash/subscription"
                    className="flex-1"
                  />
                  <Button
                    variant="secondary"
                    onClick={handleFetchImportURL}
                    loading={fetchingImportUrl}
                    disabled={!importUrl.trim()}
                  >
                    从 URL 获取
                  </Button>
                </div>
                {importResolvedUrl.trim() && (
                  <p className="text-xs text-[var(--color-success)] mt-1 break-all">
                    已绑定订阅：{importResolvedUrl}
                  </p>
                )}
                <p className="text-xs text-[var(--color-text-muted)] mt-1">获取成功后会自动回填可解析文本，并尝试自动填充 DNS 与建议分组；自动刷新时间请在列表顶部统一配置</p>
              </FormItem>
              <Textarea
                value={importText}
                onChange={e => setImportText(e.target.value)}
                rows={12}
                placeholder={`proxies:\n  - name: vless-v6\n    type: vless\n    server: example.com\n    port: 443\n    uuid: your-uuid\n    ...\n\n或粘贴 Base64 订阅 / vless:// / vmess:// / trojan:// 节点`}
              />
            </>
          )}
          {importMode === 'direct' && (
            <div className="space-y-4">
              <FormItem label="批量代理（可选）">
                <Textarea
                  value={directImportText}
                  onChange={e => setDirectImportText(e.target.value)}
                  rows={8}
                  placeholder={`124.155.254.202:8938:T2n4c7u6Q5B3:H4H6j9q5h5v8\n124.155.246.100:9752:a2G8i8f2u2N6:A9q1Q9A4h5D7\nsocks5://127.0.0.1:1080 本地SOCKS5`}
                />
                <p className="text-xs text-[var(--color-text-muted)] mt-1">每行一个代理，支持 host:port、host:port:账号:密码、标准 URL；未写协议时使用下方选择的协议</p>
              </FormItem>
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                <FormItem label="单条协议">
                  <Select
                    options={[...DIRECT_PROXY_PROTOCOL_OPTIONS]}
                    value={directImportForm.protocol}
                    onChange={e => setDirectImportForm(prev => ({ ...prev, protocol: e.target.value as DirectImportForm['protocol'] }))}
                  />
                </FormItem>
                <FormItem label="单条名称（可选）">
                  <Input
                    value={directImportForm.proxyName}
                    onChange={e => setDirectImportForm(prev => ({ ...prev, proxyName: e.target.value }))}
                    placeholder="例如：香港节点"
                  />
                </FormItem>
                <FormItem label="单条地址">
                  <Input
                    value={directImportForm.server}
                    onChange={e => setDirectImportForm(prev => ({ ...prev, server: e.target.value }))}
                    placeholder="例如：127.0.0.1 或 hk.example.com"
                  />
                </FormItem>
                <FormItem label="单条端口">
                  <Input
                    type="number"
                    min={1}
                    max={65535}
                    value={directImportForm.port}
                    onChange={e => setDirectImportForm(prev => ({ ...prev, port: e.target.value }))}
                    placeholder="例如：1080"
                  />
                </FormItem>
                <FormItem label="账号（可选）">
                  <Input
                    value={directImportForm.username}
                    onChange={e => setDirectImportForm(prev => ({ ...prev, username: e.target.value }))}
                    placeholder="留空则不使用认证"
                  />
                </FormItem>
                <FormItem label="密码（可选）">
                  <Input
                    type="password"
                    value={directImportForm.password}
                    onChange={e => setDirectImportForm(prev => ({ ...prev, password: e.target.value }))}
                    placeholder="留空则不使用密码"
                  />
                </FormItem>
              </div>
            </div>
          )}
          <FormItem label="分组名称（可选）">
            <Input
              value={importGroupName}
              onChange={e => setImportGroupName(e.target.value)}
              placeholder="例如：香港、美国、机场A"
              list="proxy-groups-datalist"
            />
            {groups.length > 0 && (
              <datalist id="proxy-groups-datalist">
                {groups.map(g => <option key={g} value={g} />)}
              </datalist>
            )}
            <p className="text-xs text-[var(--color-text-muted)] mt-1">填写后本次导入的代理将归入该分组，可按分组筛选</p>
          </FormItem>
          {importMode === 'clash' && (
            <FormItem label="名称前缀（可选）">
              <Input
                value={importNamePrefix}
                onChange={e => setImportNamePrefix(e.target.value)}
                placeholder="例如：HK、US、机场A"
              />
              <p className="text-xs text-[var(--color-text-muted)] mt-1">
                填写后代理名称将变为 <code className="px-1 bg-[var(--color-bg-secondary)] rounded">前缀-原名称</code>，留空则保持原名
              </p>
            </FormItem>
          )}
          {importMode === 'clash' && (
            <FormItem label="批量 DNS 配置（可选）">
              <Textarea value={importDnsServers} onChange={e => setImportDnsServers(e.target.value)} rows={5}
                placeholder={`dns:\n  enable: true\n  nameserver:\n    - 119.29.29.29\n    - 223.5.5.5`} />
              <p className="text-xs text-[var(--color-text-muted)] mt-1">留空则不配置 DNS，填写后将应用到本次导入的所有代理</p>
            </FormItem>
          )}
        </div>
      </Modal>

      <Modal open={previewModalOpen} onClose={() => setPreviewModalOpen(false)} title="确认导入以下代理" width="980px"
        footer={<><Button variant="secondary" onClick={() => { setPreviewModalOpen(false); setImportModalOpen(true) }}>返回修改</Button><Button onClick={handleConfirmImport} loading={importing} disabled={previewSelectedCount === 0}>导入选中{previewSelectedCount > 0 ? ` (${previewSelectedCount})` : ''}</Button></>}>
        <div className="space-y-3">
          {importMode === 'clash' && importDnsServers.trim() && (
            <p className="text-xs text-[var(--color-text-muted)] bg-[var(--color-bg-secondary)] px-3 py-2 rounded">已配置批量 DNS，将应用到以下所有代理</p>
          )}
          <div className="grid grid-cols-1 lg:grid-cols-[minmax(240px,1fr)_150px_150px_150px] gap-2">
            <Input
              value={previewKeyword}
              onChange={e => setPreviewKeyword(e.target.value)}
              placeholder="搜索名称、服务器、国家、地区、IP、运营商"
            />
            <Select
              value={previewLatencyFilter}
              onChange={e => setPreviewLatencyFilter(e.target.value as PreviewLatencyFilter)}
              options={PREVIEW_LATENCY_FILTER_OPTIONS}
            />
            <Select
              value={previewHealthFilter}
              onChange={e => setPreviewHealthFilter(e.target.value as PreviewHealthFilter)}
              options={PREVIEW_HEALTH_FILTER_OPTIONS}
            />
            <Select
              value={previewCountryFilter}
              onChange={e => setPreviewCountryFilter(e.target.value)}
              options={previewCountryOptions}
            />
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <Button size="sm" variant="secondary" onClick={handlePreviewTestAll} loading={previewTestingAll} disabled={previewTestableList.length === 0}>检测延迟</Button>
            <Button size="sm" variant="secondary" onClick={handlePreviewCheckIPHealth} loading={previewCheckingAllIPHealth} disabled={previewTestableList.length === 0}>检测IP健康</Button>
            <Button size="sm" variant="ghost" onClick={handleSelectOnlyFilteredPreview} disabled={filteredPreviewList.length === 0}>只选择当前筛选</Button>
            <Button size="sm" variant="ghost" onClick={() => setPreviewSelectedIds(new Set(previewList.map(item => item.proxyId)))} disabled={previewList.length === 0}>全选</Button>
            <Button size="sm" variant="ghost" onClick={() => setPreviewSelectedIds(new Set())} disabled={previewSelectedCount === 0}>清空选择</Button>
            <Button size="sm" variant="secondary" onClick={handleKeepFilteredPreview} disabled={!previewHasActiveFilter || filteredPreviewList.length === 0}>只保留筛选</Button>
            <Button size="sm" variant="danger" onClick={handleRemoveFilteredPreview} disabled={filteredPreviewList.length === 0}>删除筛选</Button>
          </div>
          <p className="text-xs text-[var(--color-text-muted)]">
            共 {previewList.length} 条，当前显示 {filteredPreviewList.length} 条，已选择 {previewSelectedCount} 条，已删除 {removedPreviewProxyNames.length} 条。确认导入只会导入已选择的代理。
          </p>
          <Table columns={previewColumns} data={filteredPreviewList} rowKey="proxyId" maxHeight="420px" emptyText="无代理数据" tableMinWidth="1040px" />
        </div>
      </Modal>

      <Modal open={editModalOpen} onClose={() => setEditModalOpen(false)} title="编辑代理" width="500px"
        footer={<><Button variant="secondary" onClick={() => setEditModalOpen(false)}>取消</Button><Button onClick={handleSaveProxy} loading={saving}>保存</Button></>}>
        <div className="space-y-4">
          <FormItem label="代理名称" required>
            <Input value={editForm.proxyName} onChange={e => setEditForm(prev => ({ ...prev, proxyName: e.target.value }))} placeholder="例如：香港节点" />
          </FormItem>
          <FormItem label="分组名称（可选）">
            <Input value={editForm.groupName} onChange={e => setEditForm(prev => ({ ...prev, groupName: e.target.value }))} placeholder="例如：香港、美国" list="edit-proxy-groups-datalist" />
            <datalist id="edit-proxy-groups-datalist">
              {groups.map(g => <option key={g} value={g} />)}
            </datalist>
          </FormItem>
          <FormItem label="代理配置">
            <Textarea value={editForm.proxyConfig} onChange={e => setEditForm(prev => ({ ...prev, proxyConfig: e.target.value }))} rows={10} placeholder="支持 Clash YAML、http://、https://、socks5:// 代理配置" />
          </FormItem>
          <FormItem label="DNS 服务器（可选）">
            <Textarea value={editForm.dnsServers} onChange={e => setEditForm(prev => ({ ...prev, dnsServers: e.target.value }))} rows={6}
              placeholder={`dns:\n  enable: true\n  nameserver:\n    - 119.29.29.29\n    - 223.5.5.5`} />
            <p className="text-xs text-[var(--color-text-muted)] mt-1">支持 Clash dns: YAML 格式，主要用于 Clash / 桥接代理；直连 HTTP/SOCKS5 通常不会使用这里的 DNS 配置</p>
          </FormItem>
        </div>
      </Modal>

      <Modal open={sourceEditModalOpen} onClose={() => setSourceEditModalOpen(false)} title="编辑订阅" width="560px"
        footer={<><Button variant="secondary" onClick={() => setSourceEditModalOpen(false)}>取消</Button><Button onClick={handleSaveSource}>保存</Button></>}>
        <div className="space-y-4">
          <FormItem label="订阅 URL" required>
            <Input
              value={sourceEditForm.sourceUrl}
              onChange={e => setSourceEditForm(prev => ({ ...prev, sourceUrl: e.target.value }))}
              placeholder="https://example.com/clash/subscription"
            />
          </FormItem>
          <FormItem label="分组名称（可选）">
            <Input
              value={sourceEditForm.groupName}
              onChange={e => setSourceEditForm(prev => ({ ...prev, groupName: e.target.value }))}
              placeholder="例如：香港、美国、机场A"
              list="edit-source-groups-datalist"
            />
            <datalist id="edit-source-groups-datalist">
              {groups.map(g => <option key={g} value={g} />)}
            </datalist>
          </FormItem>
          <FormItem label="名称前缀（可选）">
            <Input
              value={sourceEditForm.namePrefix}
              onChange={e => setSourceEditForm(prev => ({ ...prev, namePrefix: e.target.value }))}
              placeholder="例如：HK、US、机场A"
            />
          </FormItem>
          <FormItem label="批量 DNS 配置（可选）">
            <Textarea
              value={sourceEditForm.dnsServers}
              onChange={e => setSourceEditForm(prev => ({ ...prev, dnsServers: e.target.value }))}
              rows={5}
              placeholder={`dns:\n  enable: true\n  nameserver:\n    - 119.29.29.29\n    - 223.5.5.5`}
            />
          </FormItem>
          <p className="text-xs text-[var(--color-text-muted)]">
            保存后会同步更新该订阅下的全部节点；下次刷新订阅会继续使用这些来源配置。
          </p>
        </div>
      </Modal>

      <Modal
        open={ipHealthDetailOpen}
        onClose={() => setIPHealthDetailOpen(false)}
        title="IP健康原始返回"
        width="760px"
        footer={<Button variant="secondary" onClick={() => setIPHealthDetailOpen(false)}>关闭</Button>}
      >
        <div className="space-y-3">
          {currentIPHealthDetail && (
            <>
              <div className="text-xs text-[var(--color-text-muted)]">
                代理ID：{currentIPHealthDetail.proxyId} | 来源：{currentIPHealthDetail.source} | 时间：{currentIPHealthDetail.updatedAt}
              </div>
              {!currentIPHealthDetail.ok && (
                <div className="text-sm text-red-500">{currentIPHealthDetail.error || '检测失败'}</div>
              )}
              <pre className="max-h-[420px] overflow-auto text-xs leading-5 rounded-lg bg-[var(--color-bg-secondary)] border border-[var(--color-border)] p-3">
                {JSON.stringify(currentIPHealthDetail.rawData || {}, null, 2)}
              </pre>
            </>
          )}
        </div>
      </Modal>

      <ConfirmModal open={deleteConfirmOpen} onClose={() => setDeleteConfirmOpen(false)} onConfirm={handleDeleteConfirm}
        title="确认删除" content="确定要删除这个代理吗？此操作不可恢复。" confirmText="删除" danger />

      <ConfirmModal open={batchDeleteConfirmOpen} onClose={() => setBatchDeleteConfirmOpen(false)} onConfirm={handleBatchDeleteConfirm}
        title="批量删除" content={`确定要删除选中的 ${selectedCount} 个代理吗？此操作不可恢复。`} confirmText="删除" danger />

      <ConfirmModal open={deleteTimeoutConfirmOpen} onClose={() => setDeleteTimeoutConfirmOpen(false)} onConfirm={handleDeleteTimeoutConfirm}
        title="删除测试超时节点" content={`确定要删除 ${timeoutProxyIds.length} 个测试超时节点吗？直连和本地代理会保留，此操作不可恢复。`} confirmText="删除超时节点" danger />

      <ConfirmModal open={sourceDeleteConfirmOpen} onClose={() => setSourceDeleteConfirmOpen(false)} onConfirm={handleDeleteSourceConfirm}
        title="删除订阅" content={`确定删除订阅「${deletingSource ? sourceHostLabel(deletingSource.sourceUrl) : ''}」及其 ${deletingSource?.proxyCount || 0} 个节点吗？此操作不可恢复。`} confirmText="删除订阅" danger />
    </div>
  )
}
