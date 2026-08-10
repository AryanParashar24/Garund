"use client"

import { useEffect, useState } from "react"
import { apiUrl } from "@/lib/config"

interface WorkloadPanelProps {
  namespace: string
}

type WorkloadTab =
  | "pods"
  | "deployments"
  | "services"
  | "replicasets"

export function WorkloadPanel({
  namespace,
}: WorkloadPanelProps) {
  const [activeTab, setActiveTab] =
    useState<WorkloadTab>("pods")

  const [data, setData] = useState<any[]>([])
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    async function loadWorkloads() {
      setLoading(true)

      try {
        const response = await fetch(
          apiUrl(`/${activeTab}?namespace=${encodeURIComponent(namespace)}`)
        )

        if (!response.ok) {
          throw new Error(
            `Failed to fetch ${activeTab}`
          )
        }

        const result = await response.json()
        setData(result ?? [])
      } catch (error) {
        console.error(error)
        setData([])
      } finally {
        setLoading(false)
      }
    }

    loadWorkloads()
  }, [activeTab, namespace])

  return (
    <section className="px-8 pb-8">
      <div className="rounded-2xl border border-zinc-800 bg-zinc-900/50">

        <div className="flex items-center justify-between border-b border-zinc-800 px-6 py-4">
          <div>
            <h2 className="font-semibold">
              Workloads
            </h2>

            <p className="text-sm text-zinc-500">
              Resources in{" "}
              {namespace || "all namespaces"}
            </p>
          </div>
        </div>

        <div className="flex gap-2 border-b border-zinc-800 px-6 py-3">
          {(["pods", "deployments", "replicasets", "services"] as WorkloadTab[]).map(
            (tab) => (
              <button
                key={tab}
                onClick={() => setActiveTab(tab)}
                className={`rounded-lg px-4 py-2 text-sm capitalize transition ${
                  activeTab === tab
                    ? "bg-orange-500 text-black"
                    : "text-zinc-400 hover:bg-zinc-800 hover:text-white"
                }`}
              >
                {tab}
              </button>
            )
          )}
        </div>

        <div className="p-6">
          {loading ? (
            <p className="text-zinc-500">
              Loading {activeTab}...
            </p>
          ) : (
            <div className="space-y-2">
              {data.map((resource, index) => (
                <div
                  key={`${resource.namespace}-${resource.name}-${index}`}
                  className="flex items-center justify-between rounded-xl border border-zinc-800 bg-zinc-950/60 px-4 py-3 hover:border-zinc-700"
                >
                  <div>
                    <p className="font-medium">
                      {resource.name}
                    </p>

                    <p className="text-xs text-zinc-500">
                      {resource.namespace || "cluster"}
                    </p>
                  </div>

                  <span className="text-sm text-zinc-400">
                    {resource.status ??
                      resource.type ??
                      (resource.replicas !== undefined
                        ? `${resource.replicas} replicas`
                        : "Available")}
                  </span>
                </div>
              ))}

              {!data.length && (
                <p className="py-8 text-center text-sm text-zinc-500">
                  No {activeTab} found.
                </p>
              )}
            </div>
          )}
        </div>

      </div>
    </section>
  )
}