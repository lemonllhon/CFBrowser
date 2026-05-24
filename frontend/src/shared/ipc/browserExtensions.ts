import {
  METHOD_BROWSER_EXTENSION_ASSIGN_PROFILES,
  METHOD_BROWSER_EXTENSION_CHOOSE_ARCHIVE,
  METHOD_BROWSER_EXTENSION_CHOOSE_DIRECTORY,
  METHOD_BROWSER_EXTENSION_DELETE,
  METHOD_BROWSER_EXTENSION_GET,
  METHOD_BROWSER_EXTENSION_IMPORT_ARCHIVE,
  METHOD_BROWSER_EXTENSION_IMPORT_DIRECTORY,
  METHOD_BROWSER_EXTENSION_LIST,
  METHOD_BROWSER_EXTENSION_LIST_FOR_PROFILE,
  METHOD_BROWSER_EXTENSION_LIST_PROFILE_BINDINGS,
  METHOD_BROWSER_EXTENSION_SET_AUTO_BIND,
  METHOD_BROWSER_EXTENSION_UNASSIGN_PROFILES,
} from './envelope'
import {
  WireType,
  concatBytes,
  decodeString,
  decodeVarintField,
  encodeInt32Field,
  encodeStringField,
  readFields,
} from './protobuf'
import { decodeBrowserActionResponse } from './browser'
import { ProtoIpcClient } from './transport'

const browserExtensionProtoClient = new ProtoIpcClient()

export type ProtoBrowserExtension = {
  extensionId: string
  name: string
  version: string
  manifestVersion: number
  description: string
  sourceType: string
  sourceUrl: string
  installDir: string
  packagePath: string
  manifestJson: string
  boundCount: number
  autoBindEnabled: boolean
  autoBindMode: 'shared' | 'exclusive' | string
  createdAt: string
  updatedAt: string
}

export type ProtoBrowserExtensionBinding = {
  id: number
  profileId: string
  profileName: string
  extensionId: string
  extensionName: string
  extensionVersion: string
  mode: 'shared' | 'exclusive' | string
  enabled: boolean
  exclusiveDir: string
  createdAt: string
  updatedAt: string
}

export type ProtoBrowserExtensionImportInput = {
  path: string
  mode?: 'ask' | 'overwrite' | 'new' | 'cancel' | string
  existing?: string
}

export type ProtoBrowserExtensionImportResult = {
  cancelled: boolean
  duplicate: boolean
  message: string
  existing?: ProtoBrowserExtension | null
  extension?: ProtoBrowserExtension | null
}

export type ProtoBrowserExtensionAssignInput = {
  extensionId: string
  profileIds: string[]
  mode: 'shared' | 'exclusive' | string
  enabled: boolean
}

export type ProtoBrowserExtensionAutoBindInput = {
  extensionId: string
  enabled: boolean
  mode: 'shared' | 'exclusive' | string
}

export type ProtoBrowserExtensionUnassignInput = {
  extensionId: string
  profileIds: string[]
}

export async function listBrowserExtensions(): Promise<ProtoBrowserExtension[]> {
  const payload = await browserExtensionProtoClient.request(METHOD_BROWSER_EXTENSION_LIST, new Uint8Array())
  return decodeBrowserExtensionListResponse(payload).extensions
}

export async function getBrowserExtension(extensionId: string): Promise<ProtoBrowserExtension | null> {
  const payload = await browserExtensionProtoClient.request(METHOD_BROWSER_EXTENSION_GET, encodeBrowserExtensionIDRequest({ extensionId }))
  return decodeBrowserExtensionResponse(payload).extension
}

export async function deleteBrowserExtension(extensionId: string): Promise<boolean> {
  const payload = await browserExtensionProtoClient.request(METHOD_BROWSER_EXTENSION_DELETE, encodeBrowserExtensionIDRequest({ extensionId }))
  return decodeBrowserActionResponse(payload).ok
}

export async function chooseBrowserExtensionArchive(): Promise<{ cancelled: boolean; path: string }> {
  const payload = await browserExtensionProtoClient.request(METHOD_BROWSER_EXTENSION_CHOOSE_ARCHIVE, new Uint8Array())
  return decodeBrowserExtensionChoosePathResponse(payload)
}

export async function chooseBrowserExtensionDirectory(): Promise<{ cancelled: boolean; path: string }> {
  const payload = await browserExtensionProtoClient.request(METHOD_BROWSER_EXTENSION_CHOOSE_DIRECTORY, new Uint8Array())
  return decodeBrowserExtensionChoosePathResponse(payload)
}

export async function importBrowserExtensionArchive(input: ProtoBrowserExtensionImportInput): Promise<ProtoBrowserExtensionImportResult> {
  const payload = await browserExtensionProtoClient.request(METHOD_BROWSER_EXTENSION_IMPORT_ARCHIVE, encodeBrowserExtensionImportRequest(input))
  return decodeBrowserExtensionImportResult(payload)
}

export async function importBrowserExtensionDirectory(input: ProtoBrowserExtensionImportInput): Promise<ProtoBrowserExtensionImportResult> {
  const payload = await browserExtensionProtoClient.request(METHOD_BROWSER_EXTENSION_IMPORT_DIRECTORY, encodeBrowserExtensionImportRequest(input))
  return decodeBrowserExtensionImportResult(payload)
}

export async function listBrowserExtensionProfileBindings(extensionId: string): Promise<ProtoBrowserExtensionBinding[]> {
  const payload = await browserExtensionProtoClient.request(METHOD_BROWSER_EXTENSION_LIST_PROFILE_BINDINGS, encodeBrowserExtensionIDRequest({ extensionId }))
  return decodeBrowserExtensionBindingListResponse(payload).bindings
}

export async function listBrowserExtensionBindingsForProfile(profileId: string): Promise<ProtoBrowserExtensionBinding[]> {
  const payload = await browserExtensionProtoClient.request(METHOD_BROWSER_EXTENSION_LIST_FOR_PROFILE, encodeBrowserExtensionProfileRequest({ profileId }))
  return decodeBrowserExtensionBindingListResponse(payload).bindings
}

export async function assignBrowserExtensionProfiles(input: ProtoBrowserExtensionAssignInput): Promise<ProtoBrowserExtensionBinding[]> {
  const payload = await browserExtensionProtoClient.request(METHOD_BROWSER_EXTENSION_ASSIGN_PROFILES, encodeBrowserExtensionAssignRequest(input))
  return decodeBrowserExtensionBindingListResponse(payload).bindings
}

export async function setBrowserExtensionAutoBind(input: ProtoBrowserExtensionAutoBindInput): Promise<ProtoBrowserExtension | null> {
  const payload = await browserExtensionProtoClient.request(METHOD_BROWSER_EXTENSION_SET_AUTO_BIND, encodeBrowserExtensionAutoBindRequest(input))
  return decodeBrowserExtensionResponse(payload).extension
}

export async function unassignBrowserExtensionProfiles(input: ProtoBrowserExtensionUnassignInput): Promise<ProtoBrowserExtensionBinding[]> {
  const payload = await browserExtensionProtoClient.request(METHOD_BROWSER_EXTENSION_UNASSIGN_PROFILES, encodeBrowserExtensionUnassignRequest(input))
  return decodeBrowserExtensionBindingListResponse(payload).bindings
}

export function encodeBrowserExtensionIDRequest(message: { extensionId: string }): Uint8Array {
  return concatBytes([encodeStringField(1, message.extensionId)])
}

export function encodeBrowserExtensionProfileRequest(message: { profileId: string }): Uint8Array {
  return concatBytes([encodeStringField(1, message.profileId)])
}

export function encodeBrowserExtensionImportRequest(message: ProtoBrowserExtensionImportInput): Uint8Array {
  return concatBytes([
    encodeStringField(1, message.path),
    encodeStringField(2, message.mode ?? ''),
    encodeStringField(3, message.existing ?? ''),
  ])
}

export function encodeBrowserExtensionAssignRequest(message: ProtoBrowserExtensionAssignInput): Uint8Array {
  return concatBytes([
    encodeStringField(1, message.extensionId),
    ...message.profileIds.map(profileId => encodeStringField(2, profileId)),
    encodeStringField(3, message.mode),
    encodeInt32Field(4, message.enabled ? 1 : 0),
  ])
}

export function encodeBrowserExtensionAutoBindRequest(message: ProtoBrowserExtensionAutoBindInput): Uint8Array {
  return concatBytes([
    encodeStringField(1, message.extensionId),
    encodeInt32Field(2, message.enabled ? 1 : 0),
    encodeStringField(3, message.mode),
  ])
}

export function encodeBrowserExtensionUnassignRequest(message: ProtoBrowserExtensionUnassignInput): Uint8Array {
  return concatBytes([
    encodeStringField(1, message.extensionId),
    ...message.profileIds.map(profileId => encodeStringField(2, profileId)),
  ])
}

export function decodeBrowserExtensionListResponse(payload: Uint8Array): { extensions: ProtoBrowserExtension[] } {
  const extensions: ProtoBrowserExtension[] = []
  for (const field of readFields(payload)) {
    if (field.fieldNumber === 1 && field.wireType === WireType.LengthDelimited) {
      extensions.push(decodeBrowserExtension(field.value))
    }
  }
  return { extensions }
}

export function decodeBrowserExtensionResponse(payload: Uint8Array): { extension: ProtoBrowserExtension | null } {
  let extension: ProtoBrowserExtension | null = null
  for (const field of readFields(payload)) {
    if (field.fieldNumber === 1 && field.wireType === WireType.LengthDelimited) {
      extension = decodeBrowserExtension(field.value)
    }
  }
  return { extension }
}

export function decodeBrowserExtensionChoosePathResponse(payload: Uint8Array): { cancelled: boolean; path: string } {
  const result = { cancelled: false, path: '' }
  for (const field of readFields(payload)) {
    if (field.wireType === WireType.Varint && field.fieldNumber === 1) {
      result.cancelled = Number(decodeVarintField(field.value)) !== 0
      continue
    }
    if (field.wireType === WireType.LengthDelimited && field.fieldNumber === 2) {
      result.path = decodeString(field.value)
    }
  }
  return result
}

export function decodeBrowserExtensionImportResult(payload: Uint8Array): ProtoBrowserExtensionImportResult {
  const result: ProtoBrowserExtensionImportResult = {
    cancelled: false,
    duplicate: false,
    message: '',
    existing: null,
    extension: null,
  }
  for (const field of readFields(payload)) {
    if (field.wireType === WireType.Varint) {
      const value = Number(decodeVarintField(field.value)) !== 0
      switch (field.fieldNumber) {
        case 1:
          result.cancelled = value
          break
        case 2:
          result.duplicate = value
          break
      }
      continue
    }
    if (field.wireType === WireType.LengthDelimited) {
      switch (field.fieldNumber) {
        case 3:
          result.message = decodeString(field.value)
          break
        case 4:
          result.existing = decodeBrowserExtension(field.value)
          break
        case 5:
          result.extension = decodeBrowserExtension(field.value)
          break
      }
    }
  }
  return result
}

export function decodeBrowserExtensionBindingListResponse(payload: Uint8Array): { bindings: ProtoBrowserExtensionBinding[] } {
  const bindings: ProtoBrowserExtensionBinding[] = []
  for (const field of readFields(payload)) {
    if (field.fieldNumber === 1 && field.wireType === WireType.LengthDelimited) {
      bindings.push(decodeBrowserExtensionBinding(field.value))
    }
  }
  return { bindings }
}

function decodeBrowserExtension(payload: Uint8Array): ProtoBrowserExtension {
  const extension: ProtoBrowserExtension = {
    extensionId: '',
    name: '',
    version: '',
    manifestVersion: 0,
    description: '',
    sourceType: '',
    sourceUrl: '',
    installDir: '',
    packagePath: '',
    manifestJson: '',
    boundCount: 0,
    autoBindEnabled: false,
    autoBindMode: 'shared',
    createdAt: '',
    updatedAt: '',
  }
  for (const field of readFields(payload)) {
    if (field.wireType === WireType.LengthDelimited) {
      const text = decodeString(field.value)
      switch (field.fieldNumber) {
        case 1:
          extension.extensionId = text
          break
        case 2:
          extension.name = text
          break
        case 3:
          extension.version = text
          break
        case 5:
          extension.description = text
          break
        case 6:
          extension.sourceType = text
          break
        case 7:
          extension.sourceUrl = text
          break
        case 8:
          extension.installDir = text
          break
        case 9:
          extension.packagePath = text
          break
        case 10:
          extension.manifestJson = text
          break
        case 13:
          extension.autoBindMode = text
          break
        case 14:
          extension.createdAt = text
          break
        case 15:
          extension.updatedAt = text
          break
      }
      continue
    }
    if (field.wireType === WireType.Varint) {
      const value = Number(decodeVarintField(field.value))
      switch (field.fieldNumber) {
        case 4:
          extension.manifestVersion = value
          break
        case 11:
          extension.boundCount = value
          break
        case 12:
          extension.autoBindEnabled = value !== 0
          break
      }
    }
  }
  return extension
}

function decodeBrowserExtensionBinding(payload: Uint8Array): ProtoBrowserExtensionBinding {
  const binding: ProtoBrowserExtensionBinding = {
    id: 0,
    profileId: '',
    profileName: '',
    extensionId: '',
    extensionName: '',
    extensionVersion: '',
    mode: 'shared',
    enabled: false,
    exclusiveDir: '',
    createdAt: '',
    updatedAt: '',
  }
  for (const field of readFields(payload)) {
    if (field.wireType === WireType.LengthDelimited) {
      const text = decodeString(field.value)
      switch (field.fieldNumber) {
        case 2:
          binding.profileId = text
          break
        case 3:
          binding.profileName = text
          break
        case 4:
          binding.extensionId = text
          break
        case 5:
          binding.extensionName = text
          break
        case 6:
          binding.extensionVersion = text
          break
        case 7:
          binding.mode = text
          break
        case 9:
          binding.exclusiveDir = text
          break
        case 10:
          binding.createdAt = text
          break
        case 11:
          binding.updatedAt = text
          break
      }
      continue
    }
    if (field.wireType === WireType.Varint) {
      const value = Number(decodeVarintField(field.value))
      switch (field.fieldNumber) {
        case 1:
          binding.id = value
          break
        case 8:
          binding.enabled = value !== 0
          break
      }
    }
  }
  return binding
}
