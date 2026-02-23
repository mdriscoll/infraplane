import { useState, useEffect } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import DeploymentHistory from './DeploymentHistory'
import DeployLog from './DeployLog'
import TerraformReview from './TerraformReview'
import GCPCredentialUpload from './GCPCredentialUpload'
import Spinner from './Spinner'
import {
  useDeploy,
  useDeploymentStream,
  useDeploymentReconnect,
  useDeploymentApprove,
  useRejectDeployment,
  useValidateDeployTarget,
  useGCPProjects,
  useGCPCredentialStatus,
} from '../hooks/useApi'
import type { Deployment, DeployTarget, InfrastructurePlan } from '../api/client'

type WizardStep = 'target' | 'plan' | 'review'

const STEP_LABELS: { key: WizardStep; label: string; number: number }[] = [
  { key: 'target', label: 'Target', number: 1 },
  { key: 'plan', label: 'Plan', number: 2 },
  { key: 'review', label: 'Review & Deploy', number: 3 },
]

interface DeployWizardProps {
  appName: string
  appProvider: 'aws' | 'gcp'
  plans: InfrastructurePlan[] | undefined
  deployments: Deployment[] | undefined
  selectedPlanId: string | null
  onSelectedPlanIdChange: (id: string | null) => void
}

export default function DeployWizard({
  appName,
  appProvider,
  plans,
  deployments,
  selectedPlanId,
  onSelectedPlanIdChange,
}: DeployWizardProps) {
  const queryClient = useQueryClient()

  // Wizard step
  const [wizardStep, setWizardStep] = useState<WizardStep>('target')
  const [completedSteps, setCompletedSteps] = useState<Set<WizardStep>>(new Set())

  // Target state
  const [deployProvider, setDeployProvider] = useState<'aws' | 'gcp'>(appProvider)
  const [deployTarget, setDeployTarget] = useState<DeployTarget>({})
  const [targetValid, setTargetValid] = useState<boolean | null>(null)
  const [targetMessage, setTargetMessage] = useState('')

  // Plan generation state
  const [gitBranch, setGitBranch] = useState('main')
  const [gitCommit, setGitCommit] = useState('')
  const [streamingDeployId, setStreamingDeployId] = useState<string | null>(null)

  // Hooks
  const deployMutation = useDeploy(appName)
  const validateTarget = useValidateDeployTarget(appName)
  const { data: gcpCredStatus } = useGCPCredentialStatus()
  const { data: gcpProjects, isLoading: gcpProjectsLoading } = useGCPProjects()
  const {
    events: streamEvents,
    isStreaming,
    isComplete: streamComplete,
    isAwaitingApproval,
    resourceHCLs,
    finalStatus,
  } = useDeploymentStream(streamingDeployId)

  // Reconnection hooks for deployment history
  const [selectedDeployment, setSelectedDeployment] = useState<Deployment | null>(null)
  const {
    events: reconnectEvents,
    isStreaming: reconnectStreaming,
    isComplete: reconnectComplete,
    isAwaitingApproval: reconnectAwaitingApproval,
    resourceHCLs: reconnectResourceHCLs,
    finalStatus: reconnectFinalStatus,
    reset: reconnectReset,
  } = useDeploymentReconnect(selectedDeployment?.id ?? null)

  const approvalStream = useDeploymentApprove()
  const rejectMutation = useRejectDeployment()

  // Auto-populate GCP project from credential status
  useEffect(() => {
    if (deployProvider === 'gcp' && gcpCredStatus?.configured && gcpCredStatus.project_id && !deployTarget.gcp_project_id) {
      setDeployTarget((t) => ({ ...t, gcp_project_id: gcpCredStatus.project_id }))
    }
  }, [deployProvider, gcpCredStatus, deployTarget.gcp_project_id])

  // Auto-advance to review when generation completes
  useEffect(() => {
    if (isAwaitingApproval && wizardStep === 'plan') {
      setCompletedSteps((prev) => new Set([...prev, 'plan']))
      setWizardStep('review')
    }
  }, [isAwaitingApproval, wizardStep])

  // Auto-advance when selectedPlanId is set from Plan tab
  useEffect(() => {
    if (selectedPlanId && wizardStep === 'target') {
      setWizardStep('plan')
    }
  }, [selectedPlanId, wizardStep])

  // When reconnecting to awaiting_approval deployment, jump to review
  useEffect(() => {
    if (reconnectAwaitingApproval) {
      setWizardStep('review')
    }
  }, [reconnectAwaitingApproval])

  // Refresh deployment history when streams complete
  useEffect(() => {
    if (streamComplete) {
      queryClient.invalidateQueries({ queryKey: ['deployments', appName] })
    }
  }, [streamComplete, appName, queryClient])

  useEffect(() => {
    if (approvalStream.isComplete) {
      queryClient.invalidateQueries({ queryKey: ['deployments', appName] })
    }
  }, [approvalStream.isComplete, appName, queryClient])

  // Handlers
  const handleValidateTarget = () => {
    setTargetValid(null)
    setTargetMessage('')
    validateTarget.mutate(
      { provider: deployProvider, target: deployTarget },
      {
        onSuccess: (result) => {
          setTargetValid(result.valid)
          setTargetMessage(result.message)
        },
        onError: (err) => {
          setTargetValid(false)
          setTargetMessage(err.message)
        },
      },
    )
  }

  const handleNextToPlans = () => {
    setCompletedSteps((prev) => new Set([...prev, 'target']))
    setWizardStep('plan')
  }

  const handleBackToTarget = () => {
    setWizardStep('target')
  }

  const handleGeneratePlan = () => {
    const hasTarget = Object.values(deployTarget).some((v) => v)
    deployMutation.mutate(
      {
        gitBranch,
        gitCommit: gitCommit || undefined,
        planId: selectedPlanId || undefined,
        deployTarget: hasTarget ? deployTarget : undefined,
      },
      {
        onSuccess: (deployment) => {
          setSelectedDeployment(null)
          reconnectReset()
          setStreamingDeployId(deployment.id)
        },
      },
    )
  }

  const handleUseSavedPlan = (planId: string) => {
    onSelectedPlanIdChange(planId)
    // Create deployment with this plan and start stream
    const hasTarget = Object.values(deployTarget).some((v) => v)
    deployMutation.mutate(
      {
        gitBranch,
        gitCommit: gitCommit || undefined,
        planId,
        deployTarget: hasTarget ? deployTarget : undefined,
      },
      {
        onSuccess: (deployment) => {
          setSelectedDeployment(null)
          reconnectReset()
          setStreamingDeployId(deployment.id)
        },
      },
    )
  }

  const handleBackToPlans = () => {
    if (streamingDeployId && isAwaitingApproval) {
      if (!confirm('Going back will cancel the current plan. Continue?')) return
      rejectMutation.mutate(streamingDeployId, {
        onSuccess: () => queryClient.invalidateQueries({ queryKey: ['deployments', appName] }),
      })
    }
    setStreamingDeployId(null)
    onSelectedPlanIdChange(null)
    setWizardStep('plan')
  }

  const handleSelectDeployment = (deployment: Deployment) => {
    if (selectedDeployment?.id === deployment.id) {
      setSelectedDeployment(null)
      reconnectReset()
      return
    }
    setStreamingDeployId(null)
    setSelectedDeployment(deployment)
  }

  // Determine the active deploy ID and resource HCLs for review
  const activeDeployId = streamingDeployId || selectedDeployment?.id || null
  const activeResourceHCLs = streamingDeployId ? resourceHCLs : reconnectResourceHCLs
  const activeAwaitingApproval = streamingDeployId ? isAwaitingApproval : reconnectAwaitingApproval

  return (
    <div className="space-y-8">
      {/* Step Indicator */}
      <div className="flex items-center justify-center">
        {STEP_LABELS.map((step, idx) => {
          const isActive = wizardStep === step.key
          const isCompleted = completedSteps.has(step.key)
          return (
            <div key={step.key} className="flex items-center">
              {idx > 0 && (
                <div className={`w-12 h-0.5 mx-1 ${isCompleted || isActive ? 'bg-indigo-300' : 'bg-gray-200'}`} />
              )}
              <div className="flex items-center gap-2">
                <div
                  className={`w-7 h-7 rounded-full flex items-center justify-center text-xs font-medium transition-colors ${
                    isActive
                      ? 'bg-indigo-600 text-white'
                      : isCompleted
                        ? 'bg-green-500 text-white'
                        : 'bg-gray-200 text-gray-500'
                  }`}
                >
                  {isCompleted && !isActive ? '\u2713' : step.number}
                </div>
                <span
                  className={`text-sm font-medium ${
                    isActive ? 'text-indigo-600' : isCompleted ? 'text-green-600' : 'text-gray-400'
                  }`}
                >
                  {step.label}
                </span>
              </div>
            </div>
          )
        })}
      </div>

      {/* ===== STEP 1: TARGET ===== */}
      {wizardStep === 'target' && (
        <section className="space-y-6">
          <div>
            <h2 className="text-lg font-semibold text-gray-900 mb-1">Deployment Target</h2>
            <p className="text-sm text-gray-500">Choose your cloud provider and configure credentials.</p>
          </div>

          <div className="p-4 border border-gray-200 rounded-lg bg-white space-y-4">
            {/* Provider selector */}
            <div>
              <label className="block text-xs font-medium text-gray-600 mb-1">Cloud Provider</label>
              <div className="flex gap-2">
                <button
                  type="button"
                  onClick={() => {
                    setDeployProvider('aws')
                    setDeployTarget({})
                    setTargetValid(null)
                    setTargetMessage('')
                  }}
                  className={`flex-1 px-4 py-2.5 rounded-lg text-sm font-medium border transition-colors ${
                    deployProvider === 'aws'
                      ? 'border-orange-300 bg-orange-50 text-orange-800 ring-2 ring-orange-200'
                      : 'border-gray-200 bg-white text-gray-600 hover:bg-gray-50'
                  }`}
                >
                  AWS
                </button>
                <button
                  type="button"
                  onClick={() => {
                    setDeployProvider('gcp')
                    setDeployTarget({})
                    setTargetValid(null)
                    setTargetMessage('')
                  }}
                  className={`flex-1 px-4 py-2.5 rounded-lg text-sm font-medium border transition-colors ${
                    deployProvider === 'gcp'
                      ? 'border-blue-300 bg-blue-50 text-blue-800 ring-2 ring-blue-200'
                      : 'border-gray-200 bg-white text-gray-600 hover:bg-gray-50'
                  }`}
                >
                  GCP
                </button>
              </div>
            </div>

            {/* AWS target fields */}
            {deployProvider === 'aws' && (
              <div className="grid grid-cols-3 gap-3">
                <div>
                  <label className="block text-xs font-medium text-gray-600 mb-1">AWS Region</label>
                  <select
                    value={deployTarget.aws_region || ''}
                    onChange={(e) => {
                      setDeployTarget((t) => ({ ...t, aws_region: e.target.value }))
                      setTargetValid(null)
                    }}
                    className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
                  >
                    <option value="">Select region</option>
                    {['us-east-1', 'us-east-2', 'us-west-1', 'us-west-2', 'eu-west-1', 'eu-west-2', 'eu-central-1', 'ap-southeast-1', 'ap-northeast-1'].map((r) => (
                      <option key={r} value={r}>{r}</option>
                    ))}
                  </select>
                </div>
                <div>
                  <label className="block text-xs font-medium text-gray-600 mb-1">Account ID</label>
                  <input
                    type="text"
                    value={deployTarget.aws_account_id || ''}
                    onChange={(e) => {
                      setDeployTarget((t) => ({ ...t, aws_account_id: e.target.value }))
                      setTargetValid(null)
                    }}
                    className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
                    placeholder="123456789012"
                    maxLength={12}
                  />
                </div>
                <div>
                  <label className="block text-xs font-medium text-gray-600 mb-1">Role ARN (optional)</label>
                  <input
                    type="text"
                    value={deployTarget.aws_role_arn || ''}
                    onChange={(e) => {
                      setDeployTarget((t) => ({ ...t, aws_role_arn: e.target.value }))
                      setTargetValid(null)
                    }}
                    className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
                    placeholder="arn:aws:iam::123456789012:role/deploy"
                  />
                </div>
              </div>
            )}

            {/* GCP credentials + target fields */}
            {deployProvider === 'gcp' && (
              <>
                <GCPCredentialUpload
                  compact
                  onProjectDetected={(projectId) => {
                    setDeployTarget((t) => ({ ...t, gcp_project_id: projectId }))
                    setTargetValid(null)
                  }}
                />
                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <label className="block text-xs font-medium text-gray-600 mb-1">Project</label>
                    {gcpProjectsLoading ? (
                      <div className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm text-gray-400 bg-gray-50">
                        Loading projects...
                      </div>
                    ) : gcpProjects && gcpProjects.length > 0 ? (
                      <select
                        value={deployTarget.gcp_project_id || ''}
                        onChange={(e) => {
                          setDeployTarget((t) => ({ ...t, gcp_project_id: e.target.value }))
                          setTargetValid(null)
                        }}
                        className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
                      >
                        <option value="">Select project</option>
                        {gcpProjects.map((p) => (
                          <option key={p.project_id} value={p.project_id}>
                            {p.display_name || p.project_id} ({p.project_id})
                          </option>
                        ))}
                      </select>
                    ) : (
                      <input
                        type="text"
                        value={deployTarget.gcp_project_id || ''}
                        onChange={(e) => {
                          setDeployTarget((t) => ({ ...t, gcp_project_id: e.target.value }))
                          setTargetValid(null)
                        }}
                        className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
                        placeholder="my-project-123"
                      />
                    )}
                  </div>
                  <div>
                    <label className="block text-xs font-medium text-gray-600 mb-1">Region</label>
                    <select
                      value={deployTarget.gcp_region || ''}
                      onChange={(e) => {
                        setDeployTarget((t) => ({ ...t, gcp_region: e.target.value }))
                        setTargetValid(null)
                      }}
                      className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
                    >
                      <option value="">Select region</option>
                      {['us-central1', 'us-east1', 'us-east4', 'us-west1', 'us-west2', 'europe-west1', 'europe-west2', 'europe-west3', 'asia-east1', 'asia-southeast1', 'australia-southeast1'].map((r) => (
                        <option key={r} value={r}>{r}</option>
                      ))}
                    </select>
                  </div>
                </div>
              </>
            )}

            {/* Validate + Next */}
            <div className="flex items-center justify-between pt-2">
              <div className="flex items-center gap-3">
                <button
                  type="button"
                  onClick={handleValidateTarget}
                  disabled={validateTarget.isPending}
                  className="text-sm bg-gray-100 text-gray-700 px-3 py-1.5 rounded-lg hover:bg-gray-200 disabled:opacity-50 transition-colors"
                >
                  {validateTarget.isPending ? 'Validating...' : 'Validate Credentials'}
                </button>
                {targetValid === true && (
                  <span className="text-sm text-green-600 flex items-center gap-1">
                    <span>{'\u2713'}</span> {targetMessage}
                  </span>
                )}
                {targetValid === false && (
                  <span className="text-sm text-red-600 flex items-center gap-1">
                    <span>{'\u2717'}</span> {targetMessage}
                  </span>
                )}
              </div>
              <button
                type="button"
                onClick={handleNextToPlans}
                className="bg-indigo-600 text-white px-5 py-2 rounded-lg hover:bg-indigo-700 text-sm font-medium transition-colors"
              >
                Next
              </button>
            </div>
          </div>
        </section>
      )}

      {/* ===== STEP 2: PLAN ===== */}
      {wizardStep === 'plan' && (
        <section className="space-y-6">
          <div className="flex items-center justify-between">
            <div>
              <h2 className="text-lg font-semibold text-gray-900 mb-1">Deployment Plan</h2>
              <p className="text-sm text-gray-500">Use a saved plan or generate a new Terraform configuration.</p>
            </div>
            <button
              type="button"
              onClick={handleBackToTarget}
              className="text-sm text-gray-500 hover:text-gray-700 transition-colors"
            >
              &larr; Back to Target
            </button>
          </div>

          {/* Saved Plans */}
          {plans && plans.length > 0 && (
            <div>
              <h3 className="text-sm font-semibold text-gray-700 mb-3">Saved Plans</h3>
              <div className="space-y-2">
                {plans.map((plan) => {
                  const isSelected = selectedPlanId === plan.id
                  return (
                    <button
                      key={plan.id}
                      onClick={() => onSelectedPlanIdChange(isSelected ? null : plan.id)}
                      disabled={!!streamingDeployId}
                      className={`w-full flex items-center justify-between p-4 rounded-lg border text-left transition-colors disabled:opacity-50 ${
                        isSelected
                          ? 'border-indigo-300 bg-indigo-50 ring-2 ring-indigo-200'
                          : 'border-gray-200 bg-white hover:bg-gray-50'
                      }`}
                    >
                      <div className="flex items-center gap-3">
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
                        {plan.estimated_cost && (
                          <span className="text-sm text-gray-500">
                            ${plan.estimated_cost.monthly_cost_usd.toFixed(2)}/mo
                          </span>
                        )}
                      </div>
                      <span className="text-xs text-gray-400">
                        {new Date(plan.created_at).toLocaleDateString()}
                      </span>
                    </button>
                  )
                })}
              </div>
              {selectedPlanId && !streamingDeployId && (
                <div className="mt-3">
                  <button
                    type="button"
                    onClick={() => handleUseSavedPlan(selectedPlanId)}
                    disabled={deployMutation.isPending}
                    className="inline-flex items-center gap-2 bg-indigo-600 text-white px-5 py-2 rounded-lg hover:bg-indigo-700 disabled:opacity-50 text-sm font-medium transition-colors"
                  >
                    {deployMutation.isPending && <Spinner />}
                    {deployMutation.isPending ? 'Starting...' : 'Continue with Plan'}
                  </button>
                </div>
              )}
            </div>
          )}

          {/* Divider */}
          {plans && plans.length > 0 && (
            <div className="relative">
              <div className="absolute inset-0 flex items-center">
                <div className="w-full border-t border-gray-200" />
              </div>
              <div className="relative flex justify-center">
                <span className="bg-gray-50 px-3 text-sm text-gray-500">or</span>
              </div>
            </div>
          )}

          {/* Generate New Plan */}
          <div>
            <h3 className="text-sm font-semibold text-gray-700 mb-3">Generate New Plan</h3>
            <div className="p-4 border border-gray-200 rounded-lg bg-white space-y-3">
              <div className="flex gap-2">
                <div className="flex-1">
                  <label className="block text-xs font-medium text-gray-600 mb-1">Branch</label>
                  <input
                    type="text"
                    value={gitBranch}
                    onChange={(e) => setGitBranch(e.target.value)}
                    className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
                    placeholder="main"
                  />
                </div>
                <div className="flex-1">
                  <label className="block text-xs font-medium text-gray-600 mb-1">Commit SHA (optional)</label>
                  <input
                    type="text"
                    value={gitCommit}
                    onChange={(e) => setGitCommit(e.target.value)}
                    className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
                    placeholder="abc123"
                  />
                </div>
              </div>
              <button
                type="button"
                onClick={handleGeneratePlan}
                disabled={deployMutation.isPending || isStreaming || !gitBranch}
                className="inline-flex items-center gap-2 bg-indigo-600 text-white px-5 py-2 rounded-lg hover:bg-indigo-700 disabled:opacity-50 text-sm font-medium transition-colors"
              >
                {(deployMutation.isPending || isStreaming) && <Spinner />}
                {deployMutation.isPending ? 'Starting...' : isStreaming ? 'Generating...' : 'Generate Plan'}
              </button>
              {deployMutation.isError && (
                <p className="text-sm text-red-500">{deployMutation.error.message}</p>
              )}
            </div>
          </div>

          {/* Generation log (inline) */}
          {streamingDeployId && !isAwaitingApproval && (
            <div>
              <h3 className="text-sm font-semibold text-gray-700 mb-2">Generation Progress</h3>
              <DeployLog
                events={streamEvents}
                isStreaming={isStreaming}
                isComplete={streamComplete && !isAwaitingApproval}
                finalStatus={finalStatus}
              />
            </div>
          )}
        </section>
      )}

      {/* ===== STEP 3: REVIEW & DEPLOY ===== */}
      {wizardStep === 'review' && (
        <section className="space-y-6">
          <div className="flex items-center justify-between">
            <div>
              <h2 className="text-lg font-semibold text-gray-900 mb-1">Review & Deploy</h2>
              <p className="text-sm text-gray-500">Inspect the generated Terraform, then deploy when ready.</p>
            </div>
            {!approvalStream.isStreaming && !approvalStream.isComplete && (
              <button
                type="button"
                onClick={handleBackToPlans}
                className="text-sm text-gray-500 hover:text-gray-700 transition-colors"
              >
                &larr; Back to Plan
              </button>
            )}
          </div>

          {/* Generation log (compact) */}
          {streamingDeployId && streamEvents.length > 0 && (
            <DeployLog
              events={streamEvents}
              isStreaming={isStreaming}
              isComplete={streamComplete && !isAwaitingApproval}
              finalStatus={approvalStream.isComplete ? approvalStream.finalStatus : finalStatus}
            />
          )}

          {/* Reconnected deployment log */}
          {selectedDeployment && !streamingDeployId && reconnectEvents.length > 0 && (
            <DeployLog
              events={reconnectEvents}
              isStreaming={reconnectStreaming}
              isComplete={reconnectComplete && !reconnectAwaitingApproval}
              finalStatus={approvalStream.isComplete ? approvalStream.finalStatus : reconnectFinalStatus}
            />
          )}

          {/* Terraform Review UI */}
          {activeAwaitingApproval && !approvalStream.isStreaming && !approvalStream.isComplete && (
            <TerraformReview
              resourceHCLs={activeResourceHCLs}
              approveLabel="Deploy"
              rejectLabel="Cancel"
              onApprove={() => {
                if (activeDeployId) approvalStream.approve(activeDeployId)
              }}
              onReject={() => {
                if (activeDeployId) {
                  rejectMutation.mutate(activeDeployId, {
                    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['deployments', appName] }),
                  })
                }
              }}
              isApproving={approvalStream.isStreaming}
            />
          )}

          {/* Apply-phase log */}
          {approvalStream.events.length > 0 && (
            <DeployLog
              events={approvalStream.events}
              isStreaming={approvalStream.isStreaming}
              isComplete={approvalStream.isComplete}
              finalStatus={approvalStream.finalStatus}
            />
          )}
        </section>
      )}

      {/* Deployment History — always visible */}
      <section>
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold text-gray-900">Deployment History</h2>
          {selectedDeployment && (
            <button
              onClick={() => { setSelectedDeployment(null); reconnectReset() }}
              className="text-sm text-gray-500 hover:text-gray-700 transition-colors"
            >
              Close log
            </button>
          )}
        </div>
        <DeploymentHistory
          deployments={deployments || []}
          plans={plans || []}
          selectedDeploymentId={selectedDeployment?.id}
          onSelectDeployment={handleSelectDeployment}
        />
      </section>
    </div>
  )
}
