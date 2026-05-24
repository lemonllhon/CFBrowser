import { PROTO_SCHEMA_VERSION, decodeRpcEvent, decodeRpcResponse, encodeRpcEnvelope } from './envelope'
import type { RpcError, RpcEvent } from './envelope'

const RAW_MESSAGE_PREFIX = 'trace-proto:'
const RESPONSE_EVENT_NAME = 'trace:proto:response'
const RAW_EVENT_NAME = 'trace:proto:event'
const DEFAULT_TIMEOUT_MS = 10000
const CONFIG_WAIT_MS = 5000
const RAW_FALLBACK_CONFIG_WAIT_MS = 1200
const RAW_FALLBACK_COOLDOWN_MS = 10000
const EVENT_DEDUP_TTL_MS = 60000
const EVENT_DEDUP_MAX = 1000
const BINARY_FRAME_RESPONSE = 1
const BINARY_FRAME_EVENT = 2

type WailsEvent = {
  name?: string
  data?: unknown
}

type WailsRuntimeBridge = {
  invoke?: (message: string) => void
  dispatchWailsEvent?: (event: WailsEvent) => void
  __traceProtoDispatchHook?: boolean
  __traceProtoPreviousDispatch?: (event: WailsEvent) => void
}

type WailsWindow = Window & {
  _wails?: WailsRuntimeBridge
  __TRACE_PROTO_IPC__?: TraceProtoIpcConfig
  chrome?: {
    webview?: {
      postMessage?: (message: string) => void
    }
  }
  webkit?: {
    messageHandlers?: {
      external?: {
        postMessage?: (message: string) => void
      }
    }
  }
  wails?: {
    invoke?: (message: string) => void
  }
}

type PendingRequest = {
  resolve: (payload: Uint8Array) => void
  reject: (reason: unknown) => void
  timer: number
}

const pendingRequests = new Map<string, PendingRequest>()
const eventListeners = new Map<string, Set<(event: RpcEvent) => void>>()
const seenEventIds = new Map<string, number>()
let rawFallbackPreferredUntil = 0

export type TraceProtoIpcConfig = {
  wsUrl?: string
  rawAvailable?: boolean
}

export class ProtoIpcError extends Error {
  readonly code: string
  readonly details: string

  constructor(error: RpcError) {
    super(error.message || error.code || 'Protobuf IPC 请求失败')
    this.name = 'ProtoIpcError'
    this.code = error.code
    this.details = error.details
  }
}

export class RawProtoIpcClient {
  request(method: string, payload: Uint8Array, timeoutMs = DEFAULT_TIMEOUT_MS): Promise<Uint8Array> {
    installWailsEventHook()
    const requestId = createRequestId()
    const envelope = encodeRpcEnvelope({
      requestId,
      method,
      payload,
      schemaVersion: PROTO_SCHEMA_VERSION,
      timestampMs: Date.now(),
    })
    const message = encodeRawMessage(envelope)

    return new Promise<Uint8Array>((resolve, reject) => {
      const timer = window.setTimeout(() => {
        pendingRequests.delete(requestId)
        reject(new Error(`Protobuf IPC 请求超时: ${method}`))
      }, timeoutMs)
      pendingRequests.set(requestId, { resolve, reject, timer })

      try {
        resolveInvoke()(message)
      } catch (error) {
        window.clearTimeout(timer)
        pendingRequests.delete(requestId)
        reject(error)
      }
    })
  }
}

export class BinaryWebSocketProtoIpcClient {
  private socketPromise: Promise<WebSocket> | null = null

  private readonly pending = new Map<string, PendingRequest>()

  async ensureConnected(timeoutMs = CONFIG_WAIT_MS): Promise<void> {
    const config = getBinaryConfig() ?? await waitForBinaryConfig(timeoutMs)
    const wsUrl = config.wsUrl
    if (!wsUrl) {
      throw new Error('Protobuf IPC WebSocket 配置无效')
    }
    await this.openSocket(wsUrl)
  }

  async request(method: string, payload: Uint8Array, timeoutMs = DEFAULT_TIMEOUT_MS): Promise<Uint8Array> {
    const config = getBinaryConfig() ?? await waitForBinaryConfig(CONFIG_WAIT_MS)
    const wsUrl = config.wsUrl
    if (!wsUrl) {
      throw new Error('Protobuf IPC WebSocket 配置无效')
    }

    const requestId = createRequestId()
    const envelope = encodeRpcEnvelope({
      requestId,
      method,
      payload,
      schemaVersion: PROTO_SCHEMA_VERSION,
      timestampMs: Date.now(),
    })

    return new Promise<Uint8Array>((resolve, reject) => {
      const timer = window.setTimeout(() => {
        this.pending.delete(requestId)
        reject(new Error(`Protobuf IPC WebSocket 请求超时: ${method}`))
      }, timeoutMs)
      this.pending.set(requestId, { resolve, reject, timer })

      this.openSocket(wsUrl)
        .then(socket => {
          socket.send(envelope)
        })
        .catch(error => {
          window.clearTimeout(timer)
          this.pending.delete(requestId)
          reject(error)
        })
    })
  }

  private openSocket(wsUrl: string): Promise<WebSocket> {
    if (this.socketPromise) {
      return this.socketPromise
    }

    this.socketPromise = new Promise<WebSocket>((resolve, reject) => {
      const socket = new WebSocket(wsUrl)
      socket.binaryType = 'arraybuffer'

      socket.addEventListener('open', () => resolve(socket), { once: true })
      socket.addEventListener('error', () => {
        this.socketPromise = null
        reject(new Error('Protobuf IPC WebSocket 连接失败'))
      }, { once: true })
      socket.addEventListener('message', event => {
        void this.handleMessage(event.data)
      })
      socket.addEventListener('close', () => {
        this.socketPromise = null
        this.rejectPending(new Error('Protobuf IPC WebSocket 已关闭'))
      })
    })

    return this.socketPromise
  }

  private async handleMessage(data: unknown) {
    try {
      const bytes = await normalizeWebSocketBytes(data)
      const frame = decodeBinaryFrame(bytes)
      if (frame.frameType === BINARY_FRAME_EVENT) {
        dispatchProtoEvent(frame.payload)
        return
      }

      const response = decodeRpcResponse(frame.payload)
      const pending = this.pending.get(response.requestId)
      if (!pending) {
        return
      }

      window.clearTimeout(pending.timer)
      this.pending.delete(response.requestId)

      if (response.error) {
        pending.reject(new ProtoIpcError(response.error))
        return
      }
      pending.resolve(response.payload)
    } catch (error) {
      console.warn('Protobuf IPC WebSocket message ignored:', error)
    }
  }

  private rejectPending(error: Error) {
    for (const [requestId, pending] of this.pending) {
      window.clearTimeout(pending.timer)
      pending.reject(error)
      this.pending.delete(requestId)
    }
  }
}

function decodeBinaryFrame(bytes: Uint8Array): { frameType: number; payload: Uint8Array } {
  if (bytes.length === 0) {
    throw new Error('Protobuf IPC WebSocket 空帧')
  }
  const frameType = bytes[0]
  if (frameType === BINARY_FRAME_RESPONSE || frameType === BINARY_FRAME_EVENT) {
    return { frameType, payload: bytes.slice(1) }
  }
  return { frameType: BINARY_FRAME_RESPONSE, payload: bytes }
}

const sharedBinaryClient = new BinaryWebSocketProtoIpcClient()
const sharedRawClient = new RawProtoIpcClient()

export class ProtoIpcClient {
  private readonly binaryClient = sharedBinaryClient
  private readonly rawClient = sharedRawClient

  async request(method: string, payload: Uint8Array, timeoutMs = DEFAULT_TIMEOUT_MS): Promise<Uint8Array> {
    if (shouldPreferRawTransport()) {
      return this.rawClient.request(method, payload, timeoutMs)
    }
    if (!getBinaryConfig() && isRawChannelAvailable()) {
      try {
        await this.binaryClient.ensureConnected(RAW_FALLBACK_CONFIG_WAIT_MS)
      } catch (error) {
        rawFallbackPreferredUntil = Date.now() + RAW_FALLBACK_COOLDOWN_MS
        console.warn('Wails3 Proto IPC WebSocket config not ready, using raw Protobuf channel:', error)
        return this.rawClient.request(method, payload, timeoutMs)
      }
    }
    try {
      return await this.binaryClient.request(method, payload, timeoutMs)
    } catch (error) {
      if (!shouldUseRawFallback(error)) {
        throw error
      }
      rawFallbackPreferredUntil = Date.now() + RAW_FALLBACK_COOLDOWN_MS
      console.warn('Protobuf IPC WebSocket unavailable, using Wails3 raw Protobuf channel:', error)
      return this.rawClient.request(method, payload, timeoutMs)
    }
  }

  onEvent(eventName: string, callback: (event: RpcEvent) => void): () => void {
    installWailsEventHook()
    void this.binaryClient.ensureConnected().catch(error => {
      if (!isRawChannelAvailable()) {
        console.warn('Protobuf IPC event WebSocket unavailable:', error)
      }
    })
    return onProtoEvent(eventName, callback)
  }
}

export function protoIpcRequest(method: string, payload: Uint8Array, timeoutMs = DEFAULT_TIMEOUT_MS): Promise<Uint8Array> {
  return new ProtoIpcClient().request(method, payload, timeoutMs)
}

export function isProtoIpcAvailable(): boolean {
  return Boolean(getBinaryConfig()?.wsUrl || isRawChannelAvailable())
}

export function onProtoEvent(eventName: string, callback: (event: RpcEvent) => void): () => void {
  const listeners = eventListeners.get(eventName) ?? new Set<(event: RpcEvent) => void>()
  listeners.add(callback)
  eventListeners.set(eventName, listeners)
  return () => {
    listeners.delete(callback)
    if (listeners.size === 0) {
      eventListeners.delete(eventName)
    }
  }
}

export function encodeRawMessage(payload: Uint8Array): string {
  return RAW_MESSAGE_PREFIX + bytesToBase64(payload)
}

export function decodeRawMessage(message: string): Uint8Array {
  if (!message.startsWith(RAW_MESSAGE_PREFIX)) {
    throw new Error('Protobuf IPC 响应前缀无效')
  }
  return base64ToBytes(message.slice(RAW_MESSAGE_PREFIX.length))
}

function installWailsEventHook() {
  const bridge = ensureBridge()
  if (bridge.__traceProtoDispatchHook && bridge.dispatchWailsEvent === dispatchWailsEvent) {
    return
  }
  if (bridge.dispatchWailsEvent !== dispatchWailsEvent) {
    bridge.__traceProtoPreviousDispatch = bridge.dispatchWailsEvent
  }
  bridge.dispatchWailsEvent = dispatchWailsEvent
  bridge.__traceProtoDispatchHook = true
}

function dispatchWailsEvent(event: WailsEvent) {
  if (event.name === RESPONSE_EVENT_NAME && typeof event.data === 'string') {
    handleRawResponse(event.data)
    return
  }
  if (event.name === RAW_EVENT_NAME && typeof event.data === 'string') {
    dispatchProtoEvent(decodeRawMessage(event.data))
    return
  }
  const previous = getWailsWindow()._wails?.__traceProtoPreviousDispatch
  if (previous && previous !== dispatchWailsEvent) {
    previous(event)
  }
}

function handleRawResponse(rawMessage: string) {
  const response = decodeRpcResponse(decodeRawMessage(rawMessage))
  const pending = pendingRequests.get(response.requestId)
  if (!pending) {
    return
  }

  window.clearTimeout(pending.timer)
  pendingRequests.delete(response.requestId)

  if (response.error) {
    pending.reject(new ProtoIpcError(response.error))
    return
  }
  pending.resolve(response.payload)
}

function dispatchProtoEvent(bytes: Uint8Array) {
  const event = decodeRpcEvent(bytes)
  if (!event.eventName) {
    return
  }
  const listeners = eventListeners.get(event.eventName)
  if (!listeners) {
    return
  }
  if (isDuplicateEvent(event)) {
    return
  }
  for (const listener of listeners) {
    listener(event)
  }
}

function isDuplicateEvent(event: RpcEvent): boolean {
  if (!event.eventId) {
    return false
  }
  const now = Date.now()
  pruneSeenEventIds(now)
  if (seenEventIds.has(event.eventId)) {
    return true
  }
  seenEventIds.set(event.eventId, now)
  return false
}

function pruneSeenEventIds(now: number) {
  if (seenEventIds.size <= EVENT_DEDUP_MAX) {
    return
  }
  for (const [eventId, seenAt] of seenEventIds) {
    if (now - seenAt > EVENT_DEDUP_TTL_MS || seenEventIds.size > EVENT_DEDUP_MAX) {
      seenEventIds.delete(eventId)
    }
  }
}

function resolveInvoke(): (message: string) => void {
  const appWindow = getWailsWindow()
  const bridgeInvoke = appWindow._wails?.invoke
  if (bridgeInvoke) {
    return bridgeInvoke
  }
  const chromePostMessage = appWindow.chrome?.webview?.postMessage
  if (chromePostMessage) {
    return chromePostMessage.bind(appWindow.chrome?.webview)
  }
  const webkitPostMessage = appWindow.webkit?.messageHandlers?.external?.postMessage
  if (webkitPostMessage) {
    return webkitPostMessage.bind(appWindow.webkit?.messageHandlers?.external)
  }
  const androidInvoke = appWindow.wails?.invoke
  if (androidInvoke) {
    return androidInvoke
  }
  throw new Error('当前环境不可用 Protobuf IPC raw message 通道')
}

function isRawChannelAvailable(): boolean {
  const appWindow = getWailsWindow()
  return Boolean(
    appWindow._wails?.invoke ||
    appWindow.chrome?.webview?.postMessage ||
    appWindow.webkit?.messageHandlers?.external?.postMessage ||
    appWindow.wails?.invoke,
  )
}

function shouldUseRawFallback(error: unknown): boolean {
  if (error instanceof ProtoIpcError) {
    return false
  }
  return isRawChannelAvailable()
}

function shouldPreferRawTransport(): boolean {
  return Date.now() < rawFallbackPreferredUntil && isRawChannelAvailable()
}

function ensureBridge(): WailsRuntimeBridge {
  const appWindow = getWailsWindow()
  if (!appWindow._wails) {
    appWindow._wails = {}
  }
  return appWindow._wails
}

function getWailsWindow(): WailsWindow {
  return window as WailsWindow
}

function getBinaryConfig(): TraceProtoIpcConfig | null {
  const config = getWailsWindow().__TRACE_PROTO_IPC__
  if (!config?.wsUrl) {
    return null
  }
  return config
}

function waitForBinaryConfig(timeoutMs: number): Promise<TraceProtoIpcConfig> {
  const existing = getBinaryConfig()
  if (existing) {
    return Promise.resolve(existing)
  }

  return new Promise((resolve, reject) => {
    const timer = window.setTimeout(() => {
      window.removeEventListener('trace-proto-config', onConfig)
      reject(new Error('Protobuf IPC WebSocket 配置未注入'))
    }, timeoutMs)

    function onConfig(event: Event) {
      const detail = event instanceof CustomEvent ? event.detail as TraceProtoIpcConfig | undefined : undefined
      const config = detail?.wsUrl ? detail : getBinaryConfig()
      if (!config?.wsUrl) {
        return
      }
      window.clearTimeout(timer)
      window.removeEventListener('trace-proto-config', onConfig)
      resolve(config)
    }

    window.addEventListener('trace-proto-config', onConfig)
    const config = getBinaryConfig()
    if (config?.wsUrl) {
      window.clearTimeout(timer)
      window.removeEventListener('trace-proto-config', onConfig)
      resolve(config)
    }
  })
}

async function normalizeWebSocketBytes(data: unknown): Promise<Uint8Array> {
  if (data instanceof ArrayBuffer) {
    return new Uint8Array(data)
  }
  if (data instanceof Blob) {
    return new Uint8Array(await data.arrayBuffer())
  }
  if (typeof data === 'string') {
    return base64ToBytes(data)
  }
  throw new Error('Protobuf IPC WebSocket 响应类型无效')
}

function createRequestId(): string {
  const randomUUID = globalThis.crypto?.randomUUID?.()
  if (randomUUID) {
    return randomUUID
  }
  return `req-${Date.now().toString(36)}-${Math.random().toString(36).slice(2)}`
}

function bytesToBase64(bytes: Uint8Array): string {
  let binary = ''
  for (let index = 0; index < bytes.length; index += 1) {
    binary += String.fromCharCode(bytes[index])
  }
  return window.btoa(binary)
}

function base64ToBytes(value: string): Uint8Array {
  const binary = window.atob(value)
  const bytes = new Uint8Array(binary.length)
  for (let index = 0; index < binary.length; index += 1) {
    bytes[index] = binary.charCodeAt(index)
  }
  return bytes
}
