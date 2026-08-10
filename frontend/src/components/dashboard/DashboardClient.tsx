"use client"

import { useState } from "react"

import { SearchController } from "@/components/search/SearchController"
import { SearchResult } from "@/components/search/types"

import { TopologyView } from "@/components/topology/TopologyView"

export function DashboardClient() {
  const [
    selectedSearchResource,
    setSelectedSearchResource,
  ] = useState<SearchResult | null>(null)

  return (
    <>
      <SearchController
        onResourceSelect={
          setSelectedSearchResource
        }
      />

      <TopologyView
        selectedSearchResource={
          selectedSearchResource
        }
      />
    </>
  )
}