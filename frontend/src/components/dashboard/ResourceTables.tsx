"use client"

import { useState } from "react"

import { PodTable } from "@/components/dashboard/PodTable"
import { ServiceTable } from "@/components/dashboard/ServiceTable"
import { EventsTable } from "@/components/dashboard/EventsTable"
import { ResourceDetails } from "@/components/topology/ResourceDetails"

import {
  getResource,
  getResourceEvents,
  ResourceEvent,
} from "@/lib/api"
import { Pod } from "@/types/k8s"

interface Service {
  name: string
  namespace: string
  type: string
}

interface ResourceSelection {
  kind: string
  name: string
  namespace: string
}

interface ResourceTablesProps {
  pods: Pod[]
  services: Service[]
  events: ResourceEvent[]
}

export function ResourceTables({
  pods,
  services,
  events,
}: ResourceTablesProps) {
  const [
    selectedResource,
    setSelectedResource,
  ] = useState<Record<string, unknown> | null>(null)

  const [loading, setLoading] =
    useState(false)

  async function handleResourceSelect(
    resource: ResourceSelection
  ) {
    try {
      setLoading(true)

      const [data, events] = await Promise.all([
        getResource(resource.kind, resource.namespace, resource.name),
        getResourceEvents(resource.namespace, resource.name, resource.kind),
      ])

      setSelectedResource({ ...data, events })
    } catch (error) {
      console.error(
        "Failed to load resource:",
        error
      )

      setSelectedResource(null)
    } finally {
      setLoading(false)
    }
  }

  function handleClose() {
    setSelectedResource(null)
  }

  return (
    <>
      <div className="
        grid
        grid-cols-1
        gap-6
        xl:grid-cols-2
      ">

        {/* Pods */}

        <div className="
          overflow-hidden
          rounded-xl
          border
          border-zinc-800
          bg-zinc-900/30
        ">
          <PodTable
            pods={pods}
            onResourceSelect={
              handleResourceSelect
            }
          />
        </div>

        {/* Services */}

        <div className="
          overflow-hidden
          rounded-xl
          border
          border-zinc-800
          bg-zinc-900/30
        ">
          <ServiceTable
            services={services}
            onResourceSelect={
              handleResourceSelect
            }
          />
        </div>

      </div>

      {/* Events */}

      <div className="
        mt-6
        overflow-hidden
        rounded-xl
        border
        border-zinc-800
        bg-zinc-900/30
      ">
        <EventsTable
          events={events}
          onResourceSelect={
            handleResourceSelect
          }
        />
      </div>

      {/* Shared resource details sidebar */}

      <ResourceDetails
        resource={selectedResource}
        loading={loading}
        onClose={handleClose}
      />
    </>
  )
}
