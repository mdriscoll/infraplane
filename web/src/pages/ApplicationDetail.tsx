import { useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import ResourceList from '../components/ResourceList'
import DeploymentHistory from '../components/DeploymentHistory'
import PlanViewer from '../components/PlanViewer'
import {
  useApplication,
  useAddResource,
  useRemoveResource,
  useDeleteApplication,
  useDeployments,
  useDeploy,
  usePlans,
  useGenerateHostingPlan,
  useReanalyzeSource,
} from '../hooks/useApi'

export default function ApplicationDetail() {
  const { name } = useParams<{ name: string }>()
  const navigate = useNavigate()
  const { data, isLoading, error } = useApplication(name!)
  const { data: deployments } = useDeployments(name!)
  const { data: plans } = usePlans(name!)
  const addResource = useAddResource(name!)
  const removeResource = useRemoveResource(name!)
  const deleteApp = useDeleteApplication()
  const deployMutation = useDeploy(name!)
  const hostingPlan = useGenerateHostingPlan(name!)
  const reanalyze = useReanalyzeSource(name!)

  const [resourceDesc, setResourceDesc] = useState('')
  const [gitBranch, setGitBranch] = useState('main')
  const [gitCommit, setGitCommit] = useState('')

  if (isLoading) return <div className="text-center py-12 text-gray-500">Loading...</div>
  if (error) return <div className="text-center py-12 text-red-500">Error: {error.message}</div>
  if (!data) return null

  const { application: app, resources } = data

  const handleAddResource = (e: React.FormEvent) => {
    e.preventDefault()
    addResource.mutate(resourceDesc, { onSuccess: () => setResourceDesc('') })
  }

  const handleDeploy = (e: React.FormEvent) => {
    e.preventDefault()
    deployMutation.mutate({ gitBranch, gitCommit: gitCommit || undefined })
  }

  const handleDelete = () => {
    if (confirm(`Delete application "${app.name}"? This cannot be undone.`)) {
      deleteApp.mutate(app.name, { onSuccess: () => navigate('/') })
    }
  }

  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">{app.name}</h1>
          {app.description && <p className="text-gray-500 mt-1">{app.description}</p>}
          {app.source_path && (
            <p className="text-sm text-gray-400 mt-1 truncate" title={app.source_path}>
              Source: {app.source_path}
            </p>
          )}
          <div className="flex gap-2 mt-2">
            <span className="text-xs font-medium px-2.5 py-0.5 rounded-full bg-indigo-50 text-indigo-700">
              {app.provider.toUpperCase()}
            </span>
            <span className="text-xs font-medium px-2.5 py-0.5 rounded-full bg-gray-100 text-gray-700">
              {app.status}
            </span>
          </div>
        </div>
        <button
          onClick={handleDelete}
          className="text-sm text-red-500 hover:text-red-700 border border-red-200 px-3 py-1.5 rounded-lg hover:bg-red-50 transition-colors"
        >
          Delete
        </button>
      </div>

      {/* Resources */}
      <section>
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold text-gray-900">Resources</h2>
          {app.source_path && (
            <button
              onClick={() => reanalyze.mutate()}
              disabled={reanalyze.isPending}
              className="text-sm bg-gray-100 text-gray-700 px-3 py-1.5 rounded-lg hover:bg-gray-200 disabled:opacity-50 transition-colors"
            >
              {reanalyze.isPending ? 'Re-analyzing...' : 'Re-analyze Source'}
            </button>
          )}
        </div>
        {reanalyze.isError && (
          <p className="text-sm text-red-500 mb-4">{reanalyze.error.message}</p>
        )}
        <ResourceList
          resources={resources}
          onRemove={(id) => removeResource.mutate(id)}
          removing={removeResource.isPending}
        />
        <form onSubmit={handleAddResource} className="mt-4 flex gap-2">
          <input
            type="text"
            value={resourceDesc}
            onChange={(e) => setResourceDesc(e.target.value)}
            className="flex-1 border border-gray-300 rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
            placeholder="Describe an additional resource (e.g. 'a PostgreSQL database for user data')"
            required
          />
          <button
            type="submit"
            disabled={addResource.isPending}
            className="bg-indigo-600 text-white px-4 py-2 rounded-lg hover:bg-indigo-700 disabled:opacity-50 text-sm transition-colors"
          >
            {addResource.isPending ? 'Adding...' : 'Add Resource'}
          </button>
        </form>
        {addResource.isError && (
          <p className="mt-2 text-sm text-red-500">{addResource.error.message}</p>
        )}
      </section>

      {/* Deploy */}
      <section>
        <h2 className="text-lg font-semibold text-gray-900 mb-4">Deploy</h2>
        <form onSubmit={handleDeploy} className="flex gap-2">
          <input
            type="text"
            value={gitBranch}
            onChange={(e) => setGitBranch(e.target.value)}
            className="border border-gray-300 rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
            placeholder="Branch"
            required
          />
          <input
            type="text"
            value={gitCommit}
            onChange={(e) => setGitCommit(e.target.value)}
            className="border border-gray-300 rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
            placeholder="Commit SHA (optional)"
          />
          <button
            type="submit"
            disabled={deployMutation.isPending}
            className="bg-green-600 text-white px-4 py-2 rounded-lg hover:bg-green-700 disabled:opacity-50 text-sm transition-colors"
          >
            {deployMutation.isPending ? 'Deploying...' : 'Deploy'}
          </button>
        </form>
        {deployMutation.isError && (
          <p className="mt-2 text-sm text-red-500">{deployMutation.error.message}</p>
        )}
      </section>

      {/* Deployment History */}
      <section>
        <h2 className="text-lg font-semibold text-gray-900 mb-4">Deployment History</h2>
        <DeploymentHistory deployments={deployments || []} />
      </section>

      {/* Hosting Plan */}
      <section>
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold text-gray-900">Infrastructure Plans</h2>
          <button
            onClick={() => hostingPlan.mutate()}
            disabled={hostingPlan.isPending}
            className="text-sm bg-indigo-600 text-white px-3 py-1.5 rounded-lg hover:bg-indigo-700 disabled:opacity-50 transition-colors"
          >
            {hostingPlan.isPending ? 'Generating...' : 'Generate Hosting Plan'}
          </button>
        </div>
        {hostingPlan.isError && (
          <p className="text-sm text-red-500 mb-4">{hostingPlan.error.message}</p>
        )}
        <div className="space-y-4">
          {(plans || []).map((plan) => (
            <PlanViewer key={plan.id} plan={plan} />
          ))}
          {(!plans || plans.length === 0) && (
            <p className="text-sm text-gray-400 italic">No plans generated yet.</p>
          )}
        </div>
      </section>
    </div>
  )
}
