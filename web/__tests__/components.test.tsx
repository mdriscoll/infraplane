import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { describe, it, expect } from 'vitest'
import AppCard from '../src/components/AppCard'
import ResourceList from '../src/components/ResourceList'
import DeploymentHistory from '../src/components/DeploymentHistory'
import PlanViewer from '../src/components/PlanViewer'
import type { Application, Resource, Deployment, InfrastructurePlan } from '../src/api/client'

describe('AppCard', () => {
  const app: Application = {
    id: '123',
    name: 'my-api',
    description: 'A test API',
    git_repo_url: '',
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
})

describe('ResourceList', () => {
  it('shows empty message when no resources', () => {
    render(<ResourceList resources={[]} />)
    expect(screen.getByText('No resources added yet.')).toBeInTheDocument()
  })

  it('renders resources', () => {
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
    expect(screen.getByText('database')).toBeInTheDocument()
    expect(screen.getByText('AWS: RDS')).toBeInTheDocument()
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
