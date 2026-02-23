import { useState } from 'react'
import type { ResourceHCLDetail } from '../api/client'
import Spinner from './Spinner'

interface TerraformReviewProps {
  resourceHCLs: ResourceHCLDetail[]
  onApprove: () => void
  onReject: () => void
  isApproving: boolean
}

export default function TerraformReview({ resourceHCLs, onApprove, onReject, isApproving }: TerraformReviewProps) {
  const [expandedIds, setExpandedIds] = useState<Set<string>>(new Set())

  const toggleExpand = (id: string) => {
    setExpandedIds((prev) => {
      const next = new Set(prev)
      if (next.has(id)) {
        next.delete(id)
      } else {
        next.add(id)
      }
      return next
    })
  }

  const expandAll = () => setExpandedIds(new Set(resourceHCLs.map((r) => r.resource_id)))
  const collapseAll = () => setExpandedIds(new Set())

  if (resourceHCLs.length === 0) {
    return (
      <div className="p-4 border border-yellow-200 rounded-lg bg-yellow-50">
        <p className="text-sm text-yellow-800">No Terraform resources to review.</p>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-sm font-semibold text-gray-900">
            Review Terraform ({resourceHCLs.length} resource{resourceHCLs.length !== 1 ? 's' : ''})
          </h3>
          <p className="text-xs text-gray-500 mt-0.5">
            Review the HCL that will be applied. Expand each resource to inspect its configuration.
          </p>
        </div>
        <div className="flex gap-2 text-xs">
          <button onClick={expandAll} className="text-indigo-600 hover:text-indigo-800 transition-colors">
            Expand all
          </button>
          <span className="text-gray-300">|</span>
          <button onClick={collapseAll} className="text-indigo-600 hover:text-indigo-800 transition-colors">
            Collapse all
          </button>
        </div>
      </div>

      {/* Per-resource expandable blocks */}
      <div className="space-y-2">
        {resourceHCLs.map((rh) => {
          const isExpanded = expandedIds.has(rh.resource_id)
          const lineCount = rh.hcl.split('\n').length
          return (
            <div key={rh.resource_id} className="border border-gray-200 rounded-lg overflow-hidden bg-white">
              <button
                onClick={() => toggleExpand(rh.resource_id)}
                className="w-full flex items-center justify-between px-4 py-3 hover:bg-gray-50 transition-colors text-left"
              >
                <div className="flex items-center gap-3 min-w-0">
                  <svg
                    className={`w-4 h-4 text-gray-400 shrink-0 transition-transform duration-200 ${isExpanded ? 'rotate-90' : ''}`}
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                  >
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                  </svg>
                  <span className="text-xs font-medium px-2 py-0.5 rounded-full bg-indigo-50 text-indigo-700">
                    {rh.resource_kind}
                  </span>
                  <span className="text-sm font-medium text-gray-900">{rh.resource_name}</span>
                  <span className="text-xs text-gray-500">{rh.service_name}</span>
                </div>
                <span className="text-xs text-gray-400 shrink-0">{lineCount} line{lineCount !== 1 ? 's' : ''}</span>
              </button>
              {isExpanded && (
                <div className="border-t border-gray-200">
                  <pre className="bg-gray-900 text-gray-200 p-4 text-xs font-mono overflow-x-auto whitespace-pre">
                    {rh.hcl}
                  </pre>
                </div>
              )}
            </div>
          )
        })}
      </div>

      {/* Approve / Reject buttons */}
      <div className="flex gap-3 pt-2">
        <button
          onClick={onApprove}
          disabled={isApproving}
          className="inline-flex items-center gap-2 bg-green-600 text-white px-5 py-2.5 rounded-lg hover:bg-green-700 disabled:opacity-50 text-sm font-medium transition-colors"
        >
          {isApproving && <Spinner />}
          {isApproving ? 'Applying...' : 'Approve & Apply'}
        </button>
        <button
          onClick={onReject}
          disabled={isApproving}
          className="border border-red-200 text-red-600 px-5 py-2.5 rounded-lg hover:bg-red-50 disabled:opacity-50 text-sm font-medium transition-colors"
        >
          Reject
        </button>
      </div>
    </div>
  )
}
