"use client"

import React, { useState, useEffect } from "react"
import { SLIItem, SLIType, QueryValidationResult } from "@/lib/reliability"
import { createSLI, testSLIQuery, getPrometheusMetrics } from "@/lib/api"
import { X, Play, Code, CheckCircle, AlertTriangle, Sparkles, Layers } from "lucide-react"

interface GuidedSLIModalProps {
  clusterId: string
  isOpen: boolean
  onClose: () => void
  onSaved: () => void
}

export function GuidedSLIModal({
  clusterId,
  isOpen,
  onClose,
  onSaved,
}: GuidedSLIModalProps) {
  const [tab, setTab] = useState<"guided" | "advanced">("guided")
  const [name, setName] = useState("")
  const [description, setDescription] = useState("")
  const [service, setService] = useState("checkout")
  const [namespace, setNamespace] = useState("default")
  const [type, setType] = useState<SLIType>("availability")
  const [metric, setMetric] = useState("http_requests_total")
  const [percentile, setPercentile] = useState("p95")
  const [window, setWindow] = useState("5m")
  const [customQuery, setCustomQuery] = useState("")
  const [availableMetrics, setAvailableMetrics] = useState<string[]>([])

  const [testResult, setTestResult] = useState<QueryValidationResult | null>(null)
  const [testing, setTesting] = useState(false)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState("")

  useEffect(() => {
    if (isOpen && clusterId) {
      getPrometheusMetrics(clusterId)
        .then((res) => setAvailableMetrics(res.metrics || []))
        .catch(() => {})
    }
  }, [isOpen, clusterId])

  if (!isOpen) return null

  const handleTestQuery = async () => {
    setTesting(true)
    setError("")
    try {
      const input = {
        type,
        metric,
        percentile,
        window,
        service,
        namespace,
        customQuery: tab === "advanced" ? customQuery : undefined,
      }
      const res = await testSLIQuery(clusterId, input)
      setTestResult(res)
    } catch (e: any) {
      setError(e.message || "Failed to test query")
    } finally {
      setTesting(false)
    }
  }

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!name) {
      setError("SLI Name is required")
      return
    }

    setSaving(true)
    setError("")

    try {
      const unit = type === "availability" || type === "error_rate" ? "%" : type === "latency" ? "ms" : "req/s"
      const item: SLIItem = {
        name,
        description,
        clusterId,
        service,
        namespace,
        type: tab === "advanced" ? "custom" : type,
        query: tab === "advanced" ? customQuery : undefined,
        unit,
        evaluationWindow: window,
        enabled: true,
      }

      await createSLI(clusterId, item)
      onSaved()
      onClose()
    } catch (e: any) {
      setError(e.message || "Failed to save SLI")
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/80 backdrop-blur-md p-4 overflow-y-auto">
      <div className="relative w-full max-w-2xl bg-slate-900 border border-slate-800 rounded-2xl shadow-2xl overflow-hidden flex flex-col my-8">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-slate-800 bg-slate-950/40">
          <div className="flex items-center space-x-3">
            <div className="p-2 rounded-xl bg-indigo-500/10 text-indigo-400 border border-indigo-500/20">
              <Sparkles className="w-5 h-5" />
            </div>
            <div>
              <h3 className="text-base font-semibold text-slate-100">Guided SLI Builder</h3>
              <p className="text-xs text-slate-400">Build deterministic PromQL SLI measurements</p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="p-1.5 text-slate-400 hover:text-slate-200 hover:bg-slate-800 rounded-lg transition"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Mode Switcher */}
        <div className="flex border-b border-slate-800 bg-slate-950/20 px-6 pt-3">
          <button
            type="button"
            onClick={() => setTab("guided")}
            className={`pb-3 px-4 text-xs font-semibold flex items-center gap-2 border-b-2 transition ${
              tab === "guided"
                ? "border-indigo-500 text-indigo-400"
                : "border-transparent text-slate-400 hover:text-slate-200"
            }`}
          >
            <Layers className="w-4 h-4" />
            Guided Builder
          </button>
          <button
            type="button"
            onClick={() => setTab("advanced")}
            className={`pb-3 px-4 text-xs font-semibold flex items-center gap-2 border-b-2 transition ${
              tab === "advanced"
                ? "border-indigo-500 text-indigo-400"
                : "border-transparent text-slate-400 hover:text-slate-200"
            }`}
          >
            <Code className="w-4 h-4" />
            Advanced PromQL
          </button>
        </div>

        {/* Form Body */}
        <form onSubmit={handleSave} className="p-6 space-y-5 flex-1 overflow-y-auto">
          {error && (
            <div className="p-3.5 rounded-xl bg-rose-500/10 border border-rose-500/20 text-rose-300 text-xs flex items-center gap-2">
              <AlertTriangle className="w-4 h-4 shrink-0" />
              <span>{error}</span>
            </div>
          )}

          {/* Basic Information */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-slate-300">SLI Name *</label>
              <input
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="e.g. Checkout Availability"
                className="w-full px-3.5 py-2 rounded-xl bg-slate-950 border border-slate-800 text-slate-100 text-xs focus:outline-none focus:border-indigo-500 transition"
                required
              />
            </div>

            <div className="space-y-1.5">
              <label className="text-xs font-medium text-slate-300">Service Name *</label>
              <input
                type="text"
                value={service}
                onChange={(e) => setService(e.target.value)}
                placeholder="checkout"
                className="w-full px-3.5 py-2 rounded-xl bg-slate-950 border border-slate-800 text-slate-100 text-xs focus:outline-none focus:border-indigo-500 transition"
                required
              />
            </div>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-slate-300">Target Namespace</label>
              <input
                type="text"
                value={namespace}
                onChange={(e) => setNamespace(e.target.value)}
                placeholder="default"
                className="w-full px-3.5 py-2 rounded-xl bg-slate-950 border border-slate-800 text-slate-100 text-xs focus:outline-none focus:border-indigo-500 transition"
              />
            </div>

            <div className="space-y-1.5">
              <label className="text-xs font-medium text-slate-300">Evaluation Window</label>
              <select
                value={window}
                onChange={(e) => setWindow(e.target.value)}
                className="w-full px-3.5 py-2 rounded-xl bg-slate-950 border border-slate-800 text-slate-100 text-xs focus:outline-none focus:border-indigo-500 transition"
              >
                <option value="5m">5 Minutes (5m)</option>
                <option value="15m">15 Minutes (15m)</option>
                <option value="1h">1 Hour (1h)</option>
                <option value="24h">24 Hours (24h)</option>
              </select>
            </div>
          </div>

          {tab === "guided" ? (
            <div className="space-y-4 pt-2">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div className="space-y-1.5">
                  <label className="text-xs font-medium text-slate-300">SLI Metric Category</label>
                  <select
                    value={type}
                    onChange={(e) => setType(e.target.value as SLIType)}
                    className="w-full px-3.5 py-2 rounded-xl bg-slate-950 border border-slate-800 text-slate-100 text-xs focus:outline-none focus:border-indigo-500 transition"
                  >
                    <option value="availability">Availability (%)</option>
                    <option value="error_rate">Error Rate (%)</option>
                    <option value="latency">Response Latency (ms)</option>
                    <option value="throughput">Throughput (req/s)</option>
                    <option value="saturation">Saturation (utilization)</option>
                  </select>
                </div>

                <div className="space-y-1.5">
                  <label className="text-xs font-medium text-slate-300">Base Metric Name</label>
                  <input
                    type="text"
                    list="metrics-list"
                    value={metric}
                    onChange={(e) => setMetric(e.target.value)}
                    placeholder="http_requests_total"
                    className="w-full px-3.5 py-2 rounded-xl bg-slate-950 border border-slate-800 text-slate-100 text-xs focus:outline-none focus:border-indigo-500 transition"
                  />
                  <datalist id="metrics-list">
                    {availableMetrics.map((m) => (
                      <option key={m} value={m} />
                    ))}
                  </datalist>
                </div>
              </div>

              {type === "latency" && (
                <div className="space-y-1.5">
                  <label className="text-xs font-medium text-slate-300">Percentile Quantile</label>
                  <div className="flex space-x-2">
                    {["p50", "p90", "p95", "p99"].map((p) => (
                      <button
                        type="button"
                        key={p}
                        onClick={() => setPercentile(p)}
                        className={`px-3 py-1.5 rounded-lg text-xs font-mono font-medium border transition ${
                          percentile === p
                            ? "bg-indigo-500/20 text-indigo-300 border-indigo-500/40"
                            : "bg-slate-950 text-slate-400 border-slate-800 hover:text-slate-200"
                        }`}
                      >
                        {p.toUpperCase()}
                      </button>
                    ))}
                  </div>
                </div>
              )}
            </div>
          ) : (
            <div className="space-y-2 pt-2">
              <label className="text-xs font-medium text-slate-300">Custom PromQL Query</label>
              <textarea
                rows={4}
                value={customQuery}
                onChange={(e) => setCustomQuery(e.target.value)}
                placeholder="sum(rate(http_requests_total{status=~'2..'}[5m])) / sum(rate(http_requests_total[5m])) * 100"
                className="w-full p-3.5 rounded-xl bg-slate-950 border border-slate-800 text-emerald-300 font-mono text-xs focus:outline-none focus:border-indigo-500 transition"
              />
            </div>
          )}

          {/* Test Query Action */}
          <div className="pt-2 border-t border-slate-800/80 space-y-3">
            <div className="flex items-center justify-between">
              <span className="text-xs font-semibold text-slate-400 uppercase tracking-wider">
                Live PromQL Validation
              </span>
              <button
                type="button"
                onClick={handleTestQuery}
                disabled={testing}
                className="px-3 py-1.5 text-xs font-medium text-emerald-300 bg-emerald-500/10 hover:bg-emerald-500/20 border border-emerald-500/20 rounded-lg flex items-center gap-1.5 transition"
              >
                <Play className="w-3.5 h-3.5 fill-current" />
                {testing ? "Executing..." : "Test Query"}
              </button>
            </div>

            {testResult && (
              <div
                className={`p-4 rounded-xl border text-xs space-y-2 transition ${
                  testResult.valid
                    ? "bg-emerald-500/5 border-emerald-500/20 text-emerald-300"
                    : "bg-rose-500/5 border-rose-500/20 text-rose-300"
                }`}
              >
                <div className="flex items-center justify-between font-semibold">
                  <div className="flex items-center gap-2">
                    {testResult.valid ? (
                      <CheckCircle className="w-4 h-4 text-emerald-400" />
                    ) : (
                      <AlertTriangle className="w-4 h-4 text-rose-400" />
                    )}
                    <span>{testResult.valid ? "Query Valid & Executable" : "Query Failed"}</span>
                  </div>
                  <span className="font-mono text-slate-400">{testResult.evaluationMs}ms</span>
                </div>

                {testResult.valid && (
                  <div className="flex items-center justify-between pt-1 border-t border-slate-800 text-slate-300">
                    <span>Evaluated Value:</span>
                    <span className="font-mono font-bold text-slate-100">
                      {testResult.currentValue !== null ? testResult.currentValue.toFixed(2) : "N/A (No Series)"}
                    </span>
                  </div>
                )}

                {testResult.errorMessage && (
                  <p className="text-slate-400 font-mono text-[11px]">{testResult.errorMessage}</p>
                )}
              </div>
            )}
          </div>

          {/* Footer Actions */}
          <div className="flex items-center justify-end space-x-3 pt-4 border-t border-slate-800">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 text-xs font-medium text-slate-400 hover:text-slate-200 bg-slate-800/50 hover:bg-slate-800 rounded-xl transition"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={saving}
              className="px-5 py-2 text-xs font-semibold text-white bg-indigo-600 hover:bg-indigo-500 rounded-xl shadow-lg shadow-indigo-600/20 transition"
            >
              {saving ? "Saving..." : "Create SLI Measurement"}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
