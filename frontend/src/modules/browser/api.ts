import type { BrowserProfile, BrowserProfileInput, BrowserTab, BrowserSettings, BrowserCore, BrowserCoreInput, BrowserCoreValidateResult, BrowserProxy, BrowserCoreExtended, BrowserExtension, BrowserExtensionAssignInput, BrowserExtensionBinding, BrowserExtensionImportInput, BrowserExtensionImportResult, BrowserExtensionUnassignInput, CookieInfo, CookieImportResult, SnapshotInfo, BrowserBookmark, BrowserStartURL, BrowserGroup, BrowserGroupInput, BrowserGroupWithCount, ProxyIPHealthResult, DefaultContentRule, WindowSyncCandidate, WindowSyncLayoutSettings, WindowSyncSettings, WindowSyncStartInput, WindowSyncState } from './types'

const getBindings = async () => {
  try {
    return await import('../../wailsjs/go/main/App')
  } catch {
    return null
  }
}

let mockProfiles: BrowserProfile[] = [
  {
    profileId: 'mock-1',
    profileName: '默认指纹配置',
    userDataDir: 'data/default',
    coreId: 'default',
    fingerprintArgs: ['--fingerprint-brand=Chrome', '--fingerprint-platform=windows'],
    proxyId: '',
    proxyConfig: '',
    launchArgs: ['--disable-features=Translate'],
    tags: ['默认'],
    keywords: [],
    running: false,
    debugPort: 0,
    debugReady: false,
    pid: 0,
    runtimeWarning: '',
    lastError: '',
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
  },
]

let mockCores: BrowserCore[] = []

let mockProxies: BrowserProxy[] = []

let mockGroups: BrowserGroup[] = []

let mockDefaultContentRules: DefaultContentRule[] = []

let mockExtensions: BrowserExtension[] = []
let mockExtensionBindings: BrowserExtensionBinding[] = []

// ============================================================================
// Profile API
// ============================================================================

export async function fetchBrowserProfiles(): Promise<BrowserProfile[]> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserProfileList) {
    return (await bindings.BrowserProfileList()) || []
  }
  return mockProfiles
}

export async function fetchBrowserProfilesByTag(tag: string): Promise<BrowserProfile[]> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserProfileListByTag) {
    return (await bindings.BrowserProfileListByTag(tag)) || []
  }
  return mockProfiles.filter(p => p.tags?.includes(tag))
}

export async function fetchAllTags(): Promise<string[]> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserGetAllTags) {
    return (await bindings.BrowserGetAllTags()) || []
  }
  const set = new Set<string>()
  mockProfiles.forEach(p => p.tags?.forEach(t => set.add(t)))
  return Array.from(set).sort()
}

export async function createBrowserProfile(input: BrowserProfileInput): Promise<BrowserProfile | null> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserProfileCreate) {
    return (await bindings.BrowserProfileCreate(input)) || null
  }
  const profile: BrowserProfile = {
    profileId: `mock-${Date.now()}`,
    ...input,
    keywords: input.keywords || {},
    running: false,
    debugPort: 0,
    debugReady: false,
    pid: 0,
    runtimeWarning: '',
    lastError: '',
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
  }
  mockProfiles = [profile, ...mockProfiles]
  return profile
}

export async function updateBrowserProfile(profileId: string, input: BrowserProfileInput): Promise<BrowserProfile | null> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserProfileUpdate) {
    return (await bindings.BrowserProfileUpdate(profileId, input)) || null
  }
  const index = mockProfiles.findIndex(item => item.profileId === profileId)
  if (index === -1) return null
  mockProfiles[index] = { ...mockProfiles[index], ...input, updatedAt: new Date().toISOString() }
  return mockProfiles[index]
}

export async function deleteBrowserProfile(profileId: string): Promise<boolean> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserProfileDelete) {
    await bindings.BrowserProfileDelete(profileId)
    return true
  }
  mockProfiles = mockProfiles.filter(item => item.profileId !== profileId)
  return true
}

export async function copyBrowserProfile(profileId: string, newName: string): Promise<BrowserProfile | null> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserProfileCopy) {
    return (await bindings.BrowserProfileCopy(profileId, newName)) || null
  }
  // mock
  const src = mockProfiles.find(p => p.profileId === profileId)
  if (!src) return null
  const copy: BrowserProfile = {
    ...src,
    profileId: `mock-${Date.now()}`,
    profileName: newName || src.profileName + ' (副本)',
    userDataDir: `mock-${Date.now()}`,
    running: false,
    debugReady: false,
    runtimeWarning: '',
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
  }
  mockProfiles = [copy, ...mockProfiles]
  return copy
}

// ============================================================================
// Instance API
// ============================================================================

export async function startBrowserInstance(profileId: string): Promise<BrowserProfile | null> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserInstanceStart) {
    return (await bindings.BrowserInstanceStart(profileId)) || null
  }
  mockProfiles = mockProfiles.map(item =>
    item.profileId === profileId ? { ...item, running: true, debugPort: 9222, debugReady: true, pid: Math.floor(Math.random() * 100000), runtimeWarning: '', lastStartAt: new Date().toISOString() } : item
  )
  return mockProfiles.find(item => item.profileId === profileId) || null
}

export async function startBrowserInstanceByCode(code: string): Promise<BrowserProfile | null> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserInstanceStartByCode) {
    return (await bindings.BrowserInstanceStartByCode(code)) || null
  }
  const normalized = code.trim().toUpperCase()
  const profile = mockProfiles.find(item => (item.launchCode || '').toUpperCase() === normalized)
  if (!profile) {
    throw new Error('launch code not found')
  }
  return await startBrowserInstance(profile.profileId)
}

export async function stopBrowserInstance(profileId: string): Promise<BrowserProfile | null> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserInstanceStop) {
    return (await bindings.BrowserInstanceStop(profileId)) || null
  }
  mockProfiles = mockProfiles.map(item =>
    item.profileId === profileId ? { ...item, running: false, debugReady: false, debugPort: 0, pid: 0, runtimeWarning: '', lastStopAt: new Date().toISOString() } : item
  )
  return mockProfiles.find(item => item.profileId === profileId) || null
}

export async function restartBrowserInstance(profileId: string): Promise<BrowserProfile | null> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserInstanceRestart) {
    return (await bindings.BrowserInstanceRestart(profileId)) || null
  }
  await stopBrowserInstance(profileId)
  return await startBrowserInstance(profileId)
}

export async function pinCenterBrowserInstance(profileId: string): Promise<boolean> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserInstancePinCenter) {
    await bindings.BrowserInstancePinCenter(profileId)
    return true
  }
  const goApp = (window as any).go?.main?.App
  if (goApp?.BrowserInstancePinCenter) {
    await goApp.BrowserInstancePinCenter(profileId)
    return true
  }
  return true
}

export async function switchBrowserProfileProxyNow(profileId: string): Promise<BrowserProfile | null> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserProfileSwitchProxyNow) {
    return (await bindings.BrowserProfileSwitchProxyNow(profileId)) || null
  }
  const goApp = (window as any).go?.main?.App
  if (goApp?.BrowserProfileSwitchProxyNow) {
    return (await goApp.BrowserProfileSwitchProxyNow(profileId)) || null
  }
  mockProfiles = mockProfiles.map(item =>
    item.profileId === profileId ? { ...item, autoProxySwitchLastProxyId: `mock-proxy-${Date.now()}`, updatedAt: new Date().toISOString() } : item
  )
  return mockProfiles.find(item => item.profileId === profileId) || null
}

export async function openBrowserUrl(profileId: string, targetUrl: string): Promise<boolean> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserInstanceOpenUrl) {
    return (await bindings.BrowserInstanceOpenUrl(profileId, targetUrl)) === true
  }
  return true
}

export async function fetchBrowserTabs(profileId: string): Promise<BrowserTab[]> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserInstanceGetTabs) {
    return (await bindings.BrowserInstanceGetTabs(profileId)) || []
  }
  return [
    { tabId: 'tab-1', title: '新标签页', url: 'about:blank', active: true },
    { tabId: 'tab-2', title: '示例站点', url: 'https://example.com', active: false },
  ]
}

// ============================================================================
// Window Sync API
// ============================================================================

export async function listWindowSyncCandidates(): Promise<WindowSyncCandidate[]> {
  const bindings: any = await getBindings()
  if (bindings?.WindowSyncListCandidates) {
    return (await bindings.WindowSyncListCandidates()) || []
  }
  const goApp = (window as any).go?.main?.App
  if (goApp?.WindowSyncListCandidates) {
    return (await goApp.WindowSyncListCandidates()) || []
  }
  return mockProfiles
    .filter(profile => profile.running && profile.debugReady && profile.debugPort > 0)
    .map(profile => ({
      profileId: profile.profileId,
      profileName: profile.profileName,
      debugPort: profile.debugPort,
      pid: profile.pid,
      running: profile.running,
      debugReady: profile.debugReady,
      canSync: true,
      unavailable: '',
    }))
}

export async function startWindowSync(input: WindowSyncStartInput): Promise<WindowSyncState | null> {
  const bindings: any = await getBindings()
  if (bindings?.WindowSyncStart) {
    return (await bindings.WindowSyncStart(input)) || null
  }
  const goApp = (window as any).go?.main?.App
  if (goApp?.WindowSyncStart) {
    return (await goApp.WindowSyncStart(input)) || null
  }
  const candidates = await listWindowSyncCandidates()
  const selected = candidates.filter(item => input.profileIds.includes(item.profileId))
  const now = new Date().toISOString()
  return {
    sessionId: `mock-sync-${Date.now()}`,
    active: true,
    paused: false,
    masterProfileId: input.masterProfileId,
    profileIds: input.profileIds,
    windows: selected.map(item => ({
      ...item,
      role: item.profileId === input.masterProfileId ? 'master' : 'controlled',
      master: item.profileId === input.masterProfileId,
    })),
    masterColor: '#2563eb',
    syncKeyboard: true,
    syncMouse: true,
    layout: defaultWindowSyncLayoutSettings(),
    startedAt: now,
    updatedAt: now,
  }
}

export async function getWindowSyncState(): Promise<WindowSyncState | null> {
  const bindings: any = await getBindings()
  if (bindings?.WindowSyncGetState) {
    return (await bindings.WindowSyncGetState()) || null
  }
  const goApp = (window as any).go?.main?.App
  if (goApp?.WindowSyncGetState) {
    return (await goApp.WindowSyncGetState()) || null
  }
  return null
}

export function defaultWindowSyncSettings(): WindowSyncSettings {
  return {
    masterColor: '#2563eb',
    syncKeyboard: true,
    syncMouse: true,
  }
}

export async function getWindowSyncSettings(): Promise<WindowSyncSettings> {
  const bindings: any = await getBindings()
  if (bindings?.WindowSyncGetSettings) {
    return (await bindings.WindowSyncGetSettings()) || defaultWindowSyncSettings()
  }
  const goApp = (window as any).go?.main?.App
  if (goApp?.WindowSyncGetSettings) {
    return (await goApp.WindowSyncGetSettings()) || defaultWindowSyncSettings()
  }
  return defaultWindowSyncSettings()
}

export async function saveWindowSyncSettings(settings: WindowSyncSettings): Promise<WindowSyncState | null> {
  const bindings: any = await getBindings()
  if (bindings?.WindowSyncSaveSettings) {
    return (await bindings.WindowSyncSaveSettings(settings)) || null
  }
  const goApp = (window as any).go?.main?.App
  if (goApp?.WindowSyncSaveSettings) {
    return (await goApp.WindowSyncSaveSettings(settings)) || null
  }
  return null
}

export function defaultWindowSyncLayoutSettings(): WindowSyncLayoutSettings {
  return {
    mode: 'grid',
    width: 1500,
    height: 500,
    gapX: 10,
    gapY: 10,
    perRow: 2,
  }
}

export async function getWindowSyncLayoutSettings(): Promise<WindowSyncLayoutSettings> {
  const bindings: any = await getBindings()
  if (bindings?.WindowSyncGetLayoutSettings) {
    return (await bindings.WindowSyncGetLayoutSettings()) || defaultWindowSyncLayoutSettings()
  }
  const goApp = (window as any).go?.main?.App
  if (goApp?.WindowSyncGetLayoutSettings) {
    return (await goApp.WindowSyncGetLayoutSettings()) || defaultWindowSyncLayoutSettings()
  }
  return defaultWindowSyncLayoutSettings()
}

export async function saveWindowSyncLayoutSettings(settings: WindowSyncLayoutSettings): Promise<WindowSyncLayoutSettings> {
  const bindings: any = await getBindings()
  if (bindings?.WindowSyncSaveLayoutSettings) {
    return (await bindings.WindowSyncSaveLayoutSettings(settings)) || settings
  }
  const goApp = (window as any).go?.main?.App
  if (goApp?.WindowSyncSaveLayoutSettings) {
    return (await goApp.WindowSyncSaveLayoutSettings(settings)) || settings
  }
  return settings
}

export async function applyWindowSyncLayout(settings: WindowSyncLayoutSettings): Promise<WindowSyncState | null> {
  const bindings: any = await getBindings()
  if (bindings?.WindowSyncApplyLayout) {
    return (await bindings.WindowSyncApplyLayout(settings)) || null
  }
  const goApp = (window as any).go?.main?.App
  if (goApp?.WindowSyncApplyLayout) {
    return (await goApp.WindowSyncApplyLayout(settings)) || null
  }
  return null
}

export async function pauseWindowSync(): Promise<WindowSyncState | null> {
  const bindings: any = await getBindings()
  if (bindings?.WindowSyncPause) {
    return (await bindings.WindowSyncPause()) || null
  }
  const goApp = (window as any).go?.main?.App
  if (goApp?.WindowSyncPause) {
    return (await goApp.WindowSyncPause()) || null
  }
  return null
}

export async function resumeWindowSync(): Promise<WindowSyncState | null> {
  const bindings: any = await getBindings()
  if (bindings?.WindowSyncResume) {
    return (await bindings.WindowSyncResume()) || null
  }
  const goApp = (window as any).go?.main?.App
  if (goApp?.WindowSyncResume) {
    return (await goApp.WindowSyncResume()) || null
  }
  return null
}

export async function showAllWindowSyncWindows(): Promise<WindowSyncState | null> {
  const bindings: any = await getBindings()
  if (bindings?.WindowSyncShowAll) {
    return (await bindings.WindowSyncShowAll()) || null
  }
  const goApp = (window as any).go?.main?.App
  if (goApp?.WindowSyncShowAll) {
    return (await goApp.WindowSyncShowAll()) || null
  }
  return null
}

export async function stopWindowSync(): Promise<WindowSyncState | null> {
  const bindings: any = await getBindings()
  if (bindings?.WindowSyncStop) {
    return (await bindings.WindowSyncStop()) || null
  }
  const goApp = (window as any).go?.main?.App
  if (goApp?.WindowSyncStop) {
    return (await goApp.WindowSyncStop()) || null
  }
  return null
}

// ============================================================================
// Settings API
// ============================================================================

export async function fetchBrowserSettings(): Promise<BrowserSettings> {
  const bindings: any = await getBindings()
  if (bindings?.GetBrowserSettings) {
    return (await bindings.GetBrowserSettings()) || { userDataRoot: 'data', defaultFingerprintArgs: [], defaultLaunchArgs: [], defaultProxy: '', startReadyTimeoutMs: 3000, startStableWindowMs: 1200 }
  }
  return { userDataRoot: 'data', defaultFingerprintArgs: [], defaultLaunchArgs: [], defaultProxy: '', startReadyTimeoutMs: 3000, startStableWindowMs: 1200 }
}

export async function saveBrowserSettings(settings: BrowserSettings): Promise<boolean> {
  const bindings: any = await getBindings()
  if (bindings?.SaveBrowserSettings) {
    await bindings.SaveBrowserSettings(settings)
    return true
  }
  return true
}

// ============================================================================
// Core API
// ============================================================================

export async function fetchBrowserCores(): Promise<BrowserCore[]> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserCoreList) {
    return (await bindings.BrowserCoreList()) || []
  }
  return mockCores
}

export async function saveBrowserCore(input: BrowserCoreInput): Promise<boolean> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserCoreSave) {
    await bindings.BrowserCoreSave(input)
    return true
  }
  const index = mockCores.findIndex(c => c.coreId === input.coreId)
  if (index >= 0) {
    mockCores[index] = input
  } else {
    mockCores.push({ ...input, coreId: input.coreId || `core-${Date.now()}` })
  }
  return true
}

export async function deleteBrowserCore(coreId: string): Promise<boolean> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserCoreDelete) {
    await bindings.BrowserCoreDelete(coreId)
    return true
  }
  mockCores = mockCores.filter(c => c.coreId !== coreId)
  return true
}

export async function setDefaultBrowserCore(coreId: string): Promise<boolean> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserCoreSetDefault) {
    await bindings.BrowserCoreSetDefault(coreId)
    return true
  }
  mockCores = mockCores.map(c => ({ ...c, isDefault: c.coreId === coreId }))
  return true
}

export async function validateBrowserCorePath(corePath: string): Promise<BrowserCoreValidateResult> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserCoreValidate) {
    return (await bindings.BrowserCoreValidate(corePath)) || { valid: false, message: '验证失败' }
  }
  return { valid: true, message: '路径有效（模拟）' }
}

export async function fetchCoreExtendedInfo(): Promise<BrowserCoreExtended[]> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserCoreExtendedInfo) {
    return (await bindings.BrowserCoreExtendedInfo()) || []
  }
  return []
}

export async function scanBrowserCores(): Promise<BrowserCore[]> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserCoreScan) {
    return (await bindings.BrowserCoreScan()) || []
  }
  return mockCores
}

export async function BrowserCoreDownload(coreName: string, url: string, proxyConfig?: string): Promise<boolean> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserCoreDownload) {
    await bindings.BrowserCoreDownload(coreName, url, proxyConfig || '')
    return true
  }
  return true
}

// ============================================================================
// Extension API
// ============================================================================

export async function fetchBrowserExtensions(): Promise<BrowserExtension[]> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserExtensionList) {
    return (await bindings.BrowserExtensionList()) || []
  }
  return mockExtensions
}

export async function fetchBrowserExtension(extensionId: string): Promise<BrowserExtension | null> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserExtensionGet) {
    return (await bindings.BrowserExtensionGet(extensionId)) || null
  }
  return mockExtensions.find(item => item.extensionId === extensionId) || null
}

export async function deleteBrowserExtension(extensionId: string): Promise<boolean> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserExtensionDelete) {
    await bindings.BrowserExtensionDelete(extensionId)
    return true
  }
  const extension = mockExtensions.find(item => item.extensionId === extensionId)
  if (extension && extension.boundCount > 0) {
    throw new Error(`扩展插件已绑定 ${extension.boundCount} 个实例，不能删除`)
  }
  mockExtensions = mockExtensions.filter(item => item.extensionId !== extensionId)
  return true
}

export async function chooseBrowserExtensionArchive(): Promise<{ cancelled: boolean; path: string }> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserExtensionChooseArchive) {
    return (await bindings.BrowserExtensionChooseArchive()) || { cancelled: true, path: '' }
  }
  return { cancelled: true, path: '' }
}

export async function chooseBrowserExtensionDirectory(): Promise<{ cancelled: boolean; path: string }> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserExtensionChooseDirectory) {
    return (await bindings.BrowserExtensionChooseDirectory()) || { cancelled: true, path: '' }
  }
  return { cancelled: true, path: '' }
}

export async function importBrowserExtensionArchive(input: BrowserExtensionImportInput): Promise<BrowserExtensionImportResult> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserExtensionImportArchive) {
    return (await bindings.BrowserExtensionImportArchive(input)) || { cancelled: false, duplicate: false, message: '' }
  }
  return { cancelled: true, duplicate: false, message: '当前运行环境不支持导入扩展压缩包' }
}

export async function importBrowserExtensionDirectory(input: BrowserExtensionImportInput): Promise<BrowserExtensionImportResult> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserExtensionImportDirectory) {
    return (await bindings.BrowserExtensionImportDirectory(input)) || { cancelled: false, duplicate: false, message: '' }
  }
  return { cancelled: true, duplicate: false, message: '当前运行环境不支持导入扩展目录' }
}

export async function fetchBrowserExtensionProfileBindings(extensionId: string): Promise<BrowserExtensionBinding[]> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserExtensionListProfileBindings) {
    return (await bindings.BrowserExtensionListProfileBindings(extensionId)) || []
  }
  return mockExtensionBindings.filter(item => item.extensionId === extensionId)
}

export async function fetchBrowserExtensionBindingsForProfile(profileId: string): Promise<BrowserExtensionBinding[]> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserExtensionListForProfile) {
    return (await bindings.BrowserExtensionListForProfile(profileId)) || []
  }
  return mockExtensionBindings.filter(item => item.profileId === profileId)
}

export async function assignBrowserExtensionProfiles(input: BrowserExtensionAssignInput): Promise<BrowserExtensionBinding[]> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserExtensionAssignProfiles) {
    return (await bindings.BrowserExtensionAssignProfiles(input)) || []
  }
  const now = new Date().toISOString()
  input.profileIds.forEach(profileId => {
    const profile = mockProfiles.find(item => item.profileId === profileId)
    const extension = mockExtensions.find(item => item.extensionId === input.extensionId)
    mockExtensionBindings = mockExtensionBindings.filter(item => !(item.profileId === profileId && item.extensionId === input.extensionId))
    mockExtensionBindings.push({
      id: Date.now() + Math.floor(Math.random() * 10000),
      profileId,
      profileName: profile?.profileName || profileId,
      extensionId: input.extensionId,
      extensionName: extension?.name || input.extensionId,
      extensionVersion: extension?.version || '',
      mode: input.mode || 'shared',
      enabled: input.enabled,
      exclusiveDir: input.mode === 'exclusive' ? `data/extensions/exclusive/${profileId}/${input.extensionId}` : '',
      createdAt: now,
      updatedAt: now,
    })
  })
  return mockExtensionBindings.filter(item => item.extensionId === input.extensionId)
}

export async function unassignBrowserExtensionProfiles(input: BrowserExtensionUnassignInput): Promise<BrowserExtensionBinding[]> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserExtensionUnassignProfiles) {
    return (await bindings.BrowserExtensionUnassignProfiles(input)) || []
  }
  mockExtensionBindings = mockExtensionBindings.filter(item => !(item.extensionId === input.extensionId && input.profileIds.includes(item.profileId)))
  return mockExtensionBindings.filter(item => item.extensionId === input.extensionId)
}

// ============================================================================
// Proxy API
// ============================================================================

export async function fetchBrowserProxies(): Promise<BrowserProxy[]> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserProxyList) {
    return (await bindings.BrowserProxyList()) || []
  }
  return mockProxies
}

export async function fetchBrowserProxyGroups(): Promise<string[]> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserProxyListGroups) {
    return (await bindings.BrowserProxyListGroups()) || []
  }
  return []
}

export async function fetchBrowserProxiesByGroup(groupName: string): Promise<BrowserProxy[]> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserProxyListByGroup) {
    return (await bindings.BrowserProxyListByGroup(groupName)) || []
  }
  return mockProxies.filter(p => p.groupName === groupName)
}

export interface ClashImportURLResult {
  url: string
  content: string
  proxyCount: number
  dnsServers?: string
  suggestedGroup?: string
}

export async function fetchClashImportFromURL(targetURL: string): Promise<ClashImportURLResult> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserProxyFetchClashByURL) {
    return (await bindings.BrowserProxyFetchClashByURL(targetURL)) || {
      url: targetURL,
      content: '',
      proxyCount: 0,
    }
  }

  // 兜底：wailsjs 尚未刷新时，直接通过 window.go 调用后端绑定
  const goApp = (window as any).go?.main?.App
  if (goApp?.BrowserProxyFetchClashByURL) {
    return (await goApp.BrowserProxyFetchClashByURL(targetURL)) || {
      url: targetURL,
      content: '',
      proxyCount: 0,
    }
  }

  throw new Error('当前环境不支持 URL 导入 Clash 配置')
}

export async function saveBrowserProxies(proxies: BrowserProxy[]): Promise<boolean> {
  const bindings: any = await getBindings()
  if (bindings?.SaveBrowserProxies) {
    await bindings.SaveBrowserProxies(proxies)
    return true
  }
  mockProxies = proxies
  return true
}

export async function validateProxyConfig(proxyConfig: string, proxyId: string): Promise<{ supported: boolean; errorMsg: string }> {
  const bindings: any = await getBindings()
  if (bindings?.ValidateProxyConfig) {
    return (await bindings.ValidateProxyConfig(proxyConfig, proxyId)) || { supported: true, errorMsg: '' }
  }
  return { supported: true, errorMsg: '' }
}

export async function testProxyConnectivity(proxyId: string, proxyConfig: string): Promise<{ proxyId: string; ok: boolean; latencyMs: number; error: string }> {
  const bindings: any = await getBindings()
  if (bindings?.TestProxyConnectivity) {
    return (await bindings.TestProxyConnectivity(proxyId, proxyConfig)) || { proxyId, ok: false, latencyMs: 0, error: '调用失败' }
  }
  // mock: simulate latency
  await new Promise(r => setTimeout(r, 300 + Math.random() * 500))
  return { proxyId, ok: true, latencyMs: Math.floor(100 + Math.random() * 200), error: '' }
}

export async function testProxyRealConnectivity(proxyId: string): Promise<{ proxyId: string; ok: boolean; latencyMs: number; error: string }> {
  const bindings: any = await getBindings()
  if (bindings?.TestProxyRealConnectivity) {
    return (await bindings.TestProxyRealConnectivity(proxyId)) || { proxyId, ok: false, latencyMs: 0, error: '调用失败' }
  }
  // mock: simulate latency 300-800ms
  await new Promise(r => setTimeout(r, 300 + Math.random() * 500))
  return { proxyId, ok: true, latencyMs: Math.floor(100 + Math.random() * 400), error: '' }
}

export async function browserProxyTestSpeed(proxyId: string): Promise<{ proxyId: string; ok: boolean; latencyMs: number; error: string }> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserProxyTestSpeed) {
    return (await bindings.BrowserProxyTestSpeed(proxyId)) || { proxyId, ok: false, latencyMs: 0, error: '调用失败' }
  }
  await new Promise(r => setTimeout(r, 300 + Math.random() * 500))
  return { proxyId, ok: true, latencyMs: Math.floor(100 + Math.random() * 400), error: '' }
}

export async function browserProxyBatchTestSpeed(proxyIds: string[], concurrency: number = 20): Promise<{ proxyId: string; ok: boolean; latencyMs: number; error: string }[]> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserProxyBatchTestSpeed) {
    return (await bindings.BrowserProxyBatchTestSpeed(proxyIds, concurrency)) || []
  }
  // mock
  await new Promise(r => setTimeout(r, 1000))
  return proxyIds.map(id => ({ proxyId: id, ok: true, latencyMs: Math.floor(100 + Math.random() * 400), error: '' }))
}

export async function browserProxyCheckIPHealth(proxyId: string): Promise<ProxyIPHealthResult> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserProxyCheckIPHealth) {
    return (await bindings.BrowserProxyCheckIPHealth(proxyId)) || {
      proxyId,
      ok: false,
      source: 'ippure',
      error: '调用失败',
      ip: '',
      fraudScore: 0,
      isResidential: false,
      isBroadcast: false,
      country: '',
      region: '',
      city: '',
      asOrganization: '',
      rawData: {},
      updatedAt: new Date().toISOString(),
    }
  }
  await new Promise(r => setTimeout(r, 600))
  return {
    proxyId,
    ok: true,
    source: 'ippure',
    error: '',
    ip: '127.0.0.1',
    fraudScore: Math.floor(Math.random() * 100),
    isResidential: Math.random() > 0.5,
    isBroadcast: false,
    country: 'Mock',
    region: 'Mock',
    city: 'Mock',
    asOrganization: 'Mock ISP',
    rawData: {},
    updatedAt: new Date().toISOString(),
  }
}

export async function browserProxyBatchCheckIPHealth(proxyIds: string[], concurrency: number = 10): Promise<ProxyIPHealthResult[]> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserProxyBatchCheckIPHealth) {
    return (await bindings.BrowserProxyBatchCheckIPHealth(proxyIds, concurrency)) || []
  }
  await new Promise(r => setTimeout(r, 1200))
  return proxyIds.map(proxyId => ({
    proxyId,
    ok: true,
    source: 'ippure',
    error: '',
    ip: '127.0.0.1',
    fraudScore: Math.floor(Math.random() * 100),
    isResidential: Math.random() > 0.5,
    isBroadcast: false,
    country: 'Mock',
    region: 'Mock',
    city: 'Mock',
    asOrganization: 'Mock ISP',
    rawData: {},
    updatedAt: new Date().toISOString(),
  }))
}

export async function openUserDataDir(userDataDir: string): Promise<boolean> {
  const bindings: any = await getBindings()
  if (bindings?.OpenUserDataDir) {
    await bindings.OpenUserDataDir(userDataDir)
    return true
  }
  const goApp = (window as any).go?.main?.App
  if (goApp?.OpenUserDataDir) {
    await goApp.OpenUserDataDir(userDataDir)
    return true
  }
  return false
}

export async function openProfileUserDataDir(profileId: string): Promise<boolean> {
  const bindings: any = await getBindings()
  if (bindings?.OpenProfileUserDataDir) {
    await bindings.OpenProfileUserDataDir(profileId)
    return true
  }
  const goApp = (window as any).go?.main?.App
  if (goApp?.OpenProfileUserDataDir) {
    await goApp.OpenProfileUserDataDir(profileId)
    return true
  }
  return false
}

export async function openCorePath(corePath: string): Promise<boolean> {
  const bindings: any = await getBindings()
  if (bindings?.OpenCorePath) {
    await bindings.OpenCorePath(corePath)
    return true
  }
  return false
}

// ============================================================================
// Cookie API
// ============================================================================

export async function fetchBrowserCookies(profileId: string): Promise<CookieInfo[]> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserGetCookies) {
    return (await bindings.BrowserGetCookies(profileId)) || []
  }
  // mock data
  return [
    { name: 'session', value: 'abc123', domain: '.example.com', path: '/', expires: Date.now() / 1000 + 3600, httpOnly: true, secure: true, sameSite: 'Lax' },
    { name: 'pref', value: 'dark', domain: 'example.com', path: '/', expires: -1, httpOnly: false, secure: false, sameSite: 'None' },
  ]
}

export async function clearBrowserCookies(profileId: string): Promise<boolean> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserClearCookies) {
    await bindings.BrowserClearCookies(profileId)
    return true
  }
  return true
}

export async function exportBrowserCookies(profileId: string): Promise<string> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserExportCookies) {
    return (await bindings.BrowserExportCookies(profileId)) || ''
  }
  return '# Netscape HTTP Cookie File\n# Generated by BrowserManager\n\n.example.com\tTRUE\t/\tTRUE\t0\tsession\tabc123\n'
}

export async function importBrowserCookies(profileId: string, content: string): Promise<CookieImportResult> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserImportCookies) {
    return (await bindings.BrowserImportCookies(profileId, content)) || { imported: 0, skipped: 0 }
  }
  return { imported: 0, skipped: 0 }
}

// ============================================================================
// Snapshot API
// ============================================================================

export async function listSnapshots(profileId: string): Promise<SnapshotInfo[]> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserSnapshotList) {
    return (await bindings.BrowserSnapshotList(profileId)) || []
  }
  return []
}

export async function createSnapshot(profileId: string, name: string): Promise<SnapshotInfo | null> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserSnapshotCreate) {
    return (await bindings.BrowserSnapshotCreate(profileId, name)) || null
  }
  // mock
  return {
    snapshotId: `snap-${Date.now()}`,
    profileId,
    name,
    sizeMB: 12.5,
    createdAt: new Date().toISOString(),
  }
}

export async function restoreSnapshot(profileId: string, snapshotId: string): Promise<boolean> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserSnapshotRestore) {
    await bindings.BrowserSnapshotRestore(profileId, snapshotId)
    return true
  }
  return true
}

export async function deleteSnapshot(profileId: string, snapshotId: string): Promise<boolean> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserSnapshotDelete) {
    await bindings.BrowserSnapshotDelete(profileId, snapshotId)
    return true
  }
  return true
}

// ============================================================================
// Bookmark API
// ============================================================================

export async function fetchBookmarks(): Promise<BrowserBookmark[]> {
  const bindings: any = await getBindings()
  if (bindings?.BookmarkList) {
    return (await bindings.BookmarkList()) || []
  }
  return [
    { name: 'Google', url: 'https://www.google.com/' },
    { name: 'Gmail', url: 'https://mail.google.com/' },
    { name: 'Claude', url: 'https://claude.ai/' },
    { name: 'ChatGPT', url: 'https://chatgpt.com/' },
    { name: 'YouTube', url: 'https://www.youtube.com/' },
  ]
}

export async function saveBookmarks(items: BrowserBookmark[]): Promise<boolean> {
  const bindings: any = await getBindings()
  if (bindings?.BookmarkSave) {
    await bindings.BookmarkSave(items)
    return true
  }
  return true
}

export async function resetBookmarks(): Promise<boolean> {
  const bindings: any = await getBindings()
  if (bindings?.BookmarkReset) {
    await bindings.BookmarkReset()
    return true
  }
  return true
}

// ============================================================================
// Keywords API
// ============================================================================

export async function setProfileKeywords(profileId: string, keywords: string[]): Promise<BrowserProfile | null> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserProfileSetKeywords) {
    return (await bindings.BrowserProfileSetKeywords(profileId, keywords)) || null
  }
  mockProfiles = mockProfiles.map(p =>
    p.profileId === profileId ? { ...p, keywords, updatedAt: new Date().toISOString() } : p
  )
  return mockProfiles.find(p => p.profileId === profileId) || null
}

// ============================================================================
// LaunchCode API
// ============================================================================

export interface LaunchServerInfo {
  host: string
  port: number
  preferredPort: number
  baseUrl: string
  cdpUrl: string
  activeDebugPort: number
  ready: boolean
  apiAuth: {
    requested: boolean
    configured: boolean
    enabled: boolean
    header: string
  }
}

function normalizeLaunchServerInfo(payload: any): LaunchServerInfo {
  const host = String(payload?.host || '127.0.0.1')
  const port = Number(payload?.port) || 0
  const preferredPort = Number(payload?.preferredPort) || 0
  const fallbackPort = preferredPort > 0 ? preferredPort : 19876
  const effectivePort = port > 0 ? port : fallbackPort
  const baseUrl = String(payload?.baseUrl || (effectivePort > 0 ? `http://${host}:${effectivePort}` : ''))
  const cdpUrl = String(payload?.cdpUrl || baseUrl)
  const activeDebugPort = Number(payload?.activeDebugPort) || 0
  const apiAuthPayload = payload?.apiAuth || {}
  const apiAuth = {
    requested: !!apiAuthPayload?.requested,
    configured: !!apiAuthPayload?.configured,
    enabled: !!apiAuthPayload?.enabled,
    header: String(apiAuthPayload?.header || 'X-Ant-Api-Key'),
  }

  return {
    host,
    port: effectivePort,
    preferredPort,
    baseUrl,
    cdpUrl,
    activeDebugPort,
    ready: !!payload?.ready && port > 0,
    apiAuth,
  }
}

export async function fetchLaunchServerInfo(): Promise<LaunchServerInfo> {
  const bindings: any = await getBindings()
  if (bindings?.GetLaunchServerInfo) {
    return normalizeLaunchServerInfo(await bindings.GetLaunchServerInfo())
  }

  const goApp = (window as any).go?.main?.App
  if (goApp?.GetLaunchServerInfo) {
    return normalizeLaunchServerInfo(await goApp.GetLaunchServerInfo())
  }

  return {
    host: '127.0.0.1',
    port: 19876,
    preferredPort: 19876,
    baseUrl: 'http://127.0.0.1:19876',
    cdpUrl: 'http://127.0.0.1:19876',
    activeDebugPort: 0,
    ready: false,
    apiAuth: {
      requested: false,
      configured: false,
      enabled: false,
      header: 'X-Ant-Api-Key',
    },
  }
}

export async function getBrowserProfileCode(profileId: string): Promise<string> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserProfileGetCode) {
    return (await bindings.BrowserProfileGetCode(profileId)) || ''
  }
  return ''
}

export async function regenerateBrowserProfileCode(profileId: string): Promise<string> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserProfileRegenerateCode) {
    return (await bindings.BrowserProfileRegenerateCode(profileId)) || ''
  }
  return ''
}

export async function setBrowserProfileCode(profileId: string, code: string): Promise<string> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserProfileSetCode) {
    return (await bindings.BrowserProfileSetCode(profileId, code)) || ''
  }
  return code.trim().toUpperCase()
}


export async function batchSetProfileTags(profileIds: string[], tags: string[], replace: boolean): Promise<boolean> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserProfileBatchSetTags) {
    await bindings.BrowserProfileBatchSetTags(profileIds, tags, replace)
    return true
  }
  return true
}

export async function batchRemoveProfileTags(profileIds: string[], tags: string[]): Promise<boolean> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserProfileBatchRemoveTags) {
    await bindings.BrowserProfileBatchRemoveTags(profileIds, tags)
    return true
  }
  return true
}

export async function renameBrowserTag(oldName: string, newName: string): Promise<boolean> {
  const bindings: any = await getBindings()
  if (bindings?.BrowserRenameTag) {
    await bindings.BrowserRenameTag(oldName, newName)
    return true
  }
  return true
}

// ============================================================================
// Group API
// ============================================================================

export async function fetchGroups(): Promise<BrowserGroupWithCount[]> {
  const bindings: any = await getBindings()
  if (bindings?.ListGroups) {
    return (await bindings.ListGroups()) || []
  }
  return mockGroups.map(group => ({
    ...group,
    instanceCount: mockProfiles.filter(profile => profile.groupId === group.groupId).length,
  }))
}

export async function fetchDefaultContentRules(): Promise<DefaultContentRule[]> {
  const bindings: any = await getBindings()
  if (bindings?.DefaultContentRuleList) {
    return (await bindings.DefaultContentRuleList()) || []
  }
  return mockDefaultContentRules
}

export async function saveDefaultContentRules(items: DefaultContentRule[]): Promise<boolean> {
  const bindings: any = await getBindings()
  if (bindings?.DefaultContentRuleSave) {
    await bindings.DefaultContentRuleSave(items)
    return true
  }
  mockDefaultContentRules = items
  return true
}

export async function fetchDefaultStartURLs(): Promise<BrowserStartURL[]> {
  const bindings: any = await getBindings()
  if (bindings?.DefaultStartURLList) {
    return (await bindings.DefaultStartURLList()) || []
  }
  return [
    { name: 'IPPure', url: 'https://ippure.com/' },
    { name: 'IPLark', url: 'https://iplark.com/' },
    { name: 'Ping0', url: 'https://ping0.cc/' },
  ]
}

export async function saveDefaultStartURLs(items: BrowserStartURL[]): Promise<boolean> {
  const bindings: any = await getBindings()
  if (bindings?.DefaultStartURLSave) {
    await bindings.DefaultStartURLSave(items)
    return true
  }
  return true
}

export async function resetDefaultStartURLs(): Promise<boolean> {
  const bindings: any = await getBindings()
  if (bindings?.DefaultStartURLReset) {
    await bindings.DefaultStartURLReset()
    return true
  }
  return true
}

export async function createGroup(input: BrowserGroupInput): Promise<BrowserGroup | null> {
  const bindings: any = await getBindings()
  if (bindings?.CreateGroup) {
    return (await bindings.CreateGroup(input)) || null
  }
  if (!input.groupName.trim()) {
    throw new Error('分组名称不能为空')
  }
  if (input.parentId && !mockGroups.some(group => group.groupId === input.parentId)) {
    throw new Error('父分组不存在')
  }
  const now = new Date().toISOString()
  const group: BrowserGroup = {
    groupId: `mock-group-${Date.now()}`,
    groupName: input.groupName.trim(),
    parentId: input.parentId || '',
    sortOrder: input.sortOrder || 0,
    createdAt: now,
    updatedAt: now,
  }
  mockGroups = [...mockGroups, group]
  return group
}

export async function updateGroup(groupId: string, input: BrowserGroupInput): Promise<BrowserGroup | null> {
  const bindings: any = await getBindings()
  if (bindings?.UpdateGroup) {
    return (await bindings.UpdateGroup(groupId, input)) || null
  }
  const index = mockGroups.findIndex(group => group.groupId === groupId)
  if (index === -1) {
    throw new Error('分组不存在')
  }
  if (!input.groupName.trim()) {
    throw new Error('分组名称不能为空')
  }
  if (input.parentId === groupId) {
    throw new Error('不能将分组设为自己的子分组')
  }
  if (input.parentId && !mockGroups.some(group => group.groupId === input.parentId)) {
    throw new Error('父分组不存在')
  }
  let currentParentId = input.parentId
  while (currentParentId) {
    if (currentParentId === groupId) {
      throw new Error('不能将分组设为自己的后代分组')
    }
    currentParentId = mockGroups.find(group => group.groupId === currentParentId)?.parentId || ''
  }
  const updated = {
    ...mockGroups[index],
    groupName: input.groupName.trim(),
    parentId: input.parentId || '',
    sortOrder: input.sortOrder || 0,
    updatedAt: new Date().toISOString(),
  }
  mockGroups = mockGroups.map(group => group.groupId === groupId ? updated : group)
  return updated
}

export async function deleteGroup(groupId: string): Promise<boolean> {
  const bindings: any = await getBindings()
  if (bindings?.DeleteGroup) {
    await bindings.DeleteGroup(groupId)
    return true
  }
  const group = mockGroups.find(item => item.groupId === groupId)
  if (!group) {
    throw new Error('分组不存在')
  }
  mockGroups = mockGroups
    .filter(item => item.groupId !== groupId)
    .map(item => item.parentId === groupId ? { ...item, parentId: group.parentId, updatedAt: new Date().toISOString() } : item)
  mockProfiles = mockProfiles.map(profile => (
    profile.groupId === groupId ? { ...profile, groupId: group.parentId, updatedAt: new Date().toISOString() } : profile
  ))
  return true
}

export async function moveInstancesToGroup(profileIds: string[], groupId: string): Promise<boolean> {
  const bindings: any = await getBindings()
  if (bindings?.MoveInstancesToGroup) {
    await bindings.MoveInstancesToGroup(profileIds, groupId)
    return true
  }
  if (groupId && !mockGroups.some(group => group.groupId === groupId)) {
    throw new Error('分组不存在')
  }
  const idSet = new Set(profileIds)
  mockProfiles = mockProfiles.map(profile => (
    idSet.has(profile.profileId) ? { ...profile, groupId, updatedAt: new Date().toISOString() } : profile
  ))
  return true
}
