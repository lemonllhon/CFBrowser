import {
  METHOD_WINDOW_SYNC_BATCH_INPUT_DIFFERENT,
  METHOD_WINDOW_SYNC_BATCH_INPUT_SAME,
  METHOD_WINDOW_SYNC_CANDIDATE_LIST,
  METHOD_WINDOW_SYNC_CLOSE_BLANK_TABS,
  METHOD_WINDOW_SYNC_CLOSE_CURRENT_TAB,
  METHOD_WINDOW_SYNC_CLOSE_OTHER_TABS,
  METHOD_WINDOW_SYNC_LAYOUT_APPLY,
  METHOD_WINDOW_SYNC_LAYOUT_SETTINGS_GET,
  METHOD_WINDOW_SYNC_LAYOUT_SETTINGS_SAVE,
  METHOD_WINDOW_SYNC_OPEN_URLS,
  METHOD_WINDOW_SYNC_PAUSE,
  METHOD_WINDOW_SYNC_RESUME,
  METHOD_WINDOW_SYNC_SETTINGS_GET,
  METHOD_WINDOW_SYNC_SETTINGS_SAVE,
  METHOD_WINDOW_SYNC_SHOW_ALL,
  METHOD_WINDOW_SYNC_START,
  METHOD_WINDOW_SYNC_STATE_GET,
  METHOD_WINDOW_SYNC_STOP,
  METHOD_WINDOW_SYNC_TOOLBAR_RESIZE,
} from './envelope'
import {
  WireType,
  concatBytes,
  decodeString,
  decodeVarintField,
  encodeBoolField,
  encodeBytesField,
  encodeInt32Field,
  encodeStringField,
  readFields,
} from './protobuf'
import { ProtoIpcClient } from './transport'

const windowSyncProtoClient = new ProtoIpcClient()
const EVENT_WINDOW_SYNC_STATE_CHANGED = 'window-sync:state-changed'

export type ProtoWindowSyncCandidate = {
  profileId: string
  profileName: string
  debugPort: number
  pid: number
  running: boolean
  debugReady: boolean
  role?: 'master' | 'controlled' | string
  master?: boolean
  canSync: boolean
  canAutoStart?: boolean
  unavailable?: string
}

export type ProtoWindowSyncStartInput = {
  profileIds: string[]
  masterProfileId: string
}

export type ProtoWindowSyncLayoutSettings = {
  mode: 'grid' | 'stack' | 'custom' | string
  scope?: 'app-screen' | 'toolbar-screen' | 'all-screens' | string
  width: number
  height: number
  gapX: number
  gapY: number
  perRow: number
  updatedAt?: string
}

export type ProtoWindowSyncSettings = {
  masterColor: string
  syncKeyboard: boolean
  syncMouse: boolean
}

export type ProtoWindowSyncState = {
  sessionId: string
  active: boolean
  paused: boolean
  masterProfileId: string
  profileIds: string[]
  windows: ProtoWindowSyncCandidate[]
  masterColor: string
  syncKeyboard: boolean
  syncMouse: boolean
  layout: ProtoWindowSyncLayoutSettings
  startedAt: string
  updatedAt: string
}

export type ProtoWindowSyncBatchInputDifferentItem = {
  profileId: string
  text: string
}

export type ProtoWindowSyncBatchInputResultItem = {
  profileId: string
  profileName: string
  master: boolean
  success: boolean
  error: string
}

export type ProtoWindowSyncBatchInputResult = {
  total: number
  success: number
  failed: number
  results: ProtoWindowSyncBatchInputResultItem[]
}

export type ProtoWindowSyncActionResultItem = {
  profileId: string
  profileName: string
  master: boolean
  success: boolean
  error: string
}

export type ProtoWindowSyncActionResult = {
  total: number
  success: number
  failed: number
  results: ProtoWindowSyncActionResultItem[]
}

export type ProtoWindowSyncToolbarResizeRequest = {
  width: number
  height: number
}

export async function listWindowSyncCandidates(): Promise<ProtoWindowSyncCandidate[]> {
  const payload = await windowSyncProtoClient.request(METHOD_WINDOW_SYNC_CANDIDATE_LIST, new Uint8Array())
  return decodeWindowSyncCandidateListResponse(payload).candidates
}

export async function startWindowSync(input: ProtoWindowSyncStartInput): Promise<ProtoWindowSyncState | null> {
  const payload = await windowSyncProtoClient.request(METHOD_WINDOW_SYNC_START, encodeWindowSyncStartRequest(input), 30000)
  return decodeWindowSyncStateResponse(payload).state
}

export async function getWindowSyncState(): Promise<ProtoWindowSyncState | null> {
  const payload = await windowSyncProtoClient.request(METHOD_WINDOW_SYNC_STATE_GET, new Uint8Array())
  return decodeWindowSyncStateResponse(payload).state
}

export async function stopWindowSync(): Promise<ProtoWindowSyncState | null> {
  const payload = await windowSyncProtoClient.request(METHOD_WINDOW_SYNC_STOP, new Uint8Array())
  return decodeWindowSyncStateResponse(payload).state
}

export async function pauseWindowSync(): Promise<ProtoWindowSyncState | null> {
  const payload = await windowSyncProtoClient.request(METHOD_WINDOW_SYNC_PAUSE, new Uint8Array())
  return decodeWindowSyncStateResponse(payload).state
}

export async function resumeWindowSync(): Promise<ProtoWindowSyncState | null> {
  const payload = await windowSyncProtoClient.request(METHOD_WINDOW_SYNC_RESUME, new Uint8Array())
  return decodeWindowSyncStateResponse(payload).state
}

export async function showAllWindowSyncWindows(): Promise<ProtoWindowSyncState | null> {
  const payload = await windowSyncProtoClient.request(METHOD_WINDOW_SYNC_SHOW_ALL, new Uint8Array())
  return decodeWindowSyncStateResponse(payload).state
}

export async function getWindowSyncSettings(): Promise<ProtoWindowSyncSettings> {
  const payload = await windowSyncProtoClient.request(METHOD_WINDOW_SYNC_SETTINGS_GET, new Uint8Array())
  return decodeWindowSyncSettingsResponse(payload).settings
}

export async function saveWindowSyncSettings(settings: ProtoWindowSyncSettings): Promise<ProtoWindowSyncState | null> {
  const payload = await windowSyncProtoClient.request(METHOD_WINDOW_SYNC_SETTINGS_SAVE, encodeWindowSyncSettings(settings))
  return decodeWindowSyncStateResponse(payload).state
}

export async function getWindowSyncLayoutSettings(): Promise<ProtoWindowSyncLayoutSettings> {
  const payload = await windowSyncProtoClient.request(METHOD_WINDOW_SYNC_LAYOUT_SETTINGS_GET, new Uint8Array())
  return decodeWindowSyncLayoutSettingsResponse(payload).layout
}

export async function saveWindowSyncLayoutSettings(settings: ProtoWindowSyncLayoutSettings): Promise<ProtoWindowSyncLayoutSettings> {
  const payload = await windowSyncProtoClient.request(METHOD_WINDOW_SYNC_LAYOUT_SETTINGS_SAVE, encodeWindowSyncLayoutSettings(settings))
  return decodeWindowSyncLayoutSettingsResponse(payload).layout
}

export async function applyWindowSyncLayout(settings: ProtoWindowSyncLayoutSettings): Promise<ProtoWindowSyncState | null> {
  const payload = await windowSyncProtoClient.request(METHOD_WINDOW_SYNC_LAYOUT_APPLY, encodeWindowSyncLayoutSettings(settings), 30000)
  return decodeWindowSyncStateResponse(payload).state
}

export async function windowSyncBatchInputSame(text: string): Promise<ProtoWindowSyncBatchInputResult> {
  const payload = await windowSyncProtoClient.request(METHOD_WINDOW_SYNC_BATCH_INPUT_SAME, concatBytes([encodeStringField(1, text)]))
  return decodeWindowSyncBatchInputResult(payload)
}

export async function windowSyncBatchInputDifferent(items: ProtoWindowSyncBatchInputDifferentItem[]): Promise<ProtoWindowSyncBatchInputResult> {
  const payload = await windowSyncProtoClient.request(METHOD_WINDOW_SYNC_BATCH_INPUT_DIFFERENT, encodeWindowSyncBatchInputDifferentRequest(items))
  return decodeWindowSyncBatchInputResult(payload)
}

export async function windowSyncCloseOtherTabs(): Promise<ProtoWindowSyncActionResult> {
  const payload = await windowSyncProtoClient.request(METHOD_WINDOW_SYNC_CLOSE_OTHER_TABS, new Uint8Array())
  return decodeWindowSyncActionResult(payload)
}

export async function windowSyncCloseCurrentTab(): Promise<ProtoWindowSyncActionResult> {
  const payload = await windowSyncProtoClient.request(METHOD_WINDOW_SYNC_CLOSE_CURRENT_TAB, new Uint8Array())
  return decodeWindowSyncActionResult(payload)
}

export async function windowSyncCloseBlankTabs(): Promise<ProtoWindowSyncActionResult> {
  const payload = await windowSyncProtoClient.request(METHOD_WINDOW_SYNC_CLOSE_BLANK_TABS, new Uint8Array())
  return decodeWindowSyncActionResult(payload)
}

export async function windowSyncOpenUrls(urls: string[]): Promise<ProtoWindowSyncActionResult> {
  const payload = await windowSyncProtoClient.request(METHOD_WINDOW_SYNC_OPEN_URLS, concatBytes(urls.map(url => encodeStringField(1, url))))
  return decodeWindowSyncActionResult(payload)
}

export async function resizeWindowSyncToolbar(width: number, height: number): Promise<boolean> {
  const payload = await windowSyncProtoClient.request(METHOD_WINDOW_SYNC_TOOLBAR_RESIZE, encodeWindowSyncToolbarResizeRequest({ width, height }))
  return decodeWindowSyncToolbarResizeResponse(payload).ok
}

export function onWindowSyncStateChanged(callback: (state: ProtoWindowSyncState | null) => void): () => void {
  return windowSyncProtoClient.onEvent(EVENT_WINDOW_SYNC_STATE_CHANGED, event => {
    callback(decodeWindowSyncStateResponse(event.payload).state)
  })
}

export function encodeWindowSyncStartRequest(message: ProtoWindowSyncStartInput): Uint8Array {
  return concatBytes([
    ...message.profileIds.map(profileId => encodeStringField(1, profileId)),
    encodeStringField(2, message.masterProfileId),
  ])
}

export function encodeWindowSyncLayoutSettings(message: ProtoWindowSyncLayoutSettings): Uint8Array {
  return concatBytes([
    encodeStringField(1, message.mode),
    encodeInt32Field(2, message.width),
    encodeInt32Field(3, message.height),
    encodeInt32Field(4, message.gapX),
    encodeInt32Field(5, message.gapY),
    encodeInt32Field(6, message.perRow),
    encodeStringField(7, message.updatedAt ?? ''),
    encodeStringField(8, message.scope ?? ''),
  ])
}

export function encodeWindowSyncSettings(message: ProtoWindowSyncSettings): Uint8Array {
  return concatBytes([
    encodeStringField(1, message.masterColor),
    encodeBoolField(2, message.syncKeyboard),
    encodeBoolField(3, message.syncMouse),
  ])
}

export function encodeWindowSyncBatchInputDifferentRequest(items: ProtoWindowSyncBatchInputDifferentItem[]): Uint8Array {
  return concatBytes(items.map(item => encodeBytesField(1, encodeWindowSyncBatchInputDifferentItem(item))))
}

export function encodeWindowSyncToolbarResizeRequest(message: ProtoWindowSyncToolbarResizeRequest): Uint8Array {
  return concatBytes([
    encodeInt32Field(1, message.width),
    encodeInt32Field(2, message.height),
  ])
}

export function decodeWindowSyncCandidateListResponse(payload: Uint8Array): { candidates: ProtoWindowSyncCandidate[] } {
  const candidates: ProtoWindowSyncCandidate[] = []
  for (const field of readFields(payload)) {
    if (field.fieldNumber === 1 && field.wireType === WireType.LengthDelimited) {
      candidates.push(decodeWindowSyncCandidate(field.value))
    }
  }
  return { candidates }
}

export function decodeWindowSyncStateResponse(payload: Uint8Array): { state: ProtoWindowSyncState | null } {
  let hasState = false
  let state: ProtoWindowSyncState | null = null
  for (const field of readFields(payload)) {
    if (field.fieldNumber === 1 && field.wireType === WireType.Varint) {
      hasState = Number(decodeVarintField(field.value)) !== 0
      continue
    }
    if (field.fieldNumber === 2 && field.wireType === WireType.LengthDelimited) {
      state = decodeWindowSyncState(field.value)
      hasState = true
    }
  }
  return { state: hasState ? state ?? defaultWindowSyncState() : null }
}

export function decodeWindowSyncLayoutSettingsResponse(payload: Uint8Array): { layout: ProtoWindowSyncLayoutSettings } {
  let layout = defaultWindowSyncLayoutSettings()
  for (const field of readFields(payload)) {
    if (field.fieldNumber === 1 && field.wireType === WireType.LengthDelimited) {
      layout = decodeWindowSyncLayoutSettings(field.value)
    }
  }
  return { layout }
}

export function decodeWindowSyncSettingsResponse(payload: Uint8Array): { settings: ProtoWindowSyncSettings } {
  let settings = defaultWindowSyncSettings()
  for (const field of readFields(payload)) {
    if (field.fieldNumber === 1 && field.wireType === WireType.LengthDelimited) {
      settings = decodeWindowSyncSettings(field.value)
    }
  }
  return { settings }
}

export function decodeWindowSyncBatchInputResult(payload: Uint8Array): ProtoWindowSyncBatchInputResult {
  const result: ProtoWindowSyncBatchInputResult = { total: 0, success: 0, failed: 0, results: [] }
  for (const field of readFields(payload)) {
    if (field.wireType === WireType.Varint) {
      const value = Number(decodeVarintField(field.value))
      switch (field.fieldNumber) {
        case 1:
          result.total = value
          break
        case 2:
          result.success = value
          break
        case 3:
          result.failed = value
          break
      }
      continue
    }
    if (field.fieldNumber === 4 && field.wireType === WireType.LengthDelimited) {
      result.results.push(decodeWindowSyncBatchInputResultItem(field.value))
    }
  }
  return result
}

export function decodeWindowSyncActionResult(payload: Uint8Array): ProtoWindowSyncActionResult {
  const result: ProtoWindowSyncActionResult = { total: 0, success: 0, failed: 0, results: [] }
  for (const field of readFields(payload)) {
    if (field.wireType === WireType.Varint) {
      const value = Number(decodeVarintField(field.value))
      switch (field.fieldNumber) {
        case 1:
          result.total = value
          break
        case 2:
          result.success = value
          break
        case 3:
          result.failed = value
          break
      }
      continue
    }
    if (field.fieldNumber === 4 && field.wireType === WireType.LengthDelimited) {
      result.results.push(decodeWindowSyncActionResultItem(field.value))
    }
  }
  return result
}

export function decodeWindowSyncToolbarResizeResponse(payload: Uint8Array): { ok: boolean } {
  let ok = false
  for (const field of readFields(payload)) {
    if (field.fieldNumber === 1 && field.wireType === WireType.Varint) {
      ok = Number(decodeVarintField(field.value)) !== 0
    }
  }
  return { ok }
}

function encodeWindowSyncBatchInputDifferentItem(message: ProtoWindowSyncBatchInputDifferentItem): Uint8Array {
  return concatBytes([
    encodeStringField(1, message.profileId),
    encodeStringField(2, message.text),
  ])
}

function decodeWindowSyncCandidate(payload: Uint8Array): ProtoWindowSyncCandidate {
  const candidate: ProtoWindowSyncCandidate = {
    profileId: '',
    profileName: '',
    debugPort: 0,
    pid: 0,
    running: false,
    debugReady: false,
    role: '',
    master: false,
    canSync: false,
    canAutoStart: false,
    unavailable: '',
  }
  for (const field of readFields(payload)) {
    if (field.wireType === WireType.LengthDelimited) {
      const text = decodeString(field.value)
      switch (field.fieldNumber) {
        case 1:
          candidate.profileId = text
          break
        case 2:
          candidate.profileName = text
          break
        case 7:
          candidate.role = text
          break
        case 11:
          candidate.unavailable = text
          break
      }
      continue
    }
    if (field.wireType === WireType.Varint) {
      const value = Number(decodeVarintField(field.value))
      switch (field.fieldNumber) {
        case 3:
          candidate.debugPort = value
          break
        case 4:
          candidate.pid = value
          break
        case 5:
          candidate.running = value !== 0
          break
        case 6:
          candidate.debugReady = value !== 0
          break
        case 8:
          candidate.master = value !== 0
          break
        case 9:
          candidate.canSync = value !== 0
          break
        case 10:
          candidate.canAutoStart = value !== 0
          break
      }
    }
  }
  return candidate
}

function decodeWindowSyncLayoutSettings(payload: Uint8Array): ProtoWindowSyncLayoutSettings {
  const layout: ProtoWindowSyncLayoutSettings = {
    mode: '',
    width: 0,
    height: 0,
    gapX: 0,
    gapY: 0,
    perRow: 0,
    updatedAt: '',
    scope: '',
  }
  for (const field of readFields(payload)) {
    if (field.wireType === WireType.LengthDelimited) {
      const text = decodeString(field.value)
      switch (field.fieldNumber) {
        case 1:
          layout.mode = text
          break
        case 7:
          layout.updatedAt = text
          break
        case 8:
          layout.scope = text
          break
      }
      continue
    }
    if (field.wireType === WireType.Varint) {
      const value = Number(decodeVarintField(field.value))
      switch (field.fieldNumber) {
        case 2:
          layout.width = value
          break
        case 3:
          layout.height = value
          break
        case 4:
          layout.gapX = value
          break
        case 5:
          layout.gapY = value
          break
        case 6:
          layout.perRow = value
          break
      }
    }
  }
  return layout
}

function decodeWindowSyncSettings(payload: Uint8Array): ProtoWindowSyncSettings {
  const settings: ProtoWindowSyncSettings = {
    masterColor: '',
    syncKeyboard: false,
    syncMouse: false,
  }
  for (const field of readFields(payload)) {
    if (field.fieldNumber === 1 && field.wireType === WireType.LengthDelimited) {
      settings.masterColor = decodeString(field.value)
      continue
    }
    if (field.wireType === WireType.Varint) {
      const value = Number(decodeVarintField(field.value)) !== 0
      switch (field.fieldNumber) {
        case 2:
          settings.syncKeyboard = value
          break
        case 3:
          settings.syncMouse = value
          break
      }
    }
  }
  return settings
}

function decodeWindowSyncState(payload: Uint8Array): ProtoWindowSyncState {
  const state = defaultWindowSyncState()
  state.layout = { mode: '', scope: '', width: 0, height: 0, gapX: 0, gapY: 0, perRow: 0, updatedAt: '' }
  state.syncKeyboard = false
  state.syncMouse = false

  for (const field of readFields(payload)) {
    if (field.wireType === WireType.LengthDelimited) {
      switch (field.fieldNumber) {
        case 1:
          state.sessionId = decodeString(field.value)
          break
        case 4:
          state.masterProfileId = decodeString(field.value)
          break
        case 5:
          state.profileIds.push(decodeString(field.value))
          break
        case 6:
          state.windows.push(decodeWindowSyncCandidate(field.value))
          break
        case 7:
          state.masterColor = decodeString(field.value)
          break
        case 10:
          state.layout = decodeWindowSyncLayoutSettings(field.value)
          break
        case 11:
          state.startedAt = decodeString(field.value)
          break
        case 12:
          state.updatedAt = decodeString(field.value)
          break
      }
      continue
    }
    if (field.wireType === WireType.Varint) {
      const value = Number(decodeVarintField(field.value))
      switch (field.fieldNumber) {
        case 2:
          state.active = value !== 0
          break
        case 3:
          state.paused = value !== 0
          break
        case 8:
          state.syncKeyboard = value !== 0
          break
        case 9:
          state.syncMouse = value !== 0
          break
      }
    }
  }
  return state
}

function decodeWindowSyncBatchInputResultItem(payload: Uint8Array): ProtoWindowSyncBatchInputResultItem {
  const item: ProtoWindowSyncBatchInputResultItem = {
    profileId: '',
    profileName: '',
    master: false,
    success: false,
    error: '',
  }
  decodeWindowSyncResultItem(payload, item)
  return item
}

function decodeWindowSyncActionResultItem(payload: Uint8Array): ProtoWindowSyncActionResultItem {
  const item: ProtoWindowSyncActionResultItem = {
    profileId: '',
    profileName: '',
    master: false,
    success: false,
    error: '',
  }
  decodeWindowSyncResultItem(payload, item)
  return item
}

function decodeWindowSyncResultItem(payload: Uint8Array, item: ProtoWindowSyncBatchInputResultItem | ProtoWindowSyncActionResultItem) {
  for (const field of readFields(payload)) {
    if (field.wireType === WireType.LengthDelimited) {
      const text = decodeString(field.value)
      switch (field.fieldNumber) {
        case 1:
          item.profileId = text
          break
        case 2:
          item.profileName = text
          break
        case 5:
          item.error = text
          break
      }
      continue
    }
    if (field.wireType === WireType.Varint) {
      const value = Number(decodeVarintField(field.value)) !== 0
      switch (field.fieldNumber) {
        case 3:
          item.master = value
          break
        case 4:
          item.success = value
          break
      }
    }
  }
}

function defaultWindowSyncState(): ProtoWindowSyncState {
  return {
    sessionId: '',
    active: false,
    paused: false,
    masterProfileId: '',
    profileIds: [],
    windows: [],
    masterColor: '#2563eb',
    syncKeyboard: true,
    syncMouse: true,
    layout: defaultWindowSyncLayoutSettings(),
    startedAt: '',
    updatedAt: '',
  }
}

function defaultWindowSyncLayoutSettings(): ProtoWindowSyncLayoutSettings {
  return {
    mode: 'grid',
    scope: 'app-screen',
    width: 1500,
    height: 500,
    gapX: 10,
    gapY: 10,
    perRow: 2,
  }
}

function defaultWindowSyncSettings(): ProtoWindowSyncSettings {
  return {
    masterColor: '#2563eb',
    syncKeyboard: true,
    syncMouse: true,
  }
}
