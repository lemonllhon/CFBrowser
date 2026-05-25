import {
  METHOD_APP_UPDATE_CHECK,
  METHOD_APP_UPDATE_DOWNLOAD,
  METHOD_APP_UPDATE_DOWNLOAD_PORTABLE,
  METHOD_APP_UPDATE_INSTALL_DOWNLOADED,
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

const updateProtoClient = new ProtoIpcClient()

export type ProtoAppUpdateAsset = {
  name: string
  size: number
  downloadUrl: string
  checksum?: string
}

export type ProtoAppUpdateInfo = {
  currentVersion: string
  latestVersion: string
  releaseName: string
  releaseUrl: string
  publishedAt: string
  body: string
  hasUpdate: boolean
  asset?: ProtoAppUpdateAsset
  installerAsset?: ProtoAppUpdateAsset
  portableAsset?: ProtoAppUpdateAsset
  distributionKind: string
  recommendedPackageKind: string
  canSelfUpdatePortable: boolean
  message: string
}

export type ProtoAppUpdateDownloadResult = {
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

export type ProtoAppUpdateDownloadProgress = {
  phase: string
  progress: number
  message: string
}

export type ProtoAppUpdatePendingUpdate = {
  version?: string
  installerPath?: string
  releaseUrl?: string
  installOnNextStart?: boolean
  createdAt?: string
}

export type ProtoAppUpdatePendingNotification = {
  version?: string
  message?: string
}

export type ProtoAppUpdatePendingInstallFailed = {
  version?: string
  error?: string
}

export async function checkAppUpdate(): Promise<ProtoAppUpdateInfo> {
  const payload = await updateProtoClient.request(METHOD_APP_UPDATE_CHECK, new Uint8Array(), 30000)
  return decodeAppUpdateInfo(payload)
}

export async function downloadAppUpdate(info: ProtoAppUpdateInfo, installOnRestart: boolean): Promise<ProtoAppUpdateDownloadResult> {
  const payload = await updateProtoClient.request(
    METHOD_APP_UPDATE_DOWNLOAD,
    encodeAppUpdateDownloadRequest({ info, installOnRestart }),
    600000,
  )
  return decodeAppUpdateDownloadResult(payload)
}

export async function installDownloadedAppUpdate(installerPath = ''): Promise<boolean> {
  const payload = await updateProtoClient.request(
    METHOD_APP_UPDATE_INSTALL_DOWNLOADED,
    encodeAppUpdateInstallDownloadedRequest({ installerPath }),
    30000,
  )
  return decodeBrowserActionResponse(payload).ok
}

export async function downloadAndExtractPortableUpdate(info: ProtoAppUpdateInfo): Promise<ProtoAppUpdateDownloadResult> {
  const payload = await updateProtoClient.request(
    METHOD_APP_UPDATE_DOWNLOAD_PORTABLE,
    encodeAppUpdateInfoRequest({ info }),
    600000,
  )
  return decodeAppUpdateDownloadResult(payload)
}

export function onAppUpdateDownloadProgress(callback: (progress: ProtoAppUpdateDownloadProgress) => void): () => void {
  return updateProtoClient.onEvent('app:update:download:progress', event => callback(decodeAppUpdateDownloadProgress(event.payload)))
}

export function onAppUpdatePending(callback: (pending: ProtoAppUpdatePendingUpdate) => void): () => void {
  return updateProtoClient.onEvent('app:update:pending', event => callback(decodeAppUpdatePendingUpdate(event.payload)))
}

export function onAppUpdatePendingNotification(callback: (notification: ProtoAppUpdatePendingNotification) => void): () => void {
  return updateProtoClient.onEvent('app:update:pending:notification', event => callback(decodeAppUpdatePendingNotification(event.payload)))
}

export function onAppUpdatePendingInstallFailed(callback: (failure: ProtoAppUpdatePendingInstallFailed) => void): () => void {
  return updateProtoClient.onEvent('app:update:pending:install-failed', event => callback(decodeAppUpdatePendingInstallFailed(event.payload)))
}

export function encodeAppUpdateAsset(asset: ProtoAppUpdateAsset): Uint8Array {
  return concatBytes([
    encodeStringField(1, asset.name),
    encodeInt64Field(2, asset.size),
    encodeStringField(3, asset.downloadUrl),
    encodeStringField(4, asset.checksum || ''),
  ])
}

export function encodeAppUpdateInfo(info: ProtoAppUpdateInfo): Uint8Array {
  return concatBytes([
    encodeStringField(1, info.currentVersion),
    encodeStringField(2, info.latestVersion),
    encodeStringField(3, info.releaseName),
    encodeStringField(4, info.releaseUrl),
    encodeStringField(5, info.publishedAt),
    encodeStringField(6, info.body),
    encodeBoolField(7, info.hasUpdate),
    info.asset ? encodeBytesField(8, encodeAppUpdateAsset(info.asset)) : new Uint8Array(),
    info.installerAsset ? encodeBytesField(9, encodeAppUpdateAsset(info.installerAsset)) : new Uint8Array(),
    info.portableAsset ? encodeBytesField(10, encodeAppUpdateAsset(info.portableAsset)) : new Uint8Array(),
    encodeStringField(11, info.distributionKind),
    encodeStringField(12, info.recommendedPackageKind),
    encodeBoolField(13, info.canSelfUpdatePortable),
    encodeStringField(14, info.message),
  ])
}

export function encodeAppUpdateDownloadRequest(message: { info: ProtoAppUpdateInfo; installOnRestart: boolean }): Uint8Array {
  return concatBytes([
    encodeBytesField(1, encodeAppUpdateInfo(message.info)),
    encodeBoolField(2, message.installOnRestart),
  ])
}

export function encodeAppUpdateInfoRequest(message: { info: ProtoAppUpdateInfo }): Uint8Array {
  return concatBytes([encodeBytesField(1, encodeAppUpdateInfo(message.info))])
}

export function encodeAppUpdateInstallDownloadedRequest(message: { installerPath: string }): Uint8Array {
  return concatBytes([encodeStringField(1, message.installerPath)])
}

export function decodeAppUpdateAsset(payload: Uint8Array): ProtoAppUpdateAsset {
  const asset: ProtoAppUpdateAsset = {
    name: '',
    size: 0,
    downloadUrl: '',
  }
  for (const field of readFields(payload)) {
    if (field.wireType === WireType.LengthDelimited) {
      const text = decodeString(field.value)
      if (field.fieldNumber === 1) {
        asset.name = text
      } else if (field.fieldNumber === 3) {
        asset.downloadUrl = text
      } else if (field.fieldNumber === 4) {
        asset.checksum = text
      }
      continue
    }
    if (field.fieldNumber === 2 && field.wireType === WireType.Varint) {
      asset.size = Number(decodeVarintField(field.value))
    }
  }
  return asset
}

export function decodeAppUpdateInfo(payload: Uint8Array): ProtoAppUpdateInfo {
  const info: ProtoAppUpdateInfo = {
    currentVersion: '',
    latestVersion: '',
    releaseName: '',
    releaseUrl: '',
    publishedAt: '',
    body: '',
    hasUpdate: false,
    distributionKind: '',
    recommendedPackageKind: '',
    canSelfUpdatePortable: false,
    message: '',
  }
  for (const field of readFields(payload)) {
    if (field.wireType === WireType.LengthDelimited) {
      switch (field.fieldNumber) {
        case 1:
          info.currentVersion = decodeString(field.value)
          break
        case 2:
          info.latestVersion = decodeString(field.value)
          break
        case 3:
          info.releaseName = decodeString(field.value)
          break
        case 4:
          info.releaseUrl = decodeString(field.value)
          break
        case 5:
          info.publishedAt = decodeString(field.value)
          break
        case 6:
          info.body = decodeString(field.value)
          break
        case 8:
          info.asset = decodeAppUpdateAsset(field.value)
          break
        case 9:
          info.installerAsset = decodeAppUpdateAsset(field.value)
          break
        case 10:
          info.portableAsset = decodeAppUpdateAsset(field.value)
          break
        case 11:
          info.distributionKind = decodeString(field.value)
          break
        case 12:
          info.recommendedPackageKind = decodeString(field.value)
          break
        case 14:
          info.message = decodeString(field.value)
          break
      }
      continue
    }
    if (field.wireType !== WireType.Varint) {
      continue
    }
    const number = Number(decodeVarintField(field.value))
    if (field.fieldNumber === 7) {
      info.hasUpdate = number !== 0
    } else if (field.fieldNumber === 13) {
      info.canSelfUpdatePortable = number !== 0
    }
  }
  return info
}

export function decodeAppUpdateDownloadResult(payload: Uint8Array): ProtoAppUpdateDownloadResult {
  const result: ProtoAppUpdateDownloadResult = {}
  for (const field of readFields(payload)) {
    if (field.wireType === WireType.LengthDelimited) {
      const text = decodeString(field.value)
      switch (field.fieldNumber) {
        case 2:
          result.message = text
          break
        case 3:
          result.version = text
          break
        case 4:
          result.installerPath = text
          break
        case 5:
          result.packagePath = text
          break
        case 6:
          result.extractedPath = text
          break
        case 9:
          result.packageKind = text
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
      case 7:
        result.installOnRestart = number !== 0
        break
      case 8:
        result.restartScheduled = number !== 0
        break
    }
  }
  return result
}

export function decodeAppUpdateDownloadProgress(payload: Uint8Array): ProtoAppUpdateDownloadProgress {
  const progress: ProtoAppUpdateDownloadProgress = {
    phase: '',
    progress: 0,
    message: '',
  }
  for (const field of readFields(payload)) {
    if (field.fieldNumber === 1 && field.wireType === WireType.LengthDelimited) {
      progress.phase = decodeString(field.value)
    } else if (field.fieldNumber === 2 && field.wireType === WireType.Varint) {
      progress.progress = Number(decodeVarintField(field.value))
    } else if (field.fieldNumber === 3 && field.wireType === WireType.LengthDelimited) {
      progress.message = decodeString(field.value)
    }
  }
  return progress
}

export function decodeAppUpdatePendingUpdate(payload: Uint8Array): ProtoAppUpdatePendingUpdate {
  const pending: ProtoAppUpdatePendingUpdate = {}
  for (const field of readFields(payload)) {
    if (field.wireType === WireType.LengthDelimited) {
      const text = decodeString(field.value)
      if (field.fieldNumber === 1) {
        pending.version = text
      } else if (field.fieldNumber === 2) {
        pending.installerPath = text
      } else if (field.fieldNumber === 3) {
        pending.releaseUrl = text
      } else if (field.fieldNumber === 5) {
        pending.createdAt = text
      }
      continue
    }
    if (field.fieldNumber === 4 && field.wireType === WireType.Varint) {
      pending.installOnNextStart = Number(decodeVarintField(field.value)) !== 0
    }
  }
  return pending
}

export function decodeAppUpdatePendingNotification(payload: Uint8Array): ProtoAppUpdatePendingNotification {
  const notification: ProtoAppUpdatePendingNotification = {}
  for (const field of readFields(payload)) {
    if (field.wireType !== WireType.LengthDelimited) {
      continue
    }
    const text = decodeString(field.value)
    if (field.fieldNumber === 1) {
      notification.version = text
    } else if (field.fieldNumber === 2) {
      notification.message = text
    }
  }
  return notification
}

export function decodeAppUpdatePendingInstallFailed(payload: Uint8Array): ProtoAppUpdatePendingInstallFailed {
  const failure: ProtoAppUpdatePendingInstallFailed = {}
  for (const field of readFields(payload)) {
    if (field.wireType !== WireType.LengthDelimited) {
      continue
    }
    const text = decodeString(field.value)
    if (field.fieldNumber === 1) {
      failure.version = text
    } else if (field.fieldNumber === 2) {
      failure.error = text
    }
  }
  return failure
}
