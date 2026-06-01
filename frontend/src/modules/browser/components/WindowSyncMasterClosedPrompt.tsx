import { useEffect, useMemo, useState } from 'react'
import { AlertTriangle, GripVertical, Monitor, Power, ShieldCheck, X } from 'lucide-react'
import { Button } from '../../../shared/components'
import { stopBrowserInstance } from '../../../shared/backend/client'

type WailsRuntimeWindow = Window & {
  wails?: {
    Window?: {
      Close?: () => Promise<void>
    }
  }
}

type RemainingProfile = {
  id: string
  name: string
}

function uniqueValues(values: string[]) {
  const seen = new Set<string>()
  const result: string[] = []
  for (const raw of values) {
    const value = String(raw || '').trim()
    if (!value || seen.has(value)) continue
    seen.add(value)
    result.push(value)
  }
  return result
}

function readPromptPayload() {
  const params = new URLSearchParams(window.location.search)
  const profileId = String(params.get('profileId') || '').trim()
  const profileName = String(params.get('profileName') || profileId || '主控窗口').trim()
  const remainingIds = uniqueValues(params.getAll('remainingProfileIds'))
  const remainingNames = params.getAll('remainingProfileNames').map(name => String(name || '').trim())
  const remainingProfiles = remainingIds.map<RemainingProfile>((id, index) => ({
    id,
    name: remainingNames[index] || id,
  }))
  return {
    profileId,
    profileName: profileName || '主控窗口',
    remainingProfiles,
  }
}

async function closePromptWindow() {
  const runtimeWindow = window as WailsRuntimeWindow
  try {
    await runtimeWindow.wails?.Window?.Close?.()
    return
  } catch {
    // Fallback for browser/dev preview.
  }
  window.close()
}

export function WindowSyncMasterClosedPrompt() {
  const payload = useMemo(readPromptPayload, [])
  const [closing, setClosing] = useState(false)
  const [message, setMessage] = useState('')

  useEffect(() => {
    document.body.classList.add('window-sync-prompt-body')
    return () => {
      document.body.classList.remove('window-sync-prompt-body')
    }
  }, [])

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key !== 'Escape' || closing) return
      event.preventDefault()
      void closePromptWindow()
    }
    window.addEventListener('keydown', onKeyDown)
    return () => {
      window.removeEventListener('keydown', onKeyDown)
    }
  }, [closing])

  const closeRemainingProfiles = async () => {
    if (closing || payload.remainingProfiles.length === 0) return
    setClosing(true)
    setMessage(`正在关闭剩余 ${payload.remainingProfiles.length} 个同步实例...`)
    const results = await Promise.allSettled(
      payload.remainingProfiles.map(profile => stopBrowserInstance(profile.id)),
    )
    const failed = results.filter(result => result.status === 'rejected').length
    if (failed > 0) {
      setClosing(false)
      setMessage(`已关闭部分实例，${failed} 个实例关闭失败，可稍后在实例列表中处理。`)
      return
    }
    setMessage('剩余同步实例已关闭。')
    window.setTimeout(() => {
      void closePromptWindow()
    }, 360)
  }

  return (
    <div className="window-sync-prompt-window" data-testid="window-sync-master-closed-prompt">
      <div className="window-sync-prompt-titlebar">
        <div className="window-sync-prompt-title">
          <GripVertical className="h-4 w-4" />
          <span>窗口同步</span>
        </div>
        <button className="window-sync-prompt-close" title="关闭" onClick={() => void closePromptWindow()} disabled={closing}>
          <X className="h-4 w-4" />
        </button>
      </div>

      <main className="window-sync-prompt-content">
        <div className="window-sync-prompt-icon">
          <AlertTriangle className="h-6 w-6" />
        </div>
        <div className="min-w-0 flex-1">
          <h1 className="window-sync-prompt-heading">窗口同步已停止</h1>
          <p className="window-sync-prompt-description">
            主控实例「{payload.profileName}」已关闭，窗口同步已立即停止。
          </p>
        </div>
      </main>

      <section className="window-sync-prompt-summary">
        <div className="window-sync-prompt-summary-row">
          <span className="window-sync-prompt-summary-label">
            <ShieldCheck className="h-4 w-4" />
            当前状态
          </span>
          <span className="window-sync-prompt-status">同步已关闭</span>
        </div>
        <div className="window-sync-prompt-profile-list">
          {payload.remainingProfiles.map((profile, index) => (
            <div className="window-sync-prompt-profile" key={profile.id}>
              <span className="window-sync-prompt-profile-index">{index + 1}</span>
              <Monitor className="h-4 w-4" />
              <span className="truncate">{profile.name}</span>
            </div>
          ))}
          {payload.remainingProfiles.length === 0 && (
            <div className="window-sync-prompt-empty">没有剩余同步实例需要处理。</div>
          )}
        </div>
      </section>

      {message && (
        <div className="window-sync-prompt-message" role="status">
          {message}
        </div>
      )}

      <footer className="window-sync-prompt-actions">
        <Button variant="secondary" onClick={() => void closePromptWindow()} disabled={closing}>
          保留实例
        </Button>
        <Button variant="danger" onClick={closeRemainingProfiles} loading={closing} disabled={payload.remainingProfiles.length === 0}>
          <Power className="h-4 w-4" />
          关闭剩余实例
        </Button>
      </footer>
    </div>
  )
}
