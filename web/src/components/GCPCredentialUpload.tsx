import { useRef, useState } from 'react'
import { useGCPCredentialStatus, useUploadGCPCredentials, useDeleteGCPCredentials } from '../hooks/useApi'

interface Props {
  onProjectDetected?: (projectId: string) => void
  compact?: boolean
}

export default function GCPCredentialUpload({ onProjectDetected, compact }: Props) {
  const { data: credStatus, isLoading } = useGCPCredentialStatus()
  const uploadMutation = useUploadGCPCredentials()
  const deleteMutation = useDeleteGCPCredentials()
  const fileInputRef = useRef<HTMLInputElement>(null)
  const [showFull, setShowFull] = useState(false)

  const handleUpload = () => {
    const file = fileInputRef.current?.files?.[0]
    if (!file) return
    uploadMutation.mutate(file, {
      onSuccess: (data) => {
        if (fileInputRef.current) fileInputRef.current.value = ''
        if (data.project_id && onProjectDetected) {
          onProjectDetected(data.project_id)
        }
        setShowFull(false)
      },
    })
  }

  const handleRemove = () => {
    deleteMutation.mutate(undefined, {
      onSuccess: () => setShowFull(false),
    })
  }

  if (isLoading) {
    return (
      <div className="p-3 border border-gray-200 rounded-lg bg-gray-50">
        <p className="text-sm text-gray-400">Checking GCP credentials...</p>
      </div>
    )
  }

  // Compact mode: show single-line when credentials exist
  if (compact && credStatus?.configured && !showFull) {
    return (
      <div className="flex items-center gap-2 py-1">
        <span className="text-green-600 text-sm">{'\u2713'}</span>
        <span className="text-sm text-gray-700">
          Using <strong className="font-medium">{credStatus.client_email}</strong>
        </span>
        {credStatus.project_id && (
          <span className="text-xs text-gray-500">({credStatus.project_id})</span>
        )}
        <button
          onClick={() => setShowFull(true)}
          className="text-xs text-indigo-600 hover:text-indigo-800 underline ml-1 transition-colors"
        >
          Change
        </button>
      </div>
    )
  }

  if (credStatus?.configured) {
    return (
      <div className="p-3 border border-green-200 rounded-lg bg-green-50">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <span className="text-green-600 text-sm font-medium">{'\u2713'} GCP Credentials</span>
            <span className="text-xs text-gray-600 bg-white px-2 py-0.5 rounded-full border border-gray-200">
              {credStatus.client_email}
            </span>
            {credStatus.project_id && (
              <span className="text-xs text-gray-500">
                ({credStatus.project_id})
              </span>
            )}
          </div>
          <div className="flex items-center gap-3">
            {compact && (
              <button
                onClick={() => setShowFull(false)}
                className="text-xs text-gray-500 hover:text-gray-700 underline"
              >
                Collapse
              </button>
            )}
            <button
              onClick={handleRemove}
              disabled={deleteMutation.isPending}
              className="text-xs text-red-500 hover:text-red-700 underline disabled:opacity-50"
            >
              {deleteMutation.isPending ? 'Removing...' : 'Remove'}
            </button>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="p-3 border border-amber-200 rounded-lg bg-amber-50">
      <p className="text-sm text-amber-800 mb-2">
        No GCP credentials configured. Upload a service account key to enable project listing and deployments.
      </p>
      <div className="flex items-center gap-2">
        <input
          ref={fileInputRef}
          type="file"
          accept=".json"
          className="text-xs text-gray-600 file:mr-2 file:py-1 file:px-3 file:rounded-lg file:border-0 file:text-xs file:font-medium file:bg-blue-50 file:text-blue-700 hover:file:bg-blue-100"
        />
        <button
          onClick={handleUpload}
          disabled={uploadMutation.isPending}
          className="text-sm bg-blue-600 text-white px-3 py-1 rounded-lg hover:bg-blue-700 disabled:opacity-50 transition-colors"
        >
          {uploadMutation.isPending ? 'Uploading...' : 'Upload Key'}
        </button>
      </div>
      {uploadMutation.isError && (
        <p className="mt-2 text-xs text-red-600">{uploadMutation.error.message}</p>
      )}
    </div>
  )
}
