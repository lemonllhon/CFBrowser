import {
  METHOD_BROWSER_CORE_DELETE,
  METHOD_BROWSER_CORE_DOWNLOAD,
  METHOD_BROWSER_CORE_EXTENDED_INFO,
  METHOD_BROWSER_CORE_LIST,
  METHOD_BROWSER_CORE_OPEN_PATH,
  METHOD_BROWSER_CORE_SAVE,
  METHOD_BROWSER_CORE_SCAN,
  METHOD_BROWSER_CORE_SET_DEFAULT,
  METHOD_BROWSER_CORE_VALIDATE,
} from './envelope'
import {
  WireType,
  concatBytes,
  decodeString,
  decodeVarintField,
  encodeBytesField,
  encodeInt32Field,
  encodeStringField,
  readFields,
} from './protobuf'
import { ProtoIpcClient } from './transport'
import { decodeBrowserActionResponse } from './browser'

const coreProtoClient = new ProtoIpcClient()

export type ProtoBrowserCore = {
  coreId: string
  coreName: string
  corePath: string
  isDefault: boolean
}

export type ProtoBrowserCoreInput = ProtoBrowserCore

export type ProtoBrowserCoreValidateResult = {
  valid: boolean
  message: string
}

export type ProtoBrowserCoreExtended = {
  coreId: string
  chromeVersion: string
  instanceCount: number
}

export type ProtoBrowserCoreDownloadProgress = {
  phase: string
  progress: number
  message: string
}

export async function listBrowserCores(): Promise<ProtoBrowserCore[]> {
  const payload = await coreProtoClient.request(METHOD_BROWSER_CORE_LIST, new Uint8Array())
  return decodeBrowserCoreListResponse(payload).cores
}

export async function saveBrowserCore(core: ProtoBrowserCoreInput): Promise<boolean> {
  const payload = await coreProtoClient.request(METHOD_BROWSER_CORE_SAVE, encodeBrowserCoreSaveRequest({ core }))
  return decodeBrowserActionResponse(payload).ok
}

export async function deleteBrowserCore(coreId: string): Promise<boolean> {
  const payload = await coreProtoClient.request(METHOD_BROWSER_CORE_DELETE, encodeBrowserCoreIDRequest({ coreId }))
  return decodeBrowserActionResponse(payload).ok
}

export async function setDefaultBrowserCore(coreId: string): Promise<boolean> {
  const payload = await coreProtoClient.request(METHOD_BROWSER_CORE_SET_DEFAULT, encodeBrowserCoreIDRequest({ coreId }))
  return decodeBrowserActionResponse(payload).ok
}

export async function validateBrowserCorePath(corePath: string): Promise<ProtoBrowserCoreValidateResult> {
  const payload = await coreProtoClient.request(METHOD_BROWSER_CORE_VALIDATE, encodeBrowserCorePathRequest({ corePath }))
  return decodeBrowserCoreValidateResponse(payload)
}

export async function getBrowserCoreExtendedInfo(): Promise<ProtoBrowserCoreExtended[]> {
  const payload = await coreProtoClient.request(METHOD_BROWSER_CORE_EXTENDED_INFO, new Uint8Array())
  return decodeBrowserCoreExtendedInfoResponse(payload).items
}

export async function scanBrowserCores(): Promise<ProtoBrowserCore[]> {
  const payload = await coreProtoClient.request(METHOD_BROWSER_CORE_SCAN, new Uint8Array())
  return decodeBrowserCoreListResponse(payload).cores
}

export async function downloadBrowserCore(coreName: string, url: string, proxyConfig = ''): Promise<boolean> {
  const payload = await coreProtoClient.request(METHOD_BROWSER_CORE_DOWNLOAD, encodeBrowserCoreDownloadRequest({ coreName, url, proxyConfig }), 30000)
  return decodeBrowserActionResponse(payload).ok
}

export async function openBrowserCorePath(corePath: string): Promise<boolean> {
  const payload = await coreProtoClient.request(METHOD_BROWSER_CORE_OPEN_PATH, encodeBrowserCorePathRequest({ corePath }))
  return decodeBrowserActionResponse(payload).ok
}

export function onBrowserCoreDownloadProgress(callback: (progress: ProtoBrowserCoreDownloadProgress) => void): () => void {
  return coreProtoClient.onEvent('download:progress', event => callback(decodeBrowserCoreDownloadProgress(event.payload)))
}

export function encodeBrowserCoreSaveRequest(message: { core: ProtoBrowserCoreInput }): Uint8Array {
  return concatBytes([encodeBytesField(1, encodeBrowserCore(message.core))])
}

export function encodeBrowserCoreIDRequest(message: { coreId: string }): Uint8Array {
  return concatBytes([encodeStringField(1, message.coreId)])
}

export function encodeBrowserCorePathRequest(message: { corePath: string }): Uint8Array {
  return concatBytes([encodeStringField(1, message.corePath)])
}

export function encodeBrowserCoreDownloadRequest(message: { coreName: string; url: string; proxyConfig?: string }): Uint8Array {
  return concatBytes([
    encodeStringField(1, message.coreName),
    encodeStringField(2, message.url),
    encodeStringField(3, message.proxyConfig ?? ''),
  ])
}

export function encodeBrowserCore(core: ProtoBrowserCore): Uint8Array {
  return concatBytes([
    encodeStringField(1, core.coreId),
    encodeStringField(2, core.coreName),
    encodeStringField(3, core.corePath),
    encodeInt32Field(4, core.isDefault ? 1 : 0),
  ])
}

export function decodeBrowserCoreListResponse(payload: Uint8Array): { cores: ProtoBrowserCore[] } {
  const cores: ProtoBrowserCore[] = []
  for (const field of readFields(payload)) {
    if (field.fieldNumber === 1 && field.wireType === WireType.LengthDelimited) {
      cores.push(decodeBrowserCore(field.value))
    }
  }
  return { cores }
}

export function decodeBrowserCoreValidateResponse(payload: Uint8Array): ProtoBrowserCoreValidateResult {
  const result: ProtoBrowserCoreValidateResult = {
    valid: false,
    message: '',
  }
  for (const field of readFields(payload)) {
    if (field.fieldNumber === 1 && field.wireType === WireType.Varint) {
      result.valid = Number(decodeVarintField(field.value)) !== 0
    } else if (field.fieldNumber === 2 && field.wireType === WireType.LengthDelimited) {
      result.message = decodeString(field.value)
    }
  }
  return result
}

export function decodeBrowserCoreExtendedInfoResponse(payload: Uint8Array): { items: ProtoBrowserCoreExtended[] } {
  const items: ProtoBrowserCoreExtended[] = []
  for (const field of readFields(payload)) {
    if (field.fieldNumber === 1 && field.wireType === WireType.LengthDelimited) {
      items.push(decodeBrowserCoreExtendedInfo(field.value))
    }
  }
  return { items }
}

export function decodeBrowserCoreDownloadProgress(payload: Uint8Array): ProtoBrowserCoreDownloadProgress {
  const progress: ProtoBrowserCoreDownloadProgress = {
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

export function decodeBrowserCore(payload: Uint8Array): ProtoBrowserCore {
  const core: ProtoBrowserCore = {
    coreId: '',
    coreName: '',
    corePath: '',
    isDefault: false,
  }
  for (const field of readFields(payload)) {
    if (field.wireType === WireType.LengthDelimited) {
      const text = decodeString(field.value)
      switch (field.fieldNumber) {
        case 1:
          core.coreId = text
          break
        case 2:
          core.coreName = text
          break
        case 3:
          core.corePath = text
          break
      }
      continue
    }
    if (field.fieldNumber === 4 && field.wireType === WireType.Varint) {
      core.isDefault = Number(decodeVarintField(field.value)) !== 0
    }
  }
  return core
}

export function decodeBrowserCoreExtendedInfo(payload: Uint8Array): ProtoBrowserCoreExtended {
  const info: ProtoBrowserCoreExtended = {
    coreId: '',
    chromeVersion: '',
    instanceCount: 0,
  }
  for (const field of readFields(payload)) {
    if (field.wireType === WireType.LengthDelimited) {
      const text = decodeString(field.value)
      if (field.fieldNumber === 1) {
        info.coreId = text
      } else if (field.fieldNumber === 2) {
        info.chromeVersion = text
      }
      continue
    }
    if (field.fieldNumber === 3 && field.wireType === WireType.Varint) {
      info.instanceCount = Number(decodeVarintField(field.value))
    }
  }
  return info
}
