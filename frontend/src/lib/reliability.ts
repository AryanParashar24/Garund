export type SLIType =
  | "availability"
  | "latency"
  | "error_rate"

export interface SLI {
  name: string
  type: SLIType

  value: number | null
  target: number

  unit: string

  goodEvents: number
  totalEvents: number

  window: string

  status:
    | "healthy"
    | "warning"
    | "critical"
    | "unavailable"
}

export interface SLO {
  name: string

  service: string
  namespace: string

  target: number

  window: string

  sliType: SLIType

  current: number | null

  errorBudget: number
  errorBudgetRemaining: number
  status:
    | "healthy"
    | "warning"
    | "critical"
    | "unavailable"
}

export interface SLA {
  name: string

  service: string
  namespace: string

  availabilityTarget?: number
  latencyTargetMs?: number

  window: string

  status:
    | "compliant"
    | "at_risk"
    | "breached"
    | "unavailable"
}
