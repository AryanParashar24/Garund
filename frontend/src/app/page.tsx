import { ResourceTables } from "@/components/dashboard/ResourceTables"
import { ReliabilityPanel } from "@/components/reliability/ReliabilityPanel"
import { ResourceHealthCard } from "@/components/dashboard/ResourceHealthCard"
import { LiveEventFeed } from "@/components/events/LiveEventFeed"
import { DashboardClient } from "@/components/dashboard/DashboardClient"

import {
  getPods,
  getOverview,
  getServices,
  getEvents,
  getHealthScore,
  getNamespaces,
} from "@/lib/api"

export default async function Home() {
  const [
    pods,
    overview,
    services,
    events,
    health,
    namespaces,
  ] = await Promise.all([
    getPods(),
    getOverview(),
    getServices(),
    getEvents(),
    getHealthScore(),
    getNamespaces(),
  ])

  return (
    <main className="min-h-screen bg-zinc-950 text-zinc-100">
      <div className="mx-auto max-w-[1800px] px-6 py-6">

        {/* Header */}

        <header className="mb-6 flex items-center justify-between">
          <div>
            <div className="flex items-center gap-3">
              <h1 className="text-xl font-semibold tracking-tight">
                Garund
              </h1>

              <span
                className="
                  rounded-md
                  border
                  border-zinc-800
                  bg-zinc-900
                  px-2
                  py-1
                  text-[11px]
                  font-medium
                  text-zinc-400
                "
              >
                Kubernetes Workspace
              </span>
            </div>

            <p className="mt-1 text-sm text-zinc-500">
              Cloud-native workspace explorer
            </p>
          </div>

          <div className="flex items-center gap-5">

            {/* Cluster status */}

            <div className="flex items-center gap-2 text-xs text-zinc-500">
              <span className="h-1.5 w-1.5 rounded-full bg-emerald-500" />
              Cluster connected
            </div>

            {/* Search hint */}

            <div
              className="
                flex
                items-center
                gap-3
                rounded-lg
                border
                border-zinc-800
                bg-zinc-900/80
                px-3
                py-2
                text-sm
                text-zinc-400
              "
            >
              <span>Search resources</span>

              <kbd
                className="
                  rounded-md
                  border
                  border-zinc-700
                  bg-zinc-950
                  px-2
                  py-0.5
                  font-mono
                  text-xs
                  text-zinc-400
                "
              >
                /
              </kbd>
            </div>
          </div>
        </header>

        {/* Cluster overview */}

        <section className="mb-6">
          <div className="mb-3">
            <h2 className="text-sm font-medium text-zinc-200">
              Cluster overview
            </h2>

            <p className="mt-0.5 text-xs text-zinc-500">
              Current Kubernetes resource state
            </p>
          </div>

          <div
            className="
              grid
              grid-cols-1
              gap-3
              sm:grid-cols-2
              xl:grid-cols-5
            "
          >
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

        {/* Health */}

        <section className="mb-6">
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">

            <div
              className="
                rounded-xl
                border
                border-zinc-800
                bg-zinc-900/50
                p-4
              "
            >
              <div className="text-xs font-medium text-zinc-500">
                Health score
              </div>

              <div className="mt-2 text-2xl font-semibold text-zinc-100">
                {health.score}
              </div>

              <div className="mt-1 text-xs text-zinc-600">
                Cluster health
              </div>
            </div>

            <div
              className="
                rounded-xl
                border
                border-zinc-800
                bg-zinc-900/50
                p-4
              "
            >
              <div className="text-xs font-medium text-zinc-500">
                Namespaces
              </div>

              <div className="mt-2 text-2xl font-semibold text-zinc-100">
                {namespaces.length}
              </div>

              <div className="mt-1 text-xs text-zinc-600">
                Available namespaces
              </div>
            </div>

          </div>
        </section>

        {/* Reliability */}

        <section className="mb-6">
          <ReliabilityPanel />
        </section>

        {/* Main workspace */}

        <section className="grid grid-cols-12 gap-6">

          {/* Topology */}

          <div className="col-span-12 xl:col-span-8">
            <div
              className="
                overflow-hidden
                rounded-xl
                border
                border-zinc-800
                bg-zinc-900/30
              "
            >
              <div
                className="
                  flex
                  items-center
                  justify-between
                  border-b
                  border-zinc-800
                  px-4
                  py-3
                "
              >
                <div>
                  <h2 className="text-sm font-medium text-zinc-200">
                    Resource topology
                  </h2>

                  <p className="mt-0.5 text-xs text-zinc-500">
                    Services → Deployments → ReplicaSets → Pods
                  </p>
                </div>

                <div className="flex items-center gap-2 text-xs text-zinc-500">
                  <span className="h-1.5 w-1.5 rounded-full bg-emerald-500" />
                  Live
                </div>
              </div>

              <DashboardClient />
            </div>
          </div>

          {/* Live events */}

          <div className="col-span-12 xl:col-span-4">
            <div
              className="
                overflow-hidden
                rounded-xl
                border
                border-zinc-800
                bg-zinc-900/30
              "
            >
              <div
                className="
                  border-b
                  border-zinc-800
                  px-4
                  py-3
                "
              >
                <h2 className="text-sm font-medium text-zinc-200">
                  Live events
                </h2>

                <p className="mt-0.5 text-xs text-zinc-500">
                  Kubernetes cluster activity
                </p>
              </div>

              <LiveEventFeed />
            </div>
          </div>

        </section>

        {/* Resources + Events */}

        <section className="mt-6">

          <div className="mb-3">
            <h2 className="text-sm font-medium text-zinc-200">
              Resources & Events
            </h2>

            <p className="mt-0.5 text-xs text-zinc-500">
              Current workloads, services, and Kubernetes activity
            </p>
          </div>

          <ResourceTables
            pods={pods}
            services={services}
            events={events}
          />

        </section>

        {/* Search shortcut */}

        <footer className="flex items-center justify-center py-6">
          <div className="flex items-center gap-2 text-xs text-zinc-600">
            <span>Press</span>

            <kbd
              className="
                rounded
                border
                border-zinc-800
                bg-zinc-900
                px-1.5
                py-0.5
                font-mono
                text-zinc-400
              "
            >
              /
            </kbd>

            <span>
              to search Kubernetes resources
            </span>
          </div>
        </footer>

      </div>
    </main>
  )
}