import {
  METHOD_BROWSER_SETTINGS_GET,
  METHOD_BROWSER_SETTINGS_SAVE,
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
import { decodeBrowserActionResponse } from './browser'
import { ProtoIpcClient } from './transport'

const browserSettingsProtoClient = new ProtoIpcClient()

export type ProtoBrowserSettings = {
  userDataRoot: string
  defaultFingerprintArgs: string[]
  defaultLaunchArgs: string[]
  defaultProxy: string
  startReadyTimeoutMs: number
  startStableWindowMs: number
}

export async function getBrowserSettings(): Promise<ProtoBrowserSettings> {
  const payload = await browserSettingsProtoClient.request(METHOD_BROWSER_SETTINGS_GET, new Uint8Array())
  return decodeBrowserSettingsResponse(payload).settings
}

export async function saveBrowserSettings(settings: ProtoBrowserSettings): Promise<boolean> {
  const payload = await browserSettingsProtoClient.request(METHOD_BROWSER_SETTINGS_SAVE, encodeBrowserSettingsSaveRequest({ settings }))
  return decodeBrowserActionResponse(payload).ok
}

export function encodeBrowserSettingsSaveRequest(message: { settings: ProtoBrowserSettings }): Uint8Array {
  return concatBytes([encodeBytesField(1, encodeBrowserSettings(message.settings))])
}

export function decodeBrowserSettingsResponse(payload: Uint8Array): { settings: ProtoBrowserSettings } {
  let settings = defaultBrowserSettings()
  for (const field of readFields(payload)) {
    if (field.fieldNumber === 1 && field.wireType === WireType.LengthDelimited) {
      settings = decodeBrowserSettings(field.value)
    }
  }
  return { settings }
}

function encodeBrowserSettings(settings: ProtoBrowserSettings): Uint8Array {
  return concatBytes([
    encodeStringField(1, settings.userDataRoot),
    ...settings.defaultFingerprintArgs.map(item => encodeStringField(2, item)),
    ...settings.defaultLaunchArgs.map(item => encodeStringField(3, item)),
    encodeStringField(4, settings.defaultProxy),
    encodeInt32Field(5, settings.startReadyTimeoutMs),
    encodeInt32Field(6, settings.startStableWindowMs),
  ])
}

function decodeBrowserSettings(payload: Uint8Array): ProtoBrowserSettings {
  const settings = defaultBrowserSettings()
  for (const field of readFields(payload)) {
    if (field.wireType === WireType.LengthDelimited) {
      const text = decodeString(field.value)
      switch (field.fieldNumber) {
        case 1:
          settings.userDataRoot = text
          break
        case 2:
          settings.defaultFingerprintArgs.push(text)
          break
        case 3:
          settings.defaultLaunchArgs.push(text)
          break
        case 4:
          settings.defaultProxy = text
          break
      }
      continue
    }
    if (field.wireType === WireType.Varint) {
      const value = Number(decodeVarintField(field.value))
      switch (field.fieldNumber) {
        case 5:
          settings.startReadyTimeoutMs = value
          break
        case 6:
          settings.startStableWindowMs = value
          break
      }
    }
  }
  return settings
}

function defaultBrowserSettings(): ProtoBrowserSettings {
  return {
    userDataRoot: 'data',
    defaultFingerprintArgs: [],
    defaultLaunchArgs: [],
    defaultProxy: '',
    startReadyTimeoutMs: 3000,
    startStableWindowMs: 1200,
  }
}
