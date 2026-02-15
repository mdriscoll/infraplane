import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import * as api from '../api/client'

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
    mutationFn: ({ gitBranch, gitCommit }: { gitBranch: string; gitCommit?: string }) =>
      api.deploy(appName, gitBranch, gitCommit),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['deployments', appName] })
      queryClient.invalidateQueries({ queryKey: ['applications', appName] })
    },
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
