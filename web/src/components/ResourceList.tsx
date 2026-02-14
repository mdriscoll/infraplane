import type { Resource } from '../api/client'

const kindIcons: Record<string, string> = {
  database: 'DB',
  compute: 'CPU',
  storage: 'S3',
  cache: 'CACHE',
  queue: 'Q',
  cdn: 'CDN',
  network: 'NET',
}

interface ResourceListProps {
  resources: Resource[]
  onRemove?: (id: string) => void
  removing?: boolean
}

export default function ResourceList({ resources, onRemove, removing }: ResourceListProps) {
  if (resources.length === 0) {
    return (
      <p className="text-sm text-gray-400 italic">No resources added yet.</p>
    )
  }

  return (
    <div className="space-y-3">
      {resources.map((r) => (
        <div key={r.id} className="flex items-center justify-between bg-gray-50 rounded-lg p-4 border border-gray-100">
          <div className="flex items-center gap-3">
            <span className="inline-flex items-center justify-center w-10 h-10 rounded-lg bg-indigo-100 text-indigo-700 text-xs font-bold">
              {kindIcons[r.kind] || r.kind.slice(0, 3).toUpperCase()}
            </span>
            <div>
              <p className="font-medium text-gray-900">{r.name}</p>
              <p className="text-xs text-gray-500">{r.kind}</p>
            </div>
          </div>
          <div className="flex items-center gap-4">
            {Object.entries(r.provider_mappings || {}).map(([provider, mapping]) => (
              <span key={provider} className="text-xs text-gray-500">
                {provider.toUpperCase()}: {mapping.service_name}
              </span>
            ))}
            {onRemove && (
              <button
                onClick={() => onRemove(r.id)}
                disabled={removing}
                className="text-xs text-red-500 hover:text-red-700 disabled:opacity-50"
              >
                Remove
              </button>
            )}
          </div>
        </div>
      ))}
    </div>
  )
}
