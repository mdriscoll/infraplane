import { useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import ResourceList from '../components/ResourceList'
import LiveResourceTable from '../components/LiveResourceTable'
import DeploymentHistory from '../components/DeploymentHistory'
import PlanViewer from '../components/PlanViewer'
import InfraGraphView from '../components/InfraGraphView'
import {
  useApplication,
  useRemoveResource,
  useDeleteApplication,
  useDeployments,
  useDeploy,
  usePlans,
  useGenerateHostingPlan,
  useReanalyzeSource,
  useAnalyzeUpload,
  useDiscoverLiveResources,
  useGraph,
  useGenerateGraph,
} from '../hooks/useApi'
import { pickAndReadDirectory, isDirectoryPickerSupported } from '../lib/directoryPicker'

function Spinner({ className = '' }: { className?: string }) {
  return (
    <svg className={`animate-spin h-4 w-4 ${className}`} xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
      <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
      <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
    </svg>
  )
}

export default function ApplicationDetail() {
  const { name } = useParams<{ name: string }>()
  const navigate = useNavigate()
  const { data, isLoading, error } = useApplication(name!)
  const { data: deployments } = useDeployments(name!)
  const { data: plans } = usePlans(name!)
  const removeResource = useRemoveResource(name!)
  const deleteApp = useDeleteApplication()
  const deployMutation = useDeploy(name!)
  const hostingPlan = useGenerateHostingPlan(name!)
  const reanalyze = useReanalyzeSource(name!)
  const analyzeUpload = useAnalyzeUpload(name!)
  const discoverLive = useDiscoverLiveResources(name!)
  const { data: graph } = useGraph(name!)
  const generateGraph = useGenerateGraph(name!)

  const [gitBranch, setGitBranch] = useState('main')
  const [gitCommit, setGitCommit] = useState('')

  if (isLoading) return <div className="text-center py-12 text-gray-500">Loading...</div>
  if (error) return <div className="text-center py-12 text-red-500">Error: {error.message}</div>
  if (!data) return null

  const { application: app, resources } = data

  const handleDeploy = (e: React.FormEvent) => {
    e.preventDefault()
    deployMutation.mutate({ gitBranch, gitCommit: gitCommit || undefined })
  }

  const handleDelete = () => {
    if (confirm(`Delete application "${app.name}"? This cannot be undone.`)) {
      deleteApp.mutate(app.name, { onSuccess: () => navigate('/') })
    }
  }

  const handleReanalyze = () => {
    if (app.source_path) {
      reanalyze.mutate()
    }
  }

  const handleBrowseAndAnalyze = async () => {
    try {
      const files = await pickAndReadDirectory()
      if (files && files.length > 0) {
        analyzeUpload.mutate(files)
      }
    } catch (err) {
      console.error('Failed to read directory:', err)
    }
  }

  const isReanalyzing = reanalyze.isPending || analyzeUpload.isPending

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
          <div className="flex gap-2">
            {app.source_path && (
              <button
                onClick={handleReanalyze}
                disabled={isReanalyzing}
                className="text-sm bg-gray-100 text-gray-700 px-3 py-1.5 rounded-lg hover:bg-gray-200 disabled:opacity-50 transition-colors"
              >
                {reanalyze.isPending ? 'Re-analyzing...' : 'Re-analyze Source'}
              </button>
            )}
            {isDirectoryPickerSupported() && (
              <button
                onClick={handleBrowseAndAnalyze}
                disabled={isReanalyzing}
                className="inline-flex items-center gap-1.5 text-sm bg-gray-100 text-gray-700 px-3 py-1.5 rounded-lg hover:bg-gray-200 disabled:opacity-50 transition-colors"
              >
                <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
                </svg>
                {analyzeUpload.isPending ? 'Analyzing...' : 'Analyze Folder'}
              </button>
            )}
          </div>
        </div>
        {reanalyze.isError && (
          <p className="text-sm text-red-500 mb-4">{reanalyze.error.message}</p>
        )}
        {analyzeUpload.isError && (
          <p className="text-sm text-red-500 mb-4">{analyzeUpload.error.message}</p>
        )}
        <ResourceList
          resources={resources}
          onRemove={(id) => removeResource.mutate(id)}
          removing={removeResource.isPending}
        />
      </section>

      {/* Live Cloud Resources */}
      <section>
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold text-gray-900">Live Cloud Resources</h2>
          <button
            onClick={() => discoverLive.mutate()}
            disabled={discoverLive.isPending}
            className="inline-flex items-center gap-2 text-sm bg-emerald-600 text-white px-3 py-1.5 rounded-lg hover:bg-emerald-700 disabled:opacity-50 transition-colors"
          >
            {discoverLive.isPending && <Spinner />}
            {discoverLive.isPending ? 'Discovering...' : 'Discover Resources'}
          </button>
        </div>
        {discoverLive.isPending && (
          <div className="flex items-center gap-3 p-4 bg-emerald-50 border border-emerald-100 rounded-lg mb-4">
            <Spinner className="text-emerald-600" />
            <div>
              <p className="text-sm font-medium text-emerald-900">Scanning cloud resources...</p>
              <p className="text-xs text-emerald-600">Analyzing deploy scripts and querying cloud APIs. This may take 30-60 seconds.</p>
            </div>
          </div>
        )}
        {discoverLive.isError && (
          <p className="text-sm text-red-500 mb-4">{discoverLive.error.message}</p>
        )}
        {discoverLive.data ? (
          <LiveResourceTable
            resources={discoverLive.data.resources}
            errors={discoverLive.data.errors}
            timestamp={discoverLive.data.timestamp}
          />
        ) : !discoverLive.isPending && (
          <p className="text-sm text-gray-400 italic">
            Click &quot;Discover Resources&quot; to scan your cloud environment for live resources.
          </p>
        )}
      </section>

      {/* Infrastructure Topology */}
      <section>
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold text-gray-900">Infrastructure Topology</h2>
          <button
            onClick={() => generateGraph.mutate()}
            disabled={generateGraph.isPending}
            className="inline-flex items-center gap-2 text-sm bg-indigo-600 text-white px-3 py-1.5 rounded-lg hover:bg-indigo-700 disabled:opacity-50 transition-colors"
          >
            {generateGraph.isPending && <Spinner />}
            {generateGraph.isPending ? 'Generating...' : 'Generate Graph'}
          </button>
        </div>
        {generateGraph.isPending && (
          <div className="flex items-center gap-3 p-4 bg-indigo-50 border border-indigo-100 rounded-lg mb-4">
            <Spinner className="text-indigo-600" />
            <div>
              <p className="text-sm font-medium text-indigo-900">Generating topology graph...</p>
              <p className="text-xs text-indigo-600">Analyzing resource connections with AI. This usually takes 10-20 seconds.</p>
            </div>
          </div>
        )}
        {generateGraph.isError && (
          <p className="text-sm text-red-500 mb-4">{generateGraph.error.message}</p>
        )}
        {graph ? (
          <div className="h-[500px] border border-gray-200 rounded-lg overflow-hidden bg-white">
            <InfraGraphView graph={graph} />
          </div>
        ) : !generateGraph.isPending && (
          <p className="text-sm text-gray-400 italic">
            No topology generated yet. Click &quot;Generate Graph&quot; to visualize your infrastructure.
          </p>
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
            className="inline-flex items-center gap-2 text-sm bg-indigo-600 text-white px-3 py-1.5 rounded-lg hover:bg-indigo-700 disabled:opacity-50 transition-colors"
          >
            {hostingPlan.isPending && <Spinner />}
            {hostingPlan.isPending ? 'Generating...' : 'Generate Hosting Plan'}
          </button>
        </div>
        {hostingPlan.isPending && (
          <div className="flex items-center gap-3 p-4 bg-indigo-50 border border-indigo-100 rounded-lg mb-4">
            <Spinner className="text-indigo-600" />
            <div>
              <p className="text-sm font-medium text-indigo-900">Generating hosting plan...</p>
              <p className="text-xs text-indigo-600">AI is analyzing your {resources.length} resource{resources.length !== 1 ? 's' : ''} and building a plan. This usually takes 15-30 seconds.</p>
            </div>
          </div>
        )}
        {hostingPlan.isError && (
          <p className="text-sm text-red-500 mb-4">{hostingPlan.error.message}</p>
        )}
        <div className="space-y-4">
          {(plans || []).map((plan) => (
            <PlanViewer key={plan.id} plan={plan} />
          ))}
          {(!plans || plans.length === 0) && !hostingPlan.isPending && (
            <p className="text-sm text-gray-400 italic">No plans generated yet.</p>
          )}
        </div>
      </section>
    </div>
  )
}
