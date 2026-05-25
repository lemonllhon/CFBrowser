import {
  METHOD_BROWSER_BOOKMARK_LIST,
  METHOD_BROWSER_BOOKMARK_RESET,
  METHOD_BROWSER_BOOKMARK_SAVE,
  METHOD_BROWSER_DEFAULT_CONTENT_RULE_LIST,
  METHOD_BROWSER_DEFAULT_CONTENT_RULE_SAVE,
  METHOD_BROWSER_DEFAULT_START_URL_LIST,
  METHOD_BROWSER_DEFAULT_START_URL_RESET,
  METHOD_BROWSER_DEFAULT_START_URL_SAVE,
} from './envelope'
import {
  WireType,
  concatBytes,
  decodeString,
  decodeVarintField,
  encodeBoolField,
  encodeBytesField,
  encodeInt32Field,
  encodeStringField,
  readFields,
} from './protobuf'
import { ProtoIpcClient } from './transport'
import { decodeBrowserActionResponse } from './browser'

const browserDefaultsProtoClient = new ProtoIpcClient()

export type ProtoBrowserBookmark = {
  name: string
  url: string
}

export type ProtoBrowserStartURL = {
  name: string
  url: string
}

export type ProtoBrowserDefaultContentScope = 'tag' | 'group'

export type ProtoBrowserDefaultContentRule = {
  ruleId: string
  scope: ProtoBrowserDefaultContentScope
  targetId?: string
  targetName: string
  startUrls: ProtoBrowserStartURL[]
  bookmarks: ProtoBrowserBookmark[]
  enabled: boolean
  applyToChilds?: boolean
  includeGlobalDefaults?: boolean
}

export async function listBrowserBookmarks(): Promise<ProtoBrowserBookmark[]> {
  const payload = await browserDefaultsProtoClient.request(METHOD_BROWSER_BOOKMARK_LIST, new Uint8Array())
  return decodeBrowserBookmarkListResponse(payload).items
}

export async function saveBrowserBookmarks(items: ProtoBrowserBookmark[]): Promise<boolean> {
  const payload = await browserDefaultsProtoClient.request(METHOD_BROWSER_BOOKMARK_SAVE, encodeBrowserBookmarkSaveRequest({ items }))
  return decodeBrowserActionResponse(payload).ok
}

export async function resetBrowserBookmarks(): Promise<boolean> {
  const payload = await browserDefaultsProtoClient.request(METHOD_BROWSER_BOOKMARK_RESET, new Uint8Array())
  return decodeBrowserActionResponse(payload).ok
}

export async function listBrowserDefaultStartURLs(): Promise<ProtoBrowserStartURL[]> {
  const payload = await browserDefaultsProtoClient.request(METHOD_BROWSER_DEFAULT_START_URL_LIST, new Uint8Array())
  return decodeBrowserStartURLListResponse(payload).items
}

export async function saveBrowserDefaultStartURLs(items: ProtoBrowserStartURL[]): Promise<boolean> {
  const payload = await browserDefaultsProtoClient.request(METHOD_BROWSER_DEFAULT_START_URL_SAVE, encodeBrowserStartURLSaveRequest({ items }))
  return decodeBrowserActionResponse(payload).ok
}

export async function resetBrowserDefaultStartURLs(): Promise<boolean> {
  const payload = await browserDefaultsProtoClient.request(METHOD_BROWSER_DEFAULT_START_URL_RESET, new Uint8Array())
  return decodeBrowserActionResponse(payload).ok
}

export async function listBrowserDefaultContentRules(): Promise<ProtoBrowserDefaultContentRule[]> {
  const payload = await browserDefaultsProtoClient.request(METHOD_BROWSER_DEFAULT_CONTENT_RULE_LIST, new Uint8Array())
  return decodeBrowserDefaultContentRuleListResponse(payload).rules
}

export async function saveBrowserDefaultContentRules(rules: ProtoBrowserDefaultContentRule[]): Promise<boolean> {
  const payload = await browserDefaultsProtoClient.request(METHOD_BROWSER_DEFAULT_CONTENT_RULE_SAVE, encodeBrowserDefaultContentRuleSaveRequest({ rules }))
  return decodeBrowserActionResponse(payload).ok
}

export function encodeBrowserBookmarkSaveRequest(message: { items: ProtoBrowserBookmark[] }): Uint8Array {
  return concatBytes(message.items.map(item => encodeBytesField(1, encodeBrowserBookmark(item))))
}

export function encodeBrowserStartURLSaveRequest(message: { items: ProtoBrowserStartURL[] }): Uint8Array {
  return concatBytes(message.items.map(item => encodeBytesField(1, encodeBrowserStartURL(item))))
}

export function encodeBrowserDefaultContentRuleSaveRequest(message: { rules: ProtoBrowserDefaultContentRule[] }): Uint8Array {
  return concatBytes(message.rules.map(rule => encodeBytesField(1, encodeBrowserDefaultContentRule(rule))))
}

export function decodeBrowserBookmarkListResponse(payload: Uint8Array): { items: ProtoBrowserBookmark[] } {
  const items: ProtoBrowserBookmark[] = []
  for (const field of readFields(payload)) {
    if (field.fieldNumber === 1 && field.wireType === WireType.LengthDelimited) {
      items.push(decodeBrowserBookmark(field.value))
    }
  }
  return { items }
}

export function decodeBrowserStartURLListResponse(payload: Uint8Array): { items: ProtoBrowserStartURL[] } {
  const items: ProtoBrowserStartURL[] = []
  for (const field of readFields(payload)) {
    if (field.fieldNumber === 1 && field.wireType === WireType.LengthDelimited) {
      items.push(decodeBrowserStartURL(field.value))
    }
  }
  return { items }
}

export function decodeBrowserDefaultContentRuleListResponse(payload: Uint8Array): { rules: ProtoBrowserDefaultContentRule[] } {
  const rules: ProtoBrowserDefaultContentRule[] = []
  for (const field of readFields(payload)) {
    if (field.fieldNumber === 1 && field.wireType === WireType.LengthDelimited) {
      rules.push(decodeBrowserDefaultContentRule(field.value))
    }
  }
  return { rules }
}

function encodeBrowserBookmark(item: ProtoBrowserBookmark): Uint8Array {
  return concatBytes([
    encodeStringField(1, item.name),
    encodeStringField(2, item.url),
  ])
}

function decodeBrowserBookmark(payload: Uint8Array): ProtoBrowserBookmark {
  const item: ProtoBrowserBookmark = { name: '', url: '' }
  for (const field of readFields(payload)) {
    if (field.wireType !== WireType.LengthDelimited) {
      continue
    }
    const text = decodeString(field.value)
    switch (field.fieldNumber) {
      case 1:
        item.name = text
        break
      case 2:
        item.url = text
        break
    }
  }
  return item
}

function encodeBrowserStartURL(item: ProtoBrowserStartURL): Uint8Array {
  return concatBytes([
    encodeStringField(1, item.name),
    encodeStringField(2, item.url),
  ])
}

function decodeBrowserStartURL(payload: Uint8Array): ProtoBrowserStartURL {
  const item: ProtoBrowserStartURL = { name: '', url: '' }
  for (const field of readFields(payload)) {
    if (field.wireType !== WireType.LengthDelimited) {
      continue
    }
    const text = decodeString(field.value)
    switch (field.fieldNumber) {
      case 1:
        item.name = text
        break
      case 2:
        item.url = text
        break
    }
  }
  return item
}

function encodeBrowserDefaultContentRule(rule: ProtoBrowserDefaultContentRule): Uint8Array {
  const includeGlobalDefaultsFields = rule.includeGlobalDefaults === undefined
    ? []
    : [
        encodeBoolField(9, rule.includeGlobalDefaults),
        encodeBoolField(10, true),
      ]

  return concatBytes([
    encodeStringField(1, rule.ruleId),
    encodeStringField(2, rule.scope),
    encodeStringField(3, rule.targetId ?? ''),
    encodeStringField(4, rule.targetName),
    ...rule.startUrls.map(item => encodeBytesField(5, encodeBrowserStartURL(item))),
    ...rule.bookmarks.map(item => encodeBytesField(6, encodeBrowserBookmark(item))),
    encodeBoolField(7, rule.enabled),
    encodeBoolField(8, rule.applyToChilds === true),
    ...includeGlobalDefaultsFields,
  ])
}

function decodeBrowserDefaultContentRule(payload: Uint8Array): ProtoBrowserDefaultContentRule {
  const rule: ProtoBrowserDefaultContentRule = {
    ruleId: '',
    scope: 'tag',
    targetId: '',
    targetName: '',
    startUrls: [],
    bookmarks: [],
    enabled: false,
    applyToChilds: false,
  }
  let includeGlobalDefaults = false
  let hasIncludeGlobalDefaults = false

  for (const field of readFields(payload)) {
    if (field.wireType === WireType.LengthDelimited) {
      switch (field.fieldNumber) {
        case 1:
          rule.ruleId = decodeString(field.value)
          break
        case 2:
          rule.scope = decodeDefaultContentScope(decodeString(field.value))
          break
        case 3:
          rule.targetId = decodeString(field.value)
          break
        case 4:
          rule.targetName = decodeString(field.value)
          break
        case 5:
          rule.startUrls.push(decodeBrowserStartURL(field.value))
          break
        case 6:
          rule.bookmarks.push(decodeBrowserBookmark(field.value))
          break
      }
      continue
    }

    if (field.wireType === WireType.Varint) {
      const value = Number(decodeVarintField(field.value)) !== 0
      switch (field.fieldNumber) {
        case 7:
          rule.enabled = value
          break
        case 8:
          rule.applyToChilds = value
          break
        case 9:
          includeGlobalDefaults = value
          hasIncludeGlobalDefaults = true
          break
        case 10:
          hasIncludeGlobalDefaults = value
          break
      }
    }
  }

  if (hasIncludeGlobalDefaults) {
    rule.includeGlobalDefaults = includeGlobalDefaults
  }
  return rule
}

function decodeDefaultContentScope(value: string): ProtoBrowserDefaultContentScope {
  return value === 'group' ? 'group' : 'tag'
}
