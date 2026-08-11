import { SearchResult } from "@/components/search/types"
import { apiUrl } from "@/lib/config"
import {
  ClusterConnection,
  ClustersResponse,
  CreateClusterRequest,
  AgentManifestResponse,
} from "@/types/cluster"

import {
  SLO,
  SLA,
  SLI,
} from "./reliability"

export async function getClusters(): Promise<ClustersResponse> {
  try {
    const res = await fetch(apiUrl("/api/clusters"), { cache: "no-store" })
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    return await res.json()
  } catch {
    return {
      activeId: "local-dev",
      clusters: [
        {
          id: "local-dev",
          name: "Local Development",
          environment: "development",
          provider: "Local / Kubeconfig",
          clusterType: "Local Context",
          connectionMode: "local_kubeconfig",
          status: "CONNECTED",
          kubernetesVersion: "v1.32.0",
          latencyMs: 12,
          nodeCount: 1,
          namespaceCount: 4,
          capabilities: {
            canReadWorkloads: true,
            canReadLogs: true,
            canReadEvents: true,
            canReadTelemetry: true,
            canOperateWorkloads: true,
            canAdminister: true,
          },
          createdAt: new Date().toISOString(),
          updatedAt: new Date().toISOString(),
        },
      ],
    }
  }
}

export async function createCluster(data: CreateClusterRequest): Promise<{ cluster: ClusterConnection; agentToken: string }> {
  const res = await fetch(apiUrl("/api/clusters"), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  })
  if (!res.ok) {
    const err = await res.json().catch(() => ({}))
    throw new Error(err.error || `Failed to create cluster connection: ${res.status}`)
  }
  return res.json()
}

export async function deleteCluster(id: string): Promise<void> {
  const res = await fetch(apiUrl(`/api/clusters/${encodeURIComponent(id)}`), {
    method: "DELETE",
  })
  if (!res.ok) {
    const err = await res.json().catch(() => ({}))
    throw new Error(err.error || `Failed to delete cluster: ${res.status}`)
  }
}

export async function switchCluster(clusterId: string): Promise<void> {
  const res = await fetch(apiUrl("/api/clusters/switch"), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ clusterId }),
  })
  if (!res.ok) {
    const err = await res.json().catch(() => ({}))
    throw new Error(err.error || `Failed to switch cluster: ${res.status}`)
  }
}

export async function getAgentManifest(clusterId: string): Promise<AgentManifestResponse> {
  const res = await fetch(apiUrl(`/api/clusters/${encodeURIComponent(clusterId)}/manifest`), {
    cache: "no-store",
  })
  if (!res.ok) {
    throw new Error(`Failed to fetch agent manifest: ${res.status}`)
  }
  return res.json()
}

function withClusterQuery(url: string, clusterId?: string): string {
  if (!clusterId) return url
  const separator = url.includes("?") ? "&" : "?"
  return `${url}${separator}cluster=${encodeURIComponent(clusterId)}`
}

export async function getPods(clusterId?: string) {
  const res = await fetch(apiUrl(withClusterQuery("/pods", clusterId)), {
    cache: "no-store",
  })
  if (!res.ok) {
    throw new Error(`Failed to fetch pods: ${res.status}`)
  }
  return res.json()
}

export async function getNamespaces(clusterId?: string) {
  const res = await fetch(apiUrl(withClusterQuery("/namespaces", clusterId)), {
    cache: "no-store",
  })
  return res.json()
}

export async function getNamespaceList(clusterId?: string) {
  const res = await fetch(apiUrl(withClusterQuery("/namespace-list", clusterId)), {
    cache: "no-store",
  })
  return res.json()
}

export async function getDeployments(clusterId?: string) {
  const res = await fetch(apiUrl(withClusterQuery("/deployments", clusterId)), {
    cache: "no-store",
  })
  return res.json()
}

export async function getOverview(clusterId?: string): Promise<Overview> {
  const res = await fetch(apiUrl(withClusterQuery("/overview", clusterId)), {
    cache: "no-store",
  })

  if (!res.ok) {
    throw new Error(`Failed to fetch overview: ${res.status}`)
  }

  return res.json()
}

export async function getTopology(namespace?: string, clusterId?: string) {
  const params = new URLSearchParams()
  if (namespace) params.set("namespace", namespace)
  if (clusterId) params.set("cluster", clusterId)

  const query = params.toString() ? `?${params.toString()}` : ""
  const res = await fetch(apiUrl(`/topology${query}`), {
    cache: "no-store",
  })

  return res.json()
}

export async function getHealthScore(clusterId?: string) {
  const res = await fetch(apiUrl(withClusterQuery("/health-score", clusterId)), {
    cache: "no-store",
  })

  return res.json()
}

export async function getServices(clusterId?: string) {
  const res = await fetch(apiUrl(withClusterQuery("/services", clusterId)), {
    cache: "no-store",
  })

  if (!res.ok) {
    throw new Error(`Failed to fetch services: ${res.status}`)
  }

  return res.json()
}

export async function getEvents(clusterId?: string) {
  const res = await fetch(apiUrl(withClusterQuery("/events", clusterId)), {
    cache: "no-store",
  })

  if (!res.ok) {
    throw new Error(`Failed to fetch events: ${res.status}`)
  }

  return res.json()
}

export async function getResource(
  kind: string,
  namespace: string,
  name: string,
  clusterId?: string
) {
  const params = new URLSearchParams({
    kind,
    namespace,
    name,
  })
  if (clusterId) params.set("cluster", clusterId)

  const res = await fetch(
    apiUrl(`/resource?${params.toString()}`),
    {
      cache: "no-store",
    }
  )

  if (!res.ok) {
    throw new Error(
      `Failed to fetch ${kind}/${name}: ${res.status}`
    )
  }

  return res.json()
}

export async function getPodLogs(
  namespace: string,
  pod: string,
  container?: string,
  clusterId?: string
) {
  const params = new URLSearchParams({
    namespace,
    pod,
  })

  if (container) {
    params.set("container", container)
  }
  if (clusterId) {
    params.set("cluster", clusterId)
  }

  const res = await fetch(
    apiUrl(`/logs?${params.toString()}`),
    {
      cache: "no-store",
    }
  )

  if (!res.ok) {
    throw new Error(`Failed to fetch logs: ${res.status}`)
  }

  return res.json()
}

export async function getResourceEvents(
  namespace: string,
  name: string,
  kind?: string,
  clusterId?: string
): Promise<ResourceEvent[]> {
  const params = new URLSearchParams({
    namespace,
    name,
  })
  if (kind) params.set("kind", kind)
  if (clusterId) params.set("cluster", clusterId)

  const res = await fetch(
    apiUrl(`/resource-events?${params.toString()}`),
    {
      cache: "no-store",
    }
  )

  if (!res.ok) {
    throw new Error(`Failed to fetch resource events: ${res.status}`)
  }

  return res.json()
}

export async function analyzeResource(
  kind: string,
  namespace: string,
  name: string,
  clusterId?: string
) {
  let endpoint: string

  switch (kind.toLowerCase()) {
    case "pod":
      endpoint = "pod"
      break

    case "deployment":
      endpoint = "deployment"
      break

    default:
      throw new Error(`Analysis not supported for ${kind}`)
  }

  const params = new URLSearchParams({
    namespace,
    name,
  })
  if (clusterId) params.set("cluster", clusterId)

  const res = await fetch(
    apiUrl(`/analyze/${endpoint}?${params}`),
    {
      cache: "no-store",
    }
  )

  if (!res.ok) {
    throw new Error(`Analysis failed: ${res.status}`)
  }

  return res.json()
}

export async function restartPod(
  namespace: string,
  name: string,
  clusterId?: string
) {
  const query = clusterId ? `?cluster=${encodeURIComponent(clusterId)}` : ""
  const res = await fetch(
    apiUrl(`/pod/restart${query}`),
    {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        namespace,
        name,
      }),
    },
  )
  if (!res.ok) {
    const err = await res.json().catch(() => ({}))
    throw new Error(err.error || `Failed to restart pod: ${res.status}`)
  }
}

export async function deletePod(
  namespace: string,
  name: string,
  clusterId?: string
) {
  const params = new URLSearchParams({ namespace, name })
  if (clusterId) params.set("cluster", clusterId)

  const res = await fetch(
    apiUrl(`/pod?${params.toString()}`),
    {
      method: "DELETE",
    },
  )
  if (!res.ok) {
    const err = await res.json().catch(() => ({}))
    throw new Error(err.error || `Failed to delete pod: ${res.status}`)
  }
}

export async function getYaml(
  kind: string,
  namespace: string,
  name: string,
  clusterId?: string
) {
  const params = new URLSearchParams({ kind, namespace, name })
  if (clusterId) params.set("cluster", clusterId)
  return fetch(apiUrl(`/resource?${params.toString()}`)).then(r => r.json())
}

export async function describe(
  kind: string,
  namespace: string,
  name: string,
  clusterId?: string
) {
  const params = new URLSearchParams({ kind, namespace, name })
  if (clusterId) params.set("cluster", clusterId)
  return fetch(apiUrl(`/resource?${params.toString()}`)).then(r => r.json())
}

export async function searchResources(
  query: string,
  clusterId?: string
): Promise<SearchResult[]> {
  if (!query.trim()) {
    return []
  }

  const params = new URLSearchParams({ q: query })
  if (clusterId) params.set("cluster", clusterId)

  const response = await fetch(
    apiUrl(`/search?${params.toString()}`),
    {
      cache: "no-store",
    }
  )

  if (!response.ok) {
    throw new Error(`Search failed: ${response.status}`)
  }

  const data = await response.json()

  return data.results ?? []
}

export interface ReliabilityResponse {
  service: string
  namespace: string

  slis: SLI[]

  slo: SLO

  sla: SLA
}

export interface ResourceEvent {
  uid?: string
  type: string
  reason: string
  message: string
  count?: number
  namespace: string
  eventTime?: string
  involvedObject?: {
    kind?: string
    name?: string
    namespace?: string
  }
}

export async function getReliability(
  namespace = "",
  service = "",
  clusterId = ""
): Promise<ReliabilityResponse> {

  const params = new URLSearchParams()

  if (namespace) params.set("namespace", namespace)
  if (service) params.set("service", service)
  if (clusterId) params.set("cluster", clusterId)

  const response = await fetch(
    apiUrl(`/reliability?${params.toString()}`),
    {
      cache: "no-store",
    }
  )

  if (!response.ok) {
    throw new Error(`Failed to fetch reliability: ${response.status}`)
  }

  return response.json()
}

export interface ResourceHealth {
  total: number
  healthy: number
  unhealthy: number
}

export interface Overview {
  pods: number
  namespaces: number
  deployments: number
  replicaSets: number
  nodes: number
  services: number
  events: number

  health: {
    pods: ResourceHealth
    namespaces: ResourceHealth
    deployments: ResourceHealth
    replicaSets: ResourceHealth
    services: ResourceHealth
  }
}
