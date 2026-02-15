import type { Resource } from '../api/client'

const kindBadges: Record<string, { label: string; bg: string; text: string }> = {
  database: { label: 'DB',    bg: 'bg-green-100',  text: 'text-green-700' },
  compute:  { label: 'CPU',   bg: 'bg-purple-100', text: 'text-purple-700' },
  storage:  { label: 'S3',    bg: 'bg-gray-100',   text: 'text-gray-700' },
  cache:    { label: 'CACHE', bg: 'bg-orange-100',  text: 'text-orange-700' },
  queue:    { label: 'Q',     bg: 'bg-yellow-100', text: 'text-yellow-700' },
  cdn:      { label: 'CDN',   bg: 'bg-teal-100',   text: 'text-teal-700' },
  network:  { label: 'NET',   bg: 'bg-red-100',    text: 'text-red-700' },
  secrets:  { label: 'SEC',   bg: 'bg-amber-100',  text: 'text-amber-700' },
  policy:   { label: 'IAM',   bg: 'bg-indigo-100', text: 'text-indigo-700' },
}

function getKindBadge(kind: string) {
  return kindBadges[kind] || { label: kind.slice(0, 3).toUpperCase(), bg: 'bg-gray-100', text: 'text-gray-700' }
}

interface ResourceListProps {
  resources: Resource[]
  onRemove?: (id: string) => void
  removing?: boolean
}

export default function ResourceList({ resources, onRemove, removing }: ResourceListProps) {
  if (resources.length === 0) {
    return (
      <p className="text-sm text-gray-400 italic">No resources detected yet.</p>
    )
  }

  return (
    <div className="overflow-hidden border border-gray-200 rounded-lg">
      <table className="min-w-full divide-y divide-gray-200">
        <thead className="bg-gray-50">
          <tr>
            <th scope="col" className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Kind
            </th>
            <th scope="col" className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Name
            </th>
            <th scope="col" className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Provider Mappings
            </th>
            {onRemove && (
              <th scope="col" className="px-4 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider w-20">
                Action
              </th>
            )}
          </tr>
        </thead>
        <tbody className="bg-white divide-y divide-gray-100">
          {resources.map((r) => {
            const badge = getKindBadge(r.kind)
            return (
              <tr key={r.id} className="hover:bg-gray-50 transition-colors">
                <td className="px-4 py-3 whitespace-nowrap">
                  <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-bold ${badge.bg} ${badge.text}`}>
                    {badge.label}
                  </span>
                </td>
                <td className="px-4 py-3">
                  <span className="text-sm font-medium text-gray-900">{r.name}</span>
                </td>
                <td className="px-4 py-3">
                  <div className="flex flex-wrap gap-2">
                    {Object.entries(r.provider_mappings || {}).map(([provider, mapping]) => (
                      <span
                        key={provider}
                        className="inline-flex items-center gap-1 text-xs text-gray-600 bg-gray-100 rounded px-2 py-0.5"
                      >
                        <span className="font-semibold">{provider.toUpperCase()}</span>
                        {mapping.service_name}
                      </span>
                    ))}
                  </div>
                </td>
                {onRemove && (
                  <td className="px-4 py-3 text-right whitespace-nowrap">
                    <button
                      onClick={() => onRemove(r.id)}
                      disabled={removing}
                      className="text-xs text-red-500 hover:text-red-700 disabled:opacity-50"
                    >
                      Remove
                    </button>
                  </td>
                )}
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}
