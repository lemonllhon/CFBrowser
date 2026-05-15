import { FolderTree, Tag } from 'lucide-react'
import { useSearchParams } from 'react-router-dom'
import { Button } from '../../../shared/components'
import { GroupManagementPage } from './GroupManagementPage'
import { TagManagementPage } from './TagManagementPage'

type OrganizationTab = 'tags' | 'groups'

function normalizeTab(value: string | null): OrganizationTab {
  return value === 'groups' ? 'groups' : 'tags'
}

export function OrganizationManagementPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const activeTab = normalizeTab(searchParams.get('tab'))

  const switchTab = (tab: OrganizationTab) => {
    setSearchParams(tab === 'groups' ? { tab: 'groups' } : {})
  }

  return (
    <div className="h-full flex flex-col animate-fade-in">
      <div className="px-5 pt-5 pb-3 border-b border-[var(--color-border-muted)] bg-[var(--color-bg-base)]">
        <div className="flex items-center justify-between gap-4">
          <div>
            <h1 className="text-xl font-semibold text-[var(--color-text-primary)]">组织管理</h1>
            <p className="text-sm text-[var(--color-text-muted)] mt-1">统一维护实例标签与分组结构</p>
          </div>
          <div className="flex items-center gap-2 rounded-lg border border-[var(--color-border-default)] bg-[var(--color-bg-secondary)] p-1">
            <Button
              size="sm"
              variant={activeTab === 'tags' ? undefined : 'ghost'}
              onClick={() => switchTab('tags')}
            >
              <Tag className="w-4 h-4" />
              标签
            </Button>
            <Button
              size="sm"
              variant={activeTab === 'groups' ? undefined : 'ghost'}
              onClick={() => switchTab('groups')}
            >
              <FolderTree className="w-4 h-4" />
              分组
            </Button>
          </div>
        </div>
      </div>
      <div className="flex-1 min-h-0 overflow-hidden">
        {activeTab === 'tags' ? <TagManagementPage embedded /> : <GroupManagementPage embedded />}
      </div>
    </div>
  )
}
