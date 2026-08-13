export type SLI = EvaluatedSLI
export type SLO = EvaluatedSLO
export type SLA = EvaluatedSLA

export type SLIType =
  | "availability"
  | "latency"
  | "error_rate"
  | "throughput"
  | "saturation"
  | "custom"

export interface EvaluatedSLI {
  id: string
  name: string
  type: SLIType
  value: number | null
  target: number | null
  unit: string
  goodEvents?: number
  totalEvents?: number
  evaluationWindow: string
  status: "healthy" | "warning" | "critical" | "unavailable"
  query: string
  goodQuery?: string
  totalQuery?: string
  evaluatedAt: string
}

export interface SLIItem {
  id?: string
  name: string
  description?: string
  clusterId: string
  service: string
  namespace: string
  type: SLIType
  target?: number
  query?: string
  goodQuery?: string
  totalQuery?: string
  unit: string
  evaluationWindow: string
  enabled: boolean
  createdAt?: string
  updatedAt?: string
}

export interface MultiWindowBurnRate {
  window1h: number | null
  window6h: number | null
  window24h: number | null
}

export interface EvaluatedSLO {
  id: string
  name: string
  service: string
  namespace: string
  sliId: string
  sliName: string
  sliType: SLIType
  target: number
  window: string
  current: number | null
  allowedError: number
  errorBudgetRemaining: number
  totalBudgetMinutes: number
  remainingMinutes: number
  consumedMinutes: number
  burnRate: MultiWindowBurnRate
  status: "healthy" | "at_risk" | "exhausted" | "unavailable"
  owner?: string
  team?: string
  evaluatedAt: string
}

export interface SLOItem {
  id?: string
  name: string
  description?: string
  clusterId: string
  service: string
  namespace: string
  sliId: string
  target: number
  window: string
  owner?: string
  team?: string
  enabled: boolean
  createdAt?: string
  updatedAt?: string
}

export interface EvaluatedSLA {
  id: string
  name: string
  service: string
  namespace: string
  availabilityTarget?: number
  latencyTargetMs?: number
  window: string
  safetyMargin?: number
  status: "compliant" | "at_risk" | "breached" | "unavailable"
  evaluatedAt: string
}

export interface SLAItem {
  id?: string
  name: string
  description?: string
  clusterId: string
  service: string
  namespace: string
  availabilityTarget?: number
  latencyTargetMs?: number
  window: string
  createdAt?: string
  updatedAt?: string
}

export interface AlertPolicyItem {
  id?: string
  name: string
  clusterId: string
  service: string
  namespace: string
  sloId?: string
  sliId?: string
  conditionType: string // burn_rate, sli_threshold, slo_breach, etc.
  threshold: number
  duration: string // 5m, 15m
  severity: "P1" | "P2" | "P3" | "P4"
  destinationId?: string
  enabled: boolean
  createdAt?: string
  updatedAt?: string
}

export interface GarundAlert {
  fingerprint: string
  name: string
  clusterId: string
  service: string
  namespace: string
  sloId?: string
  sliId?: string
  severity: "P1" | "P2" | "P3" | "P4"
  status: "firing" | "resolved" | "silenced"
  summary: string
  description: string
  startsAt: string
  endsAt?: string
  updatedAt: string
  labels: Record<string, string>
  annotations: Record<string, string>
  generatorUrl?: string
}

export interface PrometheusStatus {
  clusterId: string
  url: string
  status: "CONNECTED" | "DEGRADED" | "DISCONNECTED" | "AUTH_ERROR" | "UNKNOWN"
  version?: string
  lastError?: string
}

export interface FullReliabilityOverview {
  clusterId: string
  evaluatedAt: string
  slis: EvaluatedSLI[]
  slos: EvaluatedSLO[]
  slas: EvaluatedSLA[]
  summary: {
    overallHealthScore: number
    totalSlos: number
    healthySlos: number
    atRiskSlos: number
    exhaustedSlos: number
    activeAlerts: number
  }
}

export interface PromQLInput {
  type: SLIType
  metric?: string
  goodStatuses?: string[]
  badStatuses?: string[]
  percentile?: string
  window: string
  service: string
  namespace: string
  customQuery?: string
}

export interface QueryValidationResult {
  valid: boolean
  currentValue: number | null
  seriesCount: number
  evaluationMs: number
  errorMessage?: string
  hasData: boolean
  generatedPromQL: {
    query: string
    goodQuery?: string
    totalQuery?: string
  }
}
