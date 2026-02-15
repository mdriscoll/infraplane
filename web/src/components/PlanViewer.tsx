import { useState, useCallback } from 'react'
import type { InfrastructurePlan, Resource } from '../api/client'
import { generateTerraformHCL } from '../api/client'

const kindIcons: Record<string, string> = {
  compute: '🖥️',
  database: '🗄️',
  cache: '⚡',
  storage: '📦',
  queue: '📬',
  cdn: '🌐',
  network: '🔗',
  secrets: '🔐',
  policy: '🛡️',
}

function Spinner({ className = '' }: { className?: string }) {
  return (
    <svg className={`animate-spin h-4 w-4 ${className}`} xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
      <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
      <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
    </svg>
  )
}

interface HCLResult {
  resourceId: string
  resourceName: string
  hcl: string | null
  error: string | null
  loading: boolean
}

export default function PlanViewer({ plan }: { plan: InfrastructurePlan }) {
  const [selectedResources, setSelectedResources] = useState<Set<string>>(new Set())
  const [hclResults, setHclResults] = useState<HCLResult[]>([])
  const [isGenerating, setIsGenerating] = useState(false)
  const [currentIndex, setCurrentIndex] = useState(-1)

  const provider = plan.to_provider || plan.from_provider || 'aws'
  const resources = plan.resources || []

  const toggleResource = (id: string) => {
    setSelectedResources(prev => {
      const next = new Set(prev)
      if (next.has(id)) {
        next.delete(id)
      } else {
        next.add(id)
      }
      return next
    })
  }

  const selectAll = () => {
    if (selectedResources.size === resources.length) {
      setSelectedResources(new Set())
    } else {
      setSelectedResources(new Set(resources.map(r => r.id)))
    }
  }

  const handleGenerateHCL = useCallback(async () => {
    const selected = resources.filter(r => selectedResources.has(r.id))
    if (selected.length === 0) return

    setIsGenerating(true)
    setHclResults(selected.map(r => ({
      resourceId: r.id,
      resourceName: r.name,
      hcl: null,
      error: null,
      loading: true,
    })))

    for (let i = 0; i < selected.length; i++) {
      setCurrentIndex(i)
      try {
        const result = await generateTerraformHCL(selected[i].id, provider)
        setHclResults(prev => prev.map((item, idx) =>
          idx === i ? { ...item, hcl: result.hcl, loading: false } : item
        ))
      } catch (err) {
        setHclResults(prev => prev.map((item, idx) =>
          idx === i ? { ...item, error: err instanceof Error ? err.message : 'Failed', loading: false } : item
        ))
      }
    }

    setCurrentIndex(-1)
    setIsGenerating(false)
  }, [resources, selectedResources, provider])

  const getProviderServiceName = (resource: Resource): string => {
    const mapping = resource.provider_mappings?.[provider]
    return mapping?.service_name || resource.kind
  }

  return (
    <div className="bg-white border border-gray-200 rounded-lg p-6">
      <div className="flex items-center justify-between mb-4">
        <h3 className="font-semibold text-gray-900">
          {plan.plan_type === 'hosting' ? 'Hosting Plan' : 'Migration Plan'}
        </h3>
        <span className="text-xs text-gray-500">
          {new Date(plan.created_at).toLocaleString()}
        </span>
      </div>

      {plan.from_provider && plan.to_provider && (
        <div className="mb-4 flex items-center gap-2 text-sm text-gray-600">
          <span className="uppercase font-medium">{plan.from_provider}</span>
          <span>&rarr;</span>
          <span className="uppercase font-medium">{plan.to_provider}</span>
        </div>
      )}

      {plan.estimated_cost && (
        <div className="mb-4 p-4 bg-indigo-50 rounded-lg">
          <p className="text-sm font-medium text-indigo-900">
            Estimated Cost: ${plan.estimated_cost.monthly_cost_usd.toFixed(2)}/month
          </p>
          {plan.estimated_cost.breakdown && (
            <div className="mt-2 grid grid-cols-2 gap-1 text-xs text-indigo-700">
              {Object.entries(plan.estimated_cost.breakdown).map(([key, value]) => (
                <div key={key}>
                  {key}: ${value.toFixed(2)}
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      <div className="prose prose-sm max-w-none mb-6">
        <pre className="whitespace-pre-wrap text-sm bg-gray-50 p-4 rounded-lg border border-gray-100 overflow-auto max-h-96">
          {plan.content}
        </pre>
      </div>

      {/* Terraform HCL Generation */}
      {resources.length > 0 && (
        <div className="border-t border-gray-200 pt-6">
          <div className="flex items-center justify-between mb-3">
            <h4 className="text-sm font-semibold text-gray-900">Generate Terraform Configs</h4>
            <button
              onClick={selectAll}
              className="text-xs text-indigo-600 hover:text-indigo-800"
            >
              {selectedResources.size === resources.length ? 'Deselect All' : 'Select All'}
            </button>
          </div>
          <p className="text-xs text-gray-500 mb-3">
            Select resources to generate production-ready Terraform HCL for <span className="font-medium uppercase">{provider}</span>:
          </p>

          <div className="space-y-2 mb-4">
            {resources.map(resource => (
              <label
                key={resource.id}
                className={`flex items-center gap-3 p-3 rounded-lg border cursor-pointer transition-colors ${
                  selectedResources.has(resource.id)
                    ? 'border-indigo-300 bg-indigo-50'
                    : 'border-gray-200 hover:border-gray-300 hover:bg-gray-50'
                }`}
              >
                <input
                  type="checkbox"
                  checked={selectedResources.has(resource.id)}
                  onChange={() => toggleResource(resource.id)}
                  disabled={isGenerating}
                  className="h-4 w-4 rounded border-gray-300 text-indigo-600 focus:ring-indigo-500"
                />
                <span className="text-base">{kindIcons[resource.kind] || '📋'}</span>
                <div className="flex-1 min-w-0">
                  <span className="text-sm font-medium text-gray-900">{resource.name}</span>
                  <span className="text-xs text-gray-500 ml-2">{getProviderServiceName(resource)}</span>
                </div>
                <span className="text-xs px-2 py-0.5 rounded-full bg-gray-100 text-gray-600">{resource.kind}</span>
              </label>
            ))}
          </div>

          <button
            onClick={handleGenerateHCL}
            disabled={selectedResources.size === 0 || isGenerating}
            className="inline-flex items-center gap-2 text-sm bg-emerald-600 text-white px-4 py-2 rounded-lg hover:bg-emerald-700 disabled:opacity-50 transition-colors"
          >
            {isGenerating && <Spinner />}
            {isGenerating
              ? `Generating ${currentIndex + 1} of ${selectedResources.size}...`
              : `Generate Terraform (${selectedResources.size} resource${selectedResources.size !== 1 ? 's' : ''})`
            }
          </button>

          {/* Progress bar during generation */}
          {isGenerating && (
            <div className="mt-3">
              <div className="h-1.5 bg-gray-200 rounded-full overflow-hidden">
                <div
                  className="h-full bg-emerald-500 transition-all duration-500 ease-out"
                  style={{ width: `${((currentIndex + 1) / hclResults.length) * 100}%` }}
                />
              </div>
              <p className="text-xs text-gray-500 mt-1">
                Generating HCL for {hclResults[currentIndex]?.resourceName}...
              </p>
            </div>
          )}

          {/* HCL Results */}
          {hclResults.length > 0 && !isGenerating && (
            <div className="mt-4 space-y-3">
              {hclResults.map(result => (
                <div key={result.resourceId} className="border border-gray-200 rounded-lg overflow-hidden">
                  <div className="flex items-center justify-between px-4 py-2 bg-gray-50 border-b border-gray-200">
                    <span className="text-sm font-medium text-gray-900">{result.resourceName}.tf</span>
                    {result.hcl && (
                      <button
                        onClick={() => navigator.clipboard.writeText(result.hcl!)}
                        className="text-xs text-indigo-600 hover:text-indigo-800"
                      >
                        Copy
                      </button>
                    )}
                  </div>
                  {result.error ? (
                    <p className="p-3 text-sm text-red-500">{result.error}</p>
                  ) : result.hcl ? (
                    <pre className="p-4 text-xs bg-gray-900 text-green-400 overflow-auto max-h-80 font-mono">
                      {result.hcl}
                    </pre>
                  ) : (
                    <div className="p-3 flex items-center gap-2 text-sm text-gray-500">
                      <Spinner className="text-gray-400" />
                      Generating...
                    </div>
                  )}
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  )
}
