"use client"

import React, { useState, useEffect, useRef } from "react"
import { ClusterConnection } from "@/types/cluster"
import { getClusters, switchCluster } from "@/lib/api"

interface ClusterSelectorProps {
  selectedClusterId: string
  onClusterChange: (clusterId: string, cluster: ClusterConnection) => void
  onOpenConnections: () => void
  onOpenAddWizard: () => void
}

export function ClusterSelector({
  selectedClusterId,
  onClusterChange,
  onOpenConnections,
  onOpenAddWizard,
}: ClusterSelectorProps) {
  const [clusters, setClusters] = useState<ClusterConnection[]>([])
  const [activeId, setActiveId] = useState<string>(selectedClusterId || "local-dev")
  const [isOpen, setIsOpen] = useState(false)
  const [loading, setLoading] = useState(false)
  const dropdownRef = useRef<HTMLDivElement>(null)

  const loadClusters = async () => {
    try {
      const data = await getClusters()
      setClusters(data.clusters || [])
      if (data.activeId && !selectedClusterId) {
        setActiveId(data.activeId)
      }
    } catch (e) {
      console.error("Failed to load cluster list", e)
    }
  }

  useEffect(() => {
    loadClusters()
    const interval = setInterval(loadClusters, 10000)
    return () => clearInterval(interval)
  }, [])

  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsOpen(false)
      }
    }
    document.addEventListener("mousedown", handleClickOutside)
    return () => document.removeEventListener("mousedown", handleClickOutside)
  }, [])

  const currentCluster = clusters.find((c) => c.id === (selectedClusterId || activeId)) || clusters[0] || {
    id: "local-dev",
    name: "Local Development",
    provider: "Local Kubeconfig",
    environment: "development",
    status: "CONNECTED",
    kubernetesVersion: "v1.32.0",
    nodeCount: 1,
    namespaceCount: 4,
  }

  const handleSelect = async (cluster: ClusterConnection) => {
    setLoading(true)
    try {
      await switchCluster(cluster.id)
      setActiveId(cluster.id)
      onClusterChange(cluster.id, cluster)
    } catch (e) {
      console.error("Failed to switch cluster", e)
    } finally {
      setLoading(false)
      setIsOpen(false)
    }
  }

  const getStatusColor = (status: string) => {
    switch (status) {
      case "CONNECTED":
        return "bg-emerald-500"
      case "DEGRADED":
        return "bg-amber-500"
      case "DISCONNECTED":
        return "bg-rose-500 text-rose-400"
      case "AUTH_ERROR":
        return "bg-purple-500"
      default:
        return "bg-zinc-500"
    }
  }

  return (
    <div className="relative inline-block text-left" ref={dropdownRef}>
      <button
        onClick={() => setIsOpen(!isOpen)}
        disabled={loading}
        className="
          flex
          items-center
          gap-2.5
          rounded-lg
          border
          border-zinc-800
          bg-zinc-900/90
          px-3.5
          py-2
          text-sm
          font-medium
          text-zinc-100
          shadow-lg
          shadow-black/20
          transition-all
          hover:border-zinc-700
          hover:bg-zinc-800/80
          focus:outline-none
        "
      >
        <span className={`h-2 w-2 rounded-full ${getStatusColor(currentCluster.status)} animate-pulse`} />
        
        <div className="flex items-center gap-2">
          <span className="font-semibold text-zinc-100">{currentCluster.name}</span>
          <span className="rounded bg-zinc-800 px-1.5 py-0.5 text-[10px] font-mono tracking-wide text-zinc-400 uppercase">
            {currentCluster.environment || "env"}
          </span>
        </div>

        <svg
          className={`h-4 w-4 text-zinc-400 transition-transform duration-200 ${isOpen ? "rotate-180" : ""}`}
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
        >
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
        </svg>
      </button>

      {isOpen && (
        <div
          className="
            absolute
            left-0
            z-50
            mt-2
            w-80
            origin-top-left
            rounded-xl
            border
            border-zinc-800
            bg-zinc-950/95
            p-2
            shadow-2xl
            backdrop-blur-xl
          "
        >
          <div className="px-3 py-2 text-[11px] font-medium text-zinc-500 uppercase tracking-wider">
            Connected Workspaces ({clusters.length})
          </div>

          <div className="max-h-64 overflow-y-auto space-y-1 py-1">
            {clusters.map((cluster) => {
              const isSelected = cluster.id === (selectedClusterId || activeId)
              return (
                <button
                  key={cluster.id}
                  onClick={() => handleSelect(cluster)}
                  className={`
                    flex
                    w-full
                    items-center
                    justify-between
                    rounded-lg
                    px-3
                    py-2.5
                    text-left
                    text-xs
                    transition-colors
                    ${isSelected ? "bg-zinc-900 border border-zinc-800 text-zinc-100" : "text-zinc-300 hover:bg-zinc-900/60"}
                  `}
                >
                  <div className="flex items-center gap-2.5">
                    <span className={`h-2 w-2 rounded-full ${getStatusColor(cluster.status)}`} />
                    <div>
                      <div className="font-medium text-zinc-200 flex items-center gap-1.5">
                        {cluster.name}
                        {isSelected && (
                          <span className="text-[10px] font-normal text-emerald-400">(active)</span>
                        )}
                      </div>
                      <div className="text-[11px] text-zinc-500 font-mono mt-0.5">
                        {cluster.provider} • {cluster.kubernetesVersion}
                      </div>
                    </div>
                  </div>

                  <div className="text-right text-[10px] font-mono text-zinc-500">
                    <div>{cluster.nodeCount || 0} nodes</div>
                    <div>{cluster.namespaceCount || 0} nss</div>
                  </div>
                </button>
              )
            })}
          </div>

          <div className="mt-2 border-t border-zinc-800/80 pt-2 space-y-1">
            <button
              onClick={() => {
                setIsOpen(false)
                onOpenAddWizard()
              }}
              className="
                flex
                w-full
                items-center
                gap-2
                rounded-lg
                px-3
                py-2
                text-xs
                font-medium
                text-emerald-400
                transition-colors
                hover:bg-emerald-950/30
              "
            >
              <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
              </svg>
              Add Kubernetes Connection
            </button>

            <button
              onClick={() => {
                setIsOpen(false)
                onOpenConnections()
              }}
              className="
                flex
                w-full
                items-center
                gap-2
                rounded-lg
                px-3
                py-2
                text-xs
                font-medium
                text-zinc-400
                transition-colors
                hover:bg-zinc-900
              "
            >
              <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 10h16M4 14h16M4 18h16" />
              </svg>
              Manage All Connections
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
