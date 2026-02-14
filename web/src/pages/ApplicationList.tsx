import { useState } from 'react'
import AppCard from '../components/AppCard'
import { useApplications, useRegisterApplication } from '../hooks/useApi'

export default function ApplicationList() {
  const { data: apps, isLoading, error } = useApplications()
  const registerMutation = useRegisterApplication()
  const [showForm, setShowForm] = useState(false)
  const [name, setName] = useState('')
  const [sourcePath, setSourcePath] = useState('')
  const [description, setDescription] = useState('')
  const [provider, setProvider] = useState('aws')

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    registerMutation.mutate(
      { name, source_path: sourcePath || undefined, description: description || undefined, provider },
      {
        onSuccess: () => {
          setName('')
          setSourcePath('')
          setDescription('')
          setProvider('aws')
          setShowForm(false)
        },
      }
    )
  }

  if (isLoading) {
    return <div className="text-center py-12 text-gray-500">Loading applications...</div>
  }

  if (error) {
    return <div className="text-center py-12 text-red-500">Error: {error.message}</div>
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Applications</h1>
          <p className="text-gray-500 mt-1">Manage your cloud infrastructure applications</p>
        </div>
        <button
          onClick={() => setShowForm(!showForm)}
          className="bg-indigo-600 text-white px-4 py-2 rounded-lg hover:bg-indigo-700 transition-colors"
        >
          {showForm ? 'Cancel' : 'Register Application'}
        </button>
      </div>

      {showForm && (
        <form onSubmit={handleSubmit} className="bg-white border border-gray-200 rounded-lg p-6 mb-8">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Name</label>
              <input
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value)}
                className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
                placeholder="my-api"
                required
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Provider</label>
              <select
                value={provider}
                onChange={(e) => setProvider(e.target.value)}
                className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
              >
                <option value="aws">AWS</option>
                <option value="gcp">GCP</option>
              </select>
            </div>
            <div className="md:col-span-2">
              <label className="block text-sm font-medium text-gray-700 mb-1">Source</label>
              <input
                type="text"
                value={sourcePath}
                onChange={(e) => setSourcePath(e.target.value)}
                className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
                placeholder="/path/to/project or https://github.com/org/repo"
              />
              <p className="mt-1 text-xs text-gray-400">
                Provide a local path or git URL to auto-detect infrastructure resources
              </p>
            </div>
            <div className="md:col-span-2">
              <label className="block text-sm font-medium text-gray-700 mb-1">Description</label>
              <input
                type="text"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
                placeholder="A brief description of your application (optional)"
              />
            </div>
          </div>
          <div className="mt-4 flex justify-end">
            <button
              type="submit"
              disabled={registerMutation.isPending}
              className="bg-indigo-600 text-white px-4 py-2 rounded-lg hover:bg-indigo-700 disabled:opacity-50 transition-colors"
            >
              {registerMutation.isPending ? 'Registering...' : 'Register'}
            </button>
          </div>
          {registerMutation.isError && (
            <p className="mt-2 text-sm text-red-500">{registerMutation.error.message}</p>
          )}
        </form>
      )}

      {apps && apps.length > 0 ? (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {apps.map((app) => (
            <AppCard key={app.id} app={app} />
          ))}
        </div>
      ) : (
        <div className="text-center py-12 text-gray-400">
          <p className="text-lg">No applications registered yet</p>
          <p className="text-sm mt-1">Click &quot;Register Application&quot; to get started</p>
        </div>
      )}
    </div>
  )
}
