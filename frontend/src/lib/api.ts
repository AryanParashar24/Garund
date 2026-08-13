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
  EvaluatedSLI,
  EvaluatedSLO,
  EvaluatedSLA,
  SLIItem,
  SLOItem,
  SLAItem,
  AlertPolicyItem,
  GarundAlert,
  PrometheusStatus,
  FullReliabilityOverview,
  PromQLInput,
  QueryValidationResult,
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

export async function getReliabilityOverview(clusterId: string): Promise<FullReliabilityOverview> {
  const cid = clusterId || "local-dev"
  const res = await fetch(apiUrl(`/api/clusters/${cid}/reliability/overview`), { cache: "no-store" })
  if (!res.ok) throw new Error(`Failed to fetch reliability overview: ${res.status}`)
  return res.json()
}

export async function getSLIs(clusterId: string): Promise<{ slis: EvaluatedSLI[] }> {
  const cid = clusterId || "local-dev"
  const res = await fetch(apiUrl(`/api/clusters/${cid}/slis`), { cache: "no-store" })
  if (!res.ok) throw new Error(`Failed to fetch SLIs: ${res.status}`)
  return res.json()
}

export async function createSLI(clusterId: string, sli: SLIItem): Promise<SLIItem> {
  const cid = clusterId || "local-dev"
  const res = await fetch(apiUrl(`/api/clusters/${cid}/slis`), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(sli),
  })
  if (!res.ok) throw new Error(`Failed to save SLI: ${res.status}`)
  return res.json()
}

export async function testSLIQuery(clusterId: string, input: PromQLInput): Promise<QueryValidationResult> {
  const cid = clusterId || "local-dev"
  const res = await fetch(apiUrl(`/api/clusters/${cid}/slis/test`), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input),
  })
  if (!res.ok) throw new Error(`Failed to test PromQL query: ${res.status}`)
  return res.json()
}

export async function updateSLI(clusterId: string, sliId: string, sli: Partial<SLIItem>): Promise<SLIItem> {
  const cid = clusterId || "local-dev"
  const res = await fetch(apiUrl(`/api/clusters/${cid}/slis/${sliId}`), {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(sli),
  })
  if (!res.ok) throw new Error(`Failed to update SLI: ${res.status}`)
  return res.json()
}

export async function deleteSLI(clusterId: string, sliId: string): Promise<void> {
  const cid = clusterId || "local-dev"
  const res = await fetch(apiUrl(`/api/clusters/${cid}/slis/${sliId}`), { method: "DELETE" })
  if (!res.ok) throw new Error(`Failed to delete SLI: ${res.status}`)
}

export async function getSLOs(clusterId: string): Promise<{ slos: EvaluatedSLO[] }> {
  const cid = clusterId || "local-dev"
  const res = await fetch(apiUrl(`/api/clusters/${cid}/slos`), { cache: "no-store" })
  if (!res.ok) throw new Error(`Failed to fetch SLOs: ${res.status}`)
  return res.json()
}

export async function createSLO(clusterId: string, slo: SLOItem): Promise<SLOItem> {
  const cid = clusterId || "local-dev"
  const res = await fetch(apiUrl(`/api/clusters/${cid}/slos`), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(slo),
  })
  if (!res.ok) throw new Error(`Failed to save SLO: ${res.status}`)
  return res.json()
}

export async function updateSLO(clusterId: string, sloId: string, slo: Partial<SLOItem>): Promise<SLOItem> {
  const cid = clusterId || "local-dev"
  const res = await fetch(apiUrl(`/api/clusters/${cid}/slos/${sloId}`), {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(slo),
  })
  if (!res.ok) throw new Error(`Failed to update SLO: ${res.status}`)
  return res.json()
}

export async function deleteSLO(clusterId: string, sloId: string): Promise<void> {
  const cid = clusterId || "local-dev"
  const res = await fetch(apiUrl(`/api/clusters/${cid}/slos/${sloId}`), { method: "DELETE" })
  if (!res.ok) throw new Error(`Failed to delete SLO: ${res.status}`)
}

export async function getSLAs(clusterId: string): Promise<{ slas: EvaluatedSLA[] }> {
  const cid = clusterId || "local-dev"
  const res = await fetch(apiUrl(`/api/clusters/${cid}/slas`), { cache: "no-store" })
  if (!res.ok) throw new Error(`Failed to fetch SLAs: ${res.status}`)
  return res.json()
}

export async function createSLA(clusterId: string, sla: SLAItem): Promise<SLAItem> {
  const cid = clusterId || "local-dev"
  const res = await fetch(apiUrl(`/api/clusters/${cid}/slas`), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(sla),
  })
  if (!res.ok) throw new Error(`Failed to save SLA: ${res.status}`)
  return res.json()
}

export async function updateSLA(clusterId: string, slaId: string, sla: Partial<SLAItem>): Promise<SLAItem> {
  const cid = clusterId || "local-dev"
  const res = await fetch(apiUrl(`/api/clusters/${cid}/slas/${slaId}`), {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(sla),
  })
  if (!res.ok) throw new Error(`Failed to update SLA: ${res.status}`)
  return res.json()
}

export async function deleteSLA(clusterId: string, slaId: string): Promise<void> {
  const cid = clusterId || "local-dev"
  const res = await fetch(apiUrl(`/api/clusters/${cid}/slas/${slaId}`), { method: "DELETE" })
  if (!res.ok) throw new Error(`Failed to delete SLA: ${res.status}`)
}

export async function getAlertPolicies(clusterId: string): Promise<{ policies: AlertPolicyItem[] }> {
  const cid = clusterId || "local-dev"
  const res = await fetch(apiUrl(`/api/clusters/${cid}/alert-policies`), { cache: "no-store" })
  if (!res.ok) throw new Error(`Failed to fetch alert policies: ${res.status}`)
  const data = await res.json()
  if (Array.isArray(data)) {
    return { policies: data }
  }
  return data
}

export async function createAlertPolicy(clusterId: string, policy: AlertPolicyItem): Promise<AlertPolicyItem> {
  const cid = clusterId || "local-dev"
  const res = await fetch(apiUrl(`/api/clusters/${cid}/alert-policies`), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(policy),
  })
  if (!res.ok) throw new Error(`Failed to save alert policy: ${res.status}`)
  return res.json()
}

export async function updateAlertPolicy(clusterId: string, policyId: string, policy: Partial<AlertPolicyItem>): Promise<AlertPolicyItem> {
  const cid = clusterId || "local-dev"
  const res = await fetch(apiUrl(`/api/clusters/${cid}/alert-policies/${policyId}`), {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(policy),
  })
  if (!res.ok) throw new Error(`Failed to update alert policy: ${res.status}`)
  return res.json()
}

export async function deleteAlertPolicy(clusterId: string, policyId: string): Promise<void> {
  const cid = clusterId || "local-dev"
  const res = await fetch(apiUrl(`/api/clusters/${cid}/alert-policies/${policyId}`), { method: "DELETE" })
  if (!res.ok) throw new Error(`Failed to delete alert policy: ${res.status}`)
}

export async function getActiveAlerts(clusterId: string, status = ""): Promise<{ alerts: GarundAlert[] }> {
  const cid = clusterId || "local-dev"
  const query = status ? `?status=${status}` : ""
  const res = await fetch(apiUrl(`/api/clusters/${cid}/alerts/active${query}`), { cache: "no-store" })
  if (!res.ok) throw new Error(`Failed to fetch active alerts: ${res.status}`)
  return res.json()
}

export async function getPrometheusStatus(clusterId: string): Promise<PrometheusStatus> {
  const cid = clusterId || "local-dev"
  const res = await fetch(apiUrl(`/api/clusters/${cid}/prometheus/status`), { cache: "no-store" })
  if (!res.ok) throw new Error(`Failed to fetch Prometheus status: ${res.status}`)
  return res.json()
}

export async function updatePrometheusConfig(clusterId: string, url: string): Promise<void> {
  const cid = clusterId || "local-dev"
  const res = await fetch(apiUrl(`/api/clusters/${cid}/prometheus/config`), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ url }),
  })
  if (!res.ok) throw new Error(`Failed to update Prometheus config: ${res.status}`)
}

export async function getPrometheusMetrics(clusterId: string): Promise<{ metrics: string[] }> {
  const cid = clusterId || "local-dev"
  const res = await fetch(apiUrl(`/api/clusters/${cid}/prometheus/metrics`), { cache: "no-store" })
  if (!res.ok) throw new Error(`Failed to fetch Prometheus metrics: ${res.status}`)
  return res.json()
}

export async function getReliabilityHistory(clusterId: string, sliId = ""): Promise<{ points: { timestamp: number; value: number }[] }> {
  const cid = clusterId || "local-dev"
  const q = sliId ? `?sliId=${sliId}` : ""
  const res = await fetch(apiUrl(`/api/clusters/${cid}/reliability/history${q}`), { cache: "no-store" })
  if (!res.ok) throw new Error(`Failed to fetch reliability history: ${res.status}`)
  return res.json()
}

export async function getAlertmanagerStatus(clusterId: string): Promise<{ clusterId: string; url: string; status: string; lastError?: string }> {
  const cid = clusterId || "local-dev"
  const res = await fetch(apiUrl(`/api/clusters/${cid}/alertmanager/status`), { cache: "no-store" })
  if (!res.ok) throw new Error(`Failed to fetch Alertmanager status: ${res.status}`)
  return res.json()
}

export async function updateAlertmanagerConfig(clusterId: string, url: string): Promise<void> {
  const cid = clusterId || "local-dev"
  const res = await fetch(apiUrl(`/api/clusters/${cid}/alertmanager/config`), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ url }),
  })
  if (!res.ok) throw new Error(`Failed to update Alertmanager config: ${res.status}`)
}

export async function getNotificationDestinations(clusterId: string) {
  const cid = clusterId || "local-dev"
  const res = await fetch(apiUrl(`/api/clusters/${cid}/destinations`), { cache: "no-store" })
  if (!res.ok) throw new Error(`Failed to fetch notification destinations: ${res.status}`)
  return res.json()
}

export async function createNotificationDestination(clusterId: string, dest: any) {
  const cid = clusterId || "local-dev"
  const res = await fetch(apiUrl(`/api/clusters/${cid}/destinations`), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(dest),
  })
  if (!res.ok) throw new Error(`Failed to create destination: ${res.status}`)
  return res.json()
}

export async function deleteNotificationDestination(clusterId: string, destId: string): Promise<void> {
  const cid = clusterId || "local-dev"
  const res = await fetch(apiUrl(`/api/clusters/${cid}/destinations/${destId}`), { method: "DELETE" })
  if (!res.ok) throw new Error(`Failed to delete destination: ${res.status}`)
}

export async function testAlertPolicy(clusterId: string, policyId: string): Promise<{ message: string; alert: GarundAlert }> {
  const cid = clusterId || "local-dev"
  const res = await fetch(apiUrl(`/api/clusters/${cid}/alert-policies/${policyId}/test`), { method: "POST" })
  if (!res.ok) throw new Error(`Failed to trigger test alert: ${res.status}`)
  return res.json()
}

