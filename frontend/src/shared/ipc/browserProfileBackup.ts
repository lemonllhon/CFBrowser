import {
  METHOD_BROWSER_PROFILE_BACKUP_CHOOSE_IMPORT,
  METHOD_BROWSER_PROFILE_BACKUP_EXPORT,
  METHOD_BROWSER_PROFILE_BACKUP_IMPORT,
} from './envelope'
import {
  WireType,
  concatBytes,
  decodeString,
  decodeVarintField,
  encodeBoolField,
  encodeStringField,
  readFields,
} from './protobuf'
import { ProtoIpcClient } from './transport'
import { decodeBackupProgress, type ProtoBackupProgress } from './app'

const profileBackupProtoClient = new ProtoIpcClient()

export type ProtoBrowserProfileBackupExportInput = {
  scope: 'all' | 'selected' | 'filtered' | string
  profileIds: string[]
  includeCookies: boolean
  includePlainCookiesWhenRunning: boolean
}

export type ProtoBrowserProfileBackupImportInput = {
  zipPath: string
  restoreCookies: boolean
}

export type ProtoBrowserProfileBackupSummary = {
  zipPath: string
  format: string
  version: number
  appName: string
  appVersion: string
  createdAt: string
  sourceOs: string
  profileCount: number
  cookieProfileCount: number
  includesCookies: boolean
  includesPlainCookies: boolean
  cookieNotice: string
  warnings: string[]
}

export type ProtoBrowserProfileBackupWarning = {
  profileId?: string
  profileName?: string
  message: string
}

export type ProtoBrowserProfileBackupActionResult = {
  cancelled: boolean
  message: string
  zipPath: string
  createdAt: string
  exported: number
  imported: number
  skipped: number
  failed: number
  profileCount: number
  cookieProfileCount: number
  summary: ProtoBrowserProfileBackupSummary
  warnings: ProtoBrowserProfileBackupWarning[]
}

export async function exportBrowserProfileBackup(input: ProtoBrowserProfileBackupExportInput): Promise<ProtoBrowserProfileBackupActionResult> {
  const payload = await profileBackupProtoClient.request(
    METHOD_BROWSER_PROFILE_BACKUP_EXPORT,
    encodeBrowserProfileBackupExportRequest(input),
    300000,
  )
  return decodeBrowserProfileBackupActionResult(payload)
}

export async function chooseBrowserProfileBackupImportPackage(): Promise<ProtoBrowserProfileBackupActionResult> {
  const payload = await profileBackupProtoClient.request(
    METHOD_BROWSER_PROFILE_BACKUP_CHOOSE_IMPORT,
    new Uint8Array(),
    120000,
  )
  return decodeBrowserProfileBackupActionResult(payload)
}

export async function importBrowserProfileBackup(input: ProtoBrowserProfileBackupImportInput): Promise<ProtoBrowserProfileBackupActionResult> {
  const payload = await profileBackupProtoClient.request(
    METHOD_BROWSER_PROFILE_BACKUP_IMPORT,
    encodeBrowserProfileBackupImportRequest(input),
    300000,
  )
  return decodeBrowserProfileBackupActionResult(payload)
}

export function onBrowserProfileBackupProgress(callback: (progress: ProtoBackupProgress) => void): () => void {
  return profileBackupProtoClient.onEvent('browser:profile-backup:progress', event => callback(decodeBackupProgress(event.payload)))
}

export function encodeBrowserProfileBackupExportRequest(message: ProtoBrowserProfileBackupExportInput): Uint8Array {
  return concatBytes([
    encodeStringField(1, message.scope || 'all'),
    ...message.profileIds.map(profileId => encodeStringField(2, profileId)),
    encodeBoolField(3, message.includeCookies),
    encodeBoolField(4, message.includePlainCookiesWhenRunning),
  ])
}

export function encodeBrowserProfileBackupImportRequest(message: ProtoBrowserProfileBackupImportInput): Uint8Array {
  return concatBytes([
    encodeStringField(1, message.zipPath),
    encodeBoolField(2, message.restoreCookies),
  ])
}

export function decodeBrowserProfileBackupActionResult(payload: Uint8Array): ProtoBrowserProfileBackupActionResult {
  const result: ProtoBrowserProfileBackupActionResult = {
    cancelled: false,
    message: '',
    zipPath: '',
    createdAt: '',
    exported: 0,
    imported: 0,
    skipped: 0,
    failed: 0,
    profileCount: 0,
    cookieProfileCount: 0,
    summary: emptyProfileBackupSummary(),
    warnings: [],
  }

  for (const field of readFields(payload)) {
    if (field.wireType === WireType.LengthDelimited) {
      switch (field.fieldNumber) {
        case 2:
          result.message = decodeString(field.value)
          break
        case 3:
          result.zipPath = decodeString(field.value)
          break
        case 4:
          result.createdAt = decodeString(field.value)
          break
        case 11:
          result.summary = decodeBrowserProfileBackupSummary(field.value)
          break
        case 12:
          result.warnings.push(decodeBrowserProfileBackupWarning(field.value))
          break
      }
      continue
    }

    if (field.wireType !== WireType.Varint) {
      continue
    }
    const number = Number(decodeVarintField(field.value))
    switch (field.fieldNumber) {
      case 1:
        result.cancelled = number !== 0
        break
      case 5:
        result.exported = number
        break
      case 6:
        result.imported = number
        break
      case 7:
        result.skipped = number
        break
      case 8:
        result.failed = number
        break
      case 9:
        result.profileCount = number
        break
      case 10:
        result.cookieProfileCount = number
        break
    }
  }

  if (!result.zipPath && result.summary.zipPath) {
    result.zipPath = result.summary.zipPath
  }
  if (!result.createdAt && result.summary.createdAt) {
    result.createdAt = result.summary.createdAt
  }
  return result
}

export function decodeBrowserProfileBackupSummary(payload: Uint8Array): ProtoBrowserProfileBackupSummary {
  const summary = emptyProfileBackupSummary()

  for (const field of readFields(payload)) {
    if (field.wireType === WireType.LengthDelimited) {
      const text = decodeString(field.value)
      switch (field.fieldNumber) {
        case 1:
          summary.zipPath = text
          break
        case 2:
          summary.format = text
          break
        case 4:
          summary.appName = text
          break
        case 5:
          summary.appVersion = text
          break
        case 6:
          summary.createdAt = text
          break
        case 7:
          summary.sourceOs = text
          break
        case 12:
          summary.cookieNotice = text
          break
        case 13:
          summary.warnings.push(text)
          break
      }
      continue
    }

    if (field.wireType !== WireType.Varint) {
      continue
    }
    const number = Number(decodeVarintField(field.value))
    switch (field.fieldNumber) {
      case 3:
        summary.version = number
        break
      case 8:
        summary.profileCount = number
        break
      case 9:
        summary.cookieProfileCount = number
        break
      case 10:
        summary.includesCookies = number !== 0
        break
      case 11:
        summary.includesPlainCookies = number !== 0
        break
    }
  }

  return summary
}

export function decodeBrowserProfileBackupWarning(payload: Uint8Array): ProtoBrowserProfileBackupWarning {
  const warning: ProtoBrowserProfileBackupWarning = { message: '' }
  for (const field of readFields(payload)) {
    if (field.wireType !== WireType.LengthDelimited) {
      continue
    }
    const text = decodeString(field.value)
    if (field.fieldNumber === 1) {
      warning.profileId = text
    } else if (field.fieldNumber === 2) {
      warning.profileName = text
    } else if (field.fieldNumber === 3) {
      warning.message = text
    }
  }
  return warning
}

function emptyProfileBackupSummary(): ProtoBrowserProfileBackupSummary {
  return {
    zipPath: '',
    format: '',
    version: 0,
    appName: '',
    appVersion: '',
    createdAt: '',
    sourceOs: '',
    profileCount: 0,
    cookieProfileCount: 0,
    includesCookies: false,
    includesPlainCookies: false,
    cookieNotice: '',
    warnings: [],
  }
}
