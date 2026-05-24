import {
  forceQuitApp,
  getRuntimeEnvironment as requestRuntimeEnvironment,
  getWindowSize,
  getWindowState,
  hideWindow,
  minimiseWindow,
  onAppFileDrop,
  onAppRuntimeEvent,
  openAppReleasePage,
} from '../ipc/app'

export type RuntimeUnsubscribe = () => void

export type RuntimeEnvironmentInfo = {
  buildType?: string
  platform?: string
  arch?: string
}

export type RuntimeWindowSize = {
  w: number
  h: number
}

type FileDropCallback = (x: number, y: number, paths: string[]) => void

let fileDropUnsubscribe: RuntimeUnsubscribe | null = null

function noop() {}

export function onRuntimeEvent<T = unknown>(eventName: string, callback: (payload: T, ...args: unknown[]) => void): RuntimeUnsubscribe {
  return onAppRuntimeEvent(eventName, payload => callback(payload as T))
}

export function onRuntimeEventOnce<T = unknown>(eventName: string, callback: (payload: T, ...args: unknown[]) => void): RuntimeUnsubscribe {
  let off: RuntimeUnsubscribe = noop
  off = onRuntimeEvent<T>(eventName, (payload, ...args) => {
    off()
    callback(payload, ...args)
  })
  return off
}

export function openExternalURL(url: string) {
  if (!url) {
    return
  }
  void openAppReleasePage(url).catch(() => {
    window.open(url, '_blank', 'noopener,noreferrer')
  })
}

export function hideRuntimeWindow() {
  void hideWindow()
}

export function minimiseRuntimeWindow() {
  void minimiseWindow()
}

export async function getRuntimeWindowSize(): Promise<RuntimeWindowSize> {
  try {
    const size = await getWindowSize()
    return { w: size.width, h: size.height }
  } catch {
    return { w: window.innerWidth, h: window.innerHeight }
  }
}

export async function isRuntimeWindowNormal(): Promise<boolean> {
  try {
    return (await getWindowState()).normal
  } catch {
    return true
  }
}

export async function isRuntimeWindowMaximised(): Promise<boolean> {
  try {
    return (await getWindowState()).maximised
  } catch {
    return false
  }
}

export async function isRuntimeWindowMinimised(): Promise<boolean> {
  try {
    return (await getWindowState()).minimised
  } catch {
    return false
  }
}

export async function getRuntimeEnvironment(): Promise<RuntimeEnvironmentInfo> {
  try {
    return await requestRuntimeEnvironment()
  } catch {
    return { buildType: 'browser', platform: 'browser', arch: '' }
  }
}

export function quitRuntime() {
  void forceQuitApp().catch(() => {
    window.close()
  })
}

export function onRuntimeFileDrop(callback: FileDropCallback, _useDropTarget: boolean) {
  fileDropUnsubscribe?.()
  fileDropUnsubscribe = onAppFileDrop(payload => {
    callback(payload.x, payload.y, payload.paths)
  })
}

export function clearRuntimeFileDrop() {
  fileDropUnsubscribe?.()
  fileDropUnsubscribe = null
}
