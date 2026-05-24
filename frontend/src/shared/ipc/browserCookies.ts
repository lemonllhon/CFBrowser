import {
  METHOD_BROWSER_COOKIE_CLEAR,
  METHOD_BROWSER_COOKIE_EXPORT,
  METHOD_BROWSER_COOKIE_IMPORT,
  METHOD_BROWSER_COOKIE_LIST,
} from './envelope'
import {
  WireType,
  concatBytes,
  decodeString,
  decodeVarintField,
  encodeStringField,
  readFields,
} from './protobuf'
import { ProtoIpcClient } from './transport'
import { decodeBrowserActionResponse } from './browser'

const browserCookieProtoClient = new ProtoIpcClient()

export type ProtoBrowserCookieInfo = {
  name: string
  value: string
  domain: string
  path: string
  expires: number
  httpOnly: boolean
  secure: boolean
  sameSite: string
}

export type ProtoBrowserCookieImportResult = {
  imported: number
  skipped: number
}

export async function listBrowserCookies(profileId: string): Promise<ProtoBrowserCookieInfo[]> {
  const payload = await browserCookieProtoClient.request(METHOD_BROWSER_COOKIE_LIST, encodeBrowserCookieProfileRequest({ profileId }))
  return decodeBrowserCookieListResponse(payload).cookies
}

export async function clearBrowserCookies(profileId: string): Promise<boolean> {
  const payload = await browserCookieProtoClient.request(METHOD_BROWSER_COOKIE_CLEAR, encodeBrowserCookieProfileRequest({ profileId }))
  return decodeBrowserActionResponse(payload).ok
}

export async function exportBrowserCookies(profileId: string): Promise<string> {
  const payload = await browserCookieProtoClient.request(METHOD_BROWSER_COOKIE_EXPORT, encodeBrowserCookieProfileRequest({ profileId }))
  return decodeBrowserCookieExportResponse(payload).content
}

export async function importBrowserCookies(profileId: string, content: string): Promise<ProtoBrowserCookieImportResult> {
  const payload = await browserCookieProtoClient.request(METHOD_BROWSER_COOKIE_IMPORT, encodeBrowserCookieImportRequest({ profileId, content }))
  return decodeBrowserCookieImportResult(payload)
}

export function encodeBrowserCookieProfileRequest(message: { profileId: string }): Uint8Array {
  return concatBytes([encodeStringField(1, message.profileId)])
}

export function encodeBrowserCookieImportRequest(message: { profileId: string; content: string }): Uint8Array {
  return concatBytes([
    encodeStringField(1, message.profileId),
    encodeStringField(2, message.content),
  ])
}

export function decodeBrowserCookieListResponse(payload: Uint8Array): { cookies: ProtoBrowserCookieInfo[] } {
  const cookies: ProtoBrowserCookieInfo[] = []
  for (const field of readFields(payload)) {
    if (field.fieldNumber === 1 && field.wireType === WireType.LengthDelimited) {
      cookies.push(decodeBrowserCookieInfo(field.value))
    }
  }
  return { cookies }
}

export function decodeBrowserCookieExportResponse(payload: Uint8Array): { content: string } {
  let content = ''
  for (const field of readFields(payload)) {
    if (field.fieldNumber === 1 && field.wireType === WireType.LengthDelimited) {
      content = decodeString(field.value)
    }
  }
  return { content }
}

export function decodeBrowserCookieImportResult(payload: Uint8Array): ProtoBrowserCookieImportResult {
  const result: ProtoBrowserCookieImportResult = {
    imported: 0,
    skipped: 0,
  }
  for (const field of readFields(payload)) {
    if (field.wireType !== WireType.Varint) {
      continue
    }
    const value = Number(decodeVarintField(field.value))
    switch (field.fieldNumber) {
      case 1:
        result.imported = value
        break
      case 2:
        result.skipped = value
        break
    }
  }
  return result
}

function decodeBrowserCookieInfo(payload: Uint8Array): ProtoBrowserCookieInfo {
  const cookie: ProtoBrowserCookieInfo = {
    name: '',
    value: '',
    domain: '',
    path: '',
    expires: 0,
    httpOnly: false,
    secure: false,
    sameSite: '',
  }

  for (const field of readFields(payload)) {
    if (field.wireType === WireType.LengthDelimited) {
      const text = decodeString(field.value)
      switch (field.fieldNumber) {
        case 1:
          cookie.name = text
          break
        case 2:
          cookie.value = text
          break
        case 3:
          cookie.domain = text
          break
        case 4:
          cookie.path = text
          break
        case 8:
          cookie.sameSite = text
          break
      }
      continue
    }

    if (field.wireType === WireType.Varint) {
      switch (field.fieldNumber) {
        case 5:
          cookie.expires = decodeSignedInt64(field.value)
          break
        case 6:
          cookie.httpOnly = Number(decodeVarintField(field.value)) !== 0
          break
        case 7:
          cookie.secure = Number(decodeVarintField(field.value)) !== 0
          break
      }
    }
  }

  return cookie
}

function decodeSignedInt64(value: Uint8Array): number {
  const unsigned = decodeVarintField(value)
  const signBit = 1n << 63n
  const fullRange = 1n << 64n
  const signed = unsigned >= signBit ? unsigned - fullRange : unsigned
  return Number(signed)
}
