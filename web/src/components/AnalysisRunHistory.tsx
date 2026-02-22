import { useState } from 'react'
import { useAnalysisRuns, useResourcesByRun } from '../hooks/useApi'
import ResourceList from './ResourceList'

const triggerLabels: Record<string, { label: string; bg: string; text: string }> = {
  register: { label: 'Onboard', bg: 'bg-blue-100', text: 'text-blue-700' },
  reanalyze: { label: 'Re-analyze', bg: 'bg-purple-100', text: 'text-purple-700' },
  upload: { label: 'Upload', bg: 'bg-amber-100', text: 'text-amber-700' },
}

function TriggerBadge({ trigger }: { trigger: string }) {
  const style = triggerLabels[trigger] || { label: trigger, bg: 'bg-gray-100', text: 'text-gray-700' }
  return (
    <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${style.bg} ${style.text}`}>
      {style.label}
    </span>
  )
}

function RunResources({ runId }: { runId: string }) {
  const { data: resources, isLoading } = useResourcesByRun(runId)

  if (isLoading) {
    return <p className="text-sm text-gray-400 py-2 px-4">Loading resources...</p>
  }

  if (!resources || resources.length === 0) {
    return <p className="text-sm text-gray-400 italic py-2 px-4">No resources in this run.</p>
  }

  return <ResourceList resources={resources} />
}

function formatDate(iso: string) {
  const d = new Date(iso)
  return d.toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
    hour: 'numeric',
    minute: '2-digit',
  })
}

interface AnalysisRunHistoryProps {
  appName: string
}

export default function AnalysisRunHistory({ appName }: AnalysisRunHistoryProps) {
  const { data: runs } = useAnalysisRuns(appName)
  const [expandedId, setExpandedId] = useState<string | null>(null)

  if (!runs || runs.length === 0) return null

  return (
    <div className="mt-8">
      <h3 className="text-sm font-semibold text-gray-500 uppercase tracking-wider mb-3">
        Analysis History
      </h3>
      <div className="border border-gray-200 rounded-lg divide-y divide-gray-200">
        {runs.map((run, index) => {
          const isExpanded = expandedId === run.id
          const isCurrent = index === 0

          return (
            <div key={run.id}>
              <button
                onClick={() => setExpandedId(isExpanded ? null : run.id)}
                className="w-full flex items-center justify-between px-4 py-3 hover:bg-gray-50 transition-colors text-left"
              >
                <div className="flex items-center gap-3">
                  <svg
                    className={`w-4 h-4 text-gray-400 transition-transform ${isExpanded ? 'rotate-90' : ''}`}
                    fill="none"
                    viewBox="0 0 24 24"
                    strokeWidth={2}
                    stroke="currentColor"
                  >
                    <path strokeLinecap="round" strokeLinejoin="round" d="m8.25 4.5 7.5 7.5-7.5 7.5" />
                  </svg>
                  <TriggerBadge trigger={run.trigger} />
                  <span className="text-sm text-gray-700">
                    {run.resource_count} resource{run.resource_count !== 1 ? 's' : ''}
                  </span>
                  {isCurrent && (
                    <span className="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium bg-green-100 text-green-700">
                      Current
                    </span>
                  )}
                </div>
                <span className="text-xs text-gray-400">
                  {formatDate(run.created_at)}
                </span>
              </button>
              {isExpanded && (
                <div className="px-4 pb-4 bg-gray-50">
                  <RunResources runId={run.id} />
                </div>
              )}
            </div>
          )
        })}
      </div>
    </div>
  )
}
