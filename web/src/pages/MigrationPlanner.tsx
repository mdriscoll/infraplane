import { useState } from 'react'
import PlanViewer from '../components/PlanViewer'
import { useApplications, useGenerateMigrationPlan, usePlans } from '../hooks/useApi'

export default function MigrationPlanner() {
  const { data: apps, isLoading } = useApplications()
  const [selectedApp, setSelectedApp] = useState('')
  const [fromProvider, setFromProvider] = useState('aws')
  const [toProvider, setToProvider] = useState('gcp')
  const migrateMutation = useGenerateMigrationPlan(selectedApp)
  const { data: plans } = usePlans(selectedApp)

  const migrationPlans = (plans || []).filter((p) => p.plan_type === 'migration')

  const handleGenerate = (e: React.FormEvent) => {
    e.preventDefault()
    migrateMutation.mutate({ fromProvider, toProvider })
  }

  if (isLoading) {
    return <div className="text-center py-12 text-gray-500">Loading...</div>
  }

  return (
    <div className="space-y-8">
      <div>
        <h1 className="text-2xl font-bold text-gray-900">Migration Planner</h1>
        <p className="text-gray-500 mt-1">
          Generate migration plans to move between cloud providers
        </p>
      </div>

      <div className="bg-white border border-gray-200 rounded-lg p-6">
        <form onSubmit={handleGenerate} className="grid grid-cols-1 md:grid-cols-4 gap-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Application</label>
            <select
              value={selectedApp}
              onChange={(e) => setSelectedApp(e.target.value)}
              className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
            >
              <option value="">Select an app</option>
              {(apps || []).map((app) => (
                <option key={app.id} value={app.name}>
                  {app.name} ({app.provider.toUpperCase()})
                </option>
              ))}
            </select>
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">From Provider</label>
            <select
              value={fromProvider}
              onChange={(e) => setFromProvider(e.target.value)}
              className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
            >
              <option value="aws">AWS</option>
              <option value="gcp">GCP</option>
            </select>
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">To Provider</label>
            <select
              value={toProvider}
              onChange={(e) => setToProvider(e.target.value)}
              className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
            >
              <option value="aws">AWS</option>
              <option value="gcp">GCP</option>
            </select>
          </div>
          <div className="flex items-end">
            <button
              type="submit"
              disabled={!selectedApp || fromProvider === toProvider || migrateMutation.isPending}
              className="w-full bg-indigo-600 text-white px-4 py-2 rounded-lg hover:bg-indigo-700 disabled:opacity-50 text-sm transition-colors"
            >
              {migrateMutation.isPending ? 'Generating...' : 'Generate Plan'}
            </button>
          </div>
        </form>
        {fromProvider === toProvider && selectedApp && (
          <p className="mt-3 text-sm text-amber-600">Source and target providers must be different.</p>
        )}
        {migrateMutation.isError && (
          <p className="mt-3 text-sm text-red-500">{migrateMutation.error.message}</p>
        )}
      </div>

      {/* Migration plans */}
      {selectedApp && migrationPlans.length > 0 && (
        <div>
          <h2 className="text-lg font-semibold text-gray-900 mb-4">Migration Plans</h2>
          <div className="space-y-4">
            {migrationPlans.map((plan) => (
              <PlanViewer key={plan.id} plan={plan} />
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
