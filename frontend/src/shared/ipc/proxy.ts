import {
  METHOD_BROWSER_PROXY_BATCH_CHECK_IP_HEALTH,
  METHOD_BROWSER_PROXY_BATCH_TEST_SPEED,
  METHOD_BROWSER_PROXY_CHECK_IP_HEALTH,
  METHOD_BROWSER_PROXY_FETCH_CLASH_BY_URL,
  METHOD_BROWSER_PROXY_GROUP_LIST,
  METHOD_BROWSER_PROXY_LIST,
  METHOD_BROWSER_PROXY_PREVIEW_BATCH_CHECK_IP_HEALTH,
  METHOD_BROWSER_PROXY_PREVIEW_BATCH_TEST_SPEED,
  METHOD_BROWSER_PROXY_SAVE,
  METHOD_BROWSER_PROXY_TEST_CONNECTIVITY,
  METHOD_BROWSER_PROXY_TEST_REAL_CONNECTIVITY,
  METHOD_BROWSER_PROXY_TEST_SPEED,
  METHOD_BROWSER_PROXY_VALIDATE_CONFIG,
} from './envelope'
import {
  WireType,
  concatBytes,
  decodeString,
  decodeVarintField,
  encodeBoolField,
  encodeBytesField,
  encodeInt32Field,
  encodeInt64Field,
  encodeStringField,
  readFields,
} from './protobuf'
import { ProtoIpcClient } from './transport'
import { decodeBrowserActionResponse } from './browser'

const proxyProtoClient = new ProtoIpcClient()

export type ProtoBrowserProxy = {
  proxyId: string
  proxyName: string
  proxyConfig: string
  dnsServers?: string
  groupName?: string
  sortOrder?: number
  sourceId?: string
  sourceUrl?: string
  sourceNamePrefix?: string
  sourceFilterJson?: string
  sourceAutoRefresh?: boolean
  sourceRefreshIntervalM?: number
  sourceLastRefreshAt?: string
  lastLatencyMs?: number
  lastTestOk?: boolean
  lastTestedAt?: string
  lastIPHealthJson?: string
}

export type ProtoClashImportURLResult = {
  url: string
  content: string
  proxyCount: number
  dnsServers?: string
  suggestedGroup?: string
}

export type ProtoProxyValidationResult = {
  supported: boolean
  errorMsg: string
}

export type ProtoProxyTestResult = {
  proxyId: string
  ok: boolean
  latencyMs: number
  error: string
}

export type ProtoProxyPreviewTestInput = {
  proxyId: string
  proxyConfig: string
}

export type ProtoProxyIPHealthResult = {
  proxyId: string
  ok: boolean
  source: string
  error: string
  ip: string
  fraudScore: number
  isResidential: boolean
  isBroadcast: boolean
  country: string
  region: string
  city: string
  asOrganization: string
  rawData: Record<string, unknown>
  updatedAt: string
}

export async function listBrowserProxies(groupName = ''): Promise<ProtoBrowserProxy[]> {
  const payload = await proxyProtoClient.request(METHOD_BROWSER_PROXY_LIST, encodeBrowserProxyListRequest({ groupName }))
  return decodeBrowserProxyListResponse(payload).proxies
}

export async function listBrowserProxyGroups(): Promise<string[]> {
  const payload = await proxyProtoClient.request(METHOD_BROWSER_PROXY_GROUP_LIST, new Uint8Array())
  return decodeBrowserProxyGroupListResponse(payload).groups
}

export async function saveBrowserProxies(proxies: ProtoBrowserProxy[]): Promise<boolean> {
  const payload = await proxyProtoClient.request(METHOD_BROWSER_PROXY_SAVE, encodeBrowserProxySaveRequest({ proxies: normalizeBrowserProxies(proxies) }))
  return decodeBrowserActionResponse(payload).ok
}

export async function fetchClashImportFromURL(targetURL: string): Promise<ProtoClashImportURLResult> {
  const payload = await proxyProtoClient.request(METHOD_BROWSER_PROXY_FETCH_CLASH_BY_URL, encodeBrowserProxyFetchClashByURLRequest({ url: targetURL }), 30000)
  return decodeBrowserProxyFetchClashByURLResponse(payload)
}

export async function validateProxyConfig(proxyConfig: string, proxyId: string): Promise<ProtoProxyValidationResult> {
  const payload = await proxyProtoClient.request(METHOD_BROWSER_PROXY_VALIDATE_CONFIG, encodeBrowserProxyValidateConfigRequest({ proxyConfig, proxyId }))
  return decodeBrowserProxyValidateConfigResponse(payload)
}

export async function testProxyConnectivity(proxyId: string, proxyConfig: string): Promise<ProtoProxyTestResult> {
  const payload = await proxyProtoClient.request(METHOD_BROWSER_PROXY_TEST_CONNECTIVITY, encodeBrowserProxyTestRequest({ proxyId, proxyConfig }), 30000)
  return decodeBrowserProxyTestResult(payload)
}

export async function testProxyRealConnectivity(proxyId: string): Promise<ProtoProxyTestResult> {
  const payload = await proxyProtoClient.request(METHOD_BROWSER_PROXY_TEST_REAL_CONNECTIVITY, encodeBrowserProxyTestRequest({ proxyId }), 30000)
  return decodeBrowserProxyTestResult(payload)
}

export async function testBrowserProxySpeed(proxyId: string): Promise<ProtoProxyTestResult> {
  const payload = await proxyProtoClient.request(METHOD_BROWSER_PROXY_TEST_SPEED, encodeBrowserProxyTestRequest({ proxyId }), 30000)
  return decodeBrowserProxyTestResult(payload)
}

export async function testBrowserProxyBatchSpeed(proxyIds: string[], concurrency = 20): Promise<ProtoProxyTestResult[]> {
  const payload = await proxyProtoClient.request(METHOD_BROWSER_PROXY_BATCH_TEST_SPEED, encodeBrowserProxyIDListRequest({ proxyIds, concurrency }), 120000)
  return decodeBrowserProxyTestResultListResponse(payload).results
}

export async function testBrowserProxyPreviewBatchSpeed(items: ProtoProxyPreviewTestInput[], concurrency = 20): Promise<ProtoProxyTestResult[]> {
  const payload = await proxyProtoClient.request(METHOD_BROWSER_PROXY_PREVIEW_BATCH_TEST_SPEED, encodeBrowserProxyPreviewTestRequest({ items, concurrency }), 120000)
  return decodeBrowserProxyTestResultListResponse(payload).results
}

export async function checkBrowserProxyIPHealth(proxyId: string): Promise<ProtoProxyIPHealthResult> {
  const payload = await proxyProtoClient.request(METHOD_BROWSER_PROXY_CHECK_IP_HEALTH, encodeBrowserProxyTestRequest({ proxyId }), 30000)
  return decodeBrowserProxyIPHealthResult(payload)
}

export async function checkBrowserProxyBatchIPHealth(proxyIds: string[], concurrency = 10): Promise<ProtoProxyIPHealthResult[]> {
  const payload = await proxyProtoClient.request(METHOD_BROWSER_PROXY_BATCH_CHECK_IP_HEALTH, encodeBrowserProxyIDListRequest({ proxyIds, concurrency }), 120000)
  return decodeBrowserProxyIPHealthResultListResponse(payload).results
}

export async function checkBrowserProxyPreviewBatchIPHealth(items: ProtoProxyPreviewTestInput[], concurrency = 10): Promise<ProtoProxyIPHealthResult[]> {
  const payload = await proxyProtoClient.request(METHOD_BROWSER_PROXY_PREVIEW_BATCH_CHECK_IP_HEALTH, encodeBrowserProxyPreviewTestRequest({ items, concurrency }), 120000)
  return decodeBrowserProxyIPHealthResultListResponse(payload).results
}

export function onBrowserProxySpeedResult(callback: (result: ProtoProxyTestResult) => void): () => void {
  return proxyProtoClient.onEvent('proxy:speed:result', event => callback(decodeBrowserProxyTestResult(event.payload)))
}

export function onBrowserProxyIPHealthResult(callback: (result: ProtoProxyIPHealthResult) => void): () => void {
  return proxyProtoClient.onEvent('proxy:iphealth:result', event => callback(decodeBrowserProxyIPHealthResult(event.payload)))
}

export function onBrowserProxyPreviewSpeedResult(callback: (result: ProtoProxyTestResult) => void): () => void {
  return proxyProtoClient.onEvent('proxy:preview:speed:result', event => callback(decodeBrowserProxyTestResult(event.payload)))
}

export function onBrowserProxyPreviewIPHealthResult(callback: (result: ProtoProxyIPHealthResult) => void): () => void {
  return proxyProtoClient.onEvent('proxy:preview:iphealth:result', event => callback(decodeBrowserProxyIPHealthResult(event.payload)))
}

export function encodeBrowserProxyListRequest(message: { groupName?: string }): Uint8Array {
  return concatBytes([encodeStringField(1, message.groupName ?? '')])
}

export function encodeBrowserProxySaveRequest(message: { proxies: ProtoBrowserProxy[] }): Uint8Array {
  const proxies = Array.isArray(message.proxies) ? message.proxies : []
  return concatBytes(proxies.map(item => encodeBytesField(1, encodeBrowserProxy(normalizeBrowserProxy(item)))))
}

export function encodeBrowserProxyFetchClashByURLRequest(message: { url: string }): Uint8Array {
  return concatBytes([encodeStringField(1, message.url)])
}

export function encodeBrowserProxyValidateConfigRequest(message: { proxyConfig: string; proxyId: string }): Uint8Array {
  return concatBytes([
    encodeStringField(1, message.proxyConfig),
    encodeStringField(2, message.proxyId),
  ])
}

export function encodeBrowserProxyTestRequest(message: { proxyId: string; proxyConfig?: string }): Uint8Array {
  return concatBytes([
    encodeStringField(1, message.proxyId),
    encodeStringField(2, message.proxyConfig ?? ''),
  ])
}

export function encodeBrowserProxyIDListRequest(message: { proxyIds: string[]; concurrency: number }): Uint8Array {
  return concatBytes([
    ...message.proxyIds.map(proxyId => encodeStringField(1, proxyId)),
    encodeInt32Field(2, message.concurrency),
  ])
}

export function encodeBrowserProxyPreviewTestRequest(message: { items: ProtoProxyPreviewTestInput[]; concurrency: number }): Uint8Array {
  return concatBytes([
    ...message.items.map(item => encodeBytesField(1, encodeBrowserProxyPreviewTestInput(item))),
    encodeInt32Field(2, message.concurrency),
  ])
}

export function encodeBrowserProxyPreviewTestInput(message: ProtoProxyPreviewTestInput): Uint8Array {
  return concatBytes([
    encodeStringField(1, message.proxyId),
    encodeStringField(2, message.proxyConfig),
  ])
}

export function encodeBrowserProxy(proxy: ProtoBrowserProxy): Uint8Array {
  const item = normalizeBrowserProxy(proxy)
  return concatBytes([
    encodeStringField(1, item.proxyId),
    encodeStringField(2, item.proxyName),
    encodeStringField(3, item.proxyConfig),
    encodeStringField(4, item.dnsServers ?? ''),
    encodeStringField(5, item.groupName ?? ''),
    encodeInt32Field(6, item.sortOrder ?? 0),
    encodeStringField(7, item.sourceId ?? ''),
    encodeStringField(8, item.sourceUrl ?? ''),
    encodeStringField(9, item.sourceNamePrefix ?? ''),
    encodeStringField(10, item.sourceFilterJson ?? ''),
    encodeBoolField(11, item.sourceAutoRefresh === true),
    encodeInt32Field(12, item.sourceRefreshIntervalM ?? 0),
    encodeStringField(13, item.sourceLastRefreshAt ?? ''),
    encodeInt64Field(14, item.lastLatencyMs ?? 0),
    encodeBoolField(15, item.lastTestOk === true),
    encodeStringField(16, item.lastTestedAt ?? ''),
    encodeStringField(17, item.lastIPHealthJson ?? ''),
  ])
}

function normalizeBrowserProxies(proxies: ProtoBrowserProxy[]): ProtoBrowserProxy[] {
  if (!Array.isArray(proxies)) {
    return []
  }
  return proxies.map(normalizeBrowserProxy)
}

function normalizeBrowserProxy(proxy: ProtoBrowserProxy): ProtoBrowserProxy {
  const item = proxy || ({} as ProtoBrowserProxy)
  const sourceUrl = normalizeString(item.sourceUrl)
  const sourceAutoRefresh = sourceUrl !== '' && item.sourceAutoRefresh === true
  return {
    proxyId: normalizeString(item.proxyId),
    proxyName: normalizeString(item.proxyName),
    proxyConfig: normalizeString(item.proxyConfig),
    dnsServers: normalizeOptionalString(item.dnsServers),
    groupName: normalizeOptionalString(item.groupName),
    sortOrder: normalizeInt32(item.sortOrder),
    sourceId: sourceUrl ? normalizeOptionalString(item.sourceId) : undefined,
    sourceUrl: sourceUrl || undefined,
    sourceNamePrefix: sourceUrl ? normalizeOptionalString(item.sourceNamePrefix) : undefined,
    sourceFilterJson: sourceUrl ? normalizeOptionalString(item.sourceFilterJson) : undefined,
    sourceAutoRefresh,
    sourceRefreshIntervalM: sourceAutoRefresh ? normalizeInt32(item.sourceRefreshIntervalM) : 0,
    sourceLastRefreshAt: sourceUrl ? normalizeOptionalString(item.sourceLastRefreshAt) : undefined,
    lastLatencyMs: normalizeInt64(item.lastLatencyMs),
    lastTestOk: item.lastTestOk === true,
    lastTestedAt: normalizeOptionalString(item.lastTestedAt),
    lastIPHealthJson: normalizeOptionalString(item.lastIPHealthJson),
  }
}

function normalizeString(value: unknown): string {
  return typeof value === 'string' ? value.trim() : ''
}

function normalizeOptionalString(value: unknown): string | undefined {
  const text = normalizeString(value)
  return text || undefined
}

function normalizeInt32(value: unknown): number {
  const number = typeof value === 'number' ? value : Number(value)
  if (!Number.isFinite(number)) {
    return 0
  }
  return Math.max(-2147483648, Math.min(2147483647, Math.trunc(number)))
}

function normalizeInt64(value: unknown): number {
  const number = typeof value === 'number' ? value : Number(value)
  if (!Number.isFinite(number)) {
    return 0
  }
  return Math.trunc(number)
}

export function decodeBrowserProxyFetchClashByURLResponse(payload: Uint8Array): ProtoClashImportURLResult {
  const result: ProtoClashImportURLResult = {
    url: '',
    content: '',
    proxyCount: 0,
  }

  for (const field of readFields(payload)) {
    if (field.wireType === WireType.LengthDelimited) {
      const text = decodeString(field.value)
      switch (field.fieldNumber) {
        case 1:
          result.url = text
          break
        case 2:
          result.content = text
          break
        case 4:
          result.dnsServers = text
          break
        case 5:
          result.suggestedGroup = text
          break
      }
      continue
    }
    if (field.fieldNumber === 3 && field.wireType === WireType.Varint) {
      result.proxyCount = Number(decodeVarintField(field.value))
    }
  }

  return result
}

export function decodeBrowserProxyValidateConfigResponse(payload: Uint8Array): ProtoProxyValidationResult {
  const result: ProtoProxyValidationResult = {
    supported: false,
    errorMsg: '',
  }

  for (const field of readFields(payload)) {
    if (field.fieldNumber === 1 && field.wireType === WireType.Varint) {
      result.supported = Number(decodeVarintField(field.value)) !== 0
    } else if (field.fieldNumber === 2 && field.wireType === WireType.LengthDelimited) {
      result.errorMsg = decodeString(field.value)
    }
  }

  return result
}

export function decodeBrowserProxyTestResult(payload: Uint8Array): ProtoProxyTestResult {
  const result: ProtoProxyTestResult = {
    proxyId: '',
    ok: false,
    latencyMs: 0,
    error: '',
  }

  for (const field of readFields(payload)) {
    if (field.wireType === WireType.LengthDelimited) {
      const text = decodeString(field.value)
      if (field.fieldNumber === 1) {
        result.proxyId = text
      } else if (field.fieldNumber === 4) {
        result.error = text
      }
      continue
    }
    if (field.wireType === WireType.Varint) {
      const number = Number(decodeVarintField(field.value))
      if (field.fieldNumber === 2) {
        result.ok = number !== 0
      } else if (field.fieldNumber === 3) {
        result.latencyMs = number
      }
    }
  }

  return result
}

export function decodeBrowserProxyTestResultListResponse(payload: Uint8Array): { results: ProtoProxyTestResult[] } {
  const results: ProtoProxyTestResult[] = []
  for (const field of readFields(payload)) {
    if (field.fieldNumber === 1 && field.wireType === WireType.LengthDelimited) {
      results.push(decodeBrowserProxyTestResult(field.value))
    }
  }
  return { results }
}

export function decodeBrowserProxyIPHealthResult(payload: Uint8Array): ProtoProxyIPHealthResult {
  const result: ProtoProxyIPHealthResult = {
    proxyId: '',
    ok: false,
    source: '',
    error: '',
    ip: '',
    fraudScore: 0,
    isResidential: false,
    isBroadcast: false,
    country: '',
    region: '',
    city: '',
    asOrganization: '',
    rawData: {},
    updatedAt: '',
  }

  for (const field of readFields(payload)) {
    if (field.wireType === WireType.LengthDelimited) {
      const text = decodeString(field.value)
      switch (field.fieldNumber) {
        case 1:
          result.proxyId = text
          break
        case 3:
          result.source = text
          break
        case 4:
          result.error = text
          break
        case 5:
          result.ip = text
          break
        case 9:
          result.country = text
          break
        case 10:
          result.region = text
          break
        case 11:
          result.city = text
          break
        case 12:
          result.asOrganization = text
          break
        case 13:
          result.rawData = decodeJSONRecord(text)
          break
        case 14:
          result.updatedAt = text
          break
      }
      continue
    }

    if (field.wireType === WireType.Varint) {
      const number = Number(decodeVarintField(field.value))
      switch (field.fieldNumber) {
        case 2:
          result.ok = number !== 0
          break
        case 6:
          result.fraudScore = number
          break
        case 7:
          result.isResidential = number !== 0
          break
        case 8:
          result.isBroadcast = number !== 0
          break
      }
    }
  }

  return result
}

export function decodeBrowserProxyIPHealthResultListResponse(payload: Uint8Array): { results: ProtoProxyIPHealthResult[] } {
  const results: ProtoProxyIPHealthResult[] = []
  for (const field of readFields(payload)) {
    if (field.fieldNumber === 1 && field.wireType === WireType.LengthDelimited) {
      results.push(decodeBrowserProxyIPHealthResult(field.value))
    }
  }
  return { results }
}

export function decodeBrowserProxyListResponse(payload: Uint8Array): { proxies: ProtoBrowserProxy[] } {
  const proxies: ProtoBrowserProxy[] = []
  for (const field of readFields(payload)) {
    if (field.fieldNumber === 1 && field.wireType === WireType.LengthDelimited) {
      proxies.push(decodeBrowserProxy(field.value))
    }
  }
  return { proxies }
}

export function decodeBrowserProxyGroupListResponse(payload: Uint8Array): { groups: string[] } {
  const groups: string[] = []
  for (const field of readFields(payload)) {
    if (field.fieldNumber === 1 && field.wireType === WireType.LengthDelimited) {
      groups.push(decodeString(field.value))
    }
  }
  return { groups }
}

export function decodeBrowserProxy(payload: Uint8Array): ProtoBrowserProxy {
  const proxy: ProtoBrowserProxy = {
    proxyId: '',
    proxyName: '',
    proxyConfig: '',
  }

  for (const field of readFields(payload)) {
    if (field.wireType === WireType.LengthDelimited) {
      const text = decodeString(field.value)
      switch (field.fieldNumber) {
        case 1:
          proxy.proxyId = text
          break
        case 2:
          proxy.proxyName = text
          break
        case 3:
          proxy.proxyConfig = text
          break
        case 4:
          proxy.dnsServers = text
          break
        case 5:
          proxy.groupName = text
          break
        case 7:
          proxy.sourceId = text
          break
        case 8:
          proxy.sourceUrl = text
          break
        case 9:
          proxy.sourceNamePrefix = text
          break
        case 10:
          proxy.sourceFilterJson = text
          break
        case 13:
          proxy.sourceLastRefreshAt = text
          break
        case 16:
          proxy.lastTestedAt = text
          break
        case 17:
          proxy.lastIPHealthJson = text
          break
      }
      continue
    }

    if (field.wireType === WireType.Varint) {
      const number = Number(decodeVarintField(field.value))
      switch (field.fieldNumber) {
        case 6:
          proxy.sortOrder = number
          break
        case 11:
          proxy.sourceAutoRefresh = number !== 0
          break
        case 12:
          proxy.sourceRefreshIntervalM = number
          break
        case 14:
          proxy.lastLatencyMs = number
          break
        case 15:
          proxy.lastTestOk = number !== 0
          break
      }
    }
  }

  return proxy
}

function decodeJSONRecord(value: string): Record<string, unknown> {
  if (!value) {
    return {}
  }
  try {
    const parsed = JSON.parse(value)
    if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
      return parsed as Record<string, unknown>
    }
  } catch {
    // Ignore malformed diagnostic payloads from external services.
  }
  return {}
}
