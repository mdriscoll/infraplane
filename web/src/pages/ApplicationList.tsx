import { useState } from 'react'
import { Link } from 'react-router-dom'
import AppCard from '../components/AppCard'
import { useApplications, useRegisterApplication } from '../hooks/useApi'
import { pickAndReadDirectory, isDirectoryPickerSupported } from '../lib/directoryPicker'
import type { FileContent } from '../lib/directoryPicker'

export default function ApplicationList() {
  const { data: apps, isLoading, error } = useApplications()
  const registerMutation = useRegisterApplication()
  const [showForm, setShowForm] = useState(false)
  const [name, setName] = useState('')
  const [sourcePath, setSourcePath] = useState('')
  const [description, setDescription] = useState('')
  const [provider, setProvider] = useState('aws')
  const [pickedFiles, setPickedFiles] = useState<FileContent[] | null>(null)
  const [pickedDirName, setPickedDirName] = useState('')
  const [pickError, setPickError] = useState('')

  const handleBrowse = async () => {
    setPickError('')
    try {
      const files = await pickAndReadDirectory()
      if (files) {
        setPickedFiles(files)
        setPickedDirName(`${files.length} infrastructure file${files.length !== 1 ? 's' : ''} detected`)
        // Clear the text input since we're using directory picker
        setSourcePath('')
      }
    } catch (err) {
      setPickError(err instanceof Error ? err.message : 'Failed to read directory')
    }
  }

  const clearPicked = () => {
    setPickedFiles(null)
    setPickedDirName('')
  }

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    registerMutation.mutate(
      {
        name,
        source_path: sourcePath || undefined,
        description: description || undefined,
        provider,
        files: pickedFiles && pickedFiles.length > 0 ? pickedFiles : undefined,
      },
      {
        onSuccess: () => {
          setName('')
          setSourcePath('')
          setDescription('')
          setProvider('aws')
          setPickedFiles(null)
          setPickedDirName('')
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
        <div className="flex items-center gap-3">
          <Link
            to="/onboard"
            className="inline-flex items-center gap-1.5 text-indigo-600 font-medium px-4 py-2 rounded-lg border border-indigo-200 hover:bg-indigo-50 transition-colors"
          >
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
            </svg>
            Onboarding Wizard
          </Link>
          <button
            onClick={() => setShowForm(!showForm)}
            className="bg-indigo-600 text-white px-4 py-2 rounded-lg hover:bg-indigo-700 transition-colors"
          >
            {showForm ? 'Cancel' : 'Register Application'}
          </button>
        </div>
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
              <div className="flex gap-2">
                <input
                  type="text"
                  value={pickedDirName || sourcePath}
                  onChange={(e) => {
                    setSourcePath(e.target.value)
                    clearPicked()
                  }}
                  className="flex-1 border border-gray-300 rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
                  placeholder="https://github.com/org/repo"
                  readOnly={!!pickedDirName}
                  disabled={!!pickedDirName}
                />
                {isDirectoryPickerSupported() && (
                  <button
                    type="button"
                    onClick={handleBrowse}
                    className="inline-flex items-center gap-1.5 bg-gray-100 text-gray-700 px-3 py-2 rounded-lg hover:bg-gray-200 text-sm font-medium transition-colors whitespace-nowrap"
                  >
                    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
                    </svg>
                    Browse Folder
                  </button>
                )}
              </div>
              {pickedDirName && (
                <div className="mt-1.5 flex items-center gap-2">
                  <span className="inline-flex items-center text-xs font-medium px-2 py-0.5 rounded-full bg-green-100 text-green-700">
                    {pickedDirName}
                  </span>
                  <button
                    type="button"
                    onClick={clearPicked}
                    className="text-xs text-gray-400 hover:text-gray-600"
                  >
                    Clear
                  </button>
                </div>
              )}
              {pickError && (
                <p className="mt-1 text-xs text-red-500">{pickError}</p>
              )}
              <p className="mt-1 text-xs text-gray-400">
                Browse a local folder or enter a git URL to auto-detect infrastructure resources
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
        <div className="text-center py-16">
          <div className="w-16 h-16 mx-auto mb-4 rounded-full bg-indigo-100 flex items-center justify-center">
            <svg className="w-8 h-8 text-indigo-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
            </svg>
          </div>
          <p className="text-lg font-medium text-gray-900">No applications registered yet</p>
          <p className="text-sm text-gray-500 mt-2 max-w-md mx-auto">
            Our onboarding wizard will analyze your code and generate a cloud hosting plan automatically.
          </p>
          <div className="mt-6 flex items-center justify-center gap-3">
            <Link
              to="/onboard"
              className="inline-flex items-center gap-2 bg-indigo-600 text-white px-5 py-2.5 rounded-lg hover:bg-indigo-700 transition-colors font-medium"
            >
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
              </svg>
              Start Onboarding Wizard
            </Link>
            <button
              onClick={() => setShowForm(true)}
              className="text-gray-500 hover:text-gray-700 px-4 py-2.5 rounded-lg hover:bg-gray-100 transition-colors text-sm"
            >
              or register manually
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
