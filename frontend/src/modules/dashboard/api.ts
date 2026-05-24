import type { DashboardStats } from './types'
import {
  generateCDKeys as generateCDKeysProto,
  getAppConfig,
  getDashboardStats,
  getLicenseStatus,
  redeemCDKey as redeemCDKeyProto,
  redeemGithubStar as redeemGithubStarProto,
  reloadAppConfig,
} from '../../shared/backend/client'

function normalizeAppVersion(version?: string): string {
  const value = (version || '').trim()
  return value && value.toLowerCase() !== 'unknown' ? value : ''
}

export async function fetchDashboardStats(): Promise<DashboardStats> {
  try {
    const [data, licenseStatus, appConfig] = await Promise.all([
      getDashboardStats(),
      getLicenseStatus().catch(() => ({ maxLimit: 0 })),
      getAppConfig().catch(() => ({ version: '' })),
    ])
    const appVersion = normalizeAppVersion(data.appVersion) || normalizeAppVersion(appConfig.version) || 'dev'
    return {
      totalInstances: data.totalInstances ?? 0,
      runningInstances: data.runningInstances ?? 0,
      proxyCount: data.proxyCount ?? 0,
      coreCount: data.coreCount ?? 0,
      memUsedMB: data.memUsedMB ?? 0,
      maxProfileLimit: licenseStatus.maxLimit ?? 0,
      appVersion,
    }
  } catch (e) {
    console.error('fetchDashboardStats error:', e)
  }
  return { totalInstances: 0, runningInstances: 0, proxyCount: 0, coreCount: 0, memUsedMB: 0, maxProfileLimit: 0, appVersion: 'dev' }
}

export async function redeemCDKey(cdkey: string): Promise<{ success: boolean, message?: string }> {
  try {
    await redeemCDKeyProto(cdkey)
    return { success: true }
  } catch (e: any) {
    return { success: false, message: e.message || '兑换失败' }
  }
}

export async function redeemGithubStar(): Promise<{ success: boolean, message?: string }> {
  try {
    await redeemGithubStarProto()
    return { success: true }
  } catch (e: any) {
    return { success: false, message: e.message || '领取失败' }
  }
}

export async function reloadConfig(): Promise<void> {
  try {
    await reloadAppConfig()
  } catch (e) {
    console.error('reloadConfig error:', e)
  }
}

export async function generateCDKeys(count: number): Promise<{ success: boolean, keys: string[], message?: string }> {
  try {
    const keys = await generateCDKeysProto(count)
    return { success: true, keys: keys || [] }
  } catch (e: any) {
    return { success: false, keys: [], message: e.message || '生成失败' }
  }
}
