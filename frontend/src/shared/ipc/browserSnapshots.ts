import {
  METHOD_BROWSER_SNAPSHOT_CREATE,
  METHOD_BROWSER_SNAPSHOT_DELETE,
  METHOD_BROWSER_SNAPSHOT_LIST,
  METHOD_BROWSER_SNAPSHOT_RESTORE,
} from './envelope'
import {
  WireType,
  concatBytes,
  decodeString,
  decodeVarintField,
  encodeInt64Field,
  encodeStringField,
  readFields,
} from './protobuf'
import { ProtoIpcClient } from './transport'
import { decodeBrowserActionResponse } from './browser'

const browserSnapshotProtoClient = new ProtoIpcClient()

export type ProtoBrowserSnapshotInfo = {
  snapshotId: string
  profileId: string
  name: string
  sizeMB: number
  createdAt: string
}

export async function listBrowserSnapshots(profileId: string): Promise<ProtoBrowserSnapshotInfo[]> {
  const payload = await browserSnapshotProtoClient.request(METHOD_BROWSER_SNAPSHOT_LIST, encodeBrowserSnapshotProfileRequest({ profileId }))
  return decodeBrowserSnapshotListResponse(payload).snapshots
}

export async function createBrowserSnapshot(profileId: string, name: string): Promise<ProtoBrowserSnapshotInfo | null> {
  const payload = await browserSnapshotProtoClient.request(METHOD_BROWSER_SNAPSHOT_CREATE, encodeBrowserSnapshotCreateRequest({ profileId, name }))
  return decodeBrowserSnapshotResponse(payload).snapshot
}

export async function restoreBrowserSnapshot(profileId: string, snapshotId: string): Promise<boolean> {
  const payload = await browserSnapshotProtoClient.request(METHOD_BROWSER_SNAPSHOT_RESTORE, encodeBrowserSnapshotActionRequest({ profileId, snapshotId }))
  return decodeBrowserActionResponse(payload).ok
}

export async function deleteBrowserSnapshot(profileId: string, snapshotId: string): Promise<boolean> {
  const payload = await browserSnapshotProtoClient.request(METHOD_BROWSER_SNAPSHOT_DELETE, encodeBrowserSnapshotActionRequest({ profileId, snapshotId }))
  return decodeBrowserActionResponse(payload).ok
}

export function encodeBrowserSnapshotProfileRequest(message: { profileId: string }): Uint8Array {
  return concatBytes([encodeStringField(1, message.profileId)])
}

export function encodeBrowserSnapshotCreateRequest(message: { profileId: string; name: string }): Uint8Array {
  return concatBytes([
    encodeStringField(1, message.profileId),
    encodeStringField(2, message.name),
  ])
}

export function encodeBrowserSnapshotActionRequest(message: { profileId: string; snapshotId: string }): Uint8Array {
  return concatBytes([
    encodeStringField(1, message.profileId),
    encodeStringField(2, message.snapshotId),
  ])
}

export function decodeBrowserSnapshotListResponse(payload: Uint8Array): { snapshots: ProtoBrowserSnapshotInfo[] } {
  const snapshots: ProtoBrowserSnapshotInfo[] = []
  for (const field of readFields(payload)) {
    if (field.fieldNumber === 1 && field.wireType === WireType.LengthDelimited) {
      snapshots.push(decodeBrowserSnapshotInfo(field.value))
    }
  }
  return { snapshots }
}

export function decodeBrowserSnapshotResponse(payload: Uint8Array): { snapshot: ProtoBrowserSnapshotInfo | null } {
  let snapshot: ProtoBrowserSnapshotInfo | null = null
  for (const field of readFields(payload)) {
    if (field.fieldNumber === 1 && field.wireType === WireType.LengthDelimited) {
      snapshot = decodeBrowserSnapshotInfo(field.value)
    }
  }
  return { snapshot }
}

export function encodeBrowserSnapshotInfo(snapshot: ProtoBrowserSnapshotInfo): Uint8Array {
  return concatBytes([
    encodeStringField(1, snapshot.snapshotId),
    encodeStringField(2, snapshot.profileId),
    encodeStringField(3, snapshot.name),
    encodeInt64Field(4, Math.round(snapshot.sizeMB * 1000)),
    encodeStringField(5, snapshot.createdAt),
  ])
}

function decodeBrowserSnapshotInfo(payload: Uint8Array): ProtoBrowserSnapshotInfo {
  const snapshot: ProtoBrowserSnapshotInfo = {
    snapshotId: '',
    profileId: '',
    name: '',
    sizeMB: 0,
    createdAt: '',
  }

  for (const field of readFields(payload)) {
    if (field.wireType === WireType.LengthDelimited) {
      const text = decodeString(field.value)
      switch (field.fieldNumber) {
        case 1:
          snapshot.snapshotId = text
          break
        case 2:
          snapshot.profileId = text
          break
        case 3:
          snapshot.name = text
          break
        case 5:
          snapshot.createdAt = text
          break
      }
      continue
    }

    if (field.wireType === WireType.Varint && field.fieldNumber === 4) {
      snapshot.sizeMB = Number(decodeVarintField(field.value)) / 1000
    }
  }

  return snapshot
}
