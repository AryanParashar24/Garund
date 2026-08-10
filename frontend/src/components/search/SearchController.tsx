"use client"

import { useState } from "react"

import { SearchOverlay } from "./SearchOverlay"
import { SearchResult } from "./types"

import { ResourceDetails } from "@/components/topology/ResourceDetails"

import { getResource } from "@/lib/api"

interface SearchControllerProps {
  onResourceSelect: (
    resource: SearchResult
  ) => void
}

export function SearchController({
  onResourceSelect,
}: SearchControllerProps) {
  const [
    selectedResource,
    setSelectedResource,
  ] = useState<any>(null)

  const [loading, setLoading] =
    useState(false)

  async function handleSelect(
    resource: SearchResult
  ) {
    try {
      setLoading(true)

      const data = await getResource(
        resource.kind,
        resource.namespace,
        resource.name
      )

      setSelectedResource(data)

      onResourceSelect(resource)
    } catch (error) {
      console.error(
        "Failed to load resource:",
        error
      )
    } finally {
      setLoading(false)
    }
  }

  return (
    <>
      <SearchOverlay
        onSelect={handleSelect}
      />

      <ResourceDetails
        resource={selectedResource}
        loading={loading}
        onClose={() =>
          setSelectedResource(null)
        }
      />
    </>
  )
}