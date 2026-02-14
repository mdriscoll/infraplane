import { Link } from 'react-router-dom'
import type { Application } from '../api/client'

const statusColors: Record<string, string> = {
  draft: 'bg-gray-100 text-gray-700',
  provisioned: 'bg-blue-100 text-blue-700',
  deployed: 'bg-green-100 text-green-700',
}

const providerLabels: Record<string, string> = {
  aws: 'AWS',
  gcp: 'GCP',
}

export default function AppCard({ app }: { app: Application }) {
  return (
    <Link
      to={`/applications/${app.name}`}
      className="block bg-white rounded-lg border border-gray-200 p-6 hover:shadow-md transition-shadow"
    >
      <div className="flex items-center justify-between mb-2">
        <h3 className="text-lg font-semibold text-gray-900">{app.name}</h3>
        <span className={`text-xs font-medium px-2.5 py-0.5 rounded-full ${statusColors[app.status] || statusColors.draft}`}>
          {app.status}
        </span>
      </div>
      {app.description && (
        <p className="text-sm text-gray-500 mb-3">{app.description}</p>
      )}
      <div className="flex items-center gap-2">
        <span className="inline-flex items-center text-xs font-medium px-2 py-1 rounded bg-indigo-50 text-indigo-700">
          {providerLabels[app.provider] || app.provider}
        </span>
      </div>
    </Link>
  )
}
