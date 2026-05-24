import {
  METHOD_BROWSER_PROFILE_USER_DATA_DIR_OPEN,
  METHOD_BROWSER_USER_DATA_DIR_OPEN,
} from './envelope'
import { decodeBrowserActionResponse } from './browser'
import { concatBytes, encodeStringField } from './protobuf'
import { ProtoIpcClient } from './transport'

const browserPathProtoClient = new ProtoIpcClient()

export async function openBrowserUserDataDir(userDataDir: string): Promise<boolean> {
  const payload = await browserPathProtoClient.request(METHOD_BROWSER_USER_DATA_DIR_OPEN, encodeBrowserUserDataDirOpenRequest({ userDataDir }))
  return decodeBrowserActionResponse(payload).ok
}

export async function openBrowserProfileUserDataDir(profileId: string): Promise<boolean> {
  const payload = await browserPathProtoClient.request(METHOD_BROWSER_PROFILE_USER_DATA_DIR_OPEN, encodeBrowserProfileUserDataDirOpenRequest({ profileId }))
  return decodeBrowserActionResponse(payload).ok
}

export function encodeBrowserUserDataDirOpenRequest(message: { userDataDir: string }): Uint8Array {
  return concatBytes([encodeStringField(1, message.userDataDir)])
}

export function encodeBrowserProfileUserDataDirOpenRequest(message: { profileId: string }): Uint8Array {
  return concatBytes([encodeStringField(1, message.profileId)])
}
