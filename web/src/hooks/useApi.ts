import { useState, useEffect, useRef, useCallback } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import * as api from '../api/client'
import type { DeploymentEvent, DeployTarget } from '../api/client'

// --- Application Hooks ---

export function useApplications() {
  return useQuery({
    queryKey: ['applications'],
    queryFn: api.listApplications,
  })
}

export function useApplication(name: string) {
  return useQuery({
    queryKey: ['applications', name],
    queryFn: () => api.getApplication(name),
    enabled: !!name,
  })
}

export function useRegisterApplication() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: api.registerApplication,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['applications'] })
    },
  })
}

export function useOnboardApplication() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: api.onboardApplication,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['applications'] })
    },
  })
}

export function useReanalyzeSource(appName: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () => api.reanalyzeSource(appName),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['applications', appName] })
      queryClient.invalidateQueries({ queryKey: ['resources', appName] })
      queryClient.invalidateQueries({ queryKey: ['analysis-runs', appName] })
    },
  })
}

export function useAnalyzeUpload(appName: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (files: { path: string; content: string }[]) =>
      api.analyzeUpload(appName, files),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['applications', appName] })
      queryClient.invalidateQueries({ queryKey: ['resources', appName] })
      queryClient.invalidateQueries({ queryKey: ['analysis-runs', appName] })
    },
  })
}

export function useDeleteApplication() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: api.deleteApplication,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['applications'] })
    },
  })
}

// --- Resource Hooks ---

export function useResources(appName: string) {
  return useQuery({
    queryKey: ['resources', appName],
    queryFn: () => api.listResources(appName),
    enabled: !!appName,
  })
}

export function useAddResource(appName: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (description: string) => api.addResource(appName, description),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['resources', appName] })
      queryClient.invalidateQueries({ queryKey: ['applications', appName] })
    },
  })
}

export function useRemoveResource(appName: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: api.removeResource,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['resources', appName] })
      queryClient.invalidateQueries({ queryKey: ['applications', appName] })
    },
  })
}

// --- Analysis Run Hooks ---

export function useAnalysisRuns(appName: string) {
  return useQuery({
    queryKey: ['analysis-runs', appName],
    queryFn: () => api.listAnalysisRuns(appName),
    enabled: !!appName,
  })
}

export function useResourcesByRun(runId: string | null) {
  return useQuery({
    queryKey: ['resources-by-run', runId],
    queryFn: () => api.listResourcesByRun(runId!),
    enabled: !!runId,
  })
}

// --- Plan Hooks ---

export function usePlans(appName: string) {
  return useQuery({
    queryKey: ['plans', appName],
    queryFn: () => api.listPlans(appName),
    enabled: !!appName,
  })
}

export function useGenerateHostingPlan(appName: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () => api.generateHostingPlan(appName),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['plans', appName] })
      queryClient.invalidateQueries({ queryKey: ['applications', appName] })
    },
  })
}

export function useGenerateMigrationPlan(appName: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ fromProvider, toProvider }: { fromProvider: string; toProvider: string }) =>
      api.generateMigrationPlan(appName, fromProvider, toProvider),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['plans', appName] })
    },
  })
}

// --- Deployment Hooks ---

export function useDeployments(appName: string) {
  return useQuery({
    queryKey: ['deployments', appName],
    queryFn: () => api.listDeployments(appName),
    enabled: !!appName,
  })
}

export function useLatestDeployment(appName: string) {
  return useQuery({
    queryKey: ['deployments', appName, 'latest'],
    queryFn: () => api.getLatestDeployment(appName),
    enabled: !!appName,
  })
}

export function useDeploy(appName: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ gitBranch, gitCommit, planId, deployTarget }: { gitBranch: string; gitCommit?: string; planId?: string; deployTarget?: DeployTarget }) =>
      api.deploy(appName, gitBranch, gitCommit, planId, deployTarget),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['deployments', appName] })
      queryClient.invalidateQueries({ queryKey: ['applications', appName] })
    },
  })
}

export function useValidateDeployTarget(appName: string) {
  return useMutation({
    mutationFn: ({ provider, target }: { provider: string; target: DeployTarget }) =>
      api.validateDeployTarget(appName, provider, target),
  })
}

// --- Live Resource Hooks ---

export function useDiscoverLiveResources(appName: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () => api.getLiveResources(appName),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['applications', appName] })
    },
  })
}

// --- Graph Hooks ---

export function useGraph(appName: string) {
  return useQuery({
    queryKey: ['graph', appName],
    queryFn: () => api.getLatestGraph(appName),
    enabled: !!appName,
    retry: false,
  })
}

export function useGenerateGraph(appName: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () => api.generateGraph(appName),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['graph', appName] })
    },
  })
}

// --- Compliance Framework Hooks ---

export function useComplianceFrameworks(provider?: string) {
  return useQuery({
    queryKey: ['compliance-frameworks', provider],
    queryFn: () => api.listComplianceFrameworks(provider),
  })
}

// --- GCP Project Hooks ---

export function useGCPProjects() {
  return useQuery({
    queryKey: ['gcp-projects'],
    queryFn: api.listGCPProjects,
    staleTime: 5 * 60 * 1000, // cache for 5 minutes
    retry: false, // don't retry if GCP creds are unavailable
  })
}

// --- GCP Credential Hooks ---

export function useGCPCredentialStatus() {
  return useQuery({
    queryKey: ['gcp-credentials'],
    queryFn: api.getGCPCredentialStatus,
    staleTime: 30 * 1000, // cache for 30 seconds
    retry: false,
  })
}

export function useUploadGCPCredentials() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: api.uploadGCPCredentials,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['gcp-credentials'] })
      queryClient.invalidateQueries({ queryKey: ['gcp-projects'] })
    },
  })
}

export function useDeleteGCPCredentials() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: api.deleteGCPCredentials,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['gcp-credentials'] })
      queryClient.invalidateQueries({ queryKey: ['gcp-projects'] })
    },
  })
}

// --- Deployment Stream Hook ---

export function useDeploymentStream(deploymentId: string | null) {
  const [events, setEvents] = useState<DeploymentEvent[]>([])
  const [isStreaming, setIsStreaming] = useState(false)
  const [isComplete, setIsComplete] = useState(false)
  const [finalStatus, setFinalStatus] = useState<'succeeded' | 'failed' | null>(null)
  const eventSourceRef = useRef<EventSource | null>(null)

  const reset = useCallback(() => {
    setEvents([])
    setIsStreaming(false)
    setIsComplete(false)
    setFinalStatus(null)
  }, [])

  useEffect(() => {
    if (!deploymentId) return

    reset()
    setIsStreaming(true)

    const es = new EventSource(api.getDeploymentStreamUrl(deploymentId))
    eventSourceRef.current = es

    es.onmessage = (e) => {
      const event: DeploymentEvent = JSON.parse(e.data)
      setEvents((prev) => [...prev, event])

      if (event.step === 'complete' || event.step === 'failed') {
        setIsComplete(true)
        setIsStreaming(false)
        setFinalStatus(event.status === 'succeeded' ? 'succeeded' : 'failed')
        es.close()
      }
    }

    es.onerror = () => {
      setIsStreaming(false)
      es.close()
    }

    return () => {
      es.close()
      eventSourceRef.current = null
    }
  }, [deploymentId, reset])

  return { events, isStreaming, isComplete, finalStatus, reset }
}

// --- Deployment Reconnect Stream Hook ---
// Connects to the events/stream endpoint which replays stored events
// and then streams live events for in-progress deployments.
export function useDeploymentReconnect(deploymentId: string | null) {
  const [events, setEvents] = useState<DeploymentEvent[]>([])
  const [isStreaming, setIsStreaming] = useState(false)
  const [isComplete, setIsComplete] = useState(false)
  const [finalStatus, setFinalStatus] = useState<'succeeded' | 'failed' | null>(null)
  const eventSourceRef = useRef<EventSource | null>(null)

  const reset = useCallback(() => {
    if (eventSourceRef.current) {
      eventSourceRef.current.close()
      eventSourceRef.current = null
    }
    setEvents([])
    setIsStreaming(false)
    setIsComplete(false)
    setFinalStatus(null)
  }, [])

  useEffect(() => {
    if (!deploymentId) return

    reset()
    setIsStreaming(true)

    const es = new EventSource(api.getDeploymentEventsStreamUrl(deploymentId))
    eventSourceRef.current = es

    es.onmessage = (e) => {
      const event: DeploymentEvent = JSON.parse(e.data)
      setEvents((prev) => [...prev, event])

      if (event.step === 'complete' || event.step === 'failed') {
        setIsComplete(true)
        setIsStreaming(false)
        setFinalStatus(event.status === 'succeeded' ? 'succeeded' : 'failed')
        es.close()
      }
    }

    es.onerror = () => {
      // Connection closed — could mean deployment completed (SSE closed normally)
      // or a network error. Check if we got a final event.
      setIsStreaming(false)
      es.close()
      // If we have events but no explicit complete/failed, mark as complete
      setEvents((prev) => {
        if (prev.length > 0) {
          const last = prev[prev.length - 1]
          if (last.step !== 'complete' && last.step !== 'failed') {
            setIsComplete(true)
            setFinalStatus(last.status === 'succeeded' ? 'succeeded' : last.status === 'failed' ? 'failed' : null)
          }
        }
        return prev
      })
    }

    return () => {
      es.close()
      eventSourceRef.current = null
    }
  }, [deploymentId, reset])

  return { events, isStreaming, isComplete, finalStatus, reset }
}
