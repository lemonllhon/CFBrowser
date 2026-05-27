import {
  METHOD_BROWSER_INSTANCE_RESTART,
  METHOD_BROWSER_INSTANCE_OPEN_URL,
  METHOD_BROWSER_INSTANCE_PIN_CENTER,
  METHOD_BROWSER_INSTANCE_START,
  METHOD_BROWSER_INSTANCE_START_BY_CODE,
  METHOD_BROWSER_INSTANCE_STOP,
  METHOD_BROWSER_INSTANCE_TAB_LIST,
  METHOD_BROWSER_LAUNCH_SERVER_INFO,
  METHOD_BROWSER_GROUP_CREATE,
  METHOD_BROWSER_GROUP_DELETE,
  METHOD_BROWSER_GROUP_LIST,
  METHOD_BROWSER_GROUP_MOVE_PROFILES,
  METHOD_BROWSER_GROUP_UPDATE,
  METHOD_BROWSER_PROFILE_BATCH_REMOVE_TAGS,
  METHOD_BROWSER_PROFILE_BATCH_SET_TAGS,
  METHOD_BROWSER_PROFILE_COPY,
  METHOD_BROWSER_PROFILE_CODE_GET,
  METHOD_BROWSER_PROFILE_CODE_REGENERATE,
  METHOD_BROWSER_PROFILE_CODE_SET,
  METHOD_BROWSER_PROFILE_CREATE,
  METHOD_BROWSER_PROFILE_DELETE,
  METHOD_BROWSER_PROFILE_LIST,
  METHOD_BROWSER_PROFILE_SET_KEYWORDS,
  METHOD_BROWSER_PROFILE_SWITCH_PROXY_NOW,
  METHOD_BROWSER_PROFILE_UPDATE,
  METHOD_BROWSER_TAG_LIST,
  METHOD_BROWSER_TAG_RENAME,
  decodeRpcResponse,
  encodeRpcEnvelope,
} from './envelope'
import type { RpcError } from './envelope'
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
import { ProtoIpcClient, ProtoIpcError } from './transport'

const browserProtoClient = new ProtoIpcClient()

export type ProtoBrowserProfile = {
  profileId: string
  profileName: string
  userDataDir: string
  coreId: string
  fingerprintArgs: string[]
  proxyId: string
  proxyConfig: string
  proxyBindSourceId?: string
  proxyBindSourceUrl?: string
  proxyBindName?: string
  proxyBindUpdatedAt?: string
  autoProxySwitchEnabled?: boolean
  autoProxySwitchGroupName?: string
  autoProxySwitchMode?: string
  autoProxySwitchIntervalM?: number
  autoProxySwitchRotateByGroup?: boolean
  autoProxySwitchLastProxyId?: string
  launchArgs: string[]
  tags: string[]
  keywords: string[]
  groupId?: string
  launchCode?: string
  running: boolean
  debugPort: number
  debugReady: boolean
  pid: number
  runtimeWarning: string
  lastError: string
  createdAt: string
  updatedAt: string
  lastStartAt?: string
  lastStopAt?: string
}

export type ProtoBrowserProfileInput = {
  profileName: string
  userDataDir: string
  coreId: string
  fingerprintArgs: string[]
  proxyId: string
  proxyConfig: string
  autoProxySwitchEnabled?: boolean
  autoProxySwitchGroupName?: string
  autoProxySwitchMode?: string
  autoProxySwitchIntervalM?: number
  autoProxySwitchRotateByGroup?: boolean
  launchArgs: string[]
  tags: string[]
  keywords: string[]
  groupId?: string
}

export type ProtoBrowserGroup = {
  groupId: string
  groupName: string
  parentId: string
  sortOrder: number
  createdAt: string
  updatedAt: string
  instanceCount: number
}

export type ProtoBrowserGroupInput = {
  groupName: string
  parentId: string
  sortOrder: number
}

export type ProtoBrowserTab = {
  tabId: string
  title: string
  url: string
  active: boolean
}

export type ProtoLaunchServerInfo = {
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

export async function listBrowserProfiles(tag = ''): Promise<ProtoBrowserProfile[]> {
  const payload = await browserProtoClient.request(METHOD_BROWSER_PROFILE_LIST, encodeBrowserProfileListRequest({ tag }))
  return decodeBrowserProfileListResponse(payload).profiles
}

export async function createBrowserProfile(input: ProtoBrowserProfileInput): Promise<ProtoBrowserProfile | null> {
  const payload = await browserProtoClient.request(METHOD_BROWSER_PROFILE_CREATE, encodeBrowserProfileCreateRequest({ profile: input }))
  return decodeBrowserProfileResponse(payload).profile
}

export async function updateBrowserProfile(profileId: string, input: ProtoBrowserProfileInput): Promise<ProtoBrowserProfile | null> {
  const payload = await browserProtoClient.request(METHOD_BROWSER_PROFILE_UPDATE, encodeBrowserProfileUpdateRequest({ profileId, profile: input }))
  return decodeBrowserProfileResponse(payload).profile
}

export async function deleteBrowserProfile(profileId: string): Promise<boolean> {
  const payload = await browserProtoClient.request(METHOD_BROWSER_PROFILE_DELETE, encodeBrowserProfileDeleteRequest({ profileId }))
  return decodeBrowserProfileDeleteResponse(payload).deleted
}

export async function copyBrowserProfile(profileId: string, newName: string): Promise<ProtoBrowserProfile | null> {
  const payload = await browserProtoClient.request(METHOD_BROWSER_PROFILE_COPY, encodeBrowserProfileCopyRequest({ profileId, newName }))
  return decodeBrowserProfileResponse(payload).profile
}

export async function startBrowserInstance(profileId: string): Promise<ProtoBrowserProfile | null> {
  const payload = await browserProtoClient.request(METHOD_BROWSER_INSTANCE_START, encodeBrowserInstanceProfileRequest({ profileId }))
  return decodeBrowserProfileResponse(payload).profile
}

export async function startBrowserInstanceByCode(code: string): Promise<ProtoBrowserProfile | null> {
  const payload = await browserProtoClient.request(METHOD_BROWSER_INSTANCE_START_BY_CODE, encodeBrowserInstanceStartByCodeRequest({ code }))
  return decodeBrowserProfileResponse(payload).profile
}

export async function stopBrowserInstance(profileId: string): Promise<ProtoBrowserProfile | null> {
  const payload = await browserProtoClient.request(METHOD_BROWSER_INSTANCE_STOP, encodeBrowserInstanceProfileRequest({ profileId }))
  return decodeBrowserProfileResponse(payload).profile
}

export async function restartBrowserInstance(profileId: string): Promise<ProtoBrowserProfile | null> {
  const payload = await browserProtoClient.request(METHOD_BROWSER_INSTANCE_RESTART, encodeBrowserInstanceProfileRequest({ profileId }))
  return decodeBrowserProfileResponse(payload).profile
}

export async function pinCenterBrowserInstance(profileId: string): Promise<boolean> {
  const payload = await browserProtoClient.request(METHOD_BROWSER_INSTANCE_PIN_CENTER, encodeBrowserInstanceProfileRequest({ profileId }))
  return decodeBrowserActionResponse(payload).ok
}

export async function switchBrowserProfileProxyNow(profileId: string): Promise<ProtoBrowserProfile | null> {
  const payload = await browserProtoClient.request(METHOD_BROWSER_PROFILE_SWITCH_PROXY_NOW, encodeBrowserInstanceProfileRequest({ profileId }))
  return decodeBrowserProfileResponse(payload).profile
}

export async function openBrowserURL(profileId: string, targetUrl: string): Promise<boolean> {
  const payload = await browserProtoClient.request(METHOD_BROWSER_INSTANCE_OPEN_URL, encodeBrowserInstanceOpenURLRequest({ profileId, targetUrl }))
  return decodeBrowserActionResponse(payload).ok
}

export async function listBrowserTabs(profileId: string): Promise<ProtoBrowserTab[]> {
  const payload = await browserProtoClient.request(METHOD_BROWSER_INSTANCE_TAB_LIST, encodeBrowserInstanceProfileRequest({ profileId }))
  return decodeBrowserTabListResponse(payload).tabs
}

export async function getBrowserProfileCode(profileId: string): Promise<string> {
  const payload = await browserProtoClient.request(METHOD_BROWSER_PROFILE_CODE_GET, encodeBrowserProfileCodeRequest({ profileId }))
  return decodeBrowserProfileCodeResponse(payload).code
}

export async function regenerateBrowserProfileCode(profileId: string): Promise<string> {
  const payload = await browserProtoClient.request(METHOD_BROWSER_PROFILE_CODE_REGENERATE, encodeBrowserProfileCodeRequest({ profileId }))
  return decodeBrowserProfileCodeResponse(payload).code
}

export async function setBrowserProfileCode(profileId: string, code: string): Promise<string> {
  const payload = await browserProtoClient.request(METHOD_BROWSER_PROFILE_CODE_SET, encodeBrowserProfileSetCodeRequest({ profileId, code }))
  return decodeBrowserProfileCodeResponse(payload).code
}

export async function getLaunchServerInfo(): Promise<ProtoLaunchServerInfo> {
  const payload = await browserProtoClient.request(METHOD_BROWSER_LAUNCH_SERVER_INFO, new Uint8Array())
  return decodeBrowserLaunchServerInfoResponse(payload)
}

export async function listBrowserTags(): Promise<string[]> {
  const payload = await browserProtoClient.request(METHOD_BROWSER_TAG_LIST, new Uint8Array())
  return decodeBrowserTagListResponse(payload).tags
}

export async function setBrowserProfileKeywords(profileId: string, keywords: string[]): Promise<ProtoBrowserProfile | null> {
  const payload = await browserProtoClient.request(METHOD_BROWSER_PROFILE_SET_KEYWORDS, encodeBrowserProfileSetKeywordsRequest({ profileId, keywords }))
  return decodeBrowserProfileResponse(payload).profile
}

export async function batchSetBrowserProfileTags(profileIds: string[], tags: string[], replace: boolean): Promise<boolean> {
  const payload = await browserProtoClient.request(METHOD_BROWSER_PROFILE_BATCH_SET_TAGS, encodeBrowserProfileBatchSetTagsRequest({ profileIds, tags, replace }))
  return decodeBrowserActionResponse(payload).ok
}

export async function batchRemoveBrowserProfileTags(profileIds: string[], tags: string[]): Promise<boolean> {
  const payload = await browserProtoClient.request(METHOD_BROWSER_PROFILE_BATCH_REMOVE_TAGS, encodeBrowserProfileBatchRemoveTagsRequest({ profileIds, tags }))
  return decodeBrowserActionResponse(payload).ok
}

export async function renameBrowserTag(oldName: string, newName: string): Promise<boolean> {
  const payload = await browserProtoClient.request(METHOD_BROWSER_TAG_RENAME, encodeBrowserTagRenameRequest({ oldName, newName }))
  return decodeBrowserActionResponse(payload).ok
}

export async function listBrowserGroups(): Promise<ProtoBrowserGroup[]> {
  const payload = await browserProtoClient.request(METHOD_BROWSER_GROUP_LIST, new Uint8Array())
  return decodeBrowserGroupListResponse(payload).groups
}

export async function createBrowserGroup(input: ProtoBrowserGroupInput): Promise<ProtoBrowserGroup | null> {
  const payload = await browserProtoClient.request(METHOD_BROWSER_GROUP_CREATE, encodeBrowserGroupCreateRequest({ group: input }))
  return decodeBrowserGroupResponse(payload).group
}

export async function updateBrowserGroup(groupId: string, input: ProtoBrowserGroupInput): Promise<ProtoBrowserGroup | null> {
  const payload = await browserProtoClient.request(METHOD_BROWSER_GROUP_UPDATE, encodeBrowserGroupUpdateRequest({ groupId, group: input }))
  return decodeBrowserGroupResponse(payload).group
}

export async function deleteBrowserGroup(groupId: string): Promise<boolean> {
  const payload = await browserProtoClient.request(METHOD_BROWSER_GROUP_DELETE, encodeBrowserGroupDeleteRequest({ groupId }))
  return decodeBrowserActionResponse(payload).ok
}

export async function moveBrowserProfilesToGroup(profileIds: string[], groupId: string): Promise<boolean> {
  const payload = await browserProtoClient.request(METHOD_BROWSER_GROUP_MOVE_PROFILES, encodeBrowserGroupMoveProfilesRequest({ profileIds, groupId }))
  return decodeBrowserActionResponse(payload).ok
}

export function encodeBrowserProfileListRequest(message: { tag?: string }): Uint8Array {
  return concatBytes([encodeStringField(1, message.tag ?? '')])
}

export function encodeBrowserProfileCreateRequest(message: { profile: ProtoBrowserProfileInput }): Uint8Array {
  return concatBytes([encodeBytesField(1, encodeBrowserProfileInput(message.profile))])
}

export function encodeBrowserProfileUpdateRequest(message: { profileId: string; profile: ProtoBrowserProfileInput }): Uint8Array {
  return concatBytes([
    encodeStringField(1, message.profileId),
    encodeBytesField(2, encodeBrowserProfileInput(message.profile)),
  ])
}

export function encodeBrowserProfileDeleteRequest(message: { profileId: string }): Uint8Array {
  return concatBytes([encodeStringField(1, message.profileId)])
}

export function encodeBrowserProfileCopyRequest(message: { profileId: string; newName: string }): Uint8Array {
  return concatBytes([
    encodeStringField(1, message.profileId),
    encodeStringField(2, message.newName),
  ])
}

export function encodeBrowserInstanceProfileRequest(message: { profileId: string }): Uint8Array {
  return concatBytes([encodeStringField(1, message.profileId)])
}

export function encodeBrowserInstanceStartByCodeRequest(message: { code: string }): Uint8Array {
  return concatBytes([encodeStringField(1, message.code)])
}

export function encodeBrowserInstanceOpenURLRequest(message: { profileId: string; targetUrl: string }): Uint8Array {
  return concatBytes([
    encodeStringField(1, message.profileId),
    encodeStringField(2, message.targetUrl),
  ])
}

export function encodeBrowserProfileCodeRequest(message: { profileId: string }): Uint8Array {
  return concatBytes([encodeStringField(1, message.profileId)])
}

export function encodeBrowserProfileSetCodeRequest(message: { profileId: string; code: string }): Uint8Array {
  return concatBytes([
    encodeStringField(1, message.profileId),
    encodeStringField(2, message.code),
  ])
}

export function encodeBrowserProfileSetKeywordsRequest(message: { profileId: string; keywords: string[] }): Uint8Array {
  return concatBytes([
    encodeStringField(1, message.profileId),
    ...encodeRepeatedStringFields(2, message.keywords),
  ])
}

export function encodeBrowserProfileBatchSetTagsRequest(message: { profileIds: string[]; tags: string[]; replace: boolean }): Uint8Array {
  return concatBytes([
    ...encodeRepeatedStringFields(1, message.profileIds),
    ...encodeRepeatedStringFields(2, message.tags),
    encodeBoolField(3, message.replace),
  ])
}

export function encodeBrowserProfileBatchRemoveTagsRequest(message: { profileIds: string[]; tags: string[] }): Uint8Array {
  return concatBytes([
    ...encodeRepeatedStringFields(1, message.profileIds),
    ...encodeRepeatedStringFields(2, message.tags),
  ])
}

export function encodeBrowserTagRenameRequest(message: { oldName: string; newName: string }): Uint8Array {
  return concatBytes([
    encodeStringField(1, message.oldName),
    encodeStringField(2, message.newName),
  ])
}

export function encodeBrowserGroupCreateRequest(message: { group: ProtoBrowserGroupInput }): Uint8Array {
  return concatBytes([encodeBytesField(1, encodeBrowserGroupInput(message.group))])
}

export function encodeBrowserGroupUpdateRequest(message: { groupId: string; group: ProtoBrowserGroupInput }): Uint8Array {
  return concatBytes([
    encodeStringField(1, message.groupId),
    encodeBytesField(2, encodeBrowserGroupInput(message.group)),
  ])
}

export function encodeBrowserGroupDeleteRequest(message: { groupId: string }): Uint8Array {
  return concatBytes([encodeStringField(1, message.groupId)])
}

export function encodeBrowserGroupMoveProfilesRequest(message: { profileIds: string[]; groupId: string }): Uint8Array {
  return concatBytes([
    ...encodeRepeatedStringFields(1, message.profileIds),
    encodeStringField(2, message.groupId),
  ])
}

export function encodeBrowserGroupInput(input: ProtoBrowserGroupInput): Uint8Array {
  return concatBytes([
    encodeStringField(1, input.groupName),
    encodeStringField(2, input.parentId),
    encodeInt32Field(3, input.sortOrder),
  ])
}

export function encodeBrowserProfileInput(input: ProtoBrowserProfileInput): Uint8Array {
  return concatBytes([
    encodeStringField(1, input.profileName),
    encodeStringField(2, input.userDataDir),
    encodeStringField(3, input.coreId),
    ...encodeRepeatedStringFields(4, input.fingerprintArgs),
    encodeStringField(5, input.proxyId),
    encodeStringField(6, input.proxyConfig),
    encodeBoolField(7, input.autoProxySwitchEnabled === true),
    encodeStringField(8, input.autoProxySwitchGroupName ?? ''),
    encodeStringField(9, input.autoProxySwitchMode ?? ''),
    encodeInt32Field(10, input.autoProxySwitchIntervalM ?? 0),
    ...encodeRepeatedStringFields(11, input.launchArgs),
    ...encodeRepeatedStringFields(12, input.tags),
    ...encodeRepeatedStringFields(13, input.keywords),
    encodeStringField(14, input.groupId ?? ''),
    encodeBoolField(15, input.autoProxySwitchRotateByGroup === true),
  ])
}

export function decodeBrowserProfileListResponse(payload: Uint8Array): { profiles: ProtoBrowserProfile[] } {
  const profiles: ProtoBrowserProfile[] = []
  for (const field of readFields(payload)) {
    if (field.fieldNumber === 1 && field.wireType === WireType.LengthDelimited) {
      profiles.push(decodeBrowserProfile(field.value))
    }
  }
  return { profiles }
}

export function decodeBrowserProfileResponse(payload: Uint8Array): { profile: ProtoBrowserProfile | null } {
  let profile: ProtoBrowserProfile | null = null
  for (const field of readFields(payload)) {
    if (field.fieldNumber === 1 && field.wireType === WireType.LengthDelimited) {
      profile = decodeBrowserProfile(field.value)
    }
  }
  return { profile }
}

export function decodeBrowserProfileDeleteResponse(payload: Uint8Array): { deleted: boolean } {
  let deleted = false
  for (const field of readFields(payload)) {
    if (field.fieldNumber === 1 && field.wireType === WireType.Varint) {
      deleted = Number(decodeVarintField(field.value)) !== 0
    }
  }
  return { deleted }
}

export function decodeBrowserTagListResponse(payload: Uint8Array): { tags: string[] } {
  const tags: string[] = []
  for (const field of readFields(payload)) {
    if (field.fieldNumber === 1 && field.wireType === WireType.LengthDelimited) {
      tags.push(decodeString(field.value))
    }
  }
  return { tags }
}

export function decodeBrowserActionResponse(payload: Uint8Array): { ok: boolean } {
  let ok = false
  for (const field of readFields(payload)) {
    if (field.fieldNumber === 1 && field.wireType === WireType.Varint) {
      ok = Number(decodeVarintField(field.value)) !== 0
    }
  }
  return { ok }
}

export function decodeBrowserGroupListResponse(payload: Uint8Array): { groups: ProtoBrowserGroup[] } {
  const groups: ProtoBrowserGroup[] = []
  for (const field of readFields(payload)) {
    if (field.fieldNumber === 1 && field.wireType === WireType.LengthDelimited) {
      groups.push(decodeBrowserGroup(field.value))
    }
  }
  return { groups }
}

export function decodeBrowserGroupResponse(payload: Uint8Array): { group: ProtoBrowserGroup | null } {
  let group: ProtoBrowserGroup | null = null
  for (const field of readFields(payload)) {
    if (field.fieldNumber === 1 && field.wireType === WireType.LengthDelimited) {
      group = decodeBrowserGroup(field.value)
    }
  }
  return { group }
}

export function decodeBrowserTabListResponse(payload: Uint8Array): { tabs: ProtoBrowserTab[] } {
  const tabs: ProtoBrowserTab[] = []
  for (const field of readFields(payload)) {
    if (field.fieldNumber === 1 && field.wireType === WireType.LengthDelimited) {
      tabs.push(decodeBrowserTab(field.value))
    }
  }
  return { tabs }
}

export function decodeBrowserProfileCodeResponse(payload: Uint8Array): { code: string } {
  let code = ''
  for (const field of readFields(payload)) {
    if (field.fieldNumber === 1 && field.wireType === WireType.LengthDelimited) {
      code = decodeString(field.value)
    }
  }
  return { code }
}

export function decodeBrowserLaunchServerInfoResponse(payload: Uint8Array): ProtoLaunchServerInfo {
  const info: ProtoLaunchServerInfo = {
    host: '',
    port: 0,
    preferredPort: 0,
    baseUrl: '',
    cdpUrl: '',
    activeDebugPort: 0,
    ready: false,
    apiAuth: {
      requested: false,
      configured: false,
      enabled: false,
      header: '',
    },
  }

  for (const field of readFields(payload)) {
    if (field.wireType === WireType.LengthDelimited) {
      const text = decodeString(field.value)
      switch (field.fieldNumber) {
        case 1:
          info.host = text
          break
        case 4:
          info.baseUrl = text
          break
        case 5:
          info.cdpUrl = text
          break
        case 8:
          info.apiAuth = decodeBrowserLaunchServerAPIAuth(field.value)
          break
      }
      continue
    }

    if (field.wireType === WireType.Varint) {
      const number = Number(decodeVarintField(field.value))
      switch (field.fieldNumber) {
        case 2:
          info.port = number
          break
        case 3:
          info.preferredPort = number
          break
        case 6:
          info.activeDebugPort = number
          break
        case 7:
          info.ready = number !== 0
          break
      }
    }
  }

  return info
}

function decodeBrowserLaunchServerAPIAuth(payload: Uint8Array): ProtoLaunchServerInfo['apiAuth'] {
  const apiAuth: ProtoLaunchServerInfo['apiAuth'] = {
    requested: false,
    configured: false,
    enabled: false,
    header: '',
  }

  for (const field of readFields(payload)) {
    if (field.wireType === WireType.LengthDelimited && field.fieldNumber === 4) {
      apiAuth.header = decodeString(field.value)
      continue
    }
    if (field.wireType === WireType.Varint) {
      const value = Number(decodeVarintField(field.value)) !== 0
      switch (field.fieldNumber) {
        case 1:
          apiAuth.requested = value
          break
        case 2:
          apiAuth.configured = value
          break
        case 3:
          apiAuth.enabled = value
          break
      }
    }
  }

  return apiAuth
}

function decodeBrowserTab(payload: Uint8Array): ProtoBrowserTab {
  const tab: ProtoBrowserTab = {
    tabId: '',
    title: '',
    url: '',
    active: false,
  }

  for (const field of readFields(payload)) {
    if (field.wireType === WireType.LengthDelimited) {
      const text = decodeString(field.value)
      switch (field.fieldNumber) {
        case 1:
          tab.tabId = text
          break
        case 2:
          tab.title = text
          break
        case 3:
          tab.url = text
          break
      }
      continue
    }

    if (field.wireType === WireType.Varint && field.fieldNumber === 4) {
      tab.active = Number(decodeVarintField(field.value)) !== 0
    }
  }

  return tab
}

function decodeBrowserGroup(payload: Uint8Array): ProtoBrowserGroup {
  const group: ProtoBrowserGroup = {
    groupId: '',
    groupName: '',
    parentId: '',
    sortOrder: 0,
    createdAt: '',
    updatedAt: '',
    instanceCount: 0,
  }

  for (const field of readFields(payload)) {
    if (field.wireType === WireType.LengthDelimited) {
      const text = decodeString(field.value)
      switch (field.fieldNumber) {
        case 1:
          group.groupId = text
          break
        case 2:
          group.groupName = text
          break
        case 3:
          group.parentId = text
          break
        case 5:
          group.createdAt = text
          break
        case 6:
          group.updatedAt = text
          break
      }
      continue
    }

    if (field.wireType === WireType.Varint) {
      const number = Number(decodeVarintField(field.value))
      switch (field.fieldNumber) {
        case 4:
          group.sortOrder = number
          break
        case 7:
          group.instanceCount = number
          break
      }
    }
  }

  return group
}

function decodeBrowserProfile(payload: Uint8Array): ProtoBrowserProfile {
  const profile: ProtoBrowserProfile = {
    profileId: '',
    profileName: '',
    userDataDir: '',
    coreId: '',
    fingerprintArgs: [],
    proxyId: '',
    proxyConfig: '',
    launchArgs: [],
    tags: [],
    keywords: [],
    running: false,
    debugPort: 0,
    debugReady: false,
    pid: 0,
    runtimeWarning: '',
    lastError: '',
    createdAt: '',
    updatedAt: '',
  }

  for (const field of readFields(payload)) {
    if (field.wireType === WireType.LengthDelimited) {
      const text = decodeString(field.value)
      switch (field.fieldNumber) {
        case 1:
          profile.profileId = text
          break
        case 2:
          profile.profileName = text
          break
        case 3:
          profile.userDataDir = text
          break
        case 4:
          profile.coreId = text
          break
        case 5:
          profile.fingerprintArgs.push(text)
          break
        case 6:
          profile.proxyId = text
          break
        case 7:
          profile.proxyConfig = text
          break
        case 8:
          profile.proxyBindSourceId = text
          break
        case 9:
          profile.proxyBindSourceUrl = text
          break
        case 10:
          profile.proxyBindName = text
          break
        case 11:
          profile.proxyBindUpdatedAt = text
          break
        case 13:
          profile.autoProxySwitchGroupName = text
          break
        case 14:
          profile.autoProxySwitchMode = text
          break
        case 16:
          profile.autoProxySwitchLastProxyId = text
          break
        case 17:
          profile.launchArgs.push(text)
          break
        case 18:
          profile.tags.push(text)
          break
        case 19:
          profile.keywords.push(text)
          break
        case 20:
          profile.groupId = text
          break
        case 21:
          profile.launchCode = text
          break
        case 26:
          profile.runtimeWarning = text
          break
        case 27:
          profile.lastError = text
          break
        case 28:
          profile.createdAt = text
          break
        case 29:
          profile.updatedAt = text
          break
        case 30:
          profile.lastStartAt = text
          break
        case 31:
          profile.lastStopAt = text
          break
      }
      continue
    }

    if (field.wireType === WireType.Varint) {
      const number = Number(decodeVarintField(field.value))
      switch (field.fieldNumber) {
        case 12:
          profile.autoProxySwitchEnabled = number !== 0
          break
        case 15:
          profile.autoProxySwitchIntervalM = number
          break
        case 22:
          profile.running = number !== 0
          break
        case 23:
          profile.debugPort = number
          break
        case 24:
          profile.debugReady = number !== 0
          break
        case 25:
          profile.pid = number
          break
        case 32:
          profile.autoProxySwitchRotateByGroup = number !== 0
          break
      }
    }
  }

  return profile
}

function encodeRepeatedStringFields(fieldNumber: number, values: string[]): Uint8Array[] {
  return values.map(value => encodeStringField(fieldNumber, value))
}

export function decodeBrowserProfileListEnvelope(payload: Uint8Array): ProtoBrowserProfile[] {
  const response = decodeRpcResponse(payload)
  if (response.error) {
    throw new ProtoIpcError(response.error as RpcError)
  }
  return decodeBrowserProfileListResponse(response.payload).profiles
}

export function encodeBrowserProfileListEnvelope(requestId: string, tag = ''): Uint8Array {
  return encodeRpcEnvelope({
    requestId,
    method: METHOD_BROWSER_PROFILE_LIST,
    payload: encodeBrowserProfileListRequest({ tag }),
    schemaVersion: 1,
    timestampMs: Date.now(),
  })
}
