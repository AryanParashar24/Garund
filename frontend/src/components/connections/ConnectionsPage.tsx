"use client"

import React, { useState, useEffect } from "react"
import { ClusterConnection } from "@/types/cluster"
import { getClusters, deleteCluster, switchCluster, getAgentManifest } from "@/lib/api"

interface ConnectionsPageProps {
  onSelectCluster: (clusterId: string, cluster: ClusterConnection) => void
  onOpenAddWizard: () => void
}

export function ConnectionsPage({ onSelectCluster, onOpenAddWizard }: ConnectionsPageProps) {
  const [clusters, setClusters] = useState<ClusterConnection[]>([])
  const [activeId, setActiveId] = useState<string>("")
  const [loading, setLoading] = useState(true)
  const [manifestModal, setManifestModal] = useState<{ name: string; command: string } | null>(null)

  const loadData = async () => {
    setLoading(true)
    try {
      const res = await getClusters()
      setClusters(res.clusters || [])
      setActiveId(res.activeId || "local-dev")
    } catch (e) {
      console.error("Failed to fetch connections list", e)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadData()
  }, [])

  const handleDelete = async (id: string, name: string) => {
    if (!confirm(`Are you sure you want to remove cluster connection "${name}"?`)) return
    try {
      await deleteCluster(id)
      await loadData()
    } catch (e: any) {
      alert(e.message || "Failed to remove connection")
    }
  }

  const handleSwitch = async (cluster: ClusterConnection) => {
    try {
      await switchCluster(cluster.id)
      setActiveId(cluster.id)
      onSelectCluster(cluster.id, cluster)
    } catch (e: any) {
      alert(e.message || "Failed to switch active cluster")
    }
  }

  const handleViewManifest = async (cluster: ClusterConnection) => {
    try {
      const res = await getAgentManifest(cluster.id)
      setManifestModal({ name: cluster.name, command: res.command })
    } catch (e: any) {
      alert(e.message || "Failed to generate agent manifest")
    }
  }

  const getStatusBadge = (status: string) => {
    switch (status) {
      case "CONNECTED":
        return (
          <span className="inline-flex items-center gap-1.5 rounded-full bg-emerald-950/60 px-2.5 py-1 text-xs font-medium text-emerald-400 border border-emerald-800/40">
            <span className="h-1.5 w-1.5 rounded-full bg-emerald-500 animate-pulse" />
            Connected
          </span>
        )
      case "DEGRADED":
        return (
          <span className="inline-flex items-center gap-1.5 rounded-full bg-amber-950/60 px-2.5 py-1 text-xs font-medium text-amber-400 border border-amber-800/40">
            <span className="h-1.5 w-1.5 rounded-full bg-amber-500" />
            Degraded
          </span>
        )
      case "DISCONNECTED":
        return (
          <span className="inline-flex items-center gap-1.5 rounded-full bg-rose-950/60 px-2.5 py-1 text-xs font-medium text-rose-400 border border-rose-800/40">
            <span className="h-1.5 w-1.5 rounded-full bg-rose-500" />
            Disconnected
          </span>
        )
      default:
        return (
          <span className="inline-flex items-center gap-1.5 rounded-full bg-purple-950/60 px-2.5 py-1 text-xs font-medium text-purple-400 border border-purple-800/40">
            <span className="h-1.5 w-1.5 rounded-full bg-purple-500" />
            Auth Error
          </span>
        )
    }
  }

  return (
    <div className="space-y-6">
      {/* Top Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-xl font-semibold text-zinc-100">Connection Center</h2>
          <p className="text-xs text-zinc-500 mt-1">
            Unified multi-cluster discovery, connectivity, and RBAC permissions management
          </p>
        </div>

        <button
          onClick={onOpenAddWizard}
          className="
            flex
            items-center
            gap-2
            rounded-lg
            bg-emerald-600
            px-4
            py-2.5
            text-xs
            font-semibold
            text-white
            shadow-lg
            shadow-emerald-950/50
            transition-all
            hover:bg-emerald-500
          "
        >
          <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
          </svg>
          Add Connection
        </button>
      </div>

      {loading ? (
        <div className="flex h-48 items-center justify-center text-xs text-zinc-500">
          Loading connection topology...
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
          {clusters.map((cluster) => {
            const isActive = cluster.id === activeId
            return (
              <div
                key={cluster.id}
                className={`
                  relative
                  overflow-hidden
                  rounded-2xl
                  border
                  p-5
                  transition-all
                  ${isActive ? "border-emerald-500/60 bg-zinc-900/80 shadow-xl shadow-emerald-950/20" : "border-zinc-800 bg-zinc-900/40 hover:border-zinc-700"}
                `}
              >
                {/* Active Indicator Banner */}
                {isActive && (
                  <div className="absolute top-0 right-0 rounded-bl-xl bg-emerald-500/10 px-3 py-1 text-[10px] font-semibold text-emerald-400 border-l border-b border-emerald-800/40">
                    Active Workspace
                  </div>
                )}

                {/* Card Title & Provider */}
                <div className="flex items-start justify-between pr-24">
                  <div>
                    <h3 className="text-base font-bold text-zinc-100 flex items-center gap-2">
                      {cluster.name}
                    </h3>
                    <div className="mt-1 flex items-center gap-2 text-xs text-zinc-400">
                      <span className="font-semibold text-zinc-300">{cluster.provider}</span>
                      <span>•</span>
                      <span className="font-mono text-zinc-500">{cluster.clusterType}</span>
                    </div>
                  </div>
                </div>

                <div className="mt-4 flex items-center justify-between border-t border-zinc-800/60 pt-4">
                  {getStatusBadge(cluster.status)}

                  <div className="flex items-center gap-4 text-xs font-mono text-zinc-400">
                    <div>
                      <span className="text-zinc-200 font-semibold">{cluster.kubernetesVersion || "1.32"}</span>
                    </div>
                    <div>
                      <span className="text-zinc-200 font-semibold">{cluster.nodeCount || 0}</span> nodes
                    </div>
                    <div>
                      <span className="text-zinc-200 font-semibold">{cluster.namespaceCount || 0}</span> nss
                    </div>
                  </div>
                </div>

                {/* RBAC Capabilities Scope */}
                <div className="mt-4 rounded-xl bg-zinc-950/60 p-3 border border-zinc-800/50">
                  <div className="text-[11px] font-medium text-zinc-500 mb-2">Declared RBAC Capabilities:</div>
                  <div className="flex flex-wrap gap-2 text-[11px]">
                    <span className={`px-2 py-0.5 rounded border ${cluster.capabilities?.canReadWorkloads ? "bg-emerald-950/30 text-emerald-300 border-emerald-800/30" : "bg-zinc-900 text-zinc-600 border-zinc-800"}`}>
                      ✓ Discovery & Workloads
                    </span>
                    <span className={`px-2 py-0.5 rounded border ${cluster.capabilities?.canReadLogs ? "bg-emerald-950/30 text-emerald-300 border-emerald-800/30" : "bg-zinc-900 text-zinc-600 border-zinc-800"}`}>
                      ✓ Logs & Events
                    </span>
                    <span className={`px-2 py-0.5 rounded border ${cluster.capabilities?.canReadTelemetry ? "bg-emerald-950/30 text-emerald-300 border-emerald-800/30" : "bg-zinc-900 text-zinc-600 border-zinc-800"}`}>
                      ✓ Telemetry
                    </span>
                    <span className={`px-2 py-0.5 rounded border ${cluster.capabilities?.canOperateWorkloads ? "bg-amber-950/30 text-amber-300 border-amber-800/30" : "bg-zinc-900 text-zinc-600 border-zinc-800"}`}>
                      {cluster.capabilities?.canOperateWorkloads ? "✓ Workload Operations" : "✗ Read-Only Mode"}
                    </span>
                  </div>
                </div>

                {/* Footer Controls */}
                <div className="mt-5 flex items-center justify-between border-t border-zinc-800/60 pt-4">
                  <div className="text-[11px] font-mono text-zinc-500">
                    Mode: <span className="text-zinc-400">{cluster.connectionMode}</span>
                  </div>

                  <div className="flex items-center gap-2">
                    {cluster.connectionMode === "agent" && (
                      <button
                        onClick={() => handleViewManifest(cluster)}
                        className="rounded-lg border border-zinc-800 px-3 py-1.5 text-xs text-zinc-300 hover:bg-zinc-800"
                      >
                        Agent Manifest
                      </button>
                    )}

                    {!isActive && (
                      <button
                        onClick={() => handleSwitch(cluster)}
                        className="rounded-lg bg-emerald-600/20 border border-emerald-800/40 px-3.5 py-1.5 text-xs font-semibold text-emerald-400 hover:bg-emerald-600/30"
                      >
                        Switch to Cluster
                      </button>
                    )}

                    {cluster.id !== "local-dev" && (
                      <button
                        onClick={() => handleDelete(cluster.id, cluster.name)}
                        className="rounded-lg p-1.5 text-zinc-500 hover:bg-rose-950/30 hover:text-rose-400"
                        title="Remove Connection"
                      >
                        <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                        </svg>
                      </button>
                    )}
                  </div>
                </div>
              </div>
            )
          })}
        </div>
      )}

      {/* Manifest Viewer Modal */}
      {manifestModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/80 p-4">
          <div className="w-full max-w-xl rounded-2xl border border-zinc-800 bg-zinc-950 p-6 shadow-2xl space-y-4">
            <div className="flex items-center justify-between border-b border-zinc-800 pb-3">
              <h3 className="text-sm font-semibold text-zinc-100">Agent Installation — {manifestModal.name}</h3>
              <button onClick={() => setManifestModal(null)} className="text-zinc-500 hover:text-zinc-200">
                ✕
              </button>
            </div>

            <p className="text-xs text-zinc-400">
              Run this command on your Kubernetes cluster terminal:
            </p>

            <div className="rounded-xl border border-zinc-800 bg-zinc-900 p-4 font-mono text-[11px] text-emerald-400 overflow-x-auto max-h-56">
              <pre>{manifestModal.command}</pre>
            </div>

            <div className="flex justify-end pt-2">
              <button
                onClick={() => setManifestModal(null)}
                className="rounded-lg bg-zinc-800 px-4 py-2 text-xs font-medium text-zinc-200 hover:bg-zinc-700"
              >
                Close
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
