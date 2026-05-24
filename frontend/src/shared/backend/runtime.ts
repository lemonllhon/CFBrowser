import {
  forceQuitApp,
  getRuntimeEnvironment,
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

export function EventsOn<T = unknown>(eventName: string, callback: (payload: T, ...args: unknown[]) => void): RuntimeUnsubscribe {
  return onAppRuntimeEvent(eventName, payload => callback(payload as T))
}

export function EventsOnce<T = unknown>(eventName: string, callback: (payload: T, ...args: unknown[]) => void): RuntimeUnsubscribe {
  let off: RuntimeUnsubscribe = noop
  off = EventsOn<T>(eventName, (payload, ...args) => {
    off()
    callback(payload, ...args)
  })
  return off
}

export function BrowserOpenURL(url: string) {
  if (!url) {
    return
  }
  void openAppReleasePage(url).catch(() => {
    window.open(url, '_blank', 'noopener,noreferrer')
  })
}

export function WindowHide() {
  void hideWindow()
}

export function WindowMinimise() {
  void minimiseWindow()
}

export async function WindowGetSize(): Promise<RuntimeWindowSize> {
  try {
    const size = await getWindowSize()
    return { w: size.width, h: size.height }
  } catch {
    return { w: window.innerWidth, h: window.innerHeight }
  }
}

export async function WindowIsNormal(): Promise<boolean> {
  try {
    return (await getWindowState()).normal
  } catch {
    return true
  }
}

export async function WindowIsMaximised(): Promise<boolean> {
  try {
    return (await getWindowState()).maximised
  } catch {
    return false
  }
}

export async function WindowIsMinimised(): Promise<boolean> {
  try {
    return (await getWindowState()).minimised
  } catch {
    return false
  }
}

export async function Environment(): Promise<RuntimeEnvironmentInfo> {
  try {
    return await getRuntimeEnvironment()
  } catch {
    return { buildType: 'browser', platform: 'browser', arch: '' }
  }
}

export function Quit() {
  void forceQuitApp().catch(() => {
    window.close()
  })
}

export function OnFileDrop(callback: FileDropCallback, _useDropTarget: boolean) {
  fileDropUnsubscribe?.()
  fileDropUnsubscribe = onAppFileDrop(payload => {
    callback(payload.x, payload.y, payload.paths)
  })
}

export function OnFileDropOff() {
  fileDropUnsubscribe?.()
  fileDropUnsubscribe = null
}
