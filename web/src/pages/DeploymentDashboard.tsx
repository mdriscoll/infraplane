import { useState } from 'react'
import DeploymentHistory from '../components/DeploymentHistory'
import { useApplications, useDeployments, useDeploy } from '../hooks/useApi'

export default function DeploymentDashboard() {
  const { data: apps, isLoading } = useApplications()
  const [selectedApp, setSelectedApp] = useState('')
  const { data: deployments } = useDeployments(selectedApp)
  const deployMutation = useDeploy(selectedApp)

  const [gitBranch, setGitBranch] = useState('main')
  const [gitCommit, setGitCommit] = useState('')

  const handleDeploy = (e: React.FormEvent) => {
    e.preventDefault()
    deployMutation.mutate({ gitBranch, gitCommit: gitCommit || undefined })
  }

  if (isLoading) {
    return <div className="text-center py-12 text-gray-500">Loading...</div>
  }

  return (
    <div className="space-y-8">
      <div>
        <h1 className="text-2xl font-bold text-gray-900">Deployment Dashboard</h1>
        <p className="text-gray-500 mt-1">Deploy and monitor your applications</p>
      </div>

      {/* App selector + deploy form */}
      <div className="bg-white border border-gray-200 rounded-lg p-6">
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
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
            <label className="block text-sm font-medium text-gray-700 mb-1">Branch</label>
            <input
              type="text"
              value={gitBranch}
              onChange={(e) => setGitBranch(e.target.value)}
              className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
              placeholder="main"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Commit SHA</label>
            <input
              type="text"
              value={gitCommit}
              onChange={(e) => setGitCommit(e.target.value)}
              className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
              placeholder="Optional"
            />
          </div>
          <div className="flex items-end">
            <button
              onClick={handleDeploy}
              disabled={!selectedApp || deployMutation.isPending}
              className="w-full bg-green-600 text-white px-4 py-2 rounded-lg hover:bg-green-700 disabled:opacity-50 text-sm transition-colors"
            >
              {deployMutation.isPending ? 'Deploying...' : 'Deploy'}
            </button>
          </div>
        </div>
        {deployMutation.isError && (
          <p className="mt-3 text-sm text-red-500">{deployMutation.error.message}</p>
        )}
        {deployMutation.isSuccess && (
          <p className="mt-3 text-sm text-green-600">
            Deployment started (status: {deployMutation.data.status})
          </p>
        )}
      </div>

      {/* Deployment history */}
      {selectedApp && (
        <div>
          <h2 className="text-lg font-semibold text-gray-900 mb-4">
            Deployments for {selectedApp}
          </h2>
          <DeploymentHistory deployments={deployments || []} />
        </div>
      )}
    </div>
  )
}
