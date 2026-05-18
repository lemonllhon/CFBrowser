import { useEffect, useMemo, useState } from 'react'
import { Eye, GripHorizontal, Pause, Play, Power, RefreshCw, SquareStack } from 'lucide-react'
import { Button, ToastContainer, toast } from '../../../shared/components'
import { ThemeProvider } from '../../../shared/theme'
import type { WindowSyncState } from '../types'
import { WindowSetAlwaysOnTop, WindowSetPosition } from '../../../wailsjs/runtime/runtime'

type ToolbarConfig = {
  port: number
  token: string
  width: number
  height: number
  x: number
  y: number
}

const fallbackConfig: ToolbarConfig = {
  port: 0,
  token: '',
  width: 520,
  height: 76,
  x: 360,
  y: 18,
}

function readToolbarConfig(): ToolbarConfig {
  const params = new URLSearchParams(window.location.search)
  const encoded = params.get('config')
  if (encoded) {
    try {
      return { ...fallbackConfig, ...JSON.parse(decodeURIComponent(encoded)) }
    } catch {
      // Ignore malformed startup config and fall back to environment injection.
    }
  }
  const raw = (window as any).__TRACE_WINDOW_SYNC_TOOLBAR_CONFIG__
  if (raw && typeof raw === 'object') {
    return { ...fallbackConfig, ...raw }
  }
  return fallbackConfig
}

async function readResponseText(response: Response) {
  try {
    return await response.text()
  } catch {
    return ''
  }
}

function profileCount(state: WindowSyncState | null) {
  return state?.windows?.length || state?.profileIds?.length || 0
}

export function WindowSyncFloatingToolbar() {
  const [config] = useState<ToolbarConfig>(() => readToolbarConfig())
  const [state, setState] = useState<WindowSyncState | null>(null)
  const [loadingCommand, setLoadingCommand] = useState<string>('')
  const [expanded, setExpanded] = useState(false)
  const endpoint = useMemo(() => `http://127.0.0.1:${config.port}`, [config.port])

  const request = async <T,>(path: string, init?: RequestInit): Promise<T | null> => {
    if (!config.port || !config.token) return null
    const response = await fetch(`${endpoint}${path}`, {
      ...init,
      headers: {
        'Content-Type': 'application/json',
        'X-Trace-Toolbar-Token': config.token,
        ...(init?.headers || {}),
      },
    })
    if (!response.ok) {
      const message = await readResponseText(response)
      throw new Error(message || `HTTP ${response.status}`)
    }
    return (await response.json()) as T
  }

  const loadState = async () => {
    try {
      const next = await request<WindowSyncState>('/state')
      setState(next?.active ? next : null)
    } catch {
      setState(null)
    }
  }

  const runCommand = async (command: string) => {
    setLoadingCommand(command)
    try {
      const next = await request<WindowSyncState>('/command', {
        method: 'POST',
        body: JSON.stringify({ command }),
      })
      setState(next?.active ? next : null)
      if (command === 'stop') {
        window.setTimeout(() => window.close(), 120)
      }
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '工具栏操作失败')
    } finally {
      setLoadingCommand('')
    }
  }

  useEffect(() => {
    document.body.classList.add('window-sync-toolbar-body')
    const keepTopmost = () => {
      try {
        WindowSetAlwaysOnTop(false)
        WindowSetAlwaysOnTop(true)
      } catch {
        // The toolbar can still work if the OS ignores a transient topmost refresh.
      }
    }
    try {
      WindowSetAlwaysOnTop(true)
      WindowSetPosition(Math.max(0, config.x || 0), Math.max(0, config.y || 0))
    } catch {
      // Positioning is best-effort because the backend already starts the window near the target location.
    }
    void loadState()
    const timer = window.setInterval(() => {
      void loadState()
      keepTopmost()
    }, 900)
    const topmostTimer = window.setInterval(keepTopmost, 1800)
    window.addEventListener('focus', keepTopmost)
    window.addEventListener('pointerdown', keepTopmost)
    return () => {
      document.body.classList.remove('window-sync-toolbar-body')
      window.clearInterval(timer)
      window.clearInterval(topmostTimer)
      window.removeEventListener('focus', keepTopmost)
      window.removeEventListener('pointerdown', keepTopmost)
    }
  }, [])

  const paused = !!state?.paused
  const count = profileCount(state)

  return (
    <ThemeProvider>
      <main
        className="window-sync-floating-toolbar"
        onMouseEnter={() => setExpanded(true)}
        onMouseLeave={() => setExpanded(false)}
      >
        <div className="window-sync-toolbar-drag" title="拖动工具栏">
          <GripHorizontal className="h-4 w-4" />
        </div>
        <div className="flex min-w-0 flex-1 items-center gap-3">
          <div className={`h-2.5 w-2.5 rounded-full ${paused ? 'bg-[var(--color-warning)]' : 'bg-[var(--color-success)]'}`} />
          <div className="min-w-0">
            <div className="truncate text-sm font-semibold text-[var(--color-text-primary)]">
              {paused ? '窗口同步已暂停' : state?.active ? '窗口同步中' : '等待同步状态'}
            </div>
            <div className="truncate text-[11px] text-[var(--color-text-muted)]">
              {count > 0 ? `${count} 个窗口 · 主控固定` : '连接主程序中'}
            </div>
          </div>
        </div>
        <div className="flex shrink-0 items-center gap-2">
          <Button
            size="sm"
            variant="secondary"
            title="展示窗口"
            onClick={() => void runCommand('show-all')}
            loading={loadingCommand === 'show-all'}
            className="h-9 w-9 px-0"
          >
            <Eye className="h-4 w-4" />
          </Button>
          <Button
            size="sm"
            variant="danger"
            title="停止同步"
            onClick={() => void runCommand('stop')}
            loading={loadingCommand === 'stop'}
            className="h-9 w-9 px-0"
          >
            <Power className="h-4 w-4" />
          </Button>
        </div>
        <div className={`window-sync-toolbar-expanded ${expanded ? 'is-expanded' : ''}`}>
          <Button
            size="sm"
            variant="secondary"
            title={paused ? '恢复同步' : '暂停同步'}
            onClick={() => void runCommand(paused ? 'resume' : 'pause')}
            loading={loadingCommand === 'pause' || loadingCommand === 'resume'}
            className="h-9 px-3"
          >
            {paused ? <Play className="h-4 w-4" /> : <Pause className="h-4 w-4" />}
            {paused ? '恢复' : '暂停'}
          </Button>
          <Button size="sm" variant="ghost" title="布局入口将在下一步接入" disabled className="h-9 px-3">
            <SquareStack className="h-4 w-4" />
            布局
          </Button>
          <Button size="sm" variant="ghost" title="刷新状态" onClick={() => void loadState()} className="h-9 px-3">
            <RefreshCw className="h-4 w-4" />
            刷新
          </Button>
        </div>
      </main>
      <ToastContainer />
    </ThemeProvider>
  )
}
