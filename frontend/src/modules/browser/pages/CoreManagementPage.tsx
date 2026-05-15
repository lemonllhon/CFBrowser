import { useEffect, useState, useCallback } from 'react'
import { Download, Edit2, FolderOpen, RefreshCw, Settings } from 'lucide-react'
import { Badge, Button, Card, ConfirmModal, FormItem, Input, Modal, Table, Textarea, toast } from '../../../shared/components'
import type { TableColumn } from '../../../shared/components/Table'
import type { BrowserCore, BrowserCoreInput, BrowserCoreValidateResult, BrowserSettings, BrowserCoreExtended, BrowserProxy } from '../types'
import { fetchBrowserCores, saveBrowserCore, deleteBrowserCore, setDefaultBrowserCore, validateBrowserCorePath, openCorePath, fetchBrowserSettings, saveBrowserSettings, fetchCoreExtendedInfo, scanBrowserCores, BrowserCoreDownload, fetchBrowserProxies } from '../api'
import { EventsOn, EventsOff, BrowserOpenURL } from '../../../wailsjs/runtime/runtime'

interface CoreDisplayInfo {
  coreId: string
  coreName: string
  corePath: string
  isDefault: boolean
  pathValid: boolean
  pathMessage: string
  chromeVersion: string
  instanceCount: number
}

type CoreDownloadSource = 'github' | 'custom'

interface GithubCoreAsset {
  id: number
  releaseName: string
  tagName: string
  assetName: string
  url: string
  size: number
  updatedAt: string
}

const FINGERPRINT_CHROMIUM_RELEASES_API = 'https://api.github.com/repos/adryfish/fingerprint-chromium/releases'
const FINGERPRINT_CHROMIUM_RELEASES_PAGE = 'https://github.com/adryfish/fingerprint-chromium/releases'

const formatAssetSize = (bytes: number) => {
  if (!bytes || bytes <= 0) return '-'
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`
}

const deriveCoreNameFromAsset = (asset: GithubCoreAsset) => {
  const base = asset.assetName
    .replace(/\.(zip|7z|tar\.gz|tgz)$/i, '')
    .replace(/[^\w.-]+/g, '-')
    .replace(/^-+|-+$/g, '')
  return base || asset.tagName || 'chrome-core'
}

export function CoreManagementPage() {
  const [cores, setCores] = useState<BrowserCore[]>([])
  const [displayList, setDisplayList] = useState<CoreDisplayInfo[]>([])
  const [loading, setLoading] = useState(true)
  const [scanning, setScanning] = useState(false)

  // 全局设置状态
  const [settings, setSettings] = useState<BrowserSettings>({
    userDataRoot: '',
    defaultFingerprintArgs: [],
    defaultLaunchArgs: [],
    defaultProxy: '',
    startReadyTimeoutMs: 3000,
    startStableWindowMs: 1200,
  })
  const [settingsModalOpen, setSettingsModalOpen] = useState(false)
  const [settingsForm, setSettingsForm] = useState({
    userDataRoot: '',
    defaultProxy: '',
    defaultFingerprintArgs: '',
    defaultLaunchArgs: '',
    startReadyTimeoutMs: 3000,
    startStableWindowMs: 1200,
  })
  const [savingSettings, setSavingSettings] = useState(false)

  // 编辑弹窗状态
  const [editModalOpen, setEditModalOpen] = useState(false)
  const [editingCore, setEditingCore] = useState<BrowserCore | null>(null)
  const [editForm, setEditForm] = useState({ coreName: '', corePath: '' })
  const [saving, setSaving] = useState(false)
  const [pathValidating, setPathValidating] = useState(false)
  const [pathValidResult, setPathValidResult] = useState<BrowserCoreValidateResult | null>(null)

  // 删除确认状态
  const [deleteConfirmOpen, setDeleteConfirmOpen] = useState(false)
  const [deletingCore, setDeletingCore] = useState<CoreDisplayInfo | null>(null)

  // 内核下载
  const [downloadModalOpen, setDownloadModalOpen] = useState(false)
  const [downloadForm, setDownloadForm] = useState({ name: '', url: '', proxyMode: 'system', proxyId: '', source: 'github' as CoreDownloadSource, selectedAssetUrl: '' })
  const [downloadProgress, setDownloadProgress] = useState<{ phase: string; progress: number; message: string } | null>(null)
  const [proxies, setProxies] = useState<BrowserProxy[]>([])
  const [githubAssets, setGithubAssets] = useState<GithubCoreAsset[]>([])
  const [githubLoading, setGithubLoading] = useState(false)
  const [githubError, setGithubError] = useState('')

  useEffect(() => {
    loadData()

    // 监听下载进度
    const onDownloadProgress = (data: { phase: string; progress: number; message: string }) => {
      setDownloadProgress(data)
      if (data.phase === 'done') {
        toast.success(data.message)
        setTimeout(() => {
          setDownloadModalOpen(false)
          setDownloadProgress(null)
          loadData() // 更新内核列表
        }, 1500)
      } else if (data.phase === 'error') {
        toast.error(data.message)
        setDownloadProgress(null) // 清理进度使其可以重新开始
      }
    }
    EventsOn('download:progress', onDownloadProgress)

    return () => {
      EventsOff('download:progress')
    }
  }, [])

  const loadData = async () => {
    setLoading(true)
    try {
      // 并行加载设置、内核列表和扩展信息
      const [settingsData, coreList, extendedInfo] = await Promise.all([
        fetchBrowserSettings(),
        fetchBrowserCores(),
        fetchCoreExtendedInfo(),
      ])

      setSettings(settingsData)
      setCores(coreList)

      // 创建扩展信息映射
      const extendedMap = new Map<string, BrowserCoreExtended>()
      extendedInfo.forEach(info => extendedMap.set(info.coreId, info))

      // 验证所有路径并合并扩展信息
      const displayInfoList: CoreDisplayInfo[] = await Promise.all(
        coreList.map(async (core) => {
          const result = await validateBrowserCorePath(core.corePath)
          const extended = extendedMap.get(core.coreId)
          return {
            coreId: core.coreId,
            coreName: core.coreName,
            corePath: core.corePath,
            isDefault: core.isDefault,
            pathValid: result.valid,
            pathMessage: result.message,
            chromeVersion: extended?.chromeVersion || '',
            instanceCount: extended?.instanceCount || 0,
          }
        })
      )
      setDisplayList(displayInfoList)
    } finally {
      setLoading(false)
    }
  }

  // 防抖验证路径
  const validatePath = useCallback(async (path: string) => {
    if (!path.trim()) {
      setPathValidResult(null)
      return
    }
    setPathValidating(true)
    try {
      const result = await validateBrowserCorePath(path)
      setPathValidResult(result)
    } finally {
      setPathValidating(false)
    }
  }, [])

  // 路径输入变化时触发验证（防抖）
  useEffect(() => {
    fetchBrowserProxies().then(setProxies)
    const timer = setTimeout(() => {
      if (editModalOpen && editForm.corePath) {
        validatePath(editForm.corePath)
      }
    }, 500)
    return () => clearTimeout(timer)
  }, [editForm.corePath, editModalOpen, validatePath])

  const loadGithubCoreAssets = useCallback(async () => {
    setGithubLoading(true)
    setGithubError('')
    try {
      const response = await fetch(FINGERPRINT_CHROMIUM_RELEASES_API, {
        headers: { Accept: 'application/vnd.github+json' },
      })
      if (!response.ok) {
        throw new Error(`GitHub Releases 获取失败：HTTP ${response.status}`)
      }
      const releases = await response.json()
      const assets: GithubCoreAsset[] = []
      ;(Array.isArray(releases) ? releases : []).forEach((release: any) => {
        const releaseName = String(release?.name || release?.tag_name || '未命名版本')
        const tagName = String(release?.tag_name || '')
        ;(Array.isArray(release?.assets) ? release.assets : []).forEach((asset: any) => {
          const assetName = String(asset?.name || '')
          const url = String(asset?.browser_download_url || '')
          if (!assetName || !url || !/\.(zip|7z|tar\.gz|tgz)$/i.test(assetName)) return
          assets.push({
            id: Number(asset?.id || assets.length + 1),
            releaseName,
            tagName,
            assetName,
            url,
            size: Number(asset?.size || 0),
            updatedAt: String(asset?.updated_at || release?.published_at || ''),
          })
        })
      })
      setGithubAssets(assets)
      if (assets.length === 0) {
        setGithubError('未找到可下载的压缩包资产')
      }
    } catch (error: any) {
      setGithubError(error?.message || '获取 GitHub 版本失败')
      setGithubAssets([])
    } finally {
      setGithubLoading(false)
    }
  }, [])

  useEffect(() => {
    if (downloadModalOpen && downloadForm.source === 'github' && githubAssets.length === 0 && !githubLoading && !githubError) {
      void loadGithubCoreAssets()
    }
  }, [downloadModalOpen, downloadForm.source, githubAssets.length, githubLoading, githubError, loadGithubCoreAssets])

  const handleSelectGithubAsset = (assetUrl: string) => {
    const asset = githubAssets.find(item => item.url === assetUrl)
    setDownloadForm(prev => ({
      ...prev,
      selectedAssetUrl: assetUrl,
      url: asset?.url || '',
      name: asset ? deriveCoreNameFromAsset(asset) : prev.name,
    }))
  }

  // 表格列定义
  const columns: TableColumn<CoreDisplayInfo>[] = [
    { key: 'coreName', title: '内核名称', width: '150px' },
    { key: 'corePath', title: '内核路径', width: '180px' },
    {
      key: 'chromeVersion',
      title: 'Chrome 版本',
      width: '130px',
      render: (val) => val || '-',
    },
    {
      key: 'instanceCount',
      title: '使用实例',
      width: '90px',
      render: (val) => <Badge variant="default">{val}</Badge>,
    },
    {
      key: 'isDefault',
      title: '默认',
      width: '70px',
      render: (val) => val ? <Badge variant="info">默认</Badge> : null,
    },
    {
      key: 'pathValid',
      title: '状态',
      width: '80px',
      render: (val) => (
        <Badge variant={val ? 'success' : 'error'}>
          {val ? '有效' : '无效'}
        </Badge>
      ),
    },
    {
      key: 'actions',
      title: '操作',
      width: '220px',
      render: (_, record) => (
        <div className="flex gap-2">
          <Button size="sm" variant="ghost" onClick={(e) => { e.stopPropagation(); handleOpenPath(record.corePath) }} title="打开目录">
            <FolderOpen className="w-4 h-4" />
          </Button>
          <Button size="sm" variant="ghost" onClick={(e) => { e.stopPropagation(); handleEdit(record) }}>
            编辑
          </Button>
          {!record.isDefault && (
            <Button size="sm" variant="ghost" onClick={(e) => { e.stopPropagation(); handleSetDefault(record.coreId) }}>
              设为默认
            </Button>
          )}
          <Button size="sm" variant="danger" onClick={(e) => { e.stopPropagation(); handleDeleteClick(record) }}>
            删除
          </Button>
        </div>
      ),
    },
  ]

  // 打开内核路径
  const handleOpenPath = async (corePath: string) => {
    try {
      await openCorePath(corePath)
    } catch (error: any) {
      toast.error(error?.message || '打开目录失败')
    }
  }

  // 扫描 chrome 目录，自动注册新内核
  const handleScan = async () => {
    setScanning(true)
    try {
      await scanBrowserCores()
      await loadData()
      toast.success('扫描完成')
    } catch (error: any) {
      toast.error(error?.message || '扫描失败')
    } finally {
      setScanning(false)
    }
  }

  // 新增内核
  const handleAdd = () => {
    setEditingCore(null)
    setEditForm({ coreName: '', corePath: '' })
    setPathValidResult(null)
    setEditModalOpen(true)
  }

  // 编辑内核
  const handleEdit = (record: CoreDisplayInfo) => {
    const core = cores.find(c => c.coreId === record.coreId)
    if (core) {
      setEditingCore(core)
      setEditForm({ coreName: core.coreName, corePath: core.corePath })
      setPathValidResult({ valid: record.pathValid, message: record.pathMessage })
      setEditModalOpen(true)
    }
  }

  // 保存内核
  const handleSaveCore = async () => {
    if (!editForm.coreName.trim()) {
      toast.error('请输入内核名称')
      return
    }
    if (!editForm.corePath.trim()) {
      toast.error('请输入内核路径')
      return
    }
    setSaving(true)
    try {
      const input: BrowserCoreInput = {
        coreId: editingCore?.coreId || `core-${Date.now()}`,
        coreName: editForm.coreName.trim(),
        corePath: editForm.corePath.trim(),
        isDefault: editingCore?.isDefault || false,
      }
      await saveBrowserCore(input)
      await loadData()
      setEditModalOpen(false)
      toast.success(editingCore ? '内核已更新' : '内核已添加')
    } catch (error: any) {
      toast.error(error?.message || '保存失败')
    } finally {
      setSaving(false)
    }
  }

  // 删除点击
  const handleDeleteClick = (record: CoreDisplayInfo) => {
    if (record.isDefault) {
      toast.warning('默认内核不能删除')
      return
    }
    setDeletingCore(record)
    setDeleteConfirmOpen(true)
  }

  // 确认删除
  const handleDeleteConfirm = async () => {
    if (!deletingCore) return
    try {
      await deleteBrowserCore(deletingCore.coreId)
      await loadData()
      toast.success('内核已删除')
    } catch (error: any) {
      toast.error(error?.message || '删除失败')
    }
    setDeletingCore(null)
  }

  // 设为默认
  const handleSetDefault = async (coreId: string) => {
    try {
      await setDefaultBrowserCore(coreId)
      await loadData()
      toast.success('已设为默认内核')
    } catch (error: any) {
      toast.error(error?.message || '设置失败')
    }
  }

  // 开始下载
  const handleStartDownloadCore = async () => {
    if (!downloadForm.name.trim() || !downloadForm.url.trim()) {
      toast.error('请输入名称和下载地址')
      return
    }
    if (cores.some(c => c.coreName.toLowerCase() === downloadForm.name.trim().toLowerCase())) {
      toast.error('该内核名称已存在')
      return
    }
    setDownloadProgress({ phase: 'starting', progress: 0, message: '准备下载...' })
    try {
      // 在这儿我们需要从 proxies 中寻找匹配到的代理设定，如果有则传过去的 url
      let targetProxy = ''
      if (downloadForm.proxyMode === 'system') {
        targetProxy = '__system__'
      } else if (downloadForm.proxyMode === 'direct') {
        targetProxy = '__direct__'
      } else {
        const proxyProfile = proxies.find(p => p.proxyId === downloadForm.proxyId)
        targetProxy = downloadForm.proxyId
        if (proxyProfile && proxyProfile.proxyConfig) {
          targetProxy = proxyProfile.proxyConfig
        }
      }

      await BrowserCoreDownload(downloadForm.name.trim(), downloadForm.url.trim(), targetProxy)
    } catch (err: any) {
      toast.error(err.message || '内部启动下载失败')
      setDownloadProgress(null)
    }
  }

  // 打开设置编辑弹窗
  const handleEditSettings = () => {
    setSettingsForm({
      userDataRoot: settings.userDataRoot,
      defaultProxy: settings.defaultProxy,
      defaultFingerprintArgs: settings.defaultFingerprintArgs.join('\n'),
      defaultLaunchArgs: settings.defaultLaunchArgs.join('\n'),
      startReadyTimeoutMs: settings.startReadyTimeoutMs,
      startStableWindowMs: settings.startStableWindowMs,
    })
    setSettingsModalOpen(true)
  }

  // 保存设置
  const handleSaveSettings = async () => {
    setSavingSettings(true)
    try {
      const newSettings: BrowserSettings = {
        userDataRoot: settingsForm.userDataRoot.trim(),
        defaultProxy: settingsForm.defaultProxy.trim(),
        defaultFingerprintArgs: settingsForm.defaultFingerprintArgs.split('\n').map(s => s.trim()).filter(Boolean),
        defaultLaunchArgs: settingsForm.defaultLaunchArgs.split('\n').map(s => s.trim()).filter(Boolean),
        startReadyTimeoutMs: Math.max(1000, Number(settingsForm.startReadyTimeoutMs) || 3000),
        startStableWindowMs: Math.max(0, Number(settingsForm.startStableWindowMs) || 1200),
      }
      await saveBrowserSettings(newSettings)
      setSettings(newSettings)
      setSettingsModalOpen(false)
      toast.success('设置已保存')
    } catch (error: any) {
      toast.error(error?.message || '保存失败')
    } finally {
      setSavingSettings(false)
    }
  }


  return (
    <div className="space-y-5 animate-fade-in">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold text-[var(--color-text-primary)]">内核管理</h1>
          <p className="text-sm text-[var(--color-text-muted)] mt-1">管理 Chrome 内核版本和全局设置</p>
        </div>
        <div className="flex gap-2">
          <Button size="sm" variant="secondary" onClick={() => setDownloadModalOpen(true)}>下载内核</Button>
          <Button size="sm" variant="secondary" onClick={handleScan} loading={scanning}>扫描内核</Button>
          <Button size="sm" onClick={handleAdd}>新增内核</Button>
        </div>
      </div>

      {/* 全局设置卡片 */}
      <Card>
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center gap-2">
            <Settings className="w-5 h-5 text-[var(--color-text-muted)]" />
            <h3 className="text-base font-medium text-[var(--color-text-primary)]">全局设置</h3>
          </div>
          <Button size="sm" variant="ghost" onClick={handleEditSettings}>
            <Edit2 className="w-4 h-4 mr-1" />
            编辑
          </Button>
        </div>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <p className="text-xs text-[var(--color-text-muted)] mb-1">用户数据根目录</p>
            <p className="text-sm text-[var(--color-text-primary)]">{settings.userDataRoot || '-'}</p>
          </div>
          <div>
            <p className="text-xs text-[var(--color-text-muted)] mb-1">默认代理配置</p>
            <p className="text-sm text-[var(--color-text-primary)]">{settings.defaultProxy || '-'}</p>
          </div>
          <div>
            <p className="text-xs text-[var(--color-text-muted)] mb-1">默认指纹参数</p>
            {settings.defaultFingerprintArgs.length > 0 ? (
              <pre className="text-xs text-[var(--color-text-secondary)] bg-[var(--color-bg-subtle)] p-2 rounded max-h-20 overflow-auto">
                {settings.defaultFingerprintArgs.join('\n')}
              </pre>
            ) : (
              <p className="text-sm text-[var(--color-text-primary)]">-</p>
            )}
          </div>
          <div>
            <p className="text-xs text-[var(--color-text-muted)] mb-1">默认启动参数</p>
            {settings.defaultLaunchArgs.length > 0 ? (
              <pre className="text-xs text-[var(--color-text-secondary)] bg-[var(--color-bg-subtle)] p-2 rounded max-h-20 overflow-auto">
                {settings.defaultLaunchArgs.join('\n')}
              </pre>
            ) : (
              <p className="text-sm text-[var(--color-text-primary)]">-</p>
            )}
          </div>
          <div>
            <p className="text-xs text-[var(--color-text-muted)] mb-1">启动就绪超时</p>
            <p className="text-sm text-[var(--color-text-primary)]">{settings.startReadyTimeoutMs} ms</p>
          </div>
          <div>
            <p className="text-xs text-[var(--color-text-muted)] mb-1">启动稳定窗口</p>
            <p className="text-sm text-[var(--color-text-primary)]">{settings.startStableWindowMs} ms</p>
          </div>
        </div>
      </Card>

      {/* 内核列表卡片 */}
      <Card title="内核列表" subtitle="已配置的 Chrome 内核">
        <Table
          columns={columns}
          data={displayList}
          rowKey="coreId"
          loading={loading}
          emptyText="暂无内核，请添加内核"
        />
      </Card>

      {/* 全局设置编辑弹窗 */}
      <Modal
        open={settingsModalOpen}
        onClose={() => setSettingsModalOpen(false)}
        title="编辑全局设置"
        width="550px"
        footer={
          <>
            <Button variant="secondary" onClick={() => setSettingsModalOpen(false)}>取消</Button>
            <Button onClick={handleSaveSettings} loading={savingSettings}>保存</Button>
          </>
        }
      >
        <div className="space-y-4">
          <FormItem label="用户数据根目录">
            <Input
              value={settingsForm.userDataRoot}
              onChange={e => setSettingsForm(prev => ({ ...prev, userDataRoot: e.target.value }))}
              placeholder="例如：data"
            />
          </FormItem>
          <FormItem label="默认代理配置">
            <Input
              value={settingsForm.defaultProxy}
              onChange={e => setSettingsForm(prev => ({ ...prev, defaultProxy: e.target.value }))}
              placeholder="例如：http://127.0.0.1:7890"
            />
          </FormItem>
          <FormItem label="默认指纹参数">
            <Textarea
              value={settingsForm.defaultFingerprintArgs}
              onChange={e => setSettingsForm(prev => ({ ...prev, defaultFingerprintArgs: e.target.value }))}
              rows={4}
              placeholder="每行一个参数，如 --fingerprint-brand=Chrome"
            />
          </FormItem>
          <FormItem label="默认启动参数">
            <Textarea
              value={settingsForm.defaultLaunchArgs}
              onChange={e => setSettingsForm(prev => ({ ...prev, defaultLaunchArgs: e.target.value }))}
              rows={4}
              placeholder="每行一个参数，如 --disable-sync"
            />
          </FormItem>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <FormItem label="启动就绪超时（毫秒）" hint="默认 3000，慢机器可调到 5000-10000">
              <Input
                type="number"
                min={1000}
                step={500}
                value={settingsForm.startReadyTimeoutMs}
                onChange={e => setSettingsForm(prev => ({ ...prev, startReadyTimeoutMs: Math.max(1000, Number(e.target.value) || 3000) }))}
                placeholder="3000"
              />
            </FormItem>
            <FormItem label="启动稳定窗口（毫秒）" hint="建议 1200-3000">
              <Input
                type="number"
                min={0}
                step={100}
                value={settingsForm.startStableWindowMs}
                onChange={e => setSettingsForm(prev => ({ ...prev, startStableWindowMs: Math.max(0, Number(e.target.value) || 1200) }))}
                placeholder="1200"
              />
            </FormItem>
          </div>
        </div>
      </Modal>

      {/* 新增/编辑内核弹窗 */}
      <Modal
        open={editModalOpen}
        onClose={() => setEditModalOpen(false)}
        title={editingCore ? '编辑内核' : '新增内核'}
        width="500px"
        footer={
          <>
            <Button variant="secondary" onClick={() => setEditModalOpen(false)}>取消</Button>
            <Button onClick={handleSaveCore} loading={saving}>保存</Button>
          </>
        }
      >
        <div className="space-y-4">
          <FormItem label="内核名称" required>
            <Input
              value={editForm.coreName}
              onChange={e => setEditForm(prev => ({ ...prev, coreName: e.target.value }))}
              placeholder="例如：Chrome 142"
            />
          </FormItem>
          <FormItem label="内核路径" required>
            <Input
              value={editForm.corePath}
              onChange={e => setEditForm(prev => ({ ...prev, corePath: e.target.value }))}
              placeholder="相对路径（如 chrome）或绝对路径"
            />
            {pathValidating && (
              <p className="text-xs text-[var(--color-text-muted)] mt-1">验证中...</p>
            )}
            {!pathValidating && pathValidResult && (
              <p className={`text-xs mt-1 ${pathValidResult.valid ? 'text-green-600' : 'text-red-500'}`}>
                {pathValidResult.message}
              </p>
            )}
          </FormItem>
        </div>
      </Modal>

      {/* 删除确认弹窗 */}
      <ConfirmModal
        open={deleteConfirmOpen}
        onClose={() => setDeleteConfirmOpen(false)}
        onConfirm={handleDeleteConfirm}
        title="确认删除"
        content={`确定要删除内核"${deletingCore?.coreName}"吗？此操作不可恢复。`}
        confirmText="删除"
        danger
      />

      {/* 内核下载弹窗 */}
      <Modal open={downloadModalOpen} onClose={() => {
        if (downloadProgress && downloadProgress.phase !== 'done' && downloadProgress.phase !== 'error') {
          toast.warning('正在下载中，请稍候...')
          return
        }
        setDownloadModalOpen(false)
        setDownloadProgress(null)
      }} title="下载内核" width="720px"
        footer={
          <>
            <Button variant="secondary" onClick={() => {
              if (downloadProgress && downloadProgress.phase !== 'done' && downloadProgress.phase !== 'error') return;
              setDownloadModalOpen(false)
            }} disabled={downloadProgress !== null && downloadProgress.phase !== 'error'}>取消</Button>
            <Button onClick={handleStartDownloadCore} loading={downloadProgress !== null && downloadProgress.phase !== 'error'}>开始下载</Button>
          </>
        }>
        <div className="space-y-4">
          <div className="grid grid-cols-2 gap-2">
            <Button
              variant={downloadForm.source === 'github' ? undefined : 'secondary'}
              onClick={() => setDownloadForm(prev => ({ ...prev, source: 'github' }))}
              disabled={downloadProgress !== null}
            >
              <Download className="w-4 h-4" />
              GitHub 版本
            </Button>
            <Button
              variant={downloadForm.source === 'custom' ? undefined : 'secondary'}
              onClick={() => setDownloadForm(prev => ({ ...prev, source: 'custom' }))}
              disabled={downloadProgress !== null}
            >
              自定义地址
            </Button>
          </div>

          {downloadForm.source === 'github' && (
            <Card padding="sm">
              <div className="flex items-center justify-between gap-3 mb-3">
                <div>
                  <p className="text-sm font-medium text-[var(--color-text-primary)]">fingerprint-chromium Releases</p>
                  <p className="text-xs text-[var(--color-text-muted)] mt-0.5">从 GitHub Releases 读取可下载压缩包，选择后会自动回填名称和地址</p>
                </div>
                <div className="flex gap-2 shrink-0">
                  <Button size="sm" variant="ghost" onClick={loadGithubCoreAssets} loading={githubLoading} disabled={downloadProgress !== null}>
                    {!githubLoading && <RefreshCw className="w-4 h-4" />}
                    刷新
                  </Button>
                  <Button size="sm" variant="ghost" onClick={() => BrowserOpenURL(FINGERPRINT_CHROMIUM_RELEASES_PAGE)}>打开页面</Button>
                </div>
              </div>

              {githubError ? (
                <div className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-600">
                  {githubError}
                </div>
              ) : githubLoading ? (
                <div className="rounded-md border border-[var(--color-border-default)] px-3 py-6 text-center text-sm text-[var(--color-text-muted)]">正在获取版本...</div>
              ) : (
                <div className="space-y-2 max-h-64 overflow-auto pr-1">
                  {githubAssets.map(asset => (
                    <label
                      key={`${asset.id}-${asset.url}`}
                      className={`flex items-start gap-3 rounded-lg border p-3 cursor-pointer transition-colors ${
                        downloadForm.selectedAssetUrl === asset.url
                          ? 'border-[var(--color-accent)] bg-[var(--color-accent)]/5'
                          : 'border-[var(--color-border-default)] hover:border-[var(--color-border-strong)]'
                      }`}
                    >
                      <input
                        type="radio"
                        name="github-core-asset"
                        className="mt-1 accent-[var(--color-accent)]"
                        checked={downloadForm.selectedAssetUrl === asset.url}
                        onChange={() => handleSelectGithubAsset(asset.url)}
                        disabled={downloadProgress !== null}
                      />
                      <div className="min-w-0 flex-1">
                        <div className="flex items-center justify-between gap-3">
                          <span className="text-sm font-medium text-[var(--color-text-primary)] truncate">{asset.assetName}</span>
                          <span className="text-xs text-[var(--color-text-muted)] shrink-0">{formatAssetSize(asset.size)}</span>
                        </div>
                        <div className="mt-1 text-xs text-[var(--color-text-muted)] truncate">
                          {asset.releaseName}{asset.tagName ? ` / ${asset.tagName}` : ''}
                        </div>
                      </div>
                    </label>
                  ))}
                  {githubAssets.length === 0 && (
                    <div className="rounded-md border border-[var(--color-border-default)] px-3 py-6 text-center text-sm text-[var(--color-text-muted)]">暂无可选版本</div>
                  )}
                </div>
              )}
            </Card>
          )}

          <FormItem label="内核名称" required>
            <Input
              value={downloadForm.name}
              onChange={e => setDownloadForm(prev => ({ ...prev, name: e.target.value }))}
              placeholder="例如: chrome-139"
              disabled={downloadProgress !== null}
            />
            <p className="text-xs text-[var(--color-text-muted)] mt-1">该名称将同时作为数据存放的子文件夹名。</p>
          </FormItem>
          <FormItem label={downloadForm.source === 'github' ? '下载地址（已从版本选择回填，也可手动微调）' : '下载地址（ZIP）'} required>
            <Input
              value={downloadForm.url}
              onChange={e => setDownloadForm(prev => ({ ...prev, url: e.target.value }))}
              placeholder="https://github.com/.../release.zip"
              disabled={downloadProgress !== null}
            />
            {downloadForm.source === 'custom' && (
              <p className="text-xs text-[var(--color-text-muted)] mt-1">可填写你自己构建、内部镜像或其他来源的 ZIP 内核压缩包下载地址。</p>
            )}
          </FormItem>

          <FormItem label="下载代理设置">
            <select
              value={downloadForm.proxyMode}
              onChange={e => {
                const mode = e.target.value
                setDownloadForm(prev => ({
                  ...prev,
                  proxyMode: mode,
                  proxyId: mode === 'custom' && proxies.length > 0 ? proxies[0].proxyId : ''
                }))
              }}
              className="w-full h-9 px-3 rounded-md border border-[var(--color-border-default)] bg-[var(--color-bg-primary)] text-[var(--color-text-primary)] text-sm focus:outline-none focus:ring-1 focus:ring-[var(--color-accent)] focus:border-[var(--color-accent)]"
              disabled={downloadProgress !== null}
            >
              <option value="system">跟随系统全局代理</option>
              <option value="direct">直连模式 (不使用代理)</option>
              {proxies.length > 0 && <option value="custom">指定应用代理配置...</option>}
            </select>
          </FormItem>

          {downloadForm.proxyMode === 'custom' && (
            <FormItem label="选择代理池节点" required>
              <select
                value={downloadForm.proxyId}
                onChange={e => setDownloadForm(prev => ({ ...prev, proxyId: e.target.value }))}
                className="w-full h-9 px-3 rounded-md border border-[var(--color-border-default)] bg-[var(--color-bg-primary)] text-[var(--color-text-primary)] text-sm focus:outline-none focus:ring-1 focus:ring-[var(--color-accent)] focus:border-[var(--color-accent)]"
                disabled={downloadProgress !== null}
              >
                {proxies.map(p => (
                  <option key={p.proxyId} value={p.proxyId}>
                    {p.proxyName} ({p.proxyConfig})
                  </option>
                ))}
              </select>
            </FormItem>
          )}

          {downloadProgress && (
            <div className="mt-4 p-4 border border-[var(--color-border-default)] rounded-lg bg-[var(--color-bg-secondary)]">
              <div className="flex justify-between text-sm mb-2">
                <span className="font-medium text-[var(--color-text-primary)]">{downloadProgress.message}</span>
                <span className="text-[var(--color-text-muted)]">{downloadProgress.progress}%</span>
              </div>
              <div className="w-full bg-[var(--color-bg-surface)] rounded-full h-2 overflow-hidden border border-[var(--color-border-muted)]">
                <div
                  className="bg-[var(--color-accent)] h-2 rounded-full transition-all duration-300"
                  style={{ width: `${Math.max(0, Math.min(100, downloadProgress.progress))}%` }}
                ></div>
              </div>
            </div>
          )}
        </div>
      </Modal>

    </div>
  )
}
