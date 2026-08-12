"use client"

import React, { useState, useEffect } from "react"
import {
  FullReliabilityOverview,
  EvaluatedSLI,
  EvaluatedSLO,
  EvaluatedSLA,
  GarundAlert,
  AlertPolicyItem,
  PrometheusStatus,
} from "@/lib/reliability"
import {
  getReliabilityOverview,
  getSLIs,
  getSLOs,
  getSLAs,
  getActiveAlerts,
  getPrometheusStatus,
  deleteSLI,
  deleteSLO,
  deleteSLA,
  createSLO,
  createSLA,
  createAlertPolicy,
  getAlertPolicies,
  deleteAlertPolicy,
  testAlertPolicy,
  updatePrometheusConfig,
} from "@/lib/api"
import { GuidedSLIModal } from "./GuidedSLIModal"
import { QueryExplainabilityDrawer } from "./QueryExplainabilityDrawer"
import {
  Activity,
  Shield,
  Bell,
  CheckCircle,
  AlertTriangle,
  XCircle,
  Plus,
  HelpCircle,
  Flame,
  Server,
  Zap,
  Trash2,
  ExternalLink,
  Search,
  RefreshCw,
  Clock,
  Radio,
  Sliders,
} from "lucide-react"

interface ReliabilityControlPlaneProps {
  clusterId: string
  onNavigateToResource?: (resourceType: string, name: string, namespace?: string) => void
}

type TabType = "overview" | "slis" | "slos" | "slas" | "alerts" | "integration"

export function ReliabilityControlPlane({
  clusterId,
  onNavigateToResource,
}: ReliabilityControlPlaneProps) {
  const [activeTab, setActiveTab] = useState<TabType>("overview")
  const [overview, setOverview] = useState<FullReliabilityOverview | null>(null)
  const [promStatus, setPromStatus] = useState<PrometheusStatus | null>(null)
  const [policies, setPolicies] = useState<AlertPolicyItem[]>([])
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)

  // Drawers and Modals
  const [explainableSLI, setExplainableSLI] = useState<EvaluatedSLI | null>(null)
  const [isSLIModalOpen, setIsSLIModalOpen] = useState(false)
  const [isSLOModalOpen, setIsSLOModalOpen] = useState(false)
  const [isSLAModalOpen, setIsSLAModalOpen] = useState(false)
  const [isPolicyModalOpen, setIsPolicyModalOpen] = useState(false)

  // Forms State
  const [sloName, setSloName] = useState("")
  const [sloService, setSloService] = useState("checkout")
  const [sloNamespace, setSloNamespace] = useState("default")
  const [sloSliId, setSloSliId] = useState("")
  const [sloTarget, setSloTarget] = useState(99.9)
  const [sloWindow, setSloWindow] = useState("30d")

  const [slaName, setSlaName] = useState("")
  const [slaService, setSlaService] = useState("checkout")
  const [slaNamespace, setSlaNamespace] = useState("default")
  const [slaAvailTarget, setSlaAvailTarget] = useState(99.9)
  const [slaWindow, setSlaWindow] = useState("30d")

  const [policyName, setPolicyName] = useState("")
  const [policyService, setPolicyService] = useState("checkout")
  const [policyThreshold, setPolicyThreshold] = useState(2.0)
  const [policySeverity, setPolicySeverity] = useState<"P1" | "P2" | "P3">("P1")

  const [activeAlertsList, setActiveAlertsList] = useState<GarundAlert[]>([])
  const [promUrlInput, setPromUrlInput] = useState("")

  const loadData = async () => {
    if (!clusterId) return
    try {
      setRefreshing(true)
      const [ovData, pStatus, polData, alertsData] = await Promise.all([
        getReliabilityOverview(clusterId).catch(() => {
          console.warn(`Reliability overview unavailable for cluster ${clusterId || "local-dev"} (backend offline or uninitialized)`)
          return {
            clusterId: clusterId || "local-dev",
            evaluatedAt: new Date().toISOString(),
            slis: [],
            slos: [],
            slas: [],
            summary: {
              overallHealthScore: 100,
              totalSlos: 0,
              healthySlos: 0,
              atRiskSlos: 0,
              exhaustedSlos: 0,
              activeAlerts: 0,
            },
          }
        }),
        getPrometheusStatus(clusterId).catch(() => null),
        getAlertPolicies(clusterId).catch(() => ({ policies: [] })),
        getActiveAlerts(clusterId).catch(() => ({ alerts: [] })),
      ])
      setOverview(ovData)
      setPromStatus(pStatus)
      setPolicies(polData.policies || [])
      setActiveAlertsList(alertsData.alerts || [])
      if (pStatus?.url) setPromUrlInput(pStatus.url)
    } catch (e) {
      console.error("Failed to load reliability data:", e)
    } finally {
      setLoading(false)
      setRefreshing(false)
    }
  }

  useEffect(() => {
    loadData()
    const interval = setInterval(loadData, 15000)
    return () => clearInterval(interval)
  }, [clusterId])

  const handleDeleteSLI = async (id: string) => {
    if (!confirm("Are you sure you want to delete this SLI?")) return
    await deleteSLI(clusterId, id)
    loadData()
  }

  const handleDeleteSLO = async (id: string) => {
    if (!confirm("Are you sure you want to delete this SLO?")) return
    await deleteSLO(clusterId, id)
    loadData()
  }

  const handleDeleteSLA = async (id: string) => {
    if (!confirm("Are you sure you want to delete this SLA?")) return
    await deleteSLA(clusterId, id)
    loadData()
  }

  const handleDeletePolicy = async (id: string) => {
    if (!confirm("Are you sure you want to delete this Alert Policy?")) return
    await deleteAlertPolicy(clusterId, id)
    loadData()
  }

  const handleCreateSLO = async (e: React.FormEvent) => {
    e.preventDefault()
    await createSLO(clusterId, {
      name: sloName,
      clusterId,
      service: sloService,
      namespace: sloNamespace,
      sliId: sloSliId || (overview?.slis[0]?.id || ""),
      target: Number(sloTarget),
      window: sloWindow,
      enabled: true,
    })
    setIsSLOModalOpen(false)
    loadData()
  }

  const handleCreateSLA = async (e: React.FormEvent) => {
    e.preventDefault()
    await createSLA(clusterId, {
      name: slaName,
      clusterId,
      service: slaService,
      namespace: slaNamespace,
      availabilityTarget: Number(slaAvailTarget),
      window: slaWindow,
    })
    setIsSLAModalOpen(false)
    loadData()
  }

  const handleCreatePolicy = async (e: React.FormEvent) => {
    e.preventDefault()
    await createAlertPolicy(clusterId, {
      name: policyName,
      clusterId,
      service: policyService,
      namespace: "default",
      conditionType: "burn_rate",
      threshold: Number(policyThreshold),
      duration: "15m",
      severity: policySeverity,
      enabled: true,
    })
    setIsPolicyModalOpen(false)
    loadData()
  }

  const handleUpdatePrometheus = async (e: React.FormEvent) => {
    e.preventDefault()
    await updatePrometheusConfig(clusterId, promUrlInput)
    loadData()
  }

  return (
    <div className="flex flex-col h-full bg-slate-950 text-slate-100 p-6 space-y-6 overflow-y-auto">
      {/* Header Bar */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 border-b border-slate-800 pb-5">
        <div>
          <div className="flex items-center space-x-3">
            <h1 className="text-xl font-bold tracking-tight text-slate-100 flex items-center gap-2">
              <Activity className="w-6 h-6 text-indigo-400" />
              SRE Reliability Control Plane
            </h1>
            <span className="px-2.5 py-0.5 text-xs font-mono font-medium rounded-full bg-indigo-500/10 text-indigo-400 border border-indigo-500/20">
              {clusterId}
            </span>
          </div>
          <p className="text-xs text-slate-400 mt-1">
            Telemetry, Error Budgets, Multi-Window Burn Rates, and Alertmanager Control
          </p>
        </div>

        {/* Integration Status Badge */}
        <div className="flex items-center space-x-3">
          <div className="flex items-center space-x-2 px-3 py-1.5 rounded-xl bg-slate-900 border border-slate-800 text-xs">
            <span
              className={`w-2 h-2 rounded-full ${promStatus?.status === "CONNECTED"
                ? "bg-emerald-400 animate-pulse"
                : "bg-amber-400"
                }`}
            />
            <span className="text-slate-300 font-medium">Prometheus:</span>
            <span className="font-mono text-slate-100">
              {promStatus?.status || "UNKNOWN"}
            </span>
          </div>

          <button
            onClick={loadData}
            disabled={refreshing}
            className="p-2 text-slate-400 hover:text-slate-200 bg-slate-900 hover:bg-slate-800 border border-slate-800 rounded-xl transition"
          >
            <RefreshCw className={`w-4 h-4 ${refreshing ? "animate-spin" : ""}`} />
          </button>
        </div>
      </div>

      {/* Navigation Sub-Tabs */}
      <div className="flex border-b border-slate-800 bg-slate-900/40 rounded-xl p-1">
        {[
          { id: "overview", label: "Overview", icon: Activity },
          { id: "slis", label: "SLIs & PromQL", icon: Sliders },
          { id: "slos", label: "SLOs & Error Budgets", icon: Shield },
          { id: "slas", label: "SLAs", icon: CheckCircle },
          { id: "alerts", label: "Alert Center", icon: Bell },
          { id: "integration", label: "Telemetry Connections", icon: Server },
        ].map((tab) => {
          const Icon = tab.icon
          const isActive = activeTab === tab.id
          return (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id as TabType)}
              className={`flex-1 py-2.5 px-3 text-xs font-semibold rounded-lg flex items-center justify-center gap-2 transition ${isActive
                ? "bg-indigo-600 text-white shadow-lg shadow-indigo-600/20"
                : "text-slate-400 hover:text-slate-200 hover:bg-slate-800/50"
                }`}
            >
              <Icon className="w-4 h-4" />
              {tab.label}
            </button>
          )
        })}
      </div>

      {/* TAB 1: OVERVIEW */}
      {activeTab === "overview" && (
        <div className="space-y-6">
          {/* Top Metric Cards */}
          <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
            <div className="p-5 rounded-2xl bg-gradient-to-br from-indigo-900/30 via-slate-900 to-slate-900 border border-indigo-500/20 space-y-2">
              <span className="text-xs text-slate-400 uppercase tracking-wider font-semibold">
                Overall Cluster Health
              </span>
              <div className="flex items-baseline space-x-2">
                <span className="text-3xl font-extrabold text-slate-100">
                  {overview?.summary.overallHealthScore ?? 100}%
                </span>
                <span className="text-xs text-emerald-400 font-medium">SLO Compliance</span>
              </div>
            </div>

            <div className="p-5 rounded-2xl bg-slate-900 border border-slate-800 space-y-2">
              <span className="text-xs text-slate-400 uppercase tracking-wider font-semibold">
                Active SLOs
              </span>
              <div className="flex items-baseline justify-between">
                <span className="text-3xl font-extrabold text-slate-100">
                  {overview?.summary.totalSlos ?? 0}
                </span>
                <div className="text-xs space-x-2">
                  <span className="text-emerald-400">{overview?.summary.healthySlos} Healthy</span>
                  <span className="text-amber-400">{overview?.summary.atRiskSlos} At Risk</span>
                </div>
              </div>
            </div>

            <div className="p-5 rounded-2xl bg-slate-900 border border-slate-800 space-y-2">
              <span className="text-xs text-slate-400 uppercase tracking-wider font-semibold">
                Exhausted Error Budgets
              </span>
              <div className="flex items-baseline space-x-2">
                <span className={`text-3xl font-extrabold ${(overview?.summary.exhaustedSlos ?? 0) > 0 ? "text-rose-400" : "text-slate-100"}`}>
                  {overview?.summary.exhaustedSlos ?? 0}
                </span>
                <span className="text-xs text-slate-400">SLOs Breached</span>
              </div>
            </div>

            <div className="p-5 rounded-2xl bg-slate-900 border border-slate-800 space-y-2">
              <span className="text-xs text-slate-400 uppercase tracking-wider font-semibold">
                Firing Alerts
              </span>
              <div className="flex items-baseline space-x-2">
                <span className={`text-3xl font-extrabold ${(overview?.summary.activeAlerts ?? 0) > 0 ? "text-amber-400" : "text-slate-100"}`}>
                  {overview?.summary.activeAlerts ?? 0}
                </span>
                <span className="text-xs text-slate-400">Alertmanager Incidents</span>
              </div>
            </div>
          </div>

          {/* Service Reliability Table */}
          <div className="p-6 rounded-2xl bg-slate-900 border border-slate-800 space-y-4">
            <h3 className="text-sm font-semibold text-slate-100 flex items-center gap-2">
              <Shield className="w-4 h-4 text-indigo-400" />
              Service Reliability Overview
            </h3>

            <div className="overflow-x-auto">
              <table className="w-full text-xs text-left">
                <thead className="text-slate-400 uppercase bg-slate-950/60 border-b border-slate-800">
                  <tr>
                    <th className="py-3 px-4">Service</th>
                    <th className="py-3 px-4">Namespace</th>
                    <th className="py-3 px-4">Target SLO</th>
                    <th className="py-3 px-4">Current SLI Value</th>
                    <th className="py-3 px-4">Error Budget Left</th>
                    <th className="py-3 px-4">Status</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-800/60">
                  {overview?.slos.map((slo) => (
                    <tr key={slo.id} className="hover:bg-slate-800/30 transition">
                      <td className="py-3 px-4 font-semibold text-slate-100 flex items-center gap-2">
                        <span>{slo.service}</span>
                        {onNavigateToResource && (
                          <button
                            onClick={() => onNavigateToResource("service", slo.service, slo.namespace)}
                            className="p-1 text-slate-500 hover:text-indigo-400 transition"
                            title="Navigate to Kubernetes Service"
                          >
                            <ExternalLink className="w-3.5 h-3.5" />
                          </button>
                        )}
                      </td>
                      <td className="py-3 px-4 text-slate-400 font-mono">{slo.namespace}</td>
                      <td className="py-3 px-4 font-mono text-slate-300">{slo.target}% ({slo.window})</td>
                      <td className="py-3 px-4 font-mono font-bold text-slate-100">
                        {slo.current !== null ? `${slo.current.toFixed(2)}%` : "N/A"}
                      </td>
                      <td className="py-3 px-4">
                        <div className="w-32 bg-slate-950 rounded-full h-2 overflow-hidden border border-slate-800">
                          <div
                            className={`h-full rounded-full ${slo.errorBudgetRemaining > 50
                              ? "bg-emerald-400"
                              : slo.errorBudgetRemaining > 20
                                ? "bg-amber-400"
                                : "bg-rose-500"
                              }`}
                            style={{ width: `${slo.errorBudgetRemaining}%` }}
                          />
                        </div>
                        <span className="text-[11px] font-mono text-slate-400 mt-1 block">
                          {slo.errorBudgetRemaining}%
                        </span>
                      </td>
                      <td className="py-3 px-4">
                        <span
                          className={`px-2.5 py-1 rounded-full text-[11px] font-semibold uppercase tracking-wider ${slo.status === "healthy"
                            ? "bg-emerald-500/10 text-emerald-400 border border-emerald-500/20"
                            : slo.status === "at_risk"
                              ? "bg-amber-500/10 text-amber-400 border border-amber-500/20"
                              : slo.status === "exhausted"
                                ? "bg-rose-500/10 text-rose-400 border border-rose-500/20"
                                : "bg-slate-800 text-slate-400"
                            }`}
                        >
                          {slo.status}
                        </span>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        </div>
      )}

      {/* TAB 2: SLIs & GUIDED BUILDER */}
      {activeTab === "slis" && (
        <div className="space-y-6">
          <div className="flex items-center justify-between">
            <h3 className="text-sm font-semibold text-slate-100 flex items-center gap-2">
              <Sliders className="w-4 h-4 text-indigo-400" />
              Service Level Indicators (SLIs)
            </h3>
            <button
              onClick={() => setIsSLIModalOpen(true)}
              className="px-3.5 py-2 text-xs font-semibold text-white bg-indigo-600 hover:bg-indigo-500 rounded-xl flex items-center gap-1.5 shadow-lg shadow-indigo-600/20 transition"
            >
              <Plus className="w-4 h-4" />
              Guided SLI Builder
            </button>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {overview?.slis.map((sli) => (
              <div key={sli.id} className="p-5 rounded-2xl bg-slate-900 border border-slate-800 space-y-4">
                <div className="flex items-start justify-between">
                  <div>
                    <div className="flex items-center space-x-2">
                      <h4 className="text-sm font-bold text-slate-100">{sli.name}</h4>
                      <span className="px-2 py-0.5 text-[10px] font-mono font-medium rounded uppercase bg-indigo-500/10 text-indigo-400 border border-indigo-500/20">
                        {sli.type}
                      </span>
                    </div>
                    <p className="text-xs text-slate-400 font-mono mt-0.5">
                      Window: {sli.evaluationWindow}
                    </p>
                  </div>
                  <div className="flex items-center space-x-2">
                    <button
                      onClick={() => setExplainableSLI(sli)}
                      className="p-1.5 text-slate-400 hover:text-indigo-400 bg-slate-950 border border-slate-800 rounded-lg transition"
                      title="Explain Query"
                    >
                      <HelpCircle className="w-4 h-4" />
                    </button>
                    <button
                      onClick={() => handleDeleteSLI(sli.id)}
                      className="p-1.5 text-slate-400 hover:text-rose-400 bg-slate-950 border border-slate-800 rounded-lg transition"
                      title="Delete SLI"
                    >
                      <Trash2 className="w-4 h-4" />
                    </button>
                  </div>
                </div>

                <div className="p-3.5 rounded-xl bg-slate-950 border border-slate-800/80 flex items-center justify-between">
                  <span className="text-xs text-slate-400">Current Measurement</span>
                  <span className="text-base font-extrabold font-mono text-slate-100">
                    {sli.value !== null ? `${sli.value.toFixed(2)}${sli.unit}` : "N/A"}
                  </span>
                </div>

                <div className="p-3 rounded-lg bg-slate-950/60 font-mono text-[11px] text-emerald-400/90 truncate border border-slate-800/50">
                  {sli.query}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* TAB 3: SLOs & ERROR BUDGETS */}
      {activeTab === "slos" && (
        <div className="space-y-6">
          <div className="flex items-center justify-between">
            <h3 className="text-sm font-semibold text-slate-100 flex items-center gap-2">
              <Shield className="w-4 h-4 text-indigo-400" />
              Service Level Objectives (SLOs) & Error Budgets
            </h3>
            <button
              onClick={() => setIsSLOModalOpen(true)}
              className="px-3.5 py-2 text-xs font-semibold text-white bg-indigo-600 hover:bg-indigo-500 rounded-xl flex items-center gap-1.5 shadow-lg shadow-indigo-600/20 transition"
            >
              <Plus className="w-4 h-4" />
              New SLO
            </button>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            {overview?.slos.map((slo) => (
              <div key={slo.id} className="p-6 rounded-2xl bg-slate-900 border border-slate-800 space-y-5">
                <div className="flex items-start justify-between">
                  <div>
                    <h4 className="text-base font-bold text-slate-100">{slo.name}</h4>
                    <p className="text-xs text-slate-400 mt-0.5">
                      Service: <span className="text-slate-200 font-mono">{slo.service}</span> ({slo.namespace})
                    </p>
                  </div>
                  <button
                    onClick={() => handleDeleteSLO(slo.id)}
                    className="p-1.5 text-slate-400 hover:text-rose-400 bg-slate-950 border border-slate-800 rounded-lg transition"
                  >
                    <Trash2 className="w-4 h-4" />
                  </button>
                </div>

                {/* Progress Bar */}
                <div className="space-y-2">
                  <div className="flex items-center justify-between text-xs">
                    <span className="text-slate-400">Error Budget Remaining:</span>
                    <span className="font-mono font-bold text-slate-100">
                      {slo.errorBudgetRemaining}% ({slo.remainingMinutes}m left)
                    </span>
                  </div>
                  <div className="w-full bg-slate-950 rounded-full h-3 overflow-hidden border border-slate-800">
                    <div
                      className={`h-full rounded-full transition-all duration-500 ${slo.errorBudgetRemaining > 50
                        ? "bg-emerald-400"
                        : slo.errorBudgetRemaining > 20
                          ? "bg-amber-400"
                          : "bg-rose-500"
                        }`}
                      style={{ width: `${slo.errorBudgetRemaining}%` }}
                    />
                  </div>
                </div>

                {/* Multi-Window Burn Rates */}
                <div className="p-4 rounded-xl bg-slate-950 border border-slate-800 space-y-2">
                  <span className="text-xs font-semibold text-slate-400 uppercase tracking-wider flex items-center gap-1.5">
                    <Flame className="w-3.5 h-3.5 text-amber-400" />
                    Multi-Window Burn Rates
                  </span>
                  <div className="grid grid-cols-3 gap-2 pt-1 text-center">
                    <div className="p-2 rounded-lg bg-slate-900 border border-slate-800">
                      <span className="text-[10px] text-slate-400 block">1h Burn</span>
                      <span className="text-xs font-mono font-bold text-amber-400">
                        {slo.burnRate.window1h !== null ? `${slo.burnRate.window1h}x` : "N/A"}
                      </span>
                    </div>
                    <div className="p-2 rounded-lg bg-slate-900 border border-slate-800">
                      <span className="text-[10px] text-slate-400 block">6h Burn</span>
                      <span className="text-xs font-mono font-bold text-slate-200">
                        {slo.burnRate.window6h !== null ? `${slo.burnRate.window6h}x` : "N/A"}
                      </span>
                    </div>
                    <div className="p-2 rounded-lg bg-slate-900 border border-slate-800">
                      <span className="text-[10px] text-slate-400 block">24h Burn</span>
                      <span className="text-xs font-mono font-bold text-slate-200">
                        {slo.burnRate.window24h !== null ? `${slo.burnRate.window24h}x` : "N/A"}
                      </span>
                    </div>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* TAB 4: SLAs */}
      {activeTab === "slas" && (
        <div className="space-y-6">
          <div className="flex items-center justify-between">
            <h3 className="text-sm font-semibold text-slate-100 flex items-center gap-2">
              <CheckCircle className="w-4 h-4 text-indigo-400" />
              Service Level Agreements (SLAs)
            </h3>
            <button
              onClick={() => setIsSLAModalOpen(true)}
              className="px-3.5 py-2 text-xs font-semibold text-white bg-indigo-600 hover:bg-indigo-500 rounded-xl flex items-center gap-1.5 shadow-lg shadow-indigo-600/20 transition"
            >
              <Plus className="w-4 h-4" />
              New SLA Commitment
            </button>
          </div>

          <div className="p-6 rounded-2xl bg-slate-900 border border-slate-800 space-y-4">
            <table className="w-full text-xs text-left">
              <thead className="text-slate-400 uppercase bg-slate-950/60 border-b border-slate-800">
                <tr>
                  <th className="py-3 px-4">SLA Name</th>
                  <th className="py-3 px-4">Service</th>
                  <th className="py-3 px-4">Customer Commitment</th>
                  <th className="py-3 px-4">Safety Margin</th>
                  <th className="py-3 px-4">Status</th>
                  <th className="py-3 px-4 text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-800/60">
                {overview?.slas.map((sla) => (
                  <tr key={sla.id} className="hover:bg-slate-800/30 transition">
                    <td className="py-3 px-4 font-semibold text-slate-100">{sla.name}</td>
                    <td className="py-3 px-4 text-slate-300 font-mono">{sla.service}</td>
                    <td className="py-3 px-4 font-mono text-slate-100">
                      {sla.availabilityTarget ? `${sla.availabilityTarget}%` : "N/A"} ({sla.window})
                    </td>
                    <td className="py-3 px-4 font-mono text-emerald-400 font-bold">
                      {sla.safetyMargin ? `+${sla.safetyMargin}%` : "N/A"}
                    </td>
                    <td className="py-3 px-4">
                      <span className="px-2.5 py-1 rounded-full text-[10px] font-semibold uppercase bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
                        {sla.status}
                      </span>
                    </td>
                    <td className="py-3 px-4 text-right">
                      <button
                        onClick={() => handleDeleteSLA(sla.id)}
                        className="p-1.5 text-slate-400 hover:text-rose-400 rounded-lg transition"
                      >
                        <Trash2 className="w-4 h-4" />
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* TAB 5: ALERT CENTER & POLICIES */}
      {activeTab === "alerts" && (
        <div className="space-y-6">
          <div className="flex items-center justify-between">
            <h3 className="text-sm font-semibold text-slate-100 flex items-center gap-2">
              <Bell className="w-4 h-4 text-indigo-400" />
              Alert Policies & Alertmanager Webhook Incidents
            </h3>
            <button
              onClick={() => setIsPolicyModalOpen(true)}
              className="px-3.5 py-2 text-xs font-semibold text-white bg-indigo-600 hover:bg-indigo-500 rounded-xl flex items-center gap-1.5 shadow-lg shadow-indigo-600/20 transition"
            >
              <Plus className="w-4 h-4" />
              New Alert Policy
            </button>
          </div>

          {/* Active Incidents */}
          <div className="p-6 rounded-2xl bg-slate-900 border border-slate-800 space-y-4">
            <div className="flex items-center justify-between">
              <h4 className="text-xs font-semibold text-slate-400 uppercase tracking-wider flex items-center gap-2">
                <Flame className="w-4 h-4 text-rose-400 animate-pulse" />
                Live Firing Alerts & Incidents ({activeAlertsList.length})
              </h4>
            </div>

            {activeAlertsList.length === 0 ? (
              <div className="p-8 text-center bg-slate-950/40 rounded-xl border border-slate-800/60 text-slate-400 text-xs">
                No active incidents firing for cluster <span className="font-mono text-indigo-400">{clusterId}</span>. All services operational.
              </div>
            ) : (
              <div className="space-y-3">
                {activeAlertsList.map((alert) => (
                  <div key={alert.fingerprint} className="p-4 rounded-xl bg-slate-950 border border-slate-800 flex items-start justify-between">
                    <div className="space-y-1">
                      <div className="flex items-center space-x-2">
                        <span className={`px-2 py-0.5 text-[10px] font-mono font-bold rounded ${alert.severity === "P1" ? "bg-rose-500/20 text-rose-400 border border-rose-500/30" : "bg-amber-500/20 text-amber-400 border border-amber-500/30"
                          }`}>
                          {alert.severity}
                        </span>
                        <h5 className="text-sm font-bold text-slate-100">{alert.name}</h5>
                        <span className="text-xs text-slate-400 font-mono">({alert.service} / {alert.namespace})</span>
                      </div>
                      <p className="text-xs text-slate-300">{alert.summary}</p>
                      {alert.description && (
                        <p className="text-[11px] text-slate-400 font-mono">{alert.description}</p>
                      )}
                    </div>
                    {onNavigateToResource && alert.service && (
                      <button
                        onClick={() => onNavigateToResource("service", alert.service, alert.namespace)}
                        className="px-3 py-1.5 text-xs font-medium text-indigo-400 hover:text-white bg-indigo-500/10 hover:bg-indigo-600 rounded-lg flex items-center gap-1 transition"
                      >
                        Investigate <ExternalLink className="w-3 h-3" />
                      </button>
                    )}
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* Configured Policies */}
          <div className="p-6 rounded-2xl bg-slate-900 border border-slate-800 space-y-4">
            <h4 className="text-xs font-semibold text-slate-400 uppercase tracking-wider">
              Configured Alert Policies
            </h4>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {policies.map((p) => (
                <div key={p.id} className="p-4 rounded-xl bg-slate-950 border border-slate-800 flex items-center justify-between">
                  <div>
                    <h5 className="text-sm font-bold text-slate-100">{p.name}</h5>
                    <p className="text-xs text-slate-400 font-mono mt-0.5">
                      Service: {p.service} | Condition: {p.conditionType} &gt; {p.threshold}x ({p.duration})
                    </p>
                  </div>
                  <div className="flex items-center space-x-2">
                    <span className="px-2 py-0.5 text-[10px] font-mono font-bold rounded bg-rose-500/10 text-rose-400 border border-rose-500/20">
                      {p.severity}
                    </span>
                    <button
                      onClick={async () => {
                        try {
                          await testAlertPolicy(clusterId, p.id!)
                          loadData()
                        } catch (e: any) {
                          alert(e.message)
                        }
                      }}
                      className="p-1.5 text-xs text-amber-400 hover:bg-amber-500/10 border border-amber-500/20 rounded-lg transition"
                      title="Trigger Test Alert"
                    >
                      <Zap className="w-3.5 h-3.5" />
                    </button>
                    <button
                      onClick={() => handleDeletePolicy(p.id!)}
                      className="p-1 text-slate-400 hover:text-rose-400 transition"
                    >
                      <Trash2 className="w-4 h-4" />
                    </button>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}

      {/* TAB 6: TELEMETRY CONNECTIONS */}
      {activeTab === "integration" && (
        <div className="space-y-6">
          <div className="p-6 rounded-2xl bg-slate-900 border border-slate-800 space-y-4">
            <h3 className="text-sm font-semibold text-slate-100 flex items-center gap-2">
              <Server className="w-4 h-4 text-indigo-400" />
              Prometheus & Alertmanager Connection Configuration
            </h3>

            <form onSubmit={handleUpdatePrometheus} className="space-y-4">
              <div className="space-y-1.5">
                <label className="text-xs font-medium text-slate-300">Prometheus Base URL</label>
                <div className="flex space-x-2">
                  <input
                    type="text"
                    value={promUrlInput}
                    onChange={(e) => setPromUrlInput(e.target.value)}
                    placeholder="http://localhost:9090"
                    className="flex-1 px-3.5 py-2 rounded-xl bg-slate-950 border border-slate-800 text-slate-100 text-xs font-mono focus:outline-none focus:border-indigo-500 transition"
                  />
                  <button
                    type="submit"
                    className="px-4 py-2 text-xs font-semibold text-white bg-indigo-600 hover:bg-indigo-500 rounded-xl transition"
                  >
                    Save Endpoint
                  </button>
                </div>
              </div>

              <div className="p-4 rounded-xl bg-slate-950 border border-slate-800 text-xs space-y-2">
                <div className="flex items-center justify-between text-slate-300">
                  <span>Alertmanager Webhook Webhook Ingestion URL:</span>
                  <span className="font-mono text-emerald-400">
                    http://&lt;garund-host&gt;/api/clusters/{clusterId}/alertmanager/webhook
                  </span>
                </div>
                <p className="text-slate-400 text-[11px]">
                  Configure this endpoint in your Alertmanager `receivers` block to route live incidents directly into Garund's Control Plane.
                </p>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* MODAL: GUIDED SLI */}
      <GuidedSLIModal
        clusterId={clusterId}
        isOpen={isSLIModalOpen}
        onClose={() => setIsSLIModalOpen(false)}
        onSaved={loadData}
      />

      {/* MODAL: NEW SLO */}
      {isSLOModalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/80 backdrop-blur-md p-4">
          <div className="w-full max-w-md bg-slate-900 border border-slate-800 rounded-2xl p-6 space-y-4">
            <h3 className="text-base font-bold text-slate-100">Create New SLO</h3>
            <form onSubmit={handleCreateSLO} className="space-y-4">
              <div className="space-y-1">
                <label className="text-xs text-slate-300">SLO Name</label>
                <input
                  type="text"
                  value={sloName}
                  onChange={(e) => setSloName(e.target.value)}
                  placeholder="Checkout 99.9% Availability"
                  className="w-full px-3 py-2 rounded-xl bg-slate-950 border border-slate-800 text-xs"
                  required
                />
              </div>
              <div className="space-y-1">
                <label className="text-xs text-slate-300">Target Service</label>
                <input
                  type="text"
                  value={sloService}
                  onChange={(e) => setSloService(e.target.value)}
                  className="w-full px-3 py-2 rounded-xl bg-slate-950 border border-slate-800 text-xs"
                  required
                />
              </div>
              <div className="space-y-1">
                <label className="text-xs text-slate-300">SLO Target (%)</label>
                <input
                  type="number"
                  step="0.01"
                  value={sloTarget}
                  onChange={(e) => setSloTarget(Number(e.target.value))}
                  className="w-full px-3 py-2 rounded-xl bg-slate-950 border border-slate-800 text-xs"
                  required
                />
              </div>
              <div className="flex justify-end space-x-2 pt-2">
                <button
                  type="button"
                  onClick={() => setIsSLOModalOpen(false)}
                  className="px-4 py-2 text-xs bg-slate-800 rounded-xl"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="px-4 py-2 text-xs font-semibold bg-indigo-600 text-white rounded-xl"
                >
                  Save SLO
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* DRAWER: EXPLAINABILITY */}
      <QueryExplainabilityDrawer
        sli={explainableSLI}
        onClose={() => setExplainableSLI(null)}
      />
    </div>
  )
}
