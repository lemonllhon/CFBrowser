import { useEffect, useState } from 'react'
import { GripVertical, Plus, RotateCcw, Trash2 } from 'lucide-react'
import { Button, Card, ConfirmModal, Input, toast } from '../../../shared/components'
import type { BrowserStartURL } from '../types'
import { fetchDefaultStartURLs, resetDefaultStartURLs, saveDefaultStartURLs } from '../api'

export function DefaultStartURLPage() {
  const [items, setItems] = useState<BrowserStartURL[]>([])
  const [saving, setSaving] = useState(false)
  const [resetOpen, setResetOpen] = useState(false)
  const [dragIndex, setDragIndex] = useState<number | null>(null)

  useEffect(() => {
    fetchDefaultStartURLs().then(setItems)
  }, [])

  const handleChange = (index: number, field: keyof BrowserStartURL, value: string) => {
    setItems(prev => prev.map((item, i) => i === index ? { ...item, [field]: value } : item))
  }

  const handleSave = async () => {
    const valid = items.filter(i => i.name.trim() && i.url.trim())
    if (valid.length !== items.length) {
      toast.error('存在空的名称或 URL，请填写完整后保存')
      return
    }
    setSaving(true)
    try {
      await saveDefaultStartURLs(items)
      toast.success('默认打开页已保存，下次启动实例时生效')
    } finally {
      setSaving(false)
    }
  }

  const handleReset = async () => {
    await resetDefaultStartURLs()
    setItems(await fetchDefaultStartURLs())
    setResetOpen(false)
    toast.success('已恢复默认打开页')
  }

  const handleDragOver = (e: React.DragEvent, index: number) => {
    e.preventDefault()
    if (dragIndex === null || dragIndex === index) return
    setItems(prev => {
      const next = [...prev]
      const [moved] = next.splice(dragIndex, 1)
      next.splice(index, 0, moved)
      return next
    })
    setDragIndex(index)
  }

  return (
    <div className="space-y-5 animate-fade-in">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold text-[var(--color-text-primary)]">默认打开页</h1>
          <p className="text-sm text-[var(--color-text-muted)] mt-1">实例启动时自动打开的页面，可按任务调整顺序和内容</p>
        </div>
        <div className="flex gap-2">
          <Button variant="secondary" size="sm" onClick={() => setResetOpen(true)}>
            <RotateCcw className="w-4 h-4 mr-1.5" />
            恢复默认
          </Button>
          <Button size="sm" onClick={handleSave} loading={saving}>保存</Button>
        </div>
      </div>

      <Card title={`打开页列表（${items.length} 项）`} subtitle="拖拽左侧图标可调整启动打开顺序">
        <div className="space-y-2">
          {items.map((item, index) => (
            <div
              key={index}
              draggable
              onDragStart={() => setDragIndex(index)}
              onDragOver={e => handleDragOver(e, index)}
              onDragEnd={() => setDragIndex(null)}
              className={`flex items-center gap-2 p-2 rounded-lg border transition-colors ${
                dragIndex === index
                  ? 'border-[var(--color-primary)] bg-[var(--color-bg-hover)]'
                  : 'border-[var(--color-border)] hover:border-[var(--color-border-hover)]'
              }`}
            >
              <GripVertical className="w-4 h-4 text-[var(--color-text-muted)] cursor-grab shrink-0" />
              <Input value={item.name} onChange={e => handleChange(index, 'name', e.target.value)} placeholder="名称，如 IPPure" className="w-36 shrink-0" />
              <Input value={item.url} onChange={e => handleChange(index, 'url', e.target.value)} placeholder="https://..." className="flex-1" />
              <button
                type="button"
                onClick={() => setItems(prev => prev.filter((_, i) => i !== index))}
                className="p-1.5 rounded text-[var(--color-text-muted)] hover:text-red-500 hover:bg-red-50 transition-colors shrink-0"
                title="删除"
              >
                <Trash2 className="w-4 h-4" />
              </button>
            </div>
          ))}
          {items.length === 0 && <p className="text-sm text-[var(--color-text-muted)] text-center py-6">暂无默认打开页，点击下方按钮添加</p>}
        </div>
        <button
          type="button"
          onClick={() => setItems(prev => [...prev, { name: '', url: '' }])}
          className="mt-3 w-full flex items-center justify-center gap-2 py-2 rounded-lg border border-dashed border-[var(--color-border)] text-sm text-[var(--color-text-muted)] hover:border-[var(--color-primary)] hover:text-[var(--color-primary)] transition-colors"
        >
          <Plus className="w-4 h-4" />
          添加打开页
        </button>
      </Card>

      <ConfirmModal
        open={resetOpen}
        onClose={() => setResetOpen(false)}
        onConfirm={handleReset}
        title="恢复默认打开页"
        content="将清除当前自定义打开页，恢复为 IPPure、IPLark、Ping0。确定继续？"
        confirmText="确定恢复"
        danger
      />
    </div>
  )
}
