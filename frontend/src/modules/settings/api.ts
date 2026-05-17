// Settings 模块 API
import type { AppSettings } from './types'
import { defaultSettings } from './types'

// 本地存储 key
const SETTINGS_KEY = 'app_settings'

const getBindings = async () => {
  try {
    return await import('../../wailsjs/go/main/App')
  } catch {
    return null
  }
}

export interface BackupActionResult {
  cancelled?: boolean
  message?: string
  zipPath?: string
  resetFirst?: boolean
  imported?: number
  skipped?: number
  conflicts?: number
  partial?: boolean
  componentTotal?: number
  componentSuccess?: number
  componentFailed?: number
  failedComponents?: Array<{
    componentId?: string
    componentName?: string
    error?: string
  }>
}

export interface AppConfigInfo {
  name: string
  version: string
  projectGithubUrl: string
}

export interface AppUpdateAsset {
  name: string
  size: number
  downloadUrl: string
}

export interface AppUpdateInfo {
  currentVersion: string
  latestVersion: string
  releaseName: string
  releaseUrl: string
  publishedAt: string
  body: string
  hasUpdate: boolean
  asset?: AppUpdateAsset
  message: string
}

export interface AppUpdateDownloadResult {
  cancelled?: boolean
  message?: string
  version?: string
  installerPath?: string
  installOnRestart?: boolean
}

// 获取设置
export async function fetchSettings(): Promise<AppSettings> {
  try {
    const stored = localStorage.getItem(SETTINGS_KEY)
    if (stored) {
      return normalizeImmutableAppInfo({ ...defaultSettings, ...JSON.parse(stored) })
    }
  } catch (error) {
    console.error('Failed to load settings:', error)
  }
  return defaultSettings
}

// 保存设置
export async function saveSettings(settings: AppSettings): Promise<boolean> {
  try {
    localStorage.setItem(SETTINGS_KEY, JSON.stringify(normalizeImmutableAppInfo(settings)))
    return true
  } catch (error) {
    console.error('Failed to save settings:', error)
    return false
  }
}

// 重置设置
export async function resetSettings(): Promise<AppSettings> {
  localStorage.removeItem(SETTINGS_KEY)
  return defaultSettings
}

function normalizeImmutableAppInfo(settings: AppSettings): AppSettings {
  return {
    ...settings,
    appName: defaultSettings.appName,
    appDescription: defaultSettings.appDescription,
  }
}

export async function initializeSystemData(): Promise<BackupActionResult> {
  const bindings: any = await getBindings()
  if (!bindings?.BackupInitializeSystem) {
    return { cancelled: false, message: '当前环境不支持后端初始化接口' }
  }
  return (await bindings.BackupInitializeSystem()) || {}
}

export async function exportSystemConfig(): Promise<BackupActionResult> {
  const bindings: any = await getBindings()
  if (!bindings?.BackupExportPackage) {
    return { cancelled: false, message: '当前环境不支持后端导出接口' }
  }
  return (await bindings.BackupExportPackage()) || {}
}

export async function importSystemConfig(resetFirst: boolean): Promise<BackupActionResult> {
  const bindings: any = await getBindings()
  if (!bindings?.BackupImportPackage) {
    return { cancelled: false, message: '当前环境不支持后端加载接口' }
  }
  return (await bindings.BackupImportPackage(resetFirst)) || {}
}

export async function fetchAppConfig(): Promise<AppConfigInfo> {
  const bindings: any = await getBindings()
  if (!bindings?.GetAppConfig) {
    return { name: defaultSettings.appName, version: 'dev', projectGithubUrl: 'https://github.com/lemon-casino/trace-browser-release/releases' }
  }
  const data = (await bindings.GetAppConfig()) || {}
  return {
    name: String(data.name || defaultSettings.appName),
    version: String(data.version || 'unknown'),
    projectGithubUrl: String(data.projectGithubUrl || 'https://github.com/lemon-casino/trace-browser-release/releases'),
  }
}

export async function checkAppUpdate(): Promise<AppUpdateInfo> {
  const bindings: any = await getBindings()
  if (!bindings?.CheckAppUpdate) {
    throw new Error('当前环境不支持检查更新')
  }
  return await bindings.CheckAppUpdate()
}

export async function downloadAppUpdate(info: AppUpdateInfo, installOnRestart: boolean): Promise<AppUpdateDownloadResult> {
  const bindings: any = await getBindings()
  if (!bindings?.DownloadAppUpdate) {
    throw new Error('当前环境不支持下载更新')
  }
  return (await bindings.DownloadAppUpdate(info, installOnRestart)) || {}
}

export async function installDownloadedAppUpdate(installerPath?: string): Promise<void> {
  const bindings: any = await getBindings()
  if (!bindings?.InstallDownloadedAppUpdate) {
    throw new Error('当前环境不支持安装更新')
  }
  await bindings.InstallDownloadedAppUpdate(installerPath || '')
}

export async function openAppReleasePage(url?: string): Promise<void> {
  const bindings: any = await getBindings()
  if (bindings?.OpenAppReleasePage) {
    await bindings.OpenAppReleasePage(url || '')
    return
  }
  window.open(url || 'https://github.com/lemon-casino/trace-browser-release/releases/latest', '_blank')
}
