// Settings 模块 API
import type { AppSettings } from './types'
import { defaultSettings } from './types'
import {
  checkAppUpdate as checkAppUpdateProto,
  downloadAndExtractPortableUpdate as downloadAndExtractPortableUpdateProto,
  downloadAppUpdate as downloadAppUpdateProto,
  exportSystemConfig as exportSystemConfigProto,
  getAppConfig as getAppConfigProto,
  importSystemConfig as importSystemConfigProto,
  initializeSystemData as initializeSystemDataProto,
  installDownloadedAppUpdate as installDownloadedAppUpdateProto,
  onAppUpdateDownloadProgress as onAppUpdateDownloadProgressProto,
  onAppUpdatePending as onAppUpdatePendingProto,
  onAppUpdatePendingInstallFailed as onAppUpdatePendingInstallFailedProto,
  onAppUpdatePendingNotification as onAppUpdatePendingNotificationProto,
  onBackupExportProgress as onBackupExportProgressProto,
  onBackupImportProgress as onBackupImportProgressProto,
  openAppReleasePage as openAppReleasePageProto,
  openPath as openPathProto,
  type ProtoAppUpdateDownloadProgress,
  type ProtoAppUpdateInfo,
  type ProtoAppUpdatePendingInstallFailed,
  type ProtoAppUpdatePendingNotification,
  type ProtoAppUpdatePendingUpdate,
  type ProtoBackupProgress,
} from '../../shared/backend/client'

// 本地存储 key
const SETTINGS_KEY = 'app_settings'

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
  installerAsset?: AppUpdateAsset
  portableAsset?: AppUpdateAsset
  distributionKind?: 'dev' | 'installer' | 'portable' | string
  recommendedPackageKind?: 'installer' | 'portable' | string
  canSelfUpdatePortable?: boolean
  message: string
}

export interface AppUpdateDownloadResult {
  cancelled?: boolean
  message?: string
  version?: string
  installerPath?: string
  packagePath?: string
  extractedPath?: string
  installOnRestart?: boolean
  restartScheduled?: boolean
  packageKind?: string
}

export type BackupProgress = ProtoBackupProgress
export type AppUpdateDownloadProgress = ProtoAppUpdateDownloadProgress
export type AppUpdatePendingUpdate = ProtoAppUpdatePendingUpdate
export type AppUpdatePendingNotification = ProtoAppUpdatePendingNotification
export type AppUpdatePendingInstallFailed = ProtoAppUpdatePendingInstallFailed

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
  return await initializeSystemDataProto()
}

export async function exportSystemConfig(): Promise<BackupActionResult> {
  return await exportSystemConfigProto()
}

export async function importSystemConfig(resetFirst: boolean): Promise<BackupActionResult> {
  return await importSystemConfigProto(resetFirst)
}

export async function fetchAppConfig(): Promise<AppConfigInfo> {
  const data = await getAppConfigProto()
  return {
    name: String(data.name || defaultSettings.appName),
    version: String(data.version || 'unknown'),
    projectGithubUrl: String(data.projectGithubUrl || 'https://github.com/lemon-casino/trace-browser-release/releases'),
  }
}

export async function checkAppUpdate(): Promise<AppUpdateInfo> {
  return await checkAppUpdateProto()
}

export async function downloadAppUpdate(info: AppUpdateInfo | Record<string, any>, installOnRestart: boolean): Promise<AppUpdateDownloadResult> {
  return await downloadAppUpdateProto(normalizeAppUpdateInfo(info), installOnRestart)
}

export async function installDownloadedAppUpdate(installerPath?: string): Promise<void> {
  await installDownloadedAppUpdateProto(installerPath || '')
}

export async function downloadAndExtractPortableUpdate(info: AppUpdateInfo | Record<string, any>): Promise<AppUpdateDownloadResult> {
  return await downloadAndExtractPortableUpdateProto(normalizeAppUpdateInfo(info))
}

export async function openPath(path: string): Promise<void> {
  await openPathProto(path)
}

export async function openAppReleasePage(url?: string): Promise<void> {
  await openAppReleasePageProto(url || '')
}

export function onBackupExportProgress(callback: (progress: BackupProgress) => void): () => void {
  return onBackupExportProgressProto(callback)
}

export function onBackupImportProgress(callback: (progress: BackupProgress) => void): () => void {
  return onBackupImportProgressProto(callback)
}

export function onAppUpdateDownloadProgress(callback: (progress: AppUpdateDownloadProgress) => void): () => void {
  return onAppUpdateDownloadProgressProto(callback)
}

export function onAppUpdatePending(callback: (pending: AppUpdatePendingUpdate) => void): () => void {
  return onAppUpdatePendingProto(callback)
}

export function onAppUpdatePendingNotification(callback: (notification: AppUpdatePendingNotification) => void): () => void {
  return onAppUpdatePendingNotificationProto(callback)
}

export function onAppUpdatePendingInstallFailed(callback: (failure: AppUpdatePendingInstallFailed) => void): () => void {
  return onAppUpdatePendingInstallFailedProto(callback)
}

function normalizeAppUpdateInfo(info: AppUpdateInfo | Record<string, any>): ProtoAppUpdateInfo {
  const data = info || {}
  return {
    currentVersion: String(data.currentVersion || ''),
    latestVersion: String(data.latestVersion || ''),
    releaseName: String(data.releaseName || ''),
    releaseUrl: String(data.releaseUrl || ''),
    publishedAt: String(data.publishedAt || ''),
    body: String(data.body || ''),
    hasUpdate: !!data.hasUpdate,
    asset: normalizeAppUpdateAsset(data.asset),
    installerAsset: normalizeAppUpdateAsset(data.installerAsset),
    portableAsset: normalizeAppUpdateAsset(data.portableAsset),
    distributionKind: String(data.distributionKind || ''),
    recommendedPackageKind: String(data.recommendedPackageKind || ''),
    canSelfUpdatePortable: !!data.canSelfUpdatePortable,
    message: String(data.message || ''),
  }
}

function normalizeAppUpdateAsset(asset: any): AppUpdateAsset | undefined {
  if (!asset || typeof asset !== 'object') {
    return undefined
  }
  return {
    name: String(asset.name || ''),
    size: Number(asset.size) || 0,
    downloadUrl: String(asset.downloadUrl || ''),
  }
}
