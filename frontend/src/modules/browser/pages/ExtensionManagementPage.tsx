import { useEffect, useMemo, useState } from 'react'
import { FileCode2, RefreshCw, Trash2 } from 'lucide-react'
import { Badge, Button, Card, ConfirmModal, Modal, Table, toast } from '../../../shared/components'
import type { TableColumn } from '../../../shared/components/Table'
import type { BrowserExtension } from '../types'
import { deleteBrowserExtension, fetchBrowserExtension, fetchBrowserExtensions } from '../api'

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

export function ExtensionManagementPage() {
  const [extensions, setExtensions] = useState<BrowserExtension[]>([])
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)
  const [detailOpen, setDetailOpen] = useState(false)
  const [detailLoading, setDetailLoading] = useState(false)
  const [selectedExtension, setSelectedExtension] = useState<BrowserExtension | null>(null)
  const [deleteConfirmOpen, setDeleteConfirmOpen] = useState(false)
  const [deletingExtension, setDeletingExtension] = useState<BrowserExtension | null>(null)

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
      toast.error(error?.message || '加载扩展列表失败')
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
      toast.error(error?.message || '加载扩展详情失败')
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
      toast.error(error?.message || '删除扩展插件失败')
    } finally {
      setDeletingExtension(null)
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
          <p className="text-sm text-[var(--color-text-muted)] mt-1">管理已登记的浏览器扩展插件，导入和实例绑定将在后续阶段开放</p>
        </div>
        <Button size="sm" variant="secondary" onClick={() => loadData(true)} loading={refreshing}>
          {!refreshing && <RefreshCw className="w-4 h-4" />}
          刷新
        </Button>
      </div>

      <Card title="扩展列表" subtitle="当前阶段支持查看详情和删除未绑定扩展">
        <Table
          columns={columns}
          data={extensions}
          rowKey="extensionId"
          loading={loading}
          emptyText="暂无扩展插件。后续阶段会开放压缩包、目录和地址导入。"
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
