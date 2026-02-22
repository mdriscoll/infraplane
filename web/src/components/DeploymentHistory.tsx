import type { Deployment, InfrastructurePlan } from '../api/client'

const statusStyles: Record<string, string> = {
  pending: 'bg-yellow-100 text-yellow-800',
  in_progress: 'bg-blue-100 text-blue-800',
  succeeded: 'bg-green-100 text-green-800',
  failed: 'bg-red-100 text-red-800',
}

interface DeploymentHistoryProps {
  deployments: Deployment[]
  plans?: InfrastructurePlan[]
}

export default function DeploymentHistory({ deployments, plans }: DeploymentHistoryProps) {
  if (deployments.length === 0) {
    return <p className="text-sm text-gray-400 italic">No deployments yet.</p>
  }

  const getPlanLabel = (planId?: string) => {
    if (!planId || !plans) return null
    const plan = plans.find((p) => p.id === planId)
    if (!plan) return planId.slice(0, 8)
    return plan.plan_type
  }

  return (
    <div className="overflow-hidden border border-gray-200 rounded-lg">
      <table className="min-w-full divide-y divide-gray-200">
        <thead className="bg-gray-50">
          <tr>
            <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Status</th>
            <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Branch</th>
            <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Commit</th>
            <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Provider</th>
            <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Plan</th>
            <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Started</th>
          </tr>
        </thead>
        <tbody className="bg-white divide-y divide-gray-200">
          {deployments.map((d) => (
            <tr key={d.id}>
              <td className="px-4 py-3">
                <span className={`text-xs font-medium px-2 py-1 rounded-full ${statusStyles[d.status] || ''}`}>
                  {d.status}
                </span>
              </td>
              <td className="px-4 py-3 text-sm text-gray-900 font-mono">{d.git_branch}</td>
              <td className="px-4 py-3 text-sm text-gray-500 font-mono">{d.git_commit?.slice(0, 7) || '-'}</td>
              <td className="px-4 py-3 text-sm text-gray-500 uppercase">{d.provider}</td>
              <td className="px-4 py-3">
                {d.plan_id ? (
                  <span className="text-xs font-medium px-2 py-1 rounded-full bg-indigo-50 text-indigo-700">
                    {getPlanLabel(d.plan_id)}
                  </span>
                ) : (
                  <span className="text-xs text-gray-400">-</span>
                )}
              </td>
              <td className="px-4 py-3 text-sm text-gray-500">
                {new Date(d.started_at).toLocaleString()}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
