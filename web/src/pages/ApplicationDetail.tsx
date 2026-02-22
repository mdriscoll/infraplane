import { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useQueryClient } from '@tanstack/react-query'
import ResourceList from '../components/ResourceList'
import LiveResourceTable from '../components/LiveResourceTable'
import DeploymentHistory from '../components/DeploymentHistory'
import DeployLog from '../components/DeployLog'
import PlanViewer from '../components/PlanViewer'
import InfraGraphView from '../components/InfraGraphView'
import Spinner from '../components/Spinner'
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
  useComplianceFrameworks,
  useDeploymentStream,
} from '../hooks/useApi'
import { pickAndReadDirectory, isDirectoryPickerSupported } from '../lib/directoryPicker'

type Tab = 'plan' | 'deploy' | 'monitor' | 'optimize'

const tabs: { key: Tab; label: string }[] = [
  { key: 'plan', label: 'Plan' },
  { key: 'deploy', label: 'Deploy' },
  { key: 'monitor', label: 'Monitor' },
  { key: 'optimize', label: 'Optimize' },
]

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
  const { data: frameworks } = useComplianceFrameworks(data?.application?.provider)

  const queryClient = useQueryClient()
  const [activeTab, setActiveTab] = useState<Tab>('plan')
  const [gitBranch, setGitBranch] = useState('main')
  const [gitCommit, setGitCommit] = useState('')
  const [selectedPlanId, setSelectedPlanId] = useState<string | null>(null)
  const [expandedPlanIds, setExpandedPlanIds] = useState<Set<string>>(new Set())
  const [streamingDeployId, setStreamingDeployId] = useState<string | null>(null)
  const { events: streamEvents, isStreaming, isComplete: streamComplete, finalStatus } = useDeploymentStream(streamingDeployId)

  if (isLoading) return <div className="text-center py-12 text-gray-500">Loading...</div>
  if (error) return <div className="text-center py-12 text-red-500">Error: {error.message}</div>
  if (!data) return null

  const { application: app, resources } = data

  const handleDeploy = (e: React.FormEvent) => {
    e.preventDefault()
    deployMutation.mutate(
      {
        gitBranch,
        gitCommit: gitCommit || undefined,
        planId: selectedPlanId || undefined,
      },
      {
        onSuccess: (deployment) => {
          setStreamingDeployId(deployment.id)
        },
      },
    )
  }

  // When stream completes, refresh deployment history
  useEffect(() => {
    if (streamComplete && name) {
      queryClient.invalidateQueries({ queryKey: ['deployments', name] })
    }
  }, [streamComplete, name, queryClient])

  const handleDeployPlan = (planId: string) => {
    setSelectedPlanId(planId)
    setActiveTab('deploy')
  }

  const togglePlanExpanded = (planId: string) => {
    setExpandedPlanIds((prev) => {
      const next = new Set(prev)
      if (next.has(planId)) {
        next.delete(planId)
      } else {
        next.add(planId)
      }
      return next
    })
  }

  const selectedPlan = plans?.find((p) => p.id === selectedPlanId)

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
    <div className="space-y-6">
      {/* Header — persistent across all tabs */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">{app.name}</h1>
          {app.description && <p className="text-gray-500 mt-1">{app.description}</p>}
          {app.source_path && (
            <p className="text-sm text-gray-400 mt-1 truncate" title={app.source_path}>
              Source: {app.source_path}
            </p>
          )}
          <div className="flex flex-wrap gap-2 mt-2">
            <span className="text-xs font-medium px-2.5 py-0.5 rounded-full bg-indigo-50 text-indigo-700">
              {app.provider.toUpperCase()}
            </span>
            <span className="text-xs font-medium px-2.5 py-0.5 rounded-full bg-gray-100 text-gray-700">
              {app.status}
            </span>
            {app.compliance_frameworks?.map((fw) => (
              <span key={fw} className="text-xs font-medium px-2.5 py-0.5 rounded-full bg-green-50 text-green-700">
                {fw}
              </span>
            ))}
          </div>
        </div>
        <button
          onClick={handleDelete}
          className="text-sm text-red-500 hover:text-red-700 border border-red-200 px-3 py-1.5 rounded-lg hover:bg-red-50 transition-colors"
        >
          Delete
        </button>
      </div>

      {/* Tab Bar */}
      <div className="border-b border-gray-200">
        <nav className="-mb-px flex gap-6" aria-label="Tabs">
          {tabs.map((tab) => (
            <button
              key={tab.key}
              onClick={() => setActiveTab(tab.key)}
              className={`py-3 text-sm font-medium border-b-2 transition-colors ${
                activeTab === tab.key
                  ? 'border-indigo-500 text-indigo-600'
                  : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
              }`}
            >
              {tab.label}
            </button>
          ))}
        </nav>
      </div>

      {/* Tab Content */}

      {/* ===== PLAN TAB ===== */}
      {activeTab === 'plan' && (
        <div className="space-y-8">
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

          {/* Infrastructure Plans */}
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

            {/* Expandable Plan List */}
            {plans && plans.length > 0 ? (
              <div className="space-y-2">
                {plans.map((plan) => {
                  const isExpanded = expandedPlanIds.has(plan.id)
                  return (
                    <div key={plan.id} className="border border-gray-200 rounded-lg overflow-hidden bg-white">
                      {/* Collapsed summary row */}
                      <button
                        onClick={() => togglePlanExpanded(plan.id)}
                        className="w-full flex items-center justify-between px-4 py-3 hover:bg-gray-50 transition-colors text-left"
                      >
                        <div className="flex items-center gap-3 min-w-0">
                          <svg
                            className={`w-4 h-4 text-gray-400 shrink-0 transition-transform duration-200 ${isExpanded ? 'rotate-90' : ''}`}
                            fill="none"
                            stroke="currentColor"
                            viewBox="0 0 24 24"
                          >
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                          </svg>
                          <span className={`text-xs font-medium px-2 py-0.5 rounded-full ${
                            plan.plan_type === 'hosting'
                              ? 'bg-indigo-50 text-indigo-700'
                              : 'bg-purple-50 text-purple-700'
                          }`}>
                            {plan.plan_type}
                          </span>
                          <span className="text-sm text-gray-900">
                            {plan.resources?.length ?? 0} resource{(plan.resources?.length ?? 0) !== 1 ? 's' : ''}
                          </span>
                          <span className="text-sm text-gray-500">
                            {plan.estimated_cost
                              ? `$${plan.estimated_cost.monthly_cost_usd.toFixed(2)}/mo`
                              : ''}
                          </span>
                        </div>
                        <div className="flex items-center gap-3 shrink-0">
                          <span className="text-xs text-gray-400">
                            {new Date(plan.created_at).toLocaleDateString()}
                          </span>
                          <span
                            role="button"
                            onClick={(e) => {
                              e.stopPropagation()
                              handleDeployPlan(plan.id)
                            }}
                            className="text-xs font-medium text-green-700 bg-green-50 px-3 py-1 rounded-full hover:bg-green-100 transition-colors"
                          >
                            Deploy
                          </span>
                        </div>
                      </button>

                      {/* Expanded PlanViewer card */}
                      {isExpanded && (
                        <div className="border-t border-gray-200">
                          <PlanViewer plan={plan} />
                        </div>
                      )}
                    </div>
                  )
                })}
              </div>
            ) : (
              !hostingPlan.isPending && (
                <p className="text-sm text-gray-400 italic">No plans generated yet.</p>
              )
            )}
          </section>

          {/* Compliance Frameworks */}
          <section>
            <h2 className="text-lg font-semibold text-gray-900 mb-4">Compliance Frameworks</h2>
            {app.compliance_frameworks && app.compliance_frameworks.length > 0 ? (
              <div className="space-y-3">
                {app.compliance_frameworks.map((fwId) => {
                  const detail = frameworks?.find((f) => f.id === fwId)
                  return (
                    <div key={fwId} className="p-4 border border-gray-200 rounded-lg bg-white">
                      <div className="flex items-center gap-2 mb-1">
                        <span className="text-xs font-medium px-2.5 py-0.5 rounded-full bg-green-50 text-green-700">
                          {fwId}
                        </span>
                        {detail?.version && (
                          <span className="text-xs text-gray-400">v{detail.version}</span>
                        )}
                      </div>
                      {detail && (
                        <>
                          <p className="text-sm font-medium text-gray-900">{detail.name}</p>
                          <p className="text-sm text-gray-600 mt-0.5">{detail.description}</p>
                        </>
                      )}
                    </div>
                  )
                })}
              </div>
            ) : (
              <p className="text-sm text-gray-400 italic">
                No compliance frameworks applied to this application.
              </p>
            )}
          </section>
        </div>
      )}

      {/* ===== DEPLOY TAB ===== */}
      {activeTab === 'deploy' && (
        <div className="space-y-8">
          {/* Plan Selector */}
          <section>
            <h2 className="text-lg font-semibold text-gray-900 mb-4">Infrastructure Plan</h2>
            {selectedPlan ? (
              <div className="flex items-center justify-between p-4 border border-indigo-200 bg-indigo-50 rounded-lg">
                <div className="flex items-center gap-3">
                  <span className={`text-xs font-medium px-2 py-1 rounded-full ${
                    selectedPlan.plan_type === 'hosting'
                      ? 'bg-indigo-100 text-indigo-700'
                      : 'bg-purple-100 text-purple-700'
                  }`}>
                    {selectedPlan.plan_type}
                  </span>
                  <span className="text-sm text-gray-700">
                    {selectedPlan.resources?.length ?? 0} resources
                  </span>
                  {selectedPlan.estimated_cost && (
                    <span className="text-sm text-gray-500">
                      ${selectedPlan.estimated_cost.monthly_cost_usd.toFixed(2)}/mo
                    </span>
                  )}
                  <span className="text-xs text-gray-400">
                    {new Date(selectedPlan.created_at).toLocaleDateString()}
                  </span>
                </div>
                <button
                  onClick={() => setSelectedPlanId(null)}
                  className="text-xs text-gray-500 hover:text-gray-700 underline"
                >
                  Clear
                </button>
              </div>
            ) : (
              <div className="p-4 border border-gray-200 rounded-lg bg-gray-50">
                {plans && plans.length > 0 ? (
                  <div>
                    <p className="text-sm text-gray-600 mb-2">Select an infrastructure plan to deploy, or deploy without one.</p>
                    <select
                      value=""
                      onChange={(e) => setSelectedPlanId(e.target.value || null)}
                      className="border border-gray-300 rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
                    >
                      <option value="">No plan (ad-hoc deploy)</option>
                      {plans.map((p) => (
                        <option key={p.id} value={p.id}>
                          {p.plan_type} — {p.resources?.length ?? 0} resources
                          {p.estimated_cost ? ` — $${p.estimated_cost.monthly_cost_usd.toFixed(2)}/mo` : ''}
                          {' — '}{new Date(p.created_at).toLocaleDateString()}
                        </option>
                      ))}
                    </select>
                  </div>
                ) : (
                  <p className="text-sm text-gray-400 italic">
                    No plans available. Generate a plan in the Plan tab first, or deploy without one.
                  </p>
                )}
              </div>
            )}
          </section>

          {/* Deploy Form */}
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

          {/* Streaming Deploy Log */}
          {streamingDeployId && (
            <section>
              <h2 className="text-lg font-semibold text-gray-900 mb-4">Deployment Output</h2>
              <DeployLog
                events={streamEvents}
                isStreaming={isStreaming}
                isComplete={streamComplete}
                finalStatus={finalStatus}
              />
            </section>
          )}

          {/* Deployment History */}
          <section>
            <h2 className="text-lg font-semibold text-gray-900 mb-4">Deployment History</h2>
            <DeploymentHistory deployments={deployments || []} plans={plans || []} />
          </section>
        </div>
      )}

      {/* ===== MONITOR TAB ===== */}
      {activeTab === 'monitor' && (
        <div className="space-y-8">
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
        </div>
      )}

      {/* ===== OPTIMIZE TAB ===== */}
      {activeTab === 'optimize' && (
        <div className="text-center py-16">
          <div className="w-16 h-16 mx-auto mb-4 rounded-full bg-gray-100 flex items-center justify-center">
            <svg className="w-8 h-8 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 7h8m0 0v8m0-8l-8 8-4-4-6 6" />
            </svg>
          </div>
          <h3 className="text-lg font-semibold text-gray-900 mb-2">Cost Insights</h3>
          <p className="text-sm text-gray-500 max-w-sm mx-auto">
            Cost analysis and optimization recommendations are coming soon.
            This will help you reduce cloud spending and right-size your infrastructure.
          </p>
        </div>
      )}
    </div>
  )
}
