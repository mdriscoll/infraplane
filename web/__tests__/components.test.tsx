import { render, screen, fireEvent } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import AppCard from '../src/components/AppCard'
import ResourceList from '../src/components/ResourceList'
import DeploymentHistory from '../src/components/DeploymentHistory'
import PlanViewer from '../src/components/PlanViewer'
import OnboardWizard from '../src/pages/OnboardWizard'
import ApplicationList from '../src/pages/ApplicationList'
import type { Application, Resource, Deployment, InfrastructurePlan } from '../src/api/client'

const sampleApp: Application = {
  id: '123',
  name: 'my-api',
  description: 'A test API',
  git_repo_url: '',
  source_path: '/Users/dev/my-api',
  provider: 'aws',
  status: 'deployed',
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

// Mock the API module so ApplicationList renders with test data
vi.mock('../src/hooks/useApi', async () => {
  const actual = await vi.importActual('../src/hooks/useApi')
  return {
    ...actual,
    useApplications: vi.fn(),
    useRegisterApplication: vi.fn(() => ({ mutate: vi.fn(), isPending: false, isError: false })),
    useDeleteApplication: vi.fn(() => ({ mutateAsync: vi.fn(), isPending: false, isError: false })),
  }
})

import { useApplications, useDeleteApplication } from '../src/hooks/useApi'

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
