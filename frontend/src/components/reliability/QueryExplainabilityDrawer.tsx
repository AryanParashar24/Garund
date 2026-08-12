"use client"

import React from "react"
import { EvaluatedSLI } from "@/lib/reliability"
import { X, Code, Terminal, Clock, ShieldCheck, HelpCircle } from "lucide-react"

interface QueryExplainabilityDrawerProps {
  sli: EvaluatedSLI | null
  onClose: () => void
}

export function QueryExplainabilityDrawer({
  sli,
  onClose,
}: QueryExplainabilityDrawerProps) {
  if (!sli) return null

  return (
    <div className="fixed inset-y-0 right-0 w-full max-w-xl bg-slate-900 border-l border-slate-800 shadow-2xl z-50 flex flex-col transition-all duration-300">
      {/* Header */}
      <div className="flex items-center justify-between px-6 py-4 border-b border-slate-800 bg-slate-950/50">
        <div className="flex items-center space-x-3">
          <div className="p-2 rounded-lg bg-indigo-500/10 text-indigo-400 border border-indigo-500/20">
            <HelpCircle className="w-5 h-5" />
          </div>
          <div>
            <h3 className="text-base font-semibold text-slate-100">Query Explainability</h3>
            <p className="text-xs text-slate-400">How "{sli.name}" is calculated</p>
          </div>
        </div>
        <button
          onClick={onClose}
          className="p-1.5 text-slate-400 hover:text-slate-200 hover:bg-slate-800 rounded-md transition"
        >
          <X className="w-5 h-5" />
        </button>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto p-6 space-y-6">
        {/* Metric Summary */}
        <div className="p-4 rounded-xl bg-slate-950 border border-slate-800 space-y-3">
          <div className="flex items-center justify-between text-xs">
            <span className="text-slate-400">SLI Type</span>
            <span className="font-medium text-indigo-400 uppercase tracking-wide px-2 py-0.5 rounded bg-indigo-500/10 border border-indigo-500/20">
              {sli.type}
            </span>
          </div>

          <div className="flex items-center justify-between text-xs">
            <span className="text-slate-400">Current Measurement</span>
            <span className="text-sm font-semibold text-slate-100">
              {sli.value !== null ? `${sli.value.toFixed(2)}${sli.unit}` : "N/A (Unavailable)"}
            </span>
          </div>

          <div className="flex items-center justify-between text-xs">
            <span className="text-slate-400">Evaluation Window</span>
            <span className="text-slate-300 font-mono flex items-center gap-1">
              <Clock className="w-3.5 h-3.5 text-slate-400" />
              {sli.evaluationWindow}
            </span>
          </div>

          <div className="flex items-center justify-between text-xs">
            <span className="text-slate-400">Target</span>
            <span className="text-slate-300 font-mono">{sli.target}{sli.unit}</span>
          </div>
        </div>

        {/* Primary PromQL */}
        <div className="space-y-2">
          <label className="text-xs font-semibold text-slate-300 uppercase tracking-wider flex items-center gap-1.5">
            <Terminal className="w-4 h-4 text-emerald-400" />
            Evaluated PromQL Query
          </label>
          <div className="p-3.5 rounded-lg bg-slate-950 border border-slate-800 font-mono text-xs text-emerald-300 overflow-x-auto leading-relaxed">
            {sli.query}
          </div>
        </div>

        {/* Breakdown for Ratio Queries */}
        {(sli.goodQuery || sli.totalQuery) && (
          <div className="space-y-4 pt-2">
            <h4 className="text-xs font-semibold text-slate-300 uppercase tracking-wider flex items-center gap-1.5">
              <Code className="w-4 h-4 text-sky-400" />
              Query Ratio Breakdown
            </h4>

            {sli.goodQuery && (
              <div className="space-y-1.5">
                <span className="text-xs text-slate-400">Numerator (Good Events):</span>
                <div className="p-3 rounded-lg bg-slate-950 border border-slate-800 font-mono text-xs text-sky-300 overflow-x-auto">
                  {sli.goodQuery}
                </div>
              </div>
            )}

            {sli.totalQuery && (
              <div className="space-y-1.5">
                <span className="text-xs text-slate-400">Denominator (Total Events):</span>
                <div className="p-3 rounded-lg bg-slate-950 border border-slate-800 font-mono text-xs text-sky-300 overflow-x-auto">
                  {sli.totalQuery}
                </div>
              </div>
            )}
          </div>
        )}

        {/* Trust & Methodology Note */}
        <div className="p-4 rounded-xl bg-indigo-500/5 border border-indigo-500/10 space-y-2 text-xs text-slate-400">
          <div className="flex items-center gap-2 font-medium text-indigo-300">
            <ShieldCheck className="w-4 h-4 text-indigo-400" />
            Deterministic Metric Guarantee
          </div>
          <p className="leading-relaxed">
            Garund executes this exact PromQL against your Prometheus cluster without heuristic approximations. If telemetry is unreachable or returned empty, Garund reports state as <strong className="text-amber-400 font-mono">unavailable (N/A)</strong> to preserve observability integrity.
          </p>
        </div>
      </div>

      {/* Footer */}
      <div className="p-4 border-t border-slate-800 bg-slate-950/50 flex justify-end">
        <button
          onClick={onClose}
          className="px-4 py-2 text-xs font-medium text-slate-300 bg-slate-800 hover:bg-slate-700 rounded-lg transition"
        >
          Close
        </button>
      </div>
    </div>
  )
}
