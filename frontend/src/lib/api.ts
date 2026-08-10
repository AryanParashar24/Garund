import { SearchResult } from "@/components/search/types"
import { apiUrl } from "@/lib/config"

import {
  SLO,
  SLA,
  SLI,
} from "./reliability"

export async function getPods() {
  const res = await fetch(apiUrl("/pods"), {
    cache: "no-store",
  })
  if (!res.ok) {
    throw new Error(
      `Failed to fetch pods: ${res.status}`
    )
  }

  return res.json()
}

export async function getNamespaces() {
  const res = await fetch(apiUrl("/namespaces"), {
    cache: "no-store",
  })
  return res.json()
}

export async function getNamespaceList() {
  const res = await fetch(apiUrl("/namespace-list"), {
    cache: "no-store",
  })
  return res.json()
}

export async function getDeployments() {
  const res = await fetch(apiUrl("/deployments"), {
    cache: "no-store",
  })
  return res.json()
}

export async function getOverview(): Promise<Overview> {
  const res = await fetch(apiUrl("/overview"), {
    cache: "no-store",
  })

  if (!res.ok) {
    throw new Error(
      `Failed to fetch overview: ${res.status}`
    )
  }

  return res.json()
}

export async function getTopology(namespace?: string) {
  const params = namespace
    ? `?namespace=${encodeURIComponent(namespace)}`
    : ""

  const res = await fetch(apiUrl(`/topology${params}`), {
    cache: "no-store",
  })

  return res.json()
}

export async function getHealthScore() {
  const res = await fetch(apiUrl("/health-score"), {
    cache: "no-store",
  })

  return res.json()
}

export async function getMetrics() {
  const res = await fetch(apiUrl("/metrics"), {
    cache: "no-store",
  })

  return res.json()
}

export async function getServices() {
  const res = await fetch(apiUrl("/services"), {
    cache: "no-store",
  })

  if (!res.ok) {
    throw new Error(`Failed to fetch services: ${res.status}`)
  }

  return res.json()
}

export async function getEvents() {
  const res = await fetch(apiUrl("/events"), {
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
  name: string
) {
  const params = new URLSearchParams({
    kind,
    namespace,
    name,
  })

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
  container?: string
) {
  const params = new URLSearchParams({
    namespace,
    pod,
  })

  if (container) {
    params.set("container", container)
  }

  const res = await fetch(
    apiUrl(`/logs?${params.toString()}`),
    {
      cache: "no-store",
    }
  )

  if (!res.ok) {
    throw new Error(
      `Failed to fetch logs: ${res.status}`
    )
  }

  return res.json()
}

export async function getResourceEvents(
  namespace: string,
  name: string
) {
  const params = new URLSearchParams({
    namespace,
    name,
  })

  const res = await fetch(
    apiUrl(`/resource-events?${params.toString()}`),
    {
      cache: "no-store",
    }
  )

  if (!res.ok) {
    throw new Error(
      `Failed to fetch resource events: ${res.status}`
    )
  }

  return res.json()
}

export async function analyzeResource(
  kind: string,
  namespace: string,
  name: string
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
      throw new Error(
        `Analysis not supported for ${kind}`
      )
  }

  const params = new URLSearchParams({
    namespace,
    name,
  })

  const res = await fetch(
    apiUrl(`/analyze/${endpoint}?${params}`),
    {
      cache: "no-store",
    }
  )

  if (!res.ok) {
    throw new Error(
      `Analysis failed: ${res.status}`
    )
  }

  return res.json()
}

export async function restartPod(
  namespace: string,
  name: string,
) {
  await fetch(
    apiUrl("/pod/restart"),
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
}

export async function deletePod(
  namespace: string,
  name: string,
) {
  await fetch(
    apiUrl(`/pod?namespace=${namespace}&name=${name}`),
    {
      method: "DELETE",
    },
  )
}

export async function getYaml(
  kind: string,
  namespace: string,
  name: string,
) {
  return fetch(
    apiUrl(`/resource?kind=${kind}&namespace=${namespace}&name=${name}`)
  ).then(r => r.json())
}

export async function describe(
  kind: string,
  namespace: string,
  name: string,
) {
  return fetch(
    apiUrl(`/resource?kind=${kind}&namespace=${namespace}&name=${name}`)
  ).then(r => r.json())
}

export async function searchResources(
  query: string
): Promise<SearchResult[]> {
  if (!query.trim()) {
    return []
  }

  const response = await fetch(
    apiUrl(`/search?q=${encodeURIComponent(query)}`),
    {
      cache: "no-store",
    }
  )

  if (!response.ok) {
    throw new Error(
      `Search failed: ${response.status}`
    )
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

export async function getReliability(
  namespace = "",
  service = ""
): Promise<ReliabilityResponse> {

  const params =
    new URLSearchParams()

  if (namespace) {
    params.set(
      "namespace",
      namespace
    )
  }

  if (service) {
    params.set(
      "service",
      service
    )
  }

  const response =
    await fetch(
      apiUrl(`/reliability?${params.toString()}`),
      {
        cache: "no-store",
      }
    )

  if (!response.ok) {
    throw new Error(
      `Failed to fetch reliability: ${response.status}`
    )
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
