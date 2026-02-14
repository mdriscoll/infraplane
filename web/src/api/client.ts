const API_BASE = '/api'

// --- Types ---

export interface Application {
  id: string
  name: string
  description: string
  git_repo_url: string
  provider: 'aws' | 'gcp'
  status: 'draft' | 'provisioned' | 'deployed'
  created_at: string
  updated_at: string
}

export interface Resource {
  id: string
  application_id: string
  kind: string
  name: string
  spec: Record<string, unknown>
  provider_mappings: Record<string, ProviderResource>
  created_at: string
}

export interface ProviderResource {
  service_name: string
  config: Record<string, unknown>
  terraform_hcl: string
}

export interface Deployment {
  id: string
  application_id: string
  provider: string
  git_commit: string
  git_branch: string
  status: 'pending' | 'in_progress' | 'succeeded' | 'failed'
  terraform_plan?: string
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

export interface ApplicationDetail {
  application: Application
  resources: Resource[]
  latest_deployment?: Deployment
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
  provider: string
}) => request<Application>('/applications', { method: 'POST', body: JSON.stringify(data) })

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

// Deployments
export const deploy = (appName: string, gitBranch: string, gitCommit?: string) =>
  request<Deployment>(`/applications/${appName}/deploy`, {
    method: 'POST',
    body: JSON.stringify({ git_branch: gitBranch, git_commit: gitCommit || '' }),
  })

export const listDeployments = (appName: string) =>
  request<Deployment[]>(`/applications/${appName}/deployments`)

export const getLatestDeployment = (appName: string) =>
  request<Deployment>(`/applications/${appName}/deployments/latest`)

export const getDeploymentStatus = (deploymentId: string) =>
  request<Deployment>(`/deployments/${deploymentId}`)
