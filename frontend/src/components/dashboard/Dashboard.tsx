"use client"

import { useEffect, useState } from "react"
import { NamespaceSelector } from "./NamespaceSelector"
import { StatCard } from "./StatCard"
import { TopologyView } from "@/components/topology/TopologyView"
import { WorkloadPanel } from "./WorkloadPannel"
import { useClusterWatch } from "@/hooks/useClusterWatch"
import { apiUrl } from "@/lib/config"

interface DashboardProps {
  namespaces: string[]
}

export function Dashboard({
  namespaces,
}: DashboardProps) {
    const [namespace, setNamespace] = useState("")

    const [overview, setOverview] = useState({
    pods: 0,
    deployments: 0,
    services: 0,
    nodes: 0,
    })

    useClusterWatch((event) => {
        if (
        event.type === "ADDED" ||
        event.type === "MODIFIED" ||
        event.type === "DELETED"
        ) {
        window.location.reload()
        }
    })

    const [health, setHealth] = useState({
    score: 0,
    })

    useEffect(() => {
    async function loadDashboard() {
        const [overviewRes, healthRes] = await Promise.all([
        fetch(
            apiUrl(`/overview?namespace=${encodeURIComponent(namespace)}`)
        ),
        fetch(
            apiUrl(`/health-score?namespace=${encodeURIComponent(namespace)}`)
        ),
        ])

        const overviewData = await overviewRes.json()
        const healthData = await healthRes.json()

        setOverview(overviewData)
        setHealth(healthData)
    }

    loadDashboard()
    }, [namespace])
    
  return (
    <main className="min-h-screen bg-zinc-950 text-zinc-100">
        // header
        {<header className="border-b border-zinc-800 bg-zinc-950">
            <div className="flex items-center justify-between px-8 py-5">

                <div>
                <h1 className="text-2xl font-semibold tracking-tight">
                    GARUND
                </h1>

                <p className="text-sm text-zinc-500">
                    Platform Reliability Console
                </p>
                </div>

                <div className="flex items-center gap-4">

                <div className="text-right">
                    <p className="text-xs text-zinc-500">
                    Namespace
                    </p>

                    <NamespaceSelector
                    namespaces={namespaces}
                    value={namespace}
                    onChange={setNamespace}
                    />
                </div>

                </div>
            </div>
        </header>}

        //stats
        {<section className="px-8 py-6">
            <div className="grid grid-cols-5 gap-4">
                <StatCard label="Pods" value={overview.pods} />
                <StatCard label="Deployments" value={overview.deployments} />
                <StatCard label="Services" value={overview.services} />
                <StatCard label="Nodes" value={overview.nodes} />
                <StatCard label="Health" value={`${health.score}%`} />
            </div>
        </section>}

        // topology
        {<section className="px-8 pb-8">
            <div className="
                overflow-hidden
                rounded-2xl
                border border-zinc-800
                bg-zinc-900/50
            ">
                <div className="
                flex items-center justify-between
                border-b border-zinc-800
                px-6 py-4
                ">
                <div>
                    <h2 className="font-semibold">
                    Workload Topology
                    </h2>

                    <p className="text-sm text-zinc-500">
                    Kubernetes resource relationships
                    </p>
                </div>

                <span className="
                    rounded-full
                    bg-zinc-800
                    px-3 py-1
                    text-xs text-zinc-400
                ">
                    {namespace || "All namespaces"}
                </span>
                </div>

                <TopologyView namespace={namespace} />
            </div>
        </section>}

      {<WorkloadPanel namespace={namespace} />}
    </main>
  )
}

