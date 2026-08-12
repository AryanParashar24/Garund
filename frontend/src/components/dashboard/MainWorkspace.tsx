"use client"

import React, { useState, useEffect } from "react"
import { ClusterConnection } from "@/types/cluster"
import { ClusterSelector } from "@/components/dashboard/ClusterSelector"
import { ConnectionsPage } from "@/components/connections/ConnectionsPage"
import { AddConnectionWizard } from "@/components/connections/AddConnectionWizard"
import { ResourceHealthCard } from "@/components/dashboard/ResourceHealthCard"
import { ReliabilityPanel } from "@/components/reliability/ReliabilityPanel"
import { TopologyView } from "@/components/topology/TopologyView"
import { LiveEventFeed } from "@/components/events/LiveEventFeed"
import { ResourceTables } from "@/components/dashboard/ResourceTables"
import { SearchController } from "@/components/search/SearchController"
import { SearchResult } from "@/components/search/types"
import {
  getPods,
  getOverview,
  getServices,
  getEvents,
  getHealthScore,
  getNamespaces,
  Overview,
} from "@/lib/api"

export function MainWorkspace() {
  const [activeTab, setActiveTab] = useState<"dashboard" | "reliability" | "connections">("dashboard")
  const [selectedClusterId, setSelectedClusterId] = useState<string>("")
  const [selectedCluster, setSelectedCluster] = useState<ClusterConnection | null>(null)
  const [isAddWizardOpen, setIsAddWizardOpen] = useState(false)
  const [selectedSearchResource, setSelectedSearchResource] = useState<SearchResult | null>(null)

  // Cluster workspace data
  const [overview, setOverview] = useState<Overview | null>(null)
  const [healthScore, setHealthScore] = useState<number>(100)
  const [namespaces, setNamespaces] = useState<any[]>([])
  const [pods, setPods] = useState<any[]>([])
  const [services, setServices] = useState<any[]>([])
  const [events, setEvents] = useState<any[]>([])
  const [loading, setLoading] = useState(true)

  const loadWorkspaceData = async (clusterId: string) => {
    setLoading(true)
    try {
      const [ovData, hsData, nsData, podData, svcData, evtData] = await Promise.all([
        getOverview(clusterId).catch(() => null),
        getHealthScore(clusterId).catch(() => ({ score: 100 })),
        getNamespaces(clusterId).catch(() => []),
        getPods(clusterId).catch(() => []),
        getServices(clusterId).catch(() => []),
        getEvents(clusterId).catch(() => []),
      ])

      if (ovData) setOverview(ovData)
      if (hsData) setHealthScore(hsData.score ?? 100)
      setNamespaces(nsData || [])
      setPods(podData || [])
      setServices(svcData || [])
      setEvents(evtData || [])
    } catch (e) {
      console.error("Error loading cluster workspace data:", e)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadWorkspaceData(selectedClusterId)
    const interval = setInterval(() => loadWorkspaceData(selectedClusterId), 15000)
    return () => clearInterval(interval)
  }, [selectedClusterId])

  const handleClusterChange = (clusterId: string, cluster: ClusterConnection) => {
    setSelectedClusterId(clusterId)
    setSelectedCluster(cluster)
  }

  return (
    <main className="min-h-screen bg-zinc-950 text-zinc-100">
      <div className="mx-auto max-w-[1800px] px-6 py-6 space-y-6">
        {/* Top Header */}
        <header className="flex flex-col gap-4 border-b border-zinc-800/80 pb-5 md:flex-row md:items-center md:justify-between">
          <div className="flex items-center gap-4">
            <div className="flex items-center gap-3">
              <h1 className="text-xl font-bold tracking-tight text-white flex items-center gap-2">
                <span className="bg-gradient-to-r from-emerald-400 to-teal-200 bg-clip-text text-transparent">
                  Garund
                </span>
              </h1>
              <span className="rounded-md border border-zinc-800 bg-zinc-900 px-2 py-0.5 text-[11px] font-medium text-zinc-400 font-mono">
                Multi-Cluster Workspace
              </span>
            </div>

            {/* Global Cluster Selector */}
            <ClusterSelector
              selectedClusterId={selectedClusterId}
              onClusterChange={handleClusterChange}
              onOpenConnections={() => setActiveTab("connections")}
              onOpenAddWizard={() => setIsAddWizardOpen(true)}
            />
          </div>

          <div className="flex items-center gap-4">
            {/* View Switcher Tabs */}
            <div className="flex rounded-lg border border-zinc-800 bg-zinc-900/80 p-1 text-xs">
              <button
                onClick={() => setActiveTab("dashboard")}
                className={`
                  flex items-center gap-1.5 rounded-md px-3 py-1.5 font-medium transition-colors
                  ${activeTab === "dashboard" ? "bg-zinc-800 text-zinc-100 shadow" : "text-zinc-400 hover:text-zinc-200"}
                `}
              >
                <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2V6zM14 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V6zM4 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2v-2zM14 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z" />
                </svg>
                Workspace Dashboard
              </button>

              <button
                onClick={() => setActiveTab("reliability")}
                className={`
                  flex items-center gap-1.5 rounded-md px-3 py-1.5 font-medium transition-colors
                  ${activeTab === "reliability" ? "bg-indigo-600 text-white shadow" : "text-zinc-400 hover:text-zinc-200"}
                `}
              >
                <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
                </svg>
                Reliability Control Plane
              </button>

              <button
                onClick={() => setActiveTab("connections")}
                className={`
                  flex items-center gap-1.5 rounded-md px-3 py-1.5 font-medium transition-colors
                  ${activeTab === "connections" ? "bg-zinc-800 text-zinc-100 shadow" : "text-zinc-400 hover:text-zinc-200"}
                `}
              >
                <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
                </svg>
                Connection Center
              </button>
            </div>

            {/* Quick Search trigger hint */}
            <div className="hidden sm:flex items-center gap-3 rounded-lg border border-zinc-800 bg-zinc-900/80 px-3 py-2 text-sm text-zinc-400">
              <span>Search</span>
              <kbd className="rounded-md border border-zinc-700 bg-zinc-950 px-2 py-0.5 font-mono text-xs text-zinc-400">
                /
              </kbd>
            </div>
          </div>
        </header>

        {/* Global Search Controller Modal */}
        <SearchController onResourceSelect={setSelectedSearchResource} />

        {/* CONNECTIONS CENTER VIEW */}
        {activeTab === "connections" && (
          <ConnectionsPage
            onSelectCluster={(id, c) => {
              handleClusterChange(id, c)
              setActiveTab("dashboard")
            }}
            onOpenAddWizard={() => setIsAddWizardOpen(true)}
          />
        )}

        {/* RELIABILITY CONTROL PLANE VIEW */}
        {activeTab === "reliability" && (
          <ReliabilityPanel clusterId={selectedClusterId} />
        )}

        {/* WORKSPACE DASHBOARD VIEW */}
        {activeTab === "dashboard" && (
          <div className="space-y-6">
            {/* Cluster overview metrics */}
            {overview && (
              <section className="space-y-3">
                <div className="flex items-center justify-between">
                  <div>
                    <h2 className="text-sm font-medium text-zinc-200">
                      Cluster Overview — {selectedCluster?.name || "Local Cluster"}
                    </h2>
                    <p className="text-xs text-zinc-500">Real-time resource state & status metrics</p>
                  </div>
                </div>

                <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-5">
                  <ResourceHealthCard
                    title="Pods"
                    total={overview.health.pods.total}
                    healthy={overview.health.pods.healthy}
                    unhealthy={overview.health.pods.unhealthy}
                  />
                  <ResourceHealthCard
                    title="Deployments"
                    total={overview.health.deployments.total}
                    healthy={overview.health.deployments.healthy}
                    unhealthy={overview.health.deployments.unhealthy}
                  />
                  <ResourceHealthCard
                    title="ReplicaSets"
                    total={overview.health.replicaSets.total}
                    healthy={overview.health.replicaSets.healthy}
                    unhealthy={overview.health.replicaSets.unhealthy}
                  />
                  <ResourceHealthCard
                    title="Services"
                    total={overview.health.services.total}
                    healthy={overview.health.services.healthy}
                    unhealthy={overview.health.services.unhealthy}
                  />
                  <ResourceHealthCard
                    title="Namespaces"
                    total={overview.health.namespaces.total}
                    healthy={overview.health.namespaces.healthy}
                    unhealthy={overview.health.namespaces.unhealthy}
                  />
                </div>
              </section>
            )}

            {/* Cluster Health Score & Namespaces count */}
            <section className="grid grid-cols-1 gap-3 sm:grid-cols-2">
              <div className="rounded-xl border border-zinc-800 bg-zinc-900/50 p-4">
                <div className="text-xs font-medium text-zinc-500">Cluster Health Score</div>
                <div className="mt-2 text-2xl font-semibold text-zinc-100">{healthScore} / 100</div>
                <div className="mt-1 text-xs text-zinc-600">Calculated from workload crash loops and event errors</div>
              </div>

              <div className="rounded-xl border border-zinc-800 bg-zinc-900/50 p-4">
                <div className="text-xs font-medium text-zinc-500">Active Namespaces</div>
                <div className="mt-2 text-2xl font-semibold text-zinc-100">{namespaces.length}</div>
                <div className="mt-1 text-xs text-zinc-600">Discovered namespace scopes</div>
              </div>
            </section>

            {/* Reliability Section */}
            <ReliabilityPanel clusterId={selectedClusterId} />

            {/* Topology & Events Grid */}
            <section className="grid grid-cols-12 gap-6">
              <div className="col-span-12 xl:col-span-8">
                <div className="overflow-hidden rounded-xl border border-zinc-800 bg-zinc-900/30">
                  <div className="flex items-center justify-between border-b border-zinc-800 px-4 py-3">
                    <div>
                      <h2 className="text-sm font-medium text-zinc-200">Resource Topology</h2>
                      <p className="mt-0.5 text-xs text-zinc-500">Services → Deployments → ReplicaSets → Pods</p>
                    </div>
                    <div className="flex items-center gap-2 text-xs text-zinc-500">
                      <span className="h-1.5 w-1.5 rounded-full bg-emerald-500" />
                      Live Stream
                    </div>
                  </div>
                  <TopologyView
                    selectedSearchResource={selectedSearchResource}
                    clusterId={selectedClusterId}
                  />
                </div>
              </div>

              <div className="col-span-12 xl:col-span-4">
                <div className="overflow-hidden rounded-xl border border-zinc-800 bg-zinc-900/30">
                  <div className="border-b border-zinc-800 px-4 py-3">
                    <h2 className="text-sm font-medium text-zinc-200">Live Cluster Events</h2>
                    <p className="mt-0.5 text-xs text-zinc-500">Real-time audit & warning event stream</p>
                  </div>
                  <LiveEventFeed />
                </div>
              </div>
            </section>

            {/* Resource Tables */}
            <section className="mt-6">
              <div className="mb-3">
                <h2 className="text-sm font-medium text-zinc-200">Workloads, Services & Events</h2>
                <p className="mt-0.5 text-xs text-zinc-500">Comprehensive Kubernetes resource inventory</p>
              </div>
              <ResourceTables pods={pods} services={services} events={events} />
            </section>
          </div>
        )}

        {/* Add Connection Wizard Modal */}
        <AddConnectionWizard
          isOpen={isAddWizardOpen}
          onClose={() => setIsAddWizardOpen(false)}
          onSuccess={(newCluster) => {
            setSelectedClusterId(newCluster.id)
            setSelectedCluster(newCluster)
            setActiveTab("dashboard")
          }}
        />

        {/* Search Shortcut Footer */}
        <footer className="flex items-center justify-center py-6 border-t border-zinc-900">
          <div className="flex items-center gap-2 text-xs text-zinc-600">
            <span>Press</span>
            <kbd className="rounded border border-zinc-800 bg-zinc-900 px-1.5 py-0.5 font-mono text-zinc-400">
              /
            </kbd>
            <span>to search Kubernetes resources across connected workspaces</span>
          </div>
        </footer>
      </div>
    </main>
  )
}
