const API_BASE = '/api'

// --- Types ---

export interface Application {
  id: string
  name: string
  description: string
  git_repo_url: string
  source_path: string
  provider: 'aws' | 'gcp'
  status: 'draft' | 'provisioned' | 'deployed'
  compliance_frameworks: string[]
  created_at: string
  updated_at: string
}

export interface ComplianceFrameworkInfo {
  id: string
  name: string
  version: string
  provider: string
  description: string
}

export interface Resource {
  id: string
  application_id: string
  analysis_run_id?: string
  kind: string
  name: string
  spec: Record<string, unknown>
  provider_mappings: Record<string, ProviderResource>
  created_at: string
}

export interface AnalysisRun {
  id: string
  application_id: string
  trigger: 'register' | 'reanalyze' | 'upload'
  resource_count: number
  created_at: string
}

export interface ProviderResource {
  service_name: string
  config: Record<string, unknown>
  terraform_hcl: string
}

export interface DeployTarget {
  aws_region?: string
  aws_account_id?: string
  aws_role_arn?: string
  gcp_project_id?: string
  gcp_region?: string
  gcp_folder?: string
}

export interface Deployment {
  id: string
  application_id: string
  plan_id?: string
  provider: string
  git_commit: string
  git_branch: string
  status: 'pending' | 'in_progress' | 'awaiting_approval' | 'succeeded' | 'failed'
  terraform_plan?: string
  deploy_target?: DeployTarget
  started_at: string
  completed_at?: string
}

export interface InfrastructurePlan {
  id: string
  application_id: string
  plan_type: 'hosting' | 'migration'
  from_provider?: string
  to_provider?: string
  content: string
  resources: Resource[]
  estimated_cost?: CostEstimate
  created_at: string
}

export interface CostEstimate {
  monthly_cost_usd: number
  breakdown: Record<string, number>
}

export interface GraphNode {
  id: string
  label: string
  kind: 'internet' | 'compute' | 'database' | 'cache' | 'queue' | 'storage' | 'cdn' | 'network' | 'secrets' | 'policy'
  service: string
}

export interface GraphEdge {
  id: string
  source: string
  target: string
  label: string
}

export interface InfraGraph {
  id: string
  application_id: string
  nodes: GraphNode[]
  edges: GraphEdge[]
  created_at: string
}

export interface LiveResource {
  resource_type: string
  name: string
  provider: 'aws' | 'gcp'
  region: string
  status: 'active' | 'provisioning' | 'stopped' | 'error' | 'unknown'
  details: Record<string, string>
  last_checked: string
}

export interface LiveResourceResult {
  resources: LiveResource[]
  errors?: string[]
  timestamp: string
}

export interface ApplicationDetail {
  application: Application
  resources: Resource[]
  latest_deployment?: Deployment
}

export interface OnboardResult {
  application: Application
  resources: Resource[]
  plan: InfrastructurePlan
}

export interface GCPProject {
  project_id: string
  name: string
  display_name: string
  state: string
}

// --- API Client ---

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    headers: { 'Content-Type': 'application/json' },
    ...options,
  })

  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: res.statusText }))
    throw new Error(body.error || `Request failed: ${res.status}`)
  }

  if (res.status === 204) return undefined as T
  return res.json()
}

// Applications
export const listApplications = () =>
  request<Application[]>('/applications')

export const getApplication = (name: string) =>
  request<ApplicationDetail>(`/applications/${name}`)

export const registerApplication = (data: {
  name: string
  description?: string
  git_repo_url?: string
  source_path?: string
  provider: string
  compliance_frameworks?: string[]
  files?: { path: string; content: string }[]
}) => request<Application>('/applications', { method: 'POST', body: JSON.stringify(data) })

export const onboardApplication = (data: {
  name: string
  description?: string
  provider: string
  source_path?: string
  compliance_frameworks?: string[]
  files?: { path: string; content: string }[]
}) => request<OnboardResult>('/applications/onboard', {
  method: 'POST',
  body: JSON.stringify(data),
})

export const reanalyzeSource = (appName: string) =>
  request<void>(`/applications/${appName}/reanalyze`, { method: 'POST' })

export const analyzeUpload = (appName: string, files: { path: string; content: string }[]) =>
  request<void>(`/applications/${appName}/analyze-upload`, {
    method: 'POST',
    body: JSON.stringify({ files }),
  })

export const deleteApplication = (name: string) =>
  request<void>(`/applications/${name}`, { method: 'DELETE' })

// Resources
export const listResources = (appName: string) =>
  request<Resource[]>(`/applications/${appName}/resources`)

export const addResource = (appName: string, description: string) =>
  request<Resource>(`/applications/${appName}/resources`, {
    method: 'POST',
    body: JSON.stringify({ description }),
  })

export const removeResource = (resourceId: string) =>
  request<void>(`/resources/${resourceId}`, { method: 'DELETE' })

// Analysis Runs
export const listAnalysisRuns = (appName: string) =>
  request<AnalysisRun[]>(`/applications/${appName}/analysis-runs`)

export const listResourcesByRun = (runId: string) =>
  request<Resource[]>(`/analysis-runs/${runId}/resources`)

// Plans
export const generateHostingPlan = (appName: string) =>
  request<InfrastructurePlan>(`/applications/${appName}/hosting-plan`, { method: 'POST' })

export const generateMigrationPlan = (appName: string, fromProvider: string, toProvider: string) =>
  request<InfrastructurePlan>(`/applications/${appName}/migration-plan`, {
    method: 'POST',
    body: JSON.stringify({ from_provider: fromProvider, to_provider: toProvider }),
  })

export const listPlans = (appName: string) =>
  request<InfrastructurePlan[]>(`/applications/${appName}/plans`)

// Terraform HCL
export const generateTerraformHCL = (resourceId: string, provider: string) =>
  request<{ hcl: string }>(`/resources/${resourceId}/terraform`, {
    method: 'POST',
    body: JSON.stringify({ provider }),
  })

// Deployments
export const deploy = (appName: string, gitBranch: string, gitCommit?: string, planId?: string, deployTarget?: DeployTarget) =>
  request<Deployment>(`/applications/${appName}/deploy`, {
    method: 'POST',
    body: JSON.stringify({
      git_branch: gitBranch,
      git_commit: gitCommit || '',
      ...(planId ? { plan_id: planId } : {}),
      ...(deployTarget ? { deploy_target: deployTarget } : {}),
    }),
  })

export const validateDeployTarget = (appName: string, provider: string, target: DeployTarget) =>
  request<{ valid: boolean; message: string }>(`/applications/${appName}/validate-target`, {
    method: 'POST',
    body: JSON.stringify({ provider, deploy_target: target }),
  })

export const listDeployments = (appName: string) =>
  request<Deployment[]>(`/applications/${appName}/deployments`)

export const getLatestDeployment = (appName: string) =>
  request<Deployment>(`/applications/${appName}/deployments/latest`)

export const getDeploymentStatus = (deploymentId: string) =>
  request<Deployment>(`/deployments/${deploymentId}`)

// Graphs
export const generateGraph = (appName: string) =>
  request<InfraGraph>(`/applications/${appName}/graph`, { method: 'POST' })

export const getLatestGraph = (appName: string) =>
  request<InfraGraph>(`/applications/${appName}/graph`)

// Live Resources
export const getLiveResources = (appName: string) =>
  request<LiveResourceResult>(`/applications/${appName}/live-resources`, { method: 'POST' })

// Compliance Frameworks
export const listComplianceFrameworks = (provider?: string) =>
  request<ComplianceFrameworkInfo[]>(`/compliance/frameworks${provider ? `?provider=${provider}` : ''}`)

// GCP Projects
export const listGCPProjects = () =>
  request<GCPProject[]>('/gcp/projects')

// GCP Credentials
export interface GCPCredentialStatus {
  configured: boolean
  project_id?: string
  client_email?: string
  uploaded_at?: string
}

export const getGCPCredentialStatus = () =>
  request<GCPCredentialStatus>('/gcp/credentials')

export const uploadGCPCredentials = async (file: File): Promise<GCPCredentialStatus> => {
  const formData = new FormData()
  formData.append('file', file)
  const res = await fetch(`${API_BASE}/gcp/credentials`, {
    method: 'POST',
    body: formData,
  })
  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: res.statusText }))
    throw new Error(body.error || `Upload failed: ${res.status}`)
  }
  return res.json()
}

export const deleteGCPCredentials = () =>
  request<void>('/gcp/credentials', { method: 'DELETE' })

// Deployment Streaming
export interface DeploymentEvent {
  step: 'initializing' | 'generating_terraform' | 'awaiting_approval' | 'validating_credentials' | 'validating' | 'applying' | 'complete' | 'failed'
  message: string
  timestamp: string
  status: Deployment['status']
  detail?: string
}

// Per-resource HCL detail (parsed from DeploymentEvent.detail JSON during generating_terraform step)
export interface ResourceHCLDetail {
  resource_id: string
  resource_name: string
  resource_kind: string
  service_name: string
  hcl: string
}

export const getDeploymentStreamUrl = (deploymentId: string) =>
  `${API_BASE}/deployments/${deploymentId}/stream`

export const getDeploymentEventsStreamUrl = (deploymentId: string) =>
  `${API_BASE}/deployments/${deploymentId}/events/stream`

export const getDeploymentApproveStreamUrl = (deploymentId: string) =>
  `${API_BASE}/deployments/${deploymentId}/approve`

export const getDeploymentEvents = (deploymentId: string) =>
  request<DeploymentEvent[]>(`/deployments/${deploymentId}/events`)

export const rejectDeployment = (deploymentId: string) =>
  request<{ status: string }>(`/deployments/${deploymentId}/reject`, { method: 'POST' })
