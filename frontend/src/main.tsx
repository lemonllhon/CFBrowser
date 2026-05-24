import ReactDOM from 'react-dom/client'
import App from './App'
import { WindowSyncFloatingToolbar } from './modules/browser/components/WindowSyncFloatingToolbar'
import './index.css'

type WailsRuntimeWindow = Window & {
  _wails?: {
    invoke?: (message: string) => void
  }
  chrome?: {
    webview?: {
      postMessage?: (message: string) => void
    }
  }
  webkit?: {
    messageHandlers?: {
      external?: {
        postMessage?: (message: string) => void
      }
    }
  }
  wails?: {
    invoke?: (message: string) => void
  }
}

async function loadWails3Runtime() {
  try {
    const importRuntime = new Function('specifier', 'return import(specifier)') as (specifier: string) => Promise<unknown>
    await importRuntime('/wails/runtime.js')
  } catch (error) {
    notifyWailsRuntimeReadyFallback()
    if (window.location.hostname !== '127.0.0.1' && window.location.hostname !== 'localhost') {
      console.warn('Wails3 runtime load failed:', error)
    }
  }
}

function notifyWailsRuntimeReadyFallback() {
  const appWindow = window as WailsRuntimeWindow
  const invoke = appWindow._wails?.invoke
    ?? appWindow.chrome?.webview?.postMessage?.bind(appWindow.chrome.webview)
    ?? appWindow.webkit?.messageHandlers?.external?.postMessage?.bind(appWindow.webkit.messageHandlers.external)
    ?? appWindow.wails?.invoke

  try {
    invoke?.('wails:runtime:ready')
  } catch (error) {
    if (window.location.hostname !== '127.0.0.1' && window.location.hostname !== 'localhost') {
      console.warn('Wails3 runtime ready fallback failed:', error)
    }
  }
}

async function bootstrap() {
  await loadWails3Runtime()

  ;(window as Window & { __ANT_APP_BOOTED__?: boolean }).__ANT_APP_BOOTED__ = true

  const searchParams = new URLSearchParams(window.location.search)
  const isWindowSyncToolbar = searchParams.get('toolbar') === '1'

  ReactDOM.createRoot(document.getElementById('root')!).render(
    isWindowSyncToolbar ? <WindowSyncFloatingToolbar /> : <App />,
  )
}

void bootstrap()

