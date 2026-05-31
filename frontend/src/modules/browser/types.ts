export interface BrowserProfile {
  profileId: string
  profileName: string
  userDataDir: string
  coreId: string
  fingerprintArgs: string[]
  proxyId: string
  proxyConfig: string
  proxyBindSourceId?: string
  proxyBindSourceUrl?: string
  proxyBindName?: string
  proxyBindUpdatedAt?: string
  autoProxySwitchEnabled?: boolean
  autoProxySwitchGroupName?: string
  autoProxySwitchMode?: 'interval' | 'manual' | string
  autoProxySwitchIntervalM?: number
  autoProxySwitchRotateByGroup?: boolean
  autoProxySwitchLastProxyId?: string
  launchArgs: string[]
  tags: string[]
  keywords: string[]
  groupId?: string
  running: boolean
  debugPort: number
  debugReady: boolean
  pid: number
  runtimeWarning: string
  lastError: string
  createdAt: string
  updatedAt: string
  lastStartAt?: string
  lastStopAt?: string
  launchCode?: string
  instanceMarkerIndex?: number
  instanceMarker?: string
}

export interface BrowserProfileInput {
  profileName: string
  userDataDir: string
  coreId: string
  fingerprintArgs: string[]
  proxyId: string
  proxyConfig: string
  autoProxySwitchEnabled?: boolean
  autoProxySwitchGroupName?: string
  autoProxySwitchMode?: 'interval' | 'manual' | string
  autoProxySwitchIntervalM?: number
  autoProxySwitchRotateByGroup?: boolean
  launchArgs: string[]
  tags: string[]
  keywords: string[]
  groupId?: string
}

export interface BrowserTab {
  tabId: string
  title: string
  url: string
  active: boolean
}

export interface BrowserSettings {
  userDataRoot: string
  defaultFingerprintArgs: string[]
  defaultLaunchArgs: string[]
  defaultProxy: string
  startReadyTimeoutMs: number
  startStableWindowMs: number
}

export interface BrowserStartURL {
  name: string
  url: string
}

export interface BrowserCore {
  coreId: string
  coreName: string
  corePath: string
  isDefault: boolean
}

export interface BrowserCoreInput {
  coreId: string
  coreName: string
  corePath: string
  isDefault: boolean
}

export interface BrowserCoreValidateResult {
  valid: boolean
  message: string
}

export interface BrowserProxy {
  proxyId: string
  proxyName: string
  proxyConfig: string
  dnsServers?: string
  groupName?: string
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

export interface ProxyIPHealthResult {
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
  rawData: Record<string, any>
  updatedAt: string
}

export interface BrowserCoreExtended {
  coreId: string
  chromeVersion: string
  instanceCount: number
}

export interface BrowserExtension {
  extensionId: string
  name: string
  version: string
  manifestVersion: number
  description: string
  sourceType: string
  sourceUrl: string
  installDir: string
  packagePath: string
  manifestJson: string
  boundCount: number
  autoBindEnabled: boolean
  autoBindMode: 'shared' | 'exclusive' | string
  createdAt: string
  updatedAt: string
}

export interface BrowserExtensionBinding {
  id: number
  profileId: string
  profileName: string
  extensionId: string
  extensionName: string
  extensionVersion: string
  mode: 'shared' | 'exclusive' | string
  enabled: boolean
  exclusiveDir: string
  createdAt: string
  updatedAt: string
}

export type BrowserExtensionImportMode = 'ask' | 'overwrite' | 'new' | 'cancel'

export interface BrowserExtensionImportInput {
  path: string
  mode?: BrowserExtensionImportMode | string
  existing?: string
}

export interface BrowserExtensionImportResult {
  cancelled: boolean
  duplicate: boolean
  message: string
  existing?: BrowserExtension | null
  extension?: BrowserExtension | null
}

export interface BrowserExtensionAssignInput {
  extensionId: string
  profileIds: string[]
  mode: 'shared' | 'exclusive' | string
  enabled: boolean
}

export interface BrowserExtensionAutoBindInput {
  extensionId: string
  enabled: boolean
  mode: 'shared' | 'exclusive' | string
}

export interface BrowserExtensionUnassignInput {
  extensionId: string
  profileIds: string[]
}

export interface BrowserExtensionSyncDataInput {
  extensionId: string
  sourceProfileId: string
  targetProfileIds: string[]
}

export interface CookieInfo {
  name: string
  value: string
  domain: string
  path: string
  expires: number
  httpOnly: boolean
  secure: boolean
  sameSite: string
}

export interface CookieImportResult {
  imported: number
  skipped: number
}

export interface SnapshotInfo {
  snapshotId: string
  profileId: string
  name: string
  sizeMB: number
  createdAt: string
}

export interface BrowserBookmark {
  name: string
  url: string
}

export type DefaultContentScope = 'tag' | 'group'

export interface DefaultContentRule {
  ruleId: string
  scope: DefaultContentScope
  targetId?: string
  targetName: string
  startUrls: BrowserStartURL[]
  bookmarks: BrowserBookmark[]
  enabled: boolean
  applyToChilds?: boolean
  includeGlobalDefaults?: boolean
}


// 分组相关类型
export interface BrowserGroup {
  groupId: string
  groupName: string
  parentId: string
  sortOrder: number
  createdAt: string
  updatedAt: string
}

export interface BrowserGroupInput {
  groupName: string
  parentId: string
  sortOrder: number
}

export interface BrowserGroupWithCount extends BrowserGroup {
  instanceCount: number
}

export interface WindowSyncCandidate {
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

export interface WindowSyncStartInput {
  profileIds: string[]
  masterProfileId: string
}

export type WindowSyncLayoutMode = 'grid' | 'stack' | 'custom'

export interface WindowSyncLayoutSettings {
  mode: WindowSyncLayoutMode | string
  scope?: 'app-screen' | 'toolbar-screen' | 'all-screens' | string
  width: number
  height: number
  gapX: number
  gapY: number
  perRow: number
  updatedAt?: string
}

export interface WindowSyncState {
  sessionId: string
  active: boolean
  paused: boolean
  masterProfileId: string
  profileIds: string[]
  windows: WindowSyncCandidate[]
  masterColor: string
  syncKeyboard: boolean
  syncMouse: boolean
  layout: WindowSyncLayoutSettings
  startedAt: string
  updatedAt: string
}

export interface WindowSyncSettings {
  masterColor: string
  syncKeyboard: boolean
  syncMouse: boolean
}

export interface WindowSyncBatchInputDifferentItem {
  profileId: string
  text: string
}

export interface WindowSyncBatchInputResultItem {
  profileId: string
  profileName: string
  master: boolean
  success: boolean
  error: string
}

export interface WindowSyncBatchInputResult {
  total: number
  success: number
  failed: number
  results: WindowSyncBatchInputResultItem[]
}

export interface WindowSyncActionResultItem {
  profileId: string
  profileName: string
  master: boolean
  success: boolean
  error: string
}

export interface WindowSyncActionResult {
  total: number
  success: number
  failed: number
  results: WindowSyncActionResultItem[]
}
