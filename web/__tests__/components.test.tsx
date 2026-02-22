import { render, screen, fireEvent } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import AppCard from '../src/components/AppCard'
import ResourceList from '../src/components/ResourceList'
import DeploymentHistory from '../src/components/DeploymentHistory'
import DeployLog from '../src/components/DeployLog'
import PlanViewer from '../src/components/PlanViewer'
import OnboardWizard from '../src/pages/OnboardWizard'
import ApplicationList from '../src/pages/ApplicationList'
import ApplicationDetail from '../src/pages/ApplicationDetail'
import type { Application, Resource, Deployment, InfrastructurePlan, DeploymentEvent } from '../src/api/client'

const sampleApp: Application = {
  id: '123',
  name: 'my-api',
  description: 'A test API',
  git_repo_url: '',
  source_path: '/Users/dev/my-api',
  provider: 'aws',
  status: 'deployed',
  compliance_frameworks: [],
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
}

describe('AppCard', () => {
  const app = sampleApp

  it('renders app name and description', () => {
    render(
      <MemoryRouter>
        <AppCard app={app} />
      </MemoryRouter>
    )
    expect(screen.getByText('my-api')).toBeInTheDocument()
    expect(screen.getByText('A test API')).toBeInTheDocument()
  })

  it('shows status badge', () => {
    render(
      <MemoryRouter>
        <AppCard app={app} />
      </MemoryRouter>
    )
    expect(screen.getByText('deployed')).toBeInTheDocument()
  })

  it('shows provider label', () => {
    render(
      <MemoryRouter>
        <AppCard app={app} />
      </MemoryRouter>
    )
    expect(screen.getByText('AWS')).toBeInTheDocument()
  })

  it('shows source path', () => {
    render(
      <MemoryRouter>
        <AppCard app={app} />
      </MemoryRouter>
    )
    expect(screen.getByText('/Users/dev/my-api')).toBeInTheDocument()
  })

  it('hides source path when empty', () => {
    const appNoSource = { ...app, source_path: '' }
    render(
      <MemoryRouter>
        <AppCard app={appNoSource} />
      </MemoryRouter>
    )
    expect(screen.queryByText('/Users/dev/my-api')).not.toBeInTheDocument()
  })
})

// Mock react-router-dom partially for ApplicationDetail tests
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom')
  return {
    ...actual,
    useParams: vi.fn(() => ({ name: 'my-api' })),
    useNavigate: vi.fn(() => vi.fn()),
  }
})

// Mock the API module so ApplicationList and ApplicationDetail render with test data
vi.mock('../src/hooks/useApi', async () => {
  const actual = await vi.importActual('../src/hooks/useApi')
  const mockMutation = () => ({ mutate: vi.fn(), mutateAsync: vi.fn(), isPending: false, isError: false, isSuccess: false, error: null, data: null })
  return {
    ...actual,
    useApplications: vi.fn(),
    useApplication: vi.fn(() => ({ data: null, isLoading: false, error: null })),
    useRegisterApplication: vi.fn(() => mockMutation()),
    useDeleteApplication: vi.fn(() => mockMutation()),
    useDeployments: vi.fn(() => ({ data: [] })),
    usePlans: vi.fn(() => ({ data: [] })),
    useRemoveResource: vi.fn(() => mockMutation()),
    useDeploy: vi.fn(() => mockMutation()),
    useGenerateHostingPlan: vi.fn(() => mockMutation()),
    useReanalyzeSource: vi.fn(() => mockMutation()),
    useAnalyzeUpload: vi.fn(() => mockMutation()),
    useDiscoverLiveResources: vi.fn(() => mockMutation()),
    useGraph: vi.fn(() => ({ data: null })),
    useGenerateGraph: vi.fn(() => mockMutation()),
    useComplianceFrameworks: vi.fn(() => ({ data: [] })),
    useDeploymentStream: vi.fn(() => ({ events: [], isStreaming: false, isComplete: false, finalStatus: null, reset: vi.fn() })),
  }
})

import {
  useApplications,
  useDeleteApplication,
  useApplication,
  useDeployments,
  usePlans,
  useComplianceFrameworks,
} from '../src/hooks/useApi'

const mockApps: Application[] = [
  sampleApp,
  {
    id: '456',
    name: 'frontend-app',
    description: 'React frontend',
    git_repo_url: '',
    source_path: '',
    provider: 'gcp',
    status: 'provisioned',
    compliance_frameworks: [],
    created_at: '2024-02-15T00:00:00Z',
    updated_at: '2024-02-15T00:00:00Z',
  },
  {
    id: '789',
    name: 'worker-service',
    description: '',
    git_repo_url: '',
    source_path: '',
    provider: 'aws',
    status: 'draft',
    compliance_frameworks: [],
    created_at: '2024-03-01T00:00:00Z',
    updated_at: '2024-03-01T00:00:00Z',
  },
]

describe('ApplicationList', () => {
  beforeEach(() => {
    vi.mocked(useApplications).mockReturnValue({
      data: mockApps,
      isLoading: false,
      error: null,
    } as any)
    vi.mocked(useDeleteApplication).mockReturnValue({
      mutateAsync: vi.fn().mockResolvedValue(undefined),
      isPending: false,
      isError: false,
    } as any)
  })

  it('renders a table with all applications', () => {
    renderWithProviders(<ApplicationList />)
    expect(screen.getByText('my-api')).toBeInTheDocument()
    expect(screen.getByText('frontend-app')).toBeInTheDocument()
    expect(screen.getByText('worker-service')).toBeInTheDocument()
  })

  it('shows table column headers', () => {
    renderWithProviders(<ApplicationList />)
    expect(screen.getByText('Name')).toBeInTheDocument()
    expect(screen.getByText('Status')).toBeInTheDocument()
    expect(screen.getByText('Provider')).toBeInTheDocument()
  })

  it('shows status badges and provider labels', () => {
    renderWithProviders(<ApplicationList />)
    expect(screen.getByText('deployed')).toBeInTheDocument()
    expect(screen.getByText('provisioned')).toBeInTheDocument()
    expect(screen.getByText('draft')).toBeInTheDocument()
    expect(screen.getAllByText('AWS')).toHaveLength(2)
    expect(screen.getByText('GCP')).toBeInTheDocument()
  })

  it('renders checkboxes for each application and a select-all', () => {
    renderWithProviders(<ApplicationList />)
    const checkboxes = screen.getAllByRole('checkbox')
    // 1 select-all + 3 row checkboxes
    expect(checkboxes).toHaveLength(4)
  })

  it('clicking a row checkbox selects it and shows bulk actions', () => {
    renderWithProviders(<ApplicationList />)
    const checkbox = screen.getByLabelText('Select my-api')
    fireEvent.click(checkbox)
    expect(screen.getByText('1 application selected')).toBeInTheDocument()
    expect(screen.getByText('Delete selected')).toBeInTheDocument()
  })

  it('selecting multiple apps shows correct count', () => {
    renderWithProviders(<ApplicationList />)
    fireEvent.click(screen.getByLabelText('Select my-api'))
    fireEvent.click(screen.getByLabelText('Select frontend-app'))
    expect(screen.getByText('2 applications selected')).toBeInTheDocument()
  })

  it('select all checkbox selects all applications', () => {
    renderWithProviders(<ApplicationList />)
    const selectAll = screen.getByLabelText('Select all applications')
    fireEvent.click(selectAll)
    expect(screen.getByText('3 applications selected')).toBeInTheDocument()
  })

  it('clear selection button deselects all', () => {
    renderWithProviders(<ApplicationList />)
    fireEvent.click(screen.getByLabelText('Select all applications'))
    expect(screen.getByText('3 applications selected')).toBeInTheDocument()
    fireEvent.click(screen.getByText('Clear selection'))
    expect(screen.queryByText('applications selected')).not.toBeInTheDocument()
  })

  it('delete selected opens confirmation dialog', () => {
    renderWithProviders(<ApplicationList />)
    fireEvent.click(screen.getByLabelText('Select my-api'))
    fireEvent.click(screen.getByText('Delete selected'))
    expect(screen.getByText('Delete applications')).toBeInTheDocument()
    expect(screen.getByText('This action cannot be undone.')).toBeInTheDocument()
  })

  it('shows empty state when no applications', () => {
    vi.mocked(useApplications).mockReturnValue({
      data: [],
      isLoading: false,
      error: null,
    } as any)
    renderWithProviders(<ApplicationList />)
    expect(screen.getByText('No applications registered yet')).toBeInTheDocument()
    expect(screen.getByText('Start Onboarding Wizard')).toBeInTheDocument()
  })
})

describe('ResourceList', () => {
  it('shows empty message when no resources', () => {
    render(<ResourceList resources={[]} />)
    expect(screen.getByText('No resources detected yet.')).toBeInTheDocument()
  })

  it('renders resources in a table', () => {
    const resources: Resource[] = [
      {
        id: '1',
        application_id: 'app-1',
        kind: 'database',
        name: 'user-db',
        spec: {},
        provider_mappings: {
          aws: { service_name: 'RDS', config: {}, terraform_hcl: '' },
        },
        created_at: '2024-01-01T00:00:00Z',
      },
    ]
    render(<ResourceList resources={resources} />)
    expect(screen.getByText('user-db')).toBeInTheDocument()
    // Table uses kind badges — database shows as "DB"
    expect(screen.getByText('DB')).toBeInTheDocument()
    // Provider mappings shown separately
    expect(screen.getByText('AWS')).toBeInTheDocument()
    expect(screen.getByText('RDS')).toBeInTheDocument()
  })
})

describe('DeploymentHistory', () => {
  it('shows empty message when no deployments', () => {
    render(<DeploymentHistory deployments={[]} />)
    expect(screen.getByText('No deployments yet.')).toBeInTheDocument()
  })

  it('renders deployment rows', () => {
    const deployments: Deployment[] = [
      {
        id: '1',
        application_id: 'app-1',
        provider: 'aws',
        git_commit: 'abc123def456',
        git_branch: 'main',
        status: 'succeeded',
        started_at: '2024-01-01T00:00:00Z',
      },
    ]
    render(<DeploymentHistory deployments={deployments} />)
    expect(screen.getByText('succeeded')).toBeInTheDocument()
    expect(screen.getByText('main')).toBeInTheDocument()
    expect(screen.getByText('abc123d')).toBeInTheDocument()
  })

  it('shows Plan column header', () => {
    const deployments: Deployment[] = [
      {
        id: '1',
        application_id: 'app-1',
        provider: 'aws',
        git_commit: 'abc123',
        git_branch: 'main',
        status: 'pending',
        started_at: '2024-01-01T00:00:00Z',
      },
    ]
    render(<DeploymentHistory deployments={deployments} />)
    expect(screen.getByText('Plan')).toBeInTheDocument()
  })

  it('shows plan badge when deployment has plan_id', () => {
    const plans: InfrastructurePlan[] = [
      {
        id: 'plan-1',
        application_id: 'app-1',
        plan_type: 'hosting',
        content: 'Deploy on ECS',
        resources: [],
        created_at: '2024-01-01T00:00:00Z',
      },
    ]
    const deployments: Deployment[] = [
      {
        id: '1',
        application_id: 'app-1',
        plan_id: 'plan-1',
        provider: 'aws',
        git_commit: 'abc123',
        git_branch: 'main',
        status: 'succeeded',
        started_at: '2024-01-01T00:00:00Z',
      },
    ]
    render(<DeploymentHistory deployments={deployments} plans={plans} />)
    expect(screen.getByText('hosting')).toBeInTheDocument()
  })

  it('shows dash when deployment has no plan_id', () => {
    const deployments: Deployment[] = [
      {
        id: '1',
        application_id: 'app-1',
        provider: 'aws',
        git_commit: 'abc123',
        git_branch: 'main',
        status: 'pending',
        started_at: '2024-01-01T00:00:00Z',
      },
    ]
    render(<DeploymentHistory deployments={deployments} />)
    // The plan column should show a dash
    const dashes = screen.getAllByText('-')
    expect(dashes.length).toBeGreaterThan(0)
  })
})

describe('PlanViewer', () => {
  it('renders hosting plan', () => {
    const plan: InfrastructurePlan = {
      id: '1',
      application_id: 'app-1',
      plan_type: 'hosting',
      content: 'Deploy using ECS Fargate',
      resources: [],
      estimated_cost: {
        monthly_cost_usd: 150,
        breakdown: { compute: 100, database: 50 },
      },
      created_at: '2024-01-01T00:00:00Z',
    }
    render(<PlanViewer plan={plan} />)
    expect(screen.getByText('Hosting Plan')).toBeInTheDocument()
    expect(screen.getByText('Deploy using ECS Fargate')).toBeInTheDocument()
    expect(screen.getByText('Estimated Cost: $150.00/month')).toBeInTheDocument()
  })

  it('renders migration plan with providers', () => {
    const plan: InfrastructurePlan = {
      id: '2',
      application_id: 'app-1',
      plan_type: 'migration',
      from_provider: 'aws',
      to_provider: 'gcp',
      content: 'Migrate RDS to Cloud SQL',
      resources: [],
      created_at: '2024-01-01T00:00:00Z',
    }
    render(<PlanViewer plan={plan} />)
    expect(screen.getByText('Migration Plan')).toBeInTheDocument()
    expect(screen.getByText('aws')).toBeInTheDocument()
    expect(screen.getByText('gcp')).toBeInTheDocument()
  })
})

describe('DeployLog', () => {
  it('shows waiting message when streaming with no events', () => {
    render(<DeployLog events={[]} isStreaming={true} isComplete={false} finalStatus={null} />)
    expect(screen.getByText('Waiting for output...')).toBeInTheDocument()
  })

  it('renders event messages', () => {
    const events: DeploymentEvent[] = [
      { step: 'initializing', message: 'Starting deployment...', timestamp: '2024-01-01T00:00:01Z', status: 'in_progress' },
      { step: 'generating_terraform', message: 'Generating HCL for 3 resources', timestamp: '2024-01-01T00:00:02Z', status: 'in_progress' },
    ]
    render(<DeployLog events={events} isStreaming={true} isComplete={false} finalStatus={null} />)
    expect(screen.getByText('Starting deployment...')).toBeInTheDocument()
    expect(screen.getByText('Generating HCL for 3 resources')).toBeInTheDocument()
  })

  it('shows success banner when deployment succeeds', () => {
    const events: DeploymentEvent[] = [
      { step: 'complete', message: 'Deployment succeeded', timestamp: '2024-01-01T00:00:05Z', status: 'succeeded' },
    ]
    render(<DeployLog events={events} isStreaming={false} isComplete={true} finalStatus="succeeded" />)
    expect(screen.getByText('Deployment completed successfully.')).toBeInTheDocument()
  })

  it('shows failure banner when deployment fails', () => {
    const events: DeploymentEvent[] = [
      { step: 'failed', message: 'Apply error', timestamp: '2024-01-01T00:00:05Z', status: 'failed' },
    ]
    render(<DeployLog events={events} isStreaming={false} isComplete={true} finalStatus="failed" />)
    expect(screen.getByText('Deployment failed. Check the logs above for details.')).toBeInTheDocument()
  })

  it('shows step progress with checkmarks for completed steps', () => {
    const events: DeploymentEvent[] = [
      { step: 'initializing', message: 'Init', timestamp: '2024-01-01T00:00:01Z', status: 'in_progress' },
      { step: 'generating_terraform', message: 'Gen', timestamp: '2024-01-01T00:00:02Z', status: 'in_progress' },
      { step: 'validating', message: 'Val', timestamp: '2024-01-01T00:00:03Z', status: 'in_progress' },
      { step: 'applying', message: 'Apply', timestamp: '2024-01-01T00:00:04Z', status: 'in_progress' },
      { step: 'complete', message: 'Done', timestamp: '2024-01-01T00:00:05Z', status: 'succeeded' },
    ]
    render(<DeployLog events={events} isStreaming={false} isComplete={true} finalStatus="succeeded" />)
    // All 4 steps should show checkmarks (✓) when complete
    const checkmarks = screen.getAllByText('\u2713')
    expect(checkmarks).toHaveLength(4)
    // Step labels should be present
    expect(screen.getByText('Initializing')).toBeInTheDocument()
    expect(screen.getByText('Generating Terraform')).toBeInTheDocument()
    expect(screen.getByText('Validating')).toBeInTheDocument()
    expect(screen.getByText('Applying')).toBeInTheDocument()
  })

  it('does not show waiting message when events exist', () => {
    const events: DeploymentEvent[] = [
      { step: 'initializing', message: 'Starting...', timestamp: '2024-01-01T00:00:01Z', status: 'in_progress' },
    ]
    render(<DeployLog events={events} isStreaming={true} isComplete={false} finalStatus={null} />)
    expect(screen.queryByText('Waiting for output...')).not.toBeInTheDocument()
  })

  it('does not show banner when not complete', () => {
    const events: DeploymentEvent[] = [
      { step: 'applying', message: 'Applying...', timestamp: '2024-01-01T00:00:01Z', status: 'in_progress' },
    ]
    render(<DeployLog events={events} isStreaming={true} isComplete={false} finalStatus={null} />)
    expect(screen.queryByText('Deployment completed successfully.')).not.toBeInTheDocument()
    expect(screen.queryByText(/Deployment failed/)).not.toBeInTheDocument()
  })
})

function renderWithProviders(ui: React.ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        {ui}
      </MemoryRouter>
    </QueryClientProvider>
  )
}

describe('OnboardWizard', () => {
  it('renders step 1 with provider selection', () => {
    renderWithProviders(<OnboardWizard />)
    expect(screen.getByText('Get Started with Infraplane')).toBeInTheDocument()
    expect(screen.getByText('Where do you want to host?')).toBeInTheDocument()
    expect(screen.getByText('Google Cloud')).toBeInTheDocument()
    expect(screen.getByText('Amazon Web Services')).toBeInTheDocument()
  })

  it('Next button is disabled until provider is selected', () => {
    renderWithProviders(<OnboardWizard />)
    const nextButton = screen.getByText('Next')
    expect(nextButton).toBeDisabled()
  })

  it('selecting a provider enables Next button', () => {
    renderWithProviders(<OnboardWizard />)
    const awsCard = screen.getByText('Amazon Web Services')
    fireEvent.click(awsCard)
    const nextButton = screen.getByText('Next')
    expect(nextButton).not.toBeDisabled()
  })

  it('clicking Next advances to step 2', () => {
    renderWithProviders(<OnboardWizard />)
    // Select GCP
    fireEvent.click(screen.getByText('Google Cloud'))
    // Click Next
    fireEvent.click(screen.getByText('Next'))
    // Should show step 2 content
    expect(screen.getByText('Application Name')).toBeInTheDocument()
    expect(screen.getByText('Generate My Hosting Plan')).toBeInTheDocument()
  })

  it('Back button on step 2 returns to step 1', () => {
    renderWithProviders(<OnboardWizard />)
    // Go to step 2
    fireEvent.click(screen.getByText('Google Cloud'))
    fireEvent.click(screen.getByText('Next'))
    // Click Back
    fireEvent.click(screen.getByText('← Back'))
    // Should be on step 1 again
    expect(screen.getByText('Where do you want to host?')).toBeInTheDocument()
  })

  it('shows step indicator with 4 steps', () => {
    renderWithProviders(<OnboardWizard />)
    expect(screen.getByText('Provider')).toBeInTheDocument()
    expect(screen.getByText('Your Code')).toBeInTheDocument()
    expect(screen.getByText('Analyzing')).toBeInTheDocument()
    expect(screen.getByText('Results')).toBeInTheDocument()
  })
})

describe('ApplicationDetail', () => {
  beforeEach(() => {
    vi.mocked(useApplication).mockReturnValue({
      data: {
        application: sampleApp,
        resources: [],
      },
      isLoading: false,
      error: null,
    } as any)
    vi.mocked(useDeployments).mockReturnValue({ data: [] } as any)
    vi.mocked(usePlans).mockReturnValue({ data: [] } as any)
    vi.mocked(useComplianceFrameworks).mockReturnValue({ data: [] } as any)
  })

  it('renders header and all 4 tab labels', () => {
    renderWithProviders(<ApplicationDetail />)
    expect(screen.getByText('my-api')).toBeInTheDocument()
    expect(screen.getByText('A test API')).toBeInTheDocument()
    expect(screen.getByText('Plan')).toBeInTheDocument()
    expect(screen.getByText('Deploy')).toBeInTheDocument()
    expect(screen.getByText('Monitor')).toBeInTheDocument()
    expect(screen.getByText('Optimize')).toBeInTheDocument()
  })

  it('Plan tab is active by default showing Resources heading', () => {
    renderWithProviders(<ApplicationDetail />)
    expect(screen.getByText('Resources')).toBeInTheDocument()
    expect(screen.getByText('Infrastructure Topology')).toBeInTheDocument()
    expect(screen.getByText('Infrastructure Plans')).toBeInTheDocument()
    expect(screen.getByText('Compliance Frameworks')).toBeInTheDocument()
  })

  it('clicking Deploy tab shows deploy form and hides plan content', () => {
    renderWithProviders(<ApplicationDetail />)
    fireEvent.click(screen.getByText('Deploy'))
    expect(screen.getByText('Deployment History')).toBeInTheDocument()
    expect(screen.getByPlaceholderText('Branch')).toBeInTheDocument()
    expect(screen.queryByText('Resources')).not.toBeInTheDocument()
  })

  it('clicking Monitor tab shows live resources section', () => {
    renderWithProviders(<ApplicationDetail />)
    fireEvent.click(screen.getByText('Monitor'))
    expect(screen.getByText('Live Cloud Resources')).toBeInTheDocument()
    expect(screen.queryByText('Deployment History')).not.toBeInTheDocument()
  })

  it('clicking Optimize tab shows coming soon placeholder', () => {
    renderWithProviders(<ApplicationDetail />)
    fireEvent.click(screen.getByText('Optimize'))
    expect(screen.getByText('Cost Insights')).toBeInTheDocument()
    expect(screen.getByText(/coming soon/i)).toBeInTheDocument()
  })

  it('switching back to Plan tab restores plan content', () => {
    renderWithProviders(<ApplicationDetail />)
    fireEvent.click(screen.getByText('Deploy'))
    expect(screen.queryByText('Resources')).not.toBeInTheDocument()
    fireEvent.click(screen.getByText('Plan'))
    expect(screen.getByText('Resources')).toBeInTheDocument()
  })

  it('shows expandable plan list when plans exist', () => {
    const samplePlans: InfrastructurePlan[] = [
      {
        id: 'plan-1',
        application_id: '123',
        plan_type: 'hosting',
        content: 'Deploy on ECS Fargate',
        resources: [],
        estimated_cost: { monthly_cost_usd: 250, breakdown: {} },
        created_at: '2024-06-15T00:00:00Z',
      },
    ]
    vi.mocked(usePlans).mockReturnValue({ data: samplePlans } as any)
    renderWithProviders(<ApplicationDetail />)
    // Plan row shows type badge and cost
    expect(screen.getByText('hosting')).toBeInTheDocument()
    expect(screen.getByText('$250.00/mo')).toBeInTheDocument()
  })

  it('clicking Deploy button on plan row switches to Deploy tab with plan selected', () => {
    const samplePlans: InfrastructurePlan[] = [
      {
        id: 'plan-1',
        application_id: '123',
        plan_type: 'hosting',
        content: 'Deploy on ECS Fargate',
        resources: [],
        estimated_cost: { monthly_cost_usd: 100, breakdown: {} },
        created_at: '2024-06-15T00:00:00Z',
      },
    ]
    vi.mocked(usePlans).mockReturnValue({ data: samplePlans } as any)
    renderWithProviders(<ApplicationDetail />)
    // Click the Deploy button in the expandable plan row
    const deployButtons = screen.getAllByText('Deploy')
    // First "Deploy" is the tab label, second is the button in the plan row
    const planDeployButton = deployButtons[deployButtons.length - 1]
    fireEvent.click(planDeployButton)
    // Should now be on the Deploy tab
    expect(screen.getByText('Deployment History')).toBeInTheDocument()
    // Should show the selected plan in the plan selector
    expect(screen.getByText('Clear')).toBeInTheDocument()
  })

  it('Deploy tab shows plan selector with no plans message', () => {
    vi.mocked(usePlans).mockReturnValue({ data: [] } as any)
    renderWithProviders(<ApplicationDetail />)
    fireEvent.click(screen.getByText('Deploy'))
    expect(screen.getByText('Infrastructure Plan')).toBeInTheDocument()
    expect(screen.getByText(/No plans available/i)).toBeInTheDocument()
  })

  it('Deploy tab shows plan dropdown when plans exist', () => {
    const samplePlans: InfrastructurePlan[] = [
      {
        id: 'plan-1',
        application_id: '123',
        plan_type: 'hosting',
        content: 'Deploy on ECS',
        resources: [],
        created_at: '2024-06-15T00:00:00Z',
      },
    ]
    vi.mocked(usePlans).mockReturnValue({ data: samplePlans } as any)
    renderWithProviders(<ApplicationDetail />)
    // Use getAllByText since "Deploy" appears as both tab label and plan row button
    const deployButtons = screen.getAllByText('Deploy')
    // Click the tab (first occurrence)
    fireEvent.click(deployButtons[0])
    expect(screen.getByText('Infrastructure Plan')).toBeInTheDocument()
    expect(screen.getByText(/Select an infrastructure plan/i)).toBeInTheDocument()
    expect(screen.getByText('No plan (ad-hoc deploy)')).toBeInTheDocument()
  })

  it('shows compliance framework details when app has frameworks', () => {
    vi.mocked(useApplication).mockReturnValue({
      data: {
        application: { ...sampleApp, compliance_frameworks: ['cis_gcp_v4'] },
        resources: [],
      },
      isLoading: false,
      error: null,
    } as any)
    vi.mocked(useComplianceFrameworks).mockReturnValue({
      data: [{
        id: 'cis_gcp_v4',
        name: 'CIS GCP Foundation Benchmark',
        version: '4.0.0',
        provider: 'aws',
        description: 'Security best practices for GCP',
      }],
    } as any)
    renderWithProviders(<ApplicationDetail />)
    expect(screen.getByText('CIS GCP Foundation Benchmark')).toBeInTheDocument()
    expect(screen.getByText('v4.0.0')).toBeInTheDocument()
    expect(screen.getByText('Security best practices for GCP')).toBeInTheDocument()
  })
})
