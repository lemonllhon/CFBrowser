import { useEffect, useMemo, useState } from 'react'
import { Archive, FileCode2, FolderOpen, RefreshCw, Trash2 } from 'lucide-react'
import { Badge, Button, Card, ConfirmModal, FormItem, Input, Modal, Table, toast } from '../../../shared/components'
import type { TableColumn } from '../../../shared/components/Table'
import type { BrowserExtension, BrowserExtensionImportResult } from '../types'
import { chooseBrowserExtensionArchive, chooseBrowserExtensionDirectory, deleteBrowserExtension, fetchBrowserExtension, fetchBrowserExtensions, importBrowserExtensionArchive, importBrowserExtensionDirectory } from '../api'

const sourceTypeText: Record<string, string> = {
  zip: '压缩包',
  crx: 'CRX',
  url: '地址导入',
  directory: '目录',
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
  }, [])

  const handleOpenDetail = async (record: BrowserExtension) => {
    setSelectedExtension(record)
    setDetailOpen(true)
    setDetailLoading(true)
    try {
      const detail = await fetchBrowserExtension(record.extensionId)
      if (detail) {
        setSelectedExtension(detail)
      }
    } catch (error: any) {
      toast.error(errorMessage(error, '加载扩展详情失败'), 6000)
    } finally {
      setDetailLoading(false)
    }
  }

  const handleDeleteClick = (record: BrowserExtension) => {
    if (record.boundCount > 0) {
      toast.warning(`该扩展已绑定 ${record.boundCount} 个实例，阶段 1 暂不支持删除`)
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
      }
    } catch (error: any) {
      toast.error(errorMessage(error, '导入扩展插件失败'), 6000)
    } finally {
      setImporting(false)
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
    <div className="space-y-5 animate-fade-in">
      <div className="flex items-center justify-between gap-4">
        <div>
          <h1 className="text-xl font-semibold text-[var(--color-text-primary)]">扩展插件管理</h1>
          <p className="text-sm text-[var(--color-text-muted)] mt-1">管理已登记的浏览器扩展插件，本阶段支持本地压缩包和目录导入</p>
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
        width="720px"
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
