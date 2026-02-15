import { render, screen, fireEvent } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { describe, it, expect } from 'vitest'
import AppCard from '../src/components/AppCard'
import ResourceList from '../src/components/ResourceList'
import DeploymentHistory from '../src/components/DeploymentHistory'
import PlanViewer from '../src/components/PlanViewer'
import OnboardWizard from '../src/pages/OnboardWizard'
import type { Application, Resource, Deployment, InfrastructurePlan } from '../src/api/client'

describe('AppCard', () => {
  const app: Application = {
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
