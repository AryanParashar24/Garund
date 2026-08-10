"use client"

import { useState } from "react"

import { PodTable } from "@/components/dashboard/PodTable"
import { ServiceTable } from "@/components/dashboard/ServiceTable"
import { EventsTable } from "@/components/dashboard/EventsTable"
import { ResourceDetails } from "@/components/topology/ResourceDetails"

import { getResource } from "@/lib/api"

interface ResourceSelection {
  kind: string
  name: string
  namespace: string
}

interface ResourceTablesProps {
  pods: any[]
  services: any[]
  events: any[]
}

export function ResourceTables({
  pods,
  services,
  events,
}: ResourceTablesProps) {
  const [
    selectedResource,
    setSelectedResource,
  ] = useState<any | null>(null)

  const [
    selectedResourceInfo,
    setSelectedResourceInfo,
  ] = useState<ResourceSelection | null>(null)

  const [loading, setLoading] =
    useState(false)

  async function handleResourceSelect(
    resource: ResourceSelection
  ) {
    try {
      setLoading(true)

      setSelectedResourceInfo(resource)

      const data = await getResource(
        resource.kind,
        resource.namespace,
        resource.name
      )

      setSelectedResource(data)
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
    setSelectedResourceInfo(null)
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