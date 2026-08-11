export type ClusterStatus = "CONNECTED" | "DEGRADED" | "DISCONNECTED" | "AUTH_ERROR" | "UNKNOWN"
export type ConnectionMode = "local_kubeconfig" | "agent" | "service_account_token"

export interface CapabilitySet {
  canReadWorkloads: boolean
  canReadLogs: boolean
  canReadEvents: boolean
  canReadTelemetry: boolean
  canOperateWorkloads: boolean
  canAdminister: boolean
}

export interface ClusterConnection {
  id: string
  name: string
  environment: string
  provider: string
  clusterType: string
  connectionMode: ConnectionMode
  status: ClusterStatus
  endpoint?: string
  kubernetesVersion: string
  agentVersion?: string
  lastHeartbeat?: string
  latencyMs: number
  nodeCount: number
  namespaceCount: number
  capabilities: CapabilitySet
  createdAt: string
  updatedAt: string
}

export interface ClustersResponse {
  activeId: string
  clusters: ClusterConnection[]
}

export interface CreateClusterRequest {
  name: string
  environment: string
  provider: string
  clusterType: string
  connectionMode: ConnectionMode
  endpoint?: string
  bearerToken?: string
}

export interface AgentManifestResponse {
  clusterId: string
  manifest: string
  command: string
}
