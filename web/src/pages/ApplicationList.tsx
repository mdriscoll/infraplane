import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useApplications, useRegisterApplication, useDeleteApplication } from '../hooks/useApi'
import { pickAndReadDirectory, isDirectoryPickerSupported } from '../lib/directoryPicker'
import type { FileContent } from '../lib/directoryPicker'
import type { Application } from '../api/client'

const statusColors: Record<string, string> = {
  draft: 'bg-gray-100 text-gray-700',
  provisioned: 'bg-blue-100 text-blue-700',
  deployed: 'bg-green-100 text-green-700',
}

const providerLabels: Record<string, string> = {
  aws: 'AWS',
  gcp: 'GCP',
}

function formatDate(dateStr: string): string {
  const date = new Date(dateStr)
  return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })
}

export default function ApplicationList() {
  const { data: apps, isLoading, error } = useApplications()
  const registerMutation = useRegisterApplication()
  const deleteMutation = useDeleteApplication()
  const navigate = useNavigate()

  const [showForm, setShowForm] = useState(false)
  const [name, setName] = useState('')
  const [sourcePath, setSourcePath] = useState('')
  const [description, setDescription] = useState('')
  const [provider, setProvider] = useState('aws')
  const [pickedFiles, setPickedFiles] = useState<FileContent[] | null>(null)
  const [pickedDirName, setPickedDirName] = useState('')
  const [pickError, setPickError] = useState('')

  // Selection state
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set())
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)
  const [deleteProgress, setDeleteProgress] = useState<{ done: number; total: number } | null>(null)

  const allSelected = apps && apps.length > 0 && selectedIds.size === apps.length
  const someSelected = selectedIds.size > 0

  const toggleSelectAll = () => {
    if (!apps) return
    if (allSelected) {
      setSelectedIds(new Set())
    } else {
      setSelectedIds(new Set(apps.map((a) => a.id)))
    }
  }

  const toggleSelect = (id: string) => {
    setSelectedIds((prev) => {
      const next = new Set(prev)
      if (next.has(id)) {
        next.delete(id)
      } else {
        next.add(id)
      }
      return next
    })
  }

  const handleBulkDelete = async () => {
    if (!apps) return
    const toDelete = apps.filter((a) => selectedIds.has(a.id))
    setDeleteProgress({ done: 0, total: toDelete.length })

    for (let i = 0; i < toDelete.length; i++) {
      try {
        await deleteMutation.mutateAsync(toDelete[i].name)
      } catch {
        // continue deleting the rest even if one fails
      }
      setDeleteProgress({ done: i + 1, total: toDelete.length })
    }

    setSelectedIds(new Set())
    setShowDeleteConfirm(false)
    setDeleteProgress(null)
  }

  const handleBrowse = async () => {
    setPickError('')
    try {
      const files = await pickAndReadDirectory()
      if (files) {
        setPickedFiles(files)
        setPickedDirName(`${files.length} infrastructure file${files.length !== 1 ? 's' : ''} detected`)
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
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
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

      {/* Register Form */}
      {showForm && (
        <form onSubmit={handleSubmit} className="bg-white border border-gray-200 rounded-lg p-6 mb-6">
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

      {/* Bulk Actions Bar */}
      {someSelected && (
        <div className="bg-indigo-50 border border-indigo-200 rounded-lg px-4 py-3 mb-4 flex items-center justify-between">
          <span className="text-sm font-medium text-indigo-800">
            {selectedIds.size} application{selectedIds.size !== 1 ? 's' : ''} selected
          </span>
          <div className="flex items-center gap-2">
            <button
              onClick={() => setSelectedIds(new Set())}
              className="text-sm text-indigo-600 hover:text-indigo-800 px-3 py-1.5 rounded hover:bg-indigo-100 transition-colors"
            >
              Clear selection
            </button>
            <button
              onClick={() => setShowDeleteConfirm(true)}
              className="inline-flex items-center gap-1.5 text-sm font-medium text-red-600 bg-red-50 border border-red-200 px-3 py-1.5 rounded-lg hover:bg-red-100 transition-colors"
            >
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
              </svg>
              Delete selected
            </button>
          </div>
        </div>
      )}

      {/* Delete Confirmation Modal */}
      {showDeleteConfirm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
          <div className="bg-white rounded-xl shadow-xl max-w-md w-full mx-4 p-6">
            <div className="flex items-center gap-3 mb-4">
              <div className="flex-shrink-0 w-10 h-10 rounded-full bg-red-100 flex items-center justify-center">
                <svg className="w-5 h-5 text-red-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L4.082 16.5c-.77.833.192 2.5 1.732 2.5z" />
                </svg>
              </div>
              <div>
                <h3 className="text-lg font-semibold text-gray-900">Delete applications</h3>
                <p className="text-sm text-gray-500">This action cannot be undone.</p>
              </div>
            </div>
            <p className="text-sm text-gray-700 mb-2">
              Are you sure you want to delete {selectedIds.size} application{selectedIds.size !== 1 ? 's' : ''}?
            </p>
            <ul className="text-sm text-gray-600 mb-4 max-h-40 overflow-y-auto space-y-1">
              {apps?.filter((a) => selectedIds.has(a.id)).map((a) => (
                <li key={a.id} className="flex items-center gap-2">
                  <span className="w-1.5 h-1.5 rounded-full bg-gray-400" />
                  <span className="font-medium">{a.name}</span>
                  <span className="text-gray-400">({providerLabels[a.provider] || a.provider})</span>
                </li>
              ))}
            </ul>
            {deleteProgress && (
              <div className="mb-4">
                <div className="w-full bg-gray-200 rounded-full h-2">
                  <div
                    className="bg-red-500 h-2 rounded-full transition-all"
                    style={{ width: `${(deleteProgress.done / deleteProgress.total) * 100}%` }}
                  />
                </div>
                <p className="text-xs text-gray-500 mt-1">
                  Deleting... {deleteProgress.done} of {deleteProgress.total}
                </p>
              </div>
            )}
            <div className="flex items-center justify-end gap-3">
              <button
                onClick={() => setShowDeleteConfirm(false)}
                disabled={!!deleteProgress}
                className="px-4 py-2 text-sm text-gray-700 hover:bg-gray-100 rounded-lg transition-colors disabled:opacity-50"
              >
                Cancel
              </button>
              <button
                onClick={handleBulkDelete}
                disabled={!!deleteProgress}
                className="px-4 py-2 text-sm font-medium text-white bg-red-600 rounded-lg hover:bg-red-700 transition-colors disabled:opacity-50"
              >
                {deleteProgress ? 'Deleting...' : `Delete ${selectedIds.size} application${selectedIds.size !== 1 ? 's' : ''}`}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Application List */}
      {apps && apps.length > 0 ? (
        <div className="bg-white border border-gray-200 rounded-lg overflow-hidden">
          <table className="w-full">
            <thead>
              <tr className="bg-gray-50 border-b border-gray-200">
                <th className="w-12 px-4 py-3">
                  <input
                    type="checkbox"
                    checked={allSelected}
                    ref={(el) => {
                      if (el) el.indeterminate = someSelected && !allSelected
                    }}
                    onChange={toggleSelectAll}
                    className="w-4 h-4 rounded border-gray-300 text-indigo-600 focus:ring-indigo-500"
                    aria-label="Select all applications"
                  />
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Name
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Status
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Provider
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider hidden md:table-cell">
                  Source
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider hidden lg:table-cell">
                  Created
                </th>
                <th className="w-12 px-4 py-3" />
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              {apps.map((app) => (
                <ApplicationRow
                  key={app.id}
                  app={app}
                  selected={selectedIds.has(app.id)}
                  onToggleSelect={() => toggleSelect(app.id)}
                  onNavigate={() => navigate(`/applications/${app.name}`)}
                />
              ))}
            </tbody>
          </table>
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

function ApplicationRow({
  app,
  selected,
  onToggleSelect,
  onNavigate,
}: {
  app: Application
  selected: boolean
  onToggleSelect: () => void
  onNavigate: () => void
}) {
  return (
    <tr
      className={`group transition-colors ${selected ? 'bg-indigo-50/50' : 'hover:bg-gray-50'}`}
    >
      <td className="px-4 py-3" onClick={(e) => e.stopPropagation()}>
        <input
          type="checkbox"
          checked={selected}
          onChange={onToggleSelect}
          className="w-4 h-4 rounded border-gray-300 text-indigo-600 focus:ring-indigo-500"
          aria-label={`Select ${app.name}`}
        />
      </td>
      <td className="px-4 py-3 cursor-pointer" onClick={onNavigate}>
        <div>
          <span className="text-sm font-semibold text-gray-900 group-hover:text-indigo-600 transition-colors">
            {app.name}
          </span>
          {app.description && (
            <p className="text-xs text-gray-500 mt-0.5 truncate max-w-xs">{app.description}</p>
          )}
        </div>
      </td>
      <td className="px-4 py-3 cursor-pointer" onClick={onNavigate}>
        <span className={`inline-flex text-xs font-medium px-2.5 py-0.5 rounded-full ${statusColors[app.status] || statusColors.draft}`}>
          {app.status}
        </span>
      </td>
      <td className="px-4 py-3 cursor-pointer" onClick={onNavigate}>
        <span className="inline-flex items-center text-xs font-medium px-2 py-1 rounded bg-indigo-50 text-indigo-700">
          {providerLabels[app.provider] || app.provider}
        </span>
      </td>
      <td className="px-4 py-3 cursor-pointer hidden md:table-cell" onClick={onNavigate}>
        {app.source_path ? (
          <span className="text-xs text-gray-400 truncate block max-w-[200px]" title={app.source_path}>
            {app.source_path}
          </span>
        ) : (
          <span className="text-xs text-gray-300">—</span>
        )}
      </td>
      <td className="px-4 py-3 cursor-pointer hidden lg:table-cell" onClick={onNavigate}>
        <span className="text-xs text-gray-400">{formatDate(app.created_at)}</span>
      </td>
      <td className="px-4 py-3 cursor-pointer" onClick={onNavigate}>
        <svg className="w-4 h-4 text-gray-300 group-hover:text-gray-500 transition-colors" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
        </svg>
      </td>
    </tr>
  )
}
