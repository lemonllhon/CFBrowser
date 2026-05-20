import { useEffect, useMemo, useState } from 'react'
import { Archive, FileCode2, FolderOpen, Link2, RefreshCw, Trash2, UploadCloud, Unlink } from 'lucide-react'
import { Badge, Button, Card, ConfirmModal, FormItem, Input, Modal, Select, Table, toast } from '../../../shared/components'
import type { TableColumn } from '../../../shared/components/Table'
import type { BrowserExtension, BrowserExtensionBinding, BrowserExtensionImportResult, BrowserProfile } from '../types'
import { assignBrowserExtensionProfiles, chooseBrowserExtensionArchive, chooseBrowserExtensionDirectory, deleteBrowserExtension, fetchBrowserExtension, fetchBrowserExtensionProfileBindings, fetchBrowserExtensions, fetchBrowserProfiles, importBrowserExtensionArchive, importBrowserExtensionDirectory, unassignBrowserExtensionProfiles } from '../api'
import { OnFileDrop, OnFileDropOff } from '../../../wailsjs/runtime/runtime'

const sourceTypeText: Record<string, string> = {
  zip: '压缩包',
  crx: 'CRX',
  url: '地址导入',
  directory: '目录',
}

const bindingModeText: Record<string, string> = {
  shared: '共享',
  exclusive: '独享',
}

function formatTime(value: string) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString()
}

function sourceLabel(value: string) {
  return sourceTypeText[value] || value || '-'
}

function bindingModeLabel(value: string) {
  return bindingModeText[value] || value || '-'
}

function importTypeFromPath(path: string): 'archive' | 'directory' {
  return /\.(zip|crx)$/i.test(path.trim()) ? 'archive' : 'directory'
}

function errorMessage(error: any, fallback: string) {
  if (!error) return fallback
  if (typeof error === 'string') return error || fallback
  if (typeof error?.message === 'string' && error.message.trim()) return error.message
  if (typeof error?.error === 'string' && error.error.trim()) return error.error
  try {
    return JSON.stringify(error)
  } catch {
    return fallback
  }
}

export function ExtensionManagementPage() {
  const [extensions, setExtensions] = useState<BrowserExtension[]>([])
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)
  const [detailOpen, setDetailOpen] = useState(false)
  const [detailLoading, setDetailLoading] = useState(false)
  const [selectedExtension, setSelectedExtension] = useState<BrowserExtension | null>(null)
  const [deleteConfirmOpen, setDeleteConfirmOpen] = useState(false)
  const [deletingExtension, setDeletingExtension] = useState<BrowserExtension | null>(null)
  const [importModalOpen, setImportModalOpen] = useState(false)
  const [importType, setImportType] = useState<'archive' | 'directory'>('archive')
  const [importPath, setImportPath] = useState('')
  const [importing, setImporting] = useState(false)
  const [duplicateResult, setDuplicateResult] = useState<BrowserExtensionImportResult | null>(null)
  const [dragActive, setDragActive] = useState(false)
  const [profiles, setProfiles] = useState<BrowserProfile[]>([])
  const [bindings, setBindings] = useState<BrowserExtensionBinding[]>([])
  const [bindingsLoading, setBindingsLoading] = useState(false)
  const [selectedProfileIds, setSelectedProfileIds] = useState<Set<string>>(new Set())
  const [bindingMode, setBindingMode] = useState<'shared' | 'exclusive'>('shared')
  const [bindingEnabled, setBindingEnabled] = useState(true)
  const [bindingSaving, setBindingSaving] = useState(false)

  const loadData = async (silent = false) => {
    if (silent) {
      setRefreshing(true)
    } else {
      setLoading(true)
    }
    try {
      const list = await fetchBrowserExtensions()
      setExtensions(list)
    } catch (error: any) {
      toast.error(errorMessage(error, '加载扩展列表失败'), 6000)
    } finally {
      setLoading(false)
      setRefreshing(false)
    }
  }

  useEffect(() => {
    void loadData()
    void loadProfiles()
  }, [])

  useEffect(() => {
    OnFileDrop((_, __, paths) => {
      setDragActive(false)
      void handleDroppedPaths(paths || [])
    }, false)
    return () => {
      OnFileDropOff()
    }
  }, [])

  const handleOpenDetail = async (record: BrowserExtension) => {
    setSelectedExtension(record)
    setDetailOpen(true)
    setDetailLoading(true)
    setBindings([])
    setSelectedProfileIds(new Set())
    try {
      const [detail] = await Promise.all([
        fetchBrowserExtension(record.extensionId),
        loadBindings(record.extensionId),
      ])
      if (detail) {
        setSelectedExtension(detail)
      }
    } catch (error: any) {
      toast.error(errorMessage(error, '加载扩展详情失败'), 6000)
    } finally {
      setDetailLoading(false)
    }
  }

  const loadProfiles = async () => {
    try {
      const list = await fetchBrowserProfiles()
      setProfiles(list)
    } catch (error: any) {
      toast.error(errorMessage(error, '加载实例列表失败'), 6000)
    }
  }

  const loadBindings = async (extensionId: string) => {
    const id = extensionId.trim()
    if (!id) return
    setBindingsLoading(true)
    try {
      const list = await fetchBrowserExtensionProfileBindings(id)
      setBindings(list)
    } catch (error: any) {
      toast.error(errorMessage(error, '加载扩展绑定失败'), 6000)
    } finally {
      setBindingsLoading(false)
    }
  }

  const handleDeleteClick = (record: BrowserExtension) => {
    if (record.boundCount > 0) {
      toast.warning(`该扩展已绑定 ${record.boundCount} 个实例，请先解绑后再删除`)
      return
    }
    setDeletingExtension(record)
    setDeleteConfirmOpen(true)
  }

  const handleDeleteConfirm = async () => {
    if (!deletingExtension) return
    try {
      await deleteBrowserExtension(deletingExtension.extensionId)
      await loadData(true)
      if (selectedExtension?.extensionId === deletingExtension.extensionId) {
        setDetailOpen(false)
        setSelectedExtension(null)
      }
      toast.success('扩展插件已删除')
    } catch (error: any) {
      toast.error(errorMessage(error, '删除扩展插件失败'), 6000)
    } finally {
      setDeletingExtension(null)
    }
  }

  const openImportModal = (type: 'archive' | 'directory') => {
    setImportType(type)
    setImportPath('')
    setDuplicateResult(null)
    setImportModalOpen(true)
  }

  const handleChooseImportPath = async () => {
    try {
      const result = importType === 'archive'
        ? await chooseBrowserExtensionArchive()
        : await chooseBrowserExtensionDirectory()
      if (!result?.cancelled && result?.path) {
        setImportPath(result.path)
      }
    } catch (error: any) {
      toast.error(errorMessage(error, '选择路径失败'), 6000)
    }
  }

  const runImport = async (mode: 'ask' | 'overwrite' | 'new' | 'cancel' = 'ask') => {
    const path = importPath.trim()
    if (!path) {
      toast.warning(importType === 'archive' ? '请选择扩展压缩包' : '请选择扩展目录')
      return
    }
    setImporting(true)
    try {
      const result = importType === 'archive'
        ? await importBrowserExtensionArchive({ path, mode })
        : await importBrowserExtensionDirectory({ path, mode })
      if (result.duplicate) {
        setDuplicateResult(result)
        return
      }
      if (result.cancelled) {
        toast.info(result.message || '已取消导入')
        setImportModalOpen(false)
        return
      }
      toast.success(result.message || '扩展插件导入成功')
      setImportModalOpen(false)
      setDuplicateResult(null)
      await loadData(true)
      if (result.extension) {
        setSelectedExtension(result.extension)
        setDetailOpen(true)
        await loadBindings(result.extension.extensionId)
      }
    } catch (error: any) {
      toast.error(errorMessage(error, '导入扩展插件失败'), 6000)
    } finally {
      setImporting(false)
    }
  }

  const importPathDirectly = async (path: string): Promise<{ imported: boolean; duplicate: boolean; extension?: BrowserExtension | null }> => {
    const trimmed = path.trim()
    if (!trimmed) return { imported: false, duplicate: false }
    const type = importTypeFromPath(trimmed)
    const result = type === 'archive'
      ? await importBrowserExtensionArchive({ path: trimmed, mode: 'ask' })
      : await importBrowserExtensionDirectory({ path: trimmed, mode: 'ask' })
    if (result.duplicate) {
      setImportType(type)
      setImportPath(trimmed)
      setDuplicateResult(result)
      setImportModalOpen(true)
      return { imported: false, duplicate: true }
    }
    return {
      imported: !result.cancelled,
      duplicate: false,
      extension: result.extension,
    }
  }

  const handleDroppedPaths = async (paths: string[]) => {
    const validPaths = paths.map(item => String(item || '').trim()).filter(Boolean)
    if (validPaths.length === 0) return
    setImporting(true)
    let success = 0
    let stoppedForDuplicate = false
    let lastExtension: BrowserExtension | null = null
    try {
      for (const path of validPaths) {
        const result = await importPathDirectly(path)
        if (result.duplicate) {
          stoppedForDuplicate = true
          break
        }
        if (result.imported) {
          success += 1
          if (result.extension) {
            lastExtension = result.extension
          }
        }
      }
      if (success > 0) {
        await loadData(true)
        toast.success(success === 1 ? '扩展插件导入成功' : `已导入 ${success} 个扩展插件`)
        if (lastExtension) {
          setSelectedExtension(lastExtension)
          setDetailOpen(true)
        }
      }
      if (stoppedForDuplicate) {
        toast.info('发现重复扩展，请在弹窗中选择处理方式')
      }
    } catch (error: any) {
      toast.error(errorMessage(error, '拖拽导入扩展失败'), 6000)
    } finally {
      setImporting(false)
    }
  }

  const bindingMap = useMemo(() => {
    const map = new Map<string, BrowserExtensionBinding>()
    bindings.forEach(item => map.set(item.profileId, item))
    return map
  }, [bindings])

  const boundProfileIds = useMemo(() => new Set(bindings.map(item => item.profileId)), [bindings])

  const toggleProfileSelection = (profileId: string) => {
    setSelectedProfileIds(prev => {
      const next = new Set(prev)
      if (next.has(profileId)) {
        next.delete(profileId)
      } else {
        next.add(profileId)
      }
      return next
    })
  }

  const handleSelectUnboundProfiles = () => {
    setSelectedProfileIds(new Set(profiles.filter(profile => !boundProfileIds.has(profile.profileId)).map(profile => profile.profileId)))
  }

  const handleClearProfileSelection = () => {
    setSelectedProfileIds(new Set())
  }

  const refreshSelectedExtension = async () => {
    if (!selectedExtension) return
    const detail = await fetchBrowserExtension(selectedExtension.extensionId)
    if (detail) {
      setSelectedExtension(detail)
    }
    await loadData(true)
  }

  const handleAssignSelectedProfiles = async () => {
    if (!selectedExtension) return
    const profileIds = Array.from(selectedProfileIds)
    if (profileIds.length === 0) {
      toast.warning('请选择要绑定的实例')
      return
    }
    setBindingSaving(true)
    try {
      const list = await assignBrowserExtensionProfiles({
        extensionId: selectedExtension.extensionId,
        profileIds,
        mode: bindingMode,
        enabled: bindingEnabled,
      })
      setBindings(list)
      setSelectedProfileIds(new Set())
      await refreshSelectedExtension()
      toast.success('扩展绑定已保存')
    } catch (error: any) {
      toast.error(errorMessage(error, '保存扩展绑定失败'), 6000)
    } finally {
      setBindingSaving(false)
    }
  }

  const handleUnassignProfiles = async (profileIds: string[]) => {
    if (!selectedExtension || profileIds.length === 0) return
    setBindingSaving(true)
    try {
      const list = await unassignBrowserExtensionProfiles({
        extensionId: selectedExtension.extensionId,
        profileIds,
      })
      setBindings(list)
      setSelectedProfileIds(prev => {
        const next = new Set(prev)
        profileIds.forEach(id => next.delete(id))
        return next
      })
      await refreshSelectedExtension()
      toast.success('扩展绑定已移除')
    } catch (error: any) {
      toast.error(errorMessage(error, '移除扩展绑定失败'), 6000)
    } finally {
      setBindingSaving(false)
    }
  }

  const handleToggleBindingEnabled = async (binding: BrowserExtensionBinding) => {
    if (!selectedExtension) return
    setBindingSaving(true)
    try {
      const list = await assignBrowserExtensionProfiles({
        extensionId: selectedExtension.extensionId,
        profileIds: [binding.profileId],
        mode: binding.mode || 'shared',
        enabled: !binding.enabled,
      })
      setBindings(list)
      await refreshSelectedExtension()
      toast.success(!binding.enabled ? '扩展绑定已启用' : '扩展绑定已停用')
    } catch (error: any) {
      toast.error(errorMessage(error, '更新扩展绑定失败'), 6000)
    } finally {
      setBindingSaving(false)
    }
  }

  const columns: TableColumn<BrowserExtension>[] = useMemo(() => [
    {
      key: 'name',
      title: '扩展名称',
      width: '220px',
      render: (_, record) => (
        <div className="min-w-0">
          <div className="flex items-center gap-2">
            <FileCode2 className="w-4 h-4 text-[var(--color-text-muted)] shrink-0" />
            <span className="font-medium text-[var(--color-text-primary)] truncate">{record.name || record.extensionId}</span>
          </div>
          {record.description && (
            <p className="mt-1 text-xs text-[var(--color-text-muted)] line-clamp-1">{record.description}</p>
          )}
        </div>
      ),
    },
    {
      key: 'version',
      title: '版本',
      width: '100px',
      render: (value) => value || '-',
    },
    {
      key: 'manifestVersion',
      title: 'Manifest',
      width: '95px',
      render: (value) => value ? `MV${value}` : '-',
    },
    {
      key: 'sourceType',
      title: '来源',
      width: '100px',
      render: (value) => <Badge variant="default">{sourceLabel(value)}</Badge>,
    },
    {
      key: 'boundCount',
      title: '绑定实例',
      width: '100px',
      render: (value) => <Badge variant={value > 0 ? 'info' : 'default'}>{value || 0}</Badge>,
    },
    {
      key: 'updatedAt',
      title: '更新时间',
      width: '170px',
      render: (value) => formatTime(value),
    },
    {
      key: 'actions',
      title: '操作',
      width: '160px',
      render: (_, record) => (
        <div className="flex items-center gap-2" onClick={(event) => event.stopPropagation()}>
          <Button size="sm" variant="ghost" onClick={() => handleOpenDetail(record)}>详情</Button>
          <Button
            size="sm"
            variant="danger"
            onClick={() => handleDeleteClick(record)}
            disabled={record.boundCount > 0}
            title={record.boundCount > 0 ? '已绑定实例，不能删除' : '删除扩展'}
          >
            <Trash2 className="w-4 h-4" />
          </Button>
        </div>
      ),
    },
  ], [selectedExtension?.extensionId])

  return (
    <div
      className="relative space-y-5 animate-fade-in"
      onDragEnter={() => setDragActive(true)}
      onDragOver={(event) => {
        event.preventDefault()
        setDragActive(true)
      }}
      onDragLeave={(event) => {
        if (event.currentTarget === event.target) {
          setDragActive(false)
        }
      }}
      onDrop={(event) => {
        event.preventDefault()
        setDragActive(false)
      }}
    >
      {dragActive && (
        <div className="pointer-events-none absolute inset-0 z-20 flex items-center justify-center rounded-xl border-2 border-dashed border-[var(--color-accent)] bg-[var(--color-accent)]/10 backdrop-blur-[1px]">
          <div className="flex items-center gap-3 rounded-xl border border-[var(--color-border-default)] bg-[var(--color-bg-elevated)] px-5 py-4 shadow-lg">
            <UploadCloud className="w-6 h-6 text-[var(--color-accent)]" />
            <div>
              <p className="text-sm font-semibold text-[var(--color-text-primary)]">松开后自动导入扩展</p>
              <p className="text-xs text-[var(--color-text-muted)] mt-1">支持 ZIP、CRX 和已解压的扩展目录，目录会自动识别层级</p>
            </div>
          </div>
        </div>
      )}
      <div className="flex items-center justify-between gap-4">
        <div>
          <h1 className="text-xl font-semibold text-[var(--color-text-primary)]">扩展插件管理</h1>
          <p className="text-sm text-[var(--color-text-muted)] mt-1">管理已登记的浏览器扩展插件，支持拖拽 ZIP、CRX 或扩展目录导入</p>
        </div>
        <div className="flex items-center gap-2">
          <Button size="sm" variant="secondary" onClick={() => openImportModal('directory')}>
            <FolderOpen className="w-4 h-4" />
            导入目录
          </Button>
          <Button size="sm" onClick={() => openImportModal('archive')}>
            <Archive className="w-4 h-4" />
            导入压缩包
          </Button>
          <Button size="sm" variant="secondary" onClick={() => loadData(true)} loading={refreshing}>
            {!refreshing && <RefreshCw className="w-4 h-4" />}
            刷新
          </Button>
        </div>
      </div>

      <Card title="扩展列表" subtitle="支持查看详情、导入本地扩展和删除未绑定扩展">
        <Table
          columns={columns}
          data={extensions}
          rowKey="extensionId"
          loading={loading}
          emptyText="暂无扩展插件，可以从本地压缩包或解压目录导入。"
          onRowClick={handleOpenDetail}
          tableLayout="fixed"
          tableMinWidth="940px"
        />
      </Card>

      <Modal
        open={detailOpen}
        onClose={() => setDetailOpen(false)}
        title="扩展详情"
        width="860px"
        footer={<Button variant="secondary" onClick={() => setDetailOpen(false)}>关闭</Button>}
      >
        {detailLoading ? (
          <div className="py-10 text-center text-sm text-[var(--color-text-muted)]">正在加载详情...</div>
        ) : selectedExtension ? (
          <div className="space-y-5">
            <div>
              <h3 className="text-base font-semibold text-[var(--color-text-primary)]">{selectedExtension.name || selectedExtension.extensionId}</h3>
              {selectedExtension.description && (
                <p className="text-sm text-[var(--color-text-muted)] mt-1">{selectedExtension.description}</p>
              )}
            </div>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <DetailItem label="扩展 ID" value={selectedExtension.extensionId} />
              <DetailItem label="版本" value={selectedExtension.version || '-'} />
              <DetailItem label="Manifest 版本" value={selectedExtension.manifestVersion ? `MV${selectedExtension.manifestVersion}` : '-'} />
              <DetailItem label="来源" value={sourceLabel(selectedExtension.sourceType)} />
              <DetailItem label="绑定实例数量" value={String(selectedExtension.boundCount || 0)} />
              <DetailItem label="更新时间" value={formatTime(selectedExtension.updatedAt)} />
              <DetailItem label="安装目录" value={selectedExtension.installDir || '-'} wide />
              <DetailItem label="来源地址" value={selectedExtension.sourceUrl || '-'} wide />
              <DetailItem label="原始包路径" value={selectedExtension.packagePath || '-'} wide />
            </div>
            <div>
              <div className="flex items-center justify-between gap-3 mb-3">
                <div>
                  <p className="text-sm font-semibold text-[var(--color-text-primary)]">绑定实例</p>
                  <p className="text-xs text-[var(--color-text-muted)] mt-1">共享模式复用同一份扩展，独享模式会复制实例专属目录。</p>
                </div>
                <Button size="sm" variant="secondary" onClick={() => selectedExtension && loadBindings(selectedExtension.extensionId)} loading={bindingsLoading}>
                  {!bindingsLoading && <RefreshCw className="w-4 h-4" />}
                  刷新绑定
                </Button>
              </div>

              <div className="rounded-lg border border-[var(--color-border-default)] overflow-hidden">
                <div className="grid grid-cols-[34px_1fr_82px_82px_148px] items-center gap-2 bg-[var(--color-bg-muted)] px-3 py-2 text-xs font-medium text-[var(--color-text-muted)]">
                  <span />
                  <span>实例</span>
                  <span>模式</span>
                  <span>状态</span>
                  <span className="text-right">操作</span>
                </div>
                <div className="max-h-64 overflow-auto divide-y divide-[var(--color-border-default)]">
                  {profiles.length === 0 ? (
                    <div className="px-3 py-8 text-center text-sm text-[var(--color-text-muted)]">暂无浏览器实例</div>
                  ) : profiles.map(profile => {
                    const binding = bindingMap.get(profile.profileId)
                    const checked = selectedProfileIds.has(profile.profileId)
                    return (
                      <div key={profile.profileId} className="grid grid-cols-[34px_1fr_82px_82px_148px] items-center gap-2 px-3 py-2">
                        <input
                          type="checkbox"
                          checked={checked}
                          onChange={() => toggleProfileSelection(profile.profileId)}
                          disabled={bindingSaving}
                          className="h-4 w-4"
                        />
                        <div className="min-w-0">
                          <p className="truncate text-sm text-[var(--color-text-primary)]">{profile.profileName || profile.profileId}</p>
                          <p className="truncate text-xs text-[var(--color-text-muted)]">{profile.profileId}</p>
                        </div>
                        <Badge variant={binding?.mode === 'exclusive' ? 'warning' : binding ? 'info' : 'default'}>{binding ? bindingModeLabel(binding.mode) : '未绑定'}</Badge>
                        <Badge variant={binding?.enabled ? 'success' : binding ? 'default' : 'default'}>{binding ? (binding.enabled ? '启用' : '停用') : '-'}</Badge>
                        <div className="flex justify-end gap-1" onClick={(event) => event.stopPropagation()}>
                          {binding && (
                            <>
                              <Button size="sm" variant="ghost" onClick={() => handleToggleBindingEnabled(binding)} loading={bindingSaving}>
                                {binding.enabled ? '停用' : '启用'}
                              </Button>
                              <Button size="sm" variant="danger" onClick={() => handleUnassignProfiles([profile.profileId])} disabled={bindingSaving} title="解绑">
                                <Unlink className="w-4 h-4" />
                              </Button>
                            </>
                          )}
                        </div>
                      </div>
                    )
                  })}
                </div>
              </div>

              <div className="mt-3 grid grid-cols-1 md:grid-cols-[1fr_150px_120px_auto] gap-2 items-end">
                <div className="flex flex-wrap items-center gap-2">
                  <Button size="sm" variant="secondary" onClick={handleSelectUnboundProfiles} disabled={bindingSaving || profiles.length === 0}>选择未绑定</Button>
                  <Button size="sm" variant="ghost" onClick={handleClearProfileSelection} disabled={bindingSaving || selectedProfileIds.size === 0}>清空选择</Button>
                  <span className="text-xs text-[var(--color-text-muted)]">已选 {selectedProfileIds.size} 个实例</span>
                </div>
                <FormItem label="绑定模式">
                  <Select
                    value={bindingMode}
                    onChange={(event) => setBindingMode(event.target.value as 'shared' | 'exclusive')}
                    disabled={bindingSaving}
                    options={[
                      { value: 'shared', label: '共享' },
                      { value: 'exclusive', label: '独享' },
                    ]}
                  />
                </FormItem>
                <FormItem label="状态">
                  <Select
                    value={bindingEnabled ? 'enabled' : 'disabled'}
                    onChange={(event) => setBindingEnabled(event.target.value === 'enabled')}
                    disabled={bindingSaving}
                    options={[
                      { value: 'enabled', label: '启用' },
                      { value: 'disabled', label: '停用' },
                    ]}
                  />
                </FormItem>
                <Button onClick={handleAssignSelectedProfiles} loading={bindingSaving} disabled={selectedProfileIds.size === 0}>
                  {!bindingSaving && <Link2 className="w-4 h-4" />}
                  保存绑定
                </Button>
              </div>
            </div>

            <div>
              <p className="text-xs font-medium text-[var(--color-text-muted)] mb-2">Manifest 快照</p>
              <pre className="max-h-72 overflow-auto rounded-lg border border-[var(--color-border-default)] bg-[var(--color-bg-muted)] p-3 text-xs text-[var(--color-text-secondary)] whitespace-pre-wrap break-words">
                {selectedExtension.manifestJson || '-'}
              </pre>
            </div>
          </div>
        ) : (
          <div className="py-10 text-center text-sm text-[var(--color-text-muted)]">未选择扩展</div>
        )}
      </Modal>

      <Modal
        open={importModalOpen}
        onClose={() => {
          if (!importing) setImportModalOpen(false)
        }}
        title={importType === 'archive' ? '导入扩展压缩包' : '导入扩展目录'}
        width="620px"
        footer={
          duplicateResult ? (
            <>
              <Button variant="secondary" onClick={() => runImport('cancel')} disabled={importing}>取消导入</Button>
              <Button variant="secondary" onClick={() => runImport('new')} loading={importing}>作为新扩展导入</Button>
              <Button onClick={() => runImport('overwrite')} loading={importing}>覆盖已有扩展</Button>
            </>
          ) : (
            <>
              <Button variant="secondary" onClick={() => setImportModalOpen(false)} disabled={importing}>取消</Button>
              <Button onClick={() => runImport('ask')} loading={importing}>导入</Button>
            </>
          )
        }
      >
        <div className="space-y-4">
          <div className="grid grid-cols-2 gap-2">
            <Button
              variant={importType === 'archive' ? 'primary' : 'secondary'}
              onClick={() => {
                setImportType('archive')
                setDuplicateResult(null)
              }}
              disabled={importing}
            >
              <Archive className="w-4 h-4" />
              压缩包
            </Button>
            <Button
              variant={importType === 'directory' ? 'primary' : 'secondary'}
              onClick={() => {
                setImportType('directory')
                setDuplicateResult(null)
              }}
              disabled={importing}
            >
              <FolderOpen className="w-4 h-4" />
              目录
            </Button>
          </div>

          <FormItem label={importType === 'archive' ? '扩展压缩包路径' : '扩展目录路径'} required>
            <div className="flex gap-2">
              <Input
                value={importPath}
                onChange={(event) => {
                  setImportPath(event.target.value)
                  setDuplicateResult(null)
                }}
                placeholder={importType === 'archive' ? '选择或输入 .zip / .crx 文件路径' : '选择或输入包含 manifest.json 的目录'}
                disabled={importing}
              />
              <Button variant="secondary" onClick={handleChooseImportPath} disabled={importing}>
                选择
              </Button>
            </div>
            <p className="text-xs text-[var(--color-text-muted)] mt-1">
              {importType === 'archive'
                ? '支持 ZIP 和可解包的 CRX，本地导入成功后会复制到扩展库。'
                : '目录内需要包含 manifest.json，导入时会复制一份到扩展库。'}
            </p>
          </FormItem>

          {duplicateResult?.existing && (
            <div className="rounded-lg border border-[var(--color-warning)]/30 bg-[var(--color-warning)]/10 p-3">
              <p className="text-sm font-medium text-[var(--color-text-primary)]">发现同名同版本扩展</p>
              <p className="mt-1 text-xs text-[var(--color-text-muted)]">
                已有扩展：{duplicateResult.existing.name || duplicateResult.existing.extensionId} / {duplicateResult.existing.version || '-'}
              </p>
              <p className="mt-2 text-xs text-[var(--color-text-secondary)]">
                可以覆盖已有扩展目录，也可以作为一个新的扩展副本导入。
              </p>
            </div>
          )}
        </div>
      </Modal>

      <ConfirmModal
        open={deleteConfirmOpen}
        onClose={() => setDeleteConfirmOpen(false)}
        onConfirm={handleDeleteConfirm}
        title="确认删除扩展"
        content={`确定要删除扩展插件"${deletingExtension?.name || deletingExtension?.extensionId || ''}"吗？此操作会同时删除插件库中的目录。`}
        confirmText="删除"
        danger
      />
    </div>
  )
}

function DetailItem({ label, value, wide = false }: { label: string; value: string; wide?: boolean }) {
  return (
    <div className={wide ? 'md:col-span-2' : undefined}>
      <p className="text-xs text-[var(--color-text-muted)] mb-1">{label}</p>
      <p className="text-sm text-[var(--color-text-primary)] break-words">{value}</p>
    </div>
  )
}
