import { Routes, Route, Link } from 'react-router-dom'
import ApplicationList from './pages/ApplicationList'
import ApplicationDetail from './pages/ApplicationDetail'
import DeploymentDashboard from './pages/DeploymentDashboard'
import MigrationPlanner from './pages/MigrationPlanner'
import OnboardWizard from './pages/OnboardWizard'

function Layout({ children }: { children: React.ReactNode }) {
  return (
    <div className="min-h-screen">
      <nav className="bg-white border-b border-gray-200 px-6 py-4">
        <div className="max-w-7xl mx-auto flex items-center justify-between">
          <Link to="/" className="text-xl font-bold text-indigo-600">
            Infraplane
          </Link>
          <div className="flex gap-6">
            <Link to="/" className="text-gray-600 hover:text-gray-900">
              Applications
            </Link>
            <Link to="/deployments" className="text-gray-600 hover:text-gray-900">
              Deployments
            </Link>
            <Link to="/migrate" className="text-gray-600 hover:text-gray-900">
              Migration
            </Link>
            <Link to="/onboard" className="text-indigo-600 font-medium hover:text-indigo-800">
              Onboard
            </Link>
          </div>
        </div>
      </nav>
      <main className="max-w-7xl mx-auto px-6 py-8">
        {children}
      </main>
    </div>
  )
}

export default function App() {
  return (
    <Layout>
      <Routes>
        <Route path="/" element={<ApplicationList />} />
        <Route path="/applications/:name" element={<ApplicationDetail />} />
        <Route path="/deployments" element={<DeploymentDashboard />} />
        <Route path="/migrate" element={<MigrationPlanner />} />
        <Route path="/onboard" element={<OnboardWizard />} />
      </Routes>
    </Layout>
  )
}
