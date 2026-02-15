import type { LiveResource } from '../api/client'

const statusStyles: Record<string, { bg: string; text: string; dot: string }> = {
  active:       { bg: 'bg-green-50',  text: 'text-green-700',  dot: 'bg-green-500' },
  provisioning: { bg: 'bg-blue-50',   text: 'text-blue-700',   dot: 'bg-blue-500' },
  stopped:      { bg: 'bg-gray-50',   text: 'text-gray-600',   dot: 'bg-gray-400' },
  error:        { bg: 'bg-red-50',    text: 'text-red-700',    dot: 'bg-red-500' },
  unknown:      { bg: 'bg-yellow-50', text: 'text-yellow-700', dot: 'bg-yellow-500' },
}

function getStatusStyle(status: string) {
  return statusStyles[status] || statusStyles.unknown
}

interface LiveResourceTableProps {
  resources: LiveResource[]
  errors?: string[]
  timestamp: string
}

export default function LiveResourceTable({ resources, errors, timestamp }: LiveResourceTableProps) {
  const timeStr = new Date(timestamp).toLocaleString()

  return (
    <div className="space-y-3">
      {/* Errors banner */}
      {errors && errors.length > 0 && (
        <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-3">
          <p className="text-xs font-medium text-yellow-800 mb-1">Some discovery steps had issues:</p>
          <ul className="list-disc list-inside text-xs text-yellow-700 space-y-0.5">
            {errors.map((e, i) => (
              <li key={i}>{e}</li>
            ))}
          </ul>
        </div>
      )}

      {resources.length === 0 ? (
        <p className="text-sm text-gray-400 italic">No live resources discovered.</p>
      ) : (
        <div className="overflow-hidden border border-gray-200 rounded-lg">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th scope="col" className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Status
                </th>
                <th scope="col" className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Resource Type
                </th>
                <th scope="col" className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Name
                </th>
                <th scope="col" className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Region
                </th>
                <th scope="col" className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Details
                </th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-100">
              {resources.map((r, i) => {
                const style = getStatusStyle(r.status)
                return (
                  <tr key={`${r.name}-${r.resource_type}-${i}`} className="hover:bg-gray-50 transition-colors">
                    <td className="px-4 py-3 whitespace-nowrap">
                      <span className={`inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-xs font-medium ${style.bg} ${style.text}`}>
                        <span className={`w-1.5 h-1.5 rounded-full ${style.dot}`} />
                        {r.status}
                      </span>
                    </td>
                    <td className="px-4 py-3 whitespace-nowrap">
                      <span className="text-sm font-medium text-gray-900">{r.resource_type}</span>
                    </td>
                    <td className="px-4 py-3">
                      <span className="text-sm text-gray-700 font-mono">{r.name}</span>
                    </td>
                    <td className="px-4 py-3 whitespace-nowrap">
                      <span className="text-sm text-gray-500">{r.region || '—'}</span>
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex flex-wrap gap-1.5">
                        {Object.entries(r.details || {}).map(([key, value]) => (
                          <span
                            key={key}
                            className="inline-flex items-center gap-1 text-xs text-gray-600 bg-gray-100 rounded px-2 py-0.5"
                            title={`${key}: ${value}`}
                          >
                            <span className="font-semibold">{key}:</span>
                            <span className="truncate max-w-[150px]">{value}</span>
                          </span>
                        ))}
                        {(!r.details || Object.keys(r.details).length === 0) && (
                          <span className="text-xs text-gray-400">—</span>
                        )}
                      </div>
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      )}

      <p className="text-xs text-gray-400 text-right">
        Last checked: {timeStr}
      </p>
    </div>
  )
}
