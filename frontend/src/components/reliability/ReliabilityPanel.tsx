"use client"

import { useEffect, useState } from "react"
import { getReliability, ReliabilityResponse } from "@/lib/api"
import { ReliabilityControlPlane } from "./ReliabilityControlPlane"
import { Shield, Activity, AlertTriangle, CheckCircle } from "lucide-react"

interface ReliabilityPanelProps {
  namespace?: string
  service?: string
  clusterId?: string
}

export function ReliabilityPanel({
  namespace = "",
  service = "",
  clusterId = "local-dev",
}: ReliabilityPanelProps) {
  // If no service specific filter, render full SRE Control Plane!
  if (!service && !namespace) {
    return <ReliabilityControlPlane clusterId={clusterId || "local-dev"} />
  }

  const [reliability, setReliability] = useState<ReliabilityResponse | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let cancelled = false
    async function load() {
      try {
        setLoading(true)
        const data = await getReliability(namespace, service, clusterId)
        if (!cancelled) setReliability(data)
      } catch (error) {
        console.error("Reliability fetch failed:", error)
      } finally {
        if (!cancelled) setLoading(false)
      }
    }
    load()
    return () => {
      cancelled = true
    }
  }, [namespace, service, clusterId])

  if (loading) {
    return (
      <section className="overflow-hidden rounded-2xl border border-slate-800 bg-slate-900/40 p-6">
        <div className="text-xs text-slate-400 font-medium animate-pulse flex items-center gap-2">
          <Activity className="w-4 h-4 text-indigo-400 animate-spin" />
          Loading telemetry and error budgets...
        </div>
      </section>
    )
  }

  if (!reliability) return null

  return (
    <section className="overflow-hidden rounded-2xl border border-slate-800 bg-slate-900/80 p-6 space-y-4 shadow-xl">
      <div className="flex items-center justify-between border-b border-slate-800 pb-3">
        <div className="flex items-center space-x-2">
          <Shield className="w-5 h-5 text-indigo-400" />
          <h3 className="text-sm font-bold text-slate-100">
            {reliability.service || "Service"} Reliability
          </h3>
          <span className="text-xs text-slate-400 font-mono">({reliability.namespace || "default"})</span>
        </div>
        <span className="text-xs font-mono text-slate-400">Window: {reliability.slo?.window || "30d"}</span>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        {reliability.slis.map((sli) => (
          <div key={sli.name} className="p-4 rounded-xl bg-slate-950 border border-slate-800 space-y-2">
            <div className="flex items-center justify-between">
              <span className="text-xs font-semibold text-slate-300">{sli.name}</span>
              <span
                className={`px-2 py-0.5 rounded text-[10px] font-mono font-bold uppercase ${
                  sli.status === "healthy"
                    ? "bg-emerald-500/10 text-emerald-400 border border-emerald-500/20"
                    : sli.status === "warning"
                    ? "bg-amber-500/10 text-amber-400 border border-amber-500/20"
                    : sli.status === "critical"
                    ? "bg-rose-500/10 text-rose-400 border border-rose-500/20"
                    : "bg-slate-800 text-slate-400"
                }`}
              >
                {sli.status}
              </span>
            </div>
            <div className="text-lg font-bold font-mono text-slate-100">
              {sli.value !== null ? `${sli.value.toFixed(2)}${sli.unit}` : "N/A"}
            </div>
          </div>
        ))}
      </div>
    </section>
  )
}
