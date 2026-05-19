import { useEffect, useMemo, useState } from 'react'
import { Eye, GripHorizontal, Keyboard, Link, List, Pause, Play, Power, RefreshCw, Settings, SquareStack, X } from 'lucide-react'
import { Button, ToastContainer, toast } from '../../../shared/components'
import { ThemeProvider } from '../../../shared/theme'
import type { WindowSyncActionResult, WindowSyncBatchInputDifferentItem, WindowSyncBatchInputResult, WindowSyncSettings, WindowSyncState } from '../types'
import { WindowSetAlwaysOnTop, WindowSetPosition, WindowSetSize } from '../../../wailsjs/runtime/runtime'

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
  width: 360,
  height: 76,
  x: 360,
  y: 18,
}

const expandedToolbarWidth = 780

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

function orderedWindows(state: WindowSyncState | null) {
  const windows = [...(state?.windows || [])]
  return windows.sort((a, b) => {
    if (a.master && !b.master) return -1
    if (!a.master && b.master) return 1
    return 0
  })
}

function resultMessage(result: WindowSyncBatchInputResult | null) {
  if (!result) return '批量输入已执行'
  return `批量输入完成：成功 ${result.success}/${result.total}${result.failed > 0 ? `，失败 ${result.failed}` : ''}`
}

function actionResultMessage(action: string, result: WindowSyncActionResult | null) {
  if (!result) return `${action}已执行`
  return `${action}完成：成功 ${result.success}/${result.total}${result.failed > 0 ? `，失败 ${result.failed}` : ''}`
}

function stateToSettings(state: WindowSyncState | null): WindowSyncSettings {
  return {
    masterColor: state?.masterColor || '#2563eb',
    syncKeyboard: state?.syncKeyboard ?? true,
    syncMouse: state?.syncMouse ?? true,
  }
}

function normalizeColor(value: string) {
  const trimmed = value.trim()
  if (!trimmed) return '#2563eb'
  return trimmed.startsWith('#') ? trimmed : `#${trimmed}`
}

export function WindowSyncFloatingToolbar() {
  const [config] = useState<ToolbarConfig>(() => readToolbarConfig())
  const [state, setState] = useState<WindowSyncState | null>(null)
  const [loadingCommand, setLoadingCommand] = useState<string>('')
  const [expanded, setExpanded] = useState(false)
  const [batchOpen, setBatchOpen] = useState(false)
  const [batchMode, setBatchMode] = useState<'same' | 'different'>('same')
  const [sameText, setSameText] = useState('')
  const [differentTexts, setDifferentTexts] = useState<Record<string, string>>({})
  const [batchResult, setBatchResult] = useState<WindowSyncBatchInputResult | null>(null)
  const [tabPanelOpen, setTabPanelOpen] = useState(false)
  const [openUrlsText, setOpenUrlsText] = useState('')
  const [tabResult, setTabResult] = useState<WindowSyncActionResult | null>(null)
  const [listPanelOpen, setListPanelOpen] = useState(false)
  const [settingsPanelOpen, setSettingsPanelOpen] = useState(false)
  const [settingsDraft, setSettingsDraft] = useState<WindowSyncSettings>(() => stateToSettings(null))
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

  const runBatchInput = async () => {
    const windows = orderedWindows(state)
    if (windows.length === 0) {
      toast.error('当前没有可批量输入的同步窗口')
      return
    }
    setLoadingCommand(`batch-input:${batchMode}`)
    try {
      const payload =
        batchMode === 'same'
          ? { command: 'batch-input-same', text: sameText }
          : {
              command: 'batch-input-different',
              items: windows.map<WindowSyncBatchInputDifferentItem>(window => ({
                profileId: window.profileId,
                text: differentTexts[window.profileId] || '',
              })),
            }
      const result = await request<WindowSyncBatchInputResult>('/command', {
        method: 'POST',
        body: JSON.stringify(payload),
      })
      setBatchResult(result)
      if ((result?.failed || 0) > 0) {
        toast.warning(resultMessage(result))
      } else {
        toast.success(resultMessage(result))
      }
      await loadState()
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '批量输入失败')
    } finally {
      setLoadingCommand('')
    }
  }

  const applyLayout = async (mode: 'grid' | 'stack') => {
    setLoadingCommand(`layout:${mode}`)
    try {
      const next = await request<WindowSyncState>('/command', {
        method: 'POST',
        body: JSON.stringify({ command: 'layout', mode }),
      })
      setState(next?.active ? next : null)
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '布局切换失败')
    } finally {
      setLoadingCommand('')
    }
  }

  const runTabCommand = async (command: string, label: string, urls?: string[]) => {
    setLoadingCommand(command)
    try {
      const result = await request<WindowSyncActionResult>('/command', {
        method: 'POST',
        body: JSON.stringify({ command, urls }),
      })
      setTabResult(result)
      if ((result?.failed || 0) > 0) {
        toast.warning(actionResultMessage(label, result))
      } else {
        toast.success(actionResultMessage(label, result))
      }
      await loadState()
    } catch (error) {
      toast.error(error instanceof Error ? error.message : `${label}失败`)
    } finally {
      setLoadingCommand('')
    }
  }

  const runOpenUrls = async () => {
    const urls = openUrlsText
      .split(/\r?\n/)
      .map(item => item.trim())
      .filter(Boolean)
    if (urls.length === 0) {
      toast.error('请输入需要打开的网址')
      return
    }
    await runTabCommand('open-urls', '打开网站', urls)
  }

  const openSinglePanel = (panel: 'batch' | 'tab' | 'list' | 'settings') => {
    setBatchOpen(panel === 'batch' ? open => !open : false)
    setTabPanelOpen(panel === 'tab' ? open => !open : false)
    setListPanelOpen(panel === 'list' ? open => !open : false)
    setSettingsPanelOpen(panel === 'settings' ? open => !open : false)
    if (panel === 'settings') {
      setSettingsDraft(stateToSettings(state))
    }
    setBatchResult(null)
    setTabResult(null)
  }

  const closePanelsForCollapse = () => {
    setBatchOpen(false)
    setTabPanelOpen(false)
    setListPanelOpen(false)
    setSettingsPanelOpen(false)
    setBatchResult(null)
    setTabResult(null)
  }

  const saveSettings = async () => {
    setLoadingCommand('save-settings')
    try {
      const next = await request<WindowSyncState>('/command', {
        method: 'POST',
        body: JSON.stringify({
          command: 'save-settings',
          settings: {
            ...settingsDraft,
            masterColor: normalizeColor(settingsDraft.masterColor),
          },
        }),
      })
      setState(next?.active ? next : null)
      setSettingsDraft(stateToSettings(next))
      toast.success('同步设置已保存')
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '保存同步设置失败')
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

  useEffect(() => {
    try {
      WindowSetSize(
        panelOpen || expanded ? expandedToolbarWidth : config.width || fallbackConfig.width,
        batchOpen || tabPanelOpen || listPanelOpen || settingsPanelOpen ? 430 : config.height || fallbackConfig.height,
      )
    } catch {
      // Older runtimes may ignore dynamic toolbar resizing; commands still work.
    }
  }, [batchOpen, config.height, config.width, expanded, listPanelOpen, settingsPanelOpen, tabPanelOpen])

  useEffect(() => {
    const windows = orderedWindows(state)
    setDifferentTexts(prev => {
      const next: Record<string, string> = {}
      for (const window of windows) {
        next[window.profileId] = prev[window.profileId] || ''
      }
      return next
    })
  }, [state?.sessionId, state?.windows?.length])

  useEffect(() => {
    if (settingsPanelOpen) {
      setSettingsDraft(stateToSettings(state))
    }
  }, [settingsPanelOpen, state?.masterColor, state?.syncKeyboard, state?.syncMouse])

  const paused = !!state?.paused
  const count = profileCount(state)
  const windows = orderedWindows(state)
  const batchLoading = loadingCommand === 'batch-input:same' || loadingCommand === 'batch-input:different'
  const panelOpen = batchOpen || tabPanelOpen || listPanelOpen || settingsPanelOpen
  const masterColor = normalizeColor(state?.masterColor || settingsDraft.masterColor || '#2563eb')

  return (
    <ThemeProvider>
      <main
        className={`window-sync-floating-toolbar ${panelOpen ? 'is-panel-open' : ''}`}
        onMouseEnter={() => setExpanded(true)}
        onMouseLeave={() => {
          setExpanded(false)
          closePanelsForCollapse()
        }}
      >
        <div className="window-sync-toolbar-row">
          <div className="window-sync-toolbar-drag" title="拖动工具栏">
            <GripHorizontal className="h-4 w-4" />
          </div>
          <div className="flex min-w-0 flex-1 items-center gap-3">
            <div
              className="h-2.5 w-2.5 rounded-full"
              style={{ backgroundColor: paused ? 'var(--color-warning)' : masterColor }}
            />
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
          <div className={`window-sync-toolbar-expanded ${expanded || panelOpen ? 'is-expanded' : ''}`}>
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
            <Button
              size="sm"
              variant={batchOpen ? 'secondary' : 'ghost'}
              title="批量输入"
              onClick={() => openSinglePanel('batch')}
              className="window-sync-toolbar-command"
            >
              <Keyboard className="h-4 w-4" />
              输入
            </Button>
            <Button
              size="sm"
              variant={tabPanelOpen ? 'secondary' : 'ghost'}
              title="标签控制"
              onClick={() => openSinglePanel('tab')}
              className="window-sync-toolbar-command"
            >
              <Link className="h-4 w-4" />
              标签
            </Button>
            <Button
              size="sm"
              variant={listPanelOpen ? 'secondary' : 'ghost'}
              title="同步窗口列表"
              onClick={() => openSinglePanel('list')}
              className="window-sync-toolbar-command"
            >
              <List className="h-4 w-4" />
              列表
            </Button>
            <Button
              size="sm"
              variant={settingsPanelOpen ? 'secondary' : 'ghost'}
              title="同步设置"
              onClick={() => openSinglePanel('settings')}
              className="window-sync-toolbar-command"
            >
              <Settings className="h-4 w-4" />
              设置
            </Button>
            <Button
              size="sm"
              variant="ghost"
              title="宫格布局"
              onClick={() => void applyLayout('grid')}
              loading={loadingCommand === 'layout:grid'}
              className="window-sync-toolbar-command"
            >
              <SquareStack className="h-4 w-4" />
              宫格
            </Button>
            <Button
              size="sm"
              variant="ghost"
              title="堆叠布局"
              onClick={() => void applyLayout('stack')}
              loading={loadingCommand === 'layout:stack'}
              className="window-sync-toolbar-command"
            >
              <SquareStack className="h-4 w-4" />
              堆叠
            </Button>
            <Button size="sm" variant="ghost" title="刷新状态" onClick={() => void loadState()} className="h-9 w-9 px-0">
              <RefreshCw className="h-4 w-4" />
            </Button>
          </div>
        </div>
        {batchOpen && (
          <section className="window-sync-batch-panel">
            <div className="flex items-center justify-between gap-3">
              <div>
                <div className="text-sm font-semibold text-[var(--color-text-primary)]">批量输入</div>
                <div className="text-xs text-[var(--color-text-muted)]">当前同步窗口 {windows.length} 个，差异文本必须一窗一项。</div>
              </div>
              <Button size="sm" variant="ghost" title="关闭批量输入" onClick={() => setBatchOpen(false)} className="h-8 w-8 px-0">
                <X className="h-4 w-4" />
              </Button>
            </div>
            <div className="window-sync-batch-tabs">
              <button type="button" className={batchMode === 'same' ? 'is-active' : ''} onClick={() => setBatchMode('same')}>相同文本</button>
              <button type="button" className={batchMode === 'different' ? 'is-active' : ''} onClick={() => setBatchMode('different')}>差异文本</button>
            </div>
            {batchMode === 'same' ? (
              <textarea
                className="window-sync-batch-textarea"
                value={sameText}
                onChange={event => setSameText(event.target.value)}
                placeholder="输入要填充到所有同步窗口当前焦点输入框的文本"
              />
            ) : (
              <div className="window-sync-batch-window-list">
                {windows.map(window => (
                  <label key={window.profileId} className="window-sync-batch-window-item">
                    <span className="window-sync-batch-window-label">
                      <span className={window.master ? 'text-[var(--color-accent)]' : 'text-[var(--color-text-muted)]'}>
                        {window.master ? '主控' : '被控'}
                      </span>
                      <span className="truncate">{window.profileName || window.profileId}</span>
                    </span>
                    <input
                      value={differentTexts[window.profileId] || ''}
                      onChange={event => setDifferentTexts(prev => ({ ...prev, [window.profileId]: event.target.value }))}
                      placeholder="该窗口输入内容"
                    />
                  </label>
                ))}
              </div>
            )}
            {batchResult && (
              <div className="window-sync-batch-result">
                {batchResult.results.map(item => (
                  <div key={item.profileId} className={item.success ? 'text-[var(--color-success)]' : 'text-[var(--color-danger)]'}>
                    {(item.master ? '主控' : '被控')} · {item.profileName || item.profileId}：{item.success ? '成功' : item.error || '失败'}
                  </div>
                ))}
              </div>
            )}
            <div className="flex items-center justify-end gap-2">
              <Button size="sm" variant="secondary" onClick={() => setBatchResult(null)}>清除结果</Button>
              <Button size="sm" onClick={() => void runBatchInput()} loading={batchLoading}>输入</Button>
            </div>
          </section>
        )}
        {tabPanelOpen && (
          <section className="window-sync-batch-panel">
            <div className="flex items-center justify-between gap-3">
              <div>
                <div className="text-sm font-semibold text-[var(--color-text-primary)]">标签控制</div>
                <div className="text-xs text-[var(--color-text-muted)]">当前同步窗口 {windows.length} 个，按主控当前标签执行。</div>
              </div>
              <Button size="sm" variant="ghost" title="关闭标签控制" onClick={() => setTabPanelOpen(false)} className="h-8 w-8 px-0">
                <X className="h-4 w-4" />
              </Button>
            </div>
            <div className="window-sync-tab-actions">
              <Button
                size="sm"
                variant="secondary"
                onClick={() => void runTabCommand('close-other-tabs', '关闭其他标签页')}
                loading={loadingCommand === 'close-other-tabs'}
              >
                关闭其他标签页
              </Button>
              <Button
                size="sm"
                variant="secondary"
                onClick={() => void runTabCommand('close-current-tab', '关闭当前标签页')}
                loading={loadingCommand === 'close-current-tab'}
              >
                关闭当前标签页
              </Button>
              <Button
                size="sm"
                variant="secondary"
                onClick={() => void runTabCommand('close-blank-tabs', '关闭空白标签页')}
                loading={loadingCommand === 'close-blank-tabs'}
              >
                关闭空白标签页
              </Button>
            </div>
            <textarea
              className="window-sync-batch-textarea window-sync-open-urls-textarea"
              value={openUrlsText}
              onChange={event => setOpenUrlsText(event.target.value)}
              placeholder="输入要在所有同步窗口打开的网址，多个网址换行区分"
            />
            {tabResult && (
              <div className="window-sync-batch-result">
                {tabResult.results.map(item => (
                  <div key={item.profileId} className={item.success ? 'text-[var(--color-success)]' : 'text-[var(--color-danger)]'}>
                    {(item.master ? '主控' : '被控')} · {item.profileName || item.profileId}：{item.success ? '成功' : item.error || '失败'}
                  </div>
                ))}
              </div>
            )}
            <div className="flex items-center justify-end gap-2">
              <Button size="sm" variant="secondary" onClick={() => setTabResult(null)}>清除结果</Button>
              <Button size="sm" onClick={() => void runOpenUrls()} loading={loadingCommand === 'open-urls'}>打开网站</Button>
            </div>
          </section>
        )}
        {listPanelOpen && (
          <section className="window-sync-batch-panel">
            <div className="flex items-center justify-between gap-3">
              <div>
                <div className="text-sm font-semibold text-[var(--color-text-primary)]">同步窗口列表</div>
                <div className="text-xs text-[var(--color-text-muted)]">当前同步窗口 {windows.length} 个，主控窗口固定。</div>
              </div>
              <Button size="sm" variant="ghost" title="关闭窗口列表" onClick={() => setListPanelOpen(false)} className="h-8 w-8 px-0">
                <X className="h-4 w-4" />
              </Button>
            </div>
            <div className="window-sync-window-list">
              {windows.map(window => (
                <div key={window.profileId} className="window-sync-window-list-item">
                  <div className="min-w-0">
                    <div className="flex min-w-0 items-center gap-2">
                      <span
                        className={window.master ? 'window-sync-role-badge is-master' : 'window-sync-role-badge'}
                        style={window.master ? { color: masterColor, backgroundColor: `${masterColor}1f` } : undefined}
                      >
                        {window.master ? '主控' : '被控'}
                      </span>
                      <span className="truncate text-sm font-medium text-[var(--color-text-primary)]">{window.profileName || window.profileId}</span>
                    </div>
                    <div className="mt-1 truncate text-xs text-[var(--color-text-muted)]">
                      PID {window.pid || '-'} · 调试端口 {window.debugPort || '-'}
                    </div>
                  </div>
                  <div className="shrink-0 text-right">
                    <div className={window.running ? 'text-xs text-[var(--color-success)]' : 'text-xs text-[var(--color-danger)]'}>
                      {window.running ? '运行中' : '未运行'}
                    </div>
                    <div className={window.debugReady ? 'mt-1 text-xs text-[var(--color-success)]' : 'mt-1 text-xs text-[var(--color-text-muted)]'}>
                      {window.debugReady ? '调试就绪' : '调试未就绪'}
                    </div>
                  </div>
                </div>
              ))}
            </div>
            <div className="flex items-center justify-end gap-2">
              <Button size="sm" variant="secondary" onClick={() => void loadState()}>刷新</Button>
              <Button size="sm" variant="secondary" onClick={() => void runCommand('show-all')} loading={loadingCommand === 'show-all'}>展示窗口</Button>
            </div>
          </section>
        )}
        {settingsPanelOpen && (
          <section className="window-sync-batch-panel">
            <div className="flex items-center justify-between gap-3">
              <div>
                <div className="text-sm font-semibold text-[var(--color-text-primary)]">同步设置</div>
                <div className="text-xs text-[var(--color-text-muted)]">保存后立即应用到当前同步会话。</div>
              </div>
              <Button size="sm" variant="ghost" title="关闭同步设置" onClick={() => setSettingsPanelOpen(false)} className="h-8 w-8 px-0">
                <X className="h-4 w-4" />
              </Button>
            </div>
            <div className="window-sync-settings-panel">
              <label className="window-sync-setting-row">
                <span>
                  <span className="block text-sm font-medium text-[var(--color-text-primary)]">主控窗口颜色</span>
                  <span className="block text-xs text-[var(--color-text-muted)]">用于主控窗口标识。</span>
                </span>
                <span className="flex items-center gap-2">
                  <input
                    type="color"
                    value={normalizeColor(settingsDraft.masterColor)}
                    onChange={event => setSettingsDraft(prev => ({ ...prev, masterColor: event.target.value }))}
                    className="window-sync-color-input"
                  />
                  <input
                    value={settingsDraft.masterColor}
                    onChange={event => setSettingsDraft(prev => ({ ...prev, masterColor: event.target.value }))}
                    className="window-sync-color-text"
                    placeholder="#2563eb"
                  />
                </span>
              </label>
              <label className="window-sync-setting-row">
                <span>
                  <span className="block text-sm font-medium text-[var(--color-text-primary)]">同步键盘输入</span>
                  <span className="block text-xs text-[var(--color-text-muted)]">开启后主控按键会同步到被控窗口。</span>
                </span>
                <input
                  type="checkbox"
                  checked={settingsDraft.syncKeyboard}
                  onChange={event => setSettingsDraft(prev => ({ ...prev, syncKeyboard: event.target.checked }))}
                  className="window-sync-toggle"
                />
              </label>
              <label className="window-sync-setting-row">
                <span>
                  <span className="block text-sm font-medium text-[var(--color-text-primary)]">同步鼠标输入</span>
                  <span className="block text-xs text-[var(--color-text-muted)]">开启后点击、滚动和拖动会同步到被控窗口。</span>
                </span>
                <input
                  type="checkbox"
                  checked={settingsDraft.syncMouse}
                  onChange={event => setSettingsDraft(prev => ({ ...prev, syncMouse: event.target.checked }))}
                  className="window-sync-toggle"
                />
              </label>
            </div>
            <div className="flex items-center justify-end gap-2">
              <Button size="sm" variant="secondary" onClick={() => setSettingsDraft(stateToSettings(state))}>重置</Button>
              <Button size="sm" onClick={() => void saveSettings()} loading={loadingCommand === 'save-settings'}>保存</Button>
            </div>
          </section>
        )}
      </main>
      <ToastContainer />
    </ThemeProvider>
  )
}
