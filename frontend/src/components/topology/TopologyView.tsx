"use client"

import {
  useCallback,
  useEffect,
  useRef,
  useState,
} from "react"

import {
  useNodesState,
  useEdgesState,
} from "reactflow"

import ReactFlow, {
  Background,
  Controls,
  MiniMap,
  Node,
  Edge,
} from "reactflow"

import "reactflow/dist/style.css"

import { useClusterWatch } from "@/hooks/useClusterWatch"

import { ResourceNode } from "./ResourceNode"
import { ResourceDetails } from "./ResourceDetails"

import { getResource } from "@/lib/api"
import { apiUrl } from "@/lib/config"

import { SearchResult } from "@/components/search/types"

interface TopologyResponse {
  nodes: Node[]
  edges: Edge[]
  stats?: {
    services: number
    deployments: number
    replicaSets: number
    pods: number
    edges: number
  }
}

interface TopologyViewProps {
  namespace?: string
  search?: string
  selectedSearchResource?: SearchResult | null
}

const nodeTypes = {
  service: ResourceNode,
  deployment: ResourceNode,
  replicaset: ResourceNode,
  pod: ResourceNode,
}

export function TopologyView({
  namespace = "",
  search = "",
  selectedSearchResource = null,
}: TopologyViewProps) {
  const [
    nodes,
    setNodes,
    onNodesChange,
  ] = useNodesState([])

  const [
    edges,
    setEdges,
    onEdgesChange,
  ] = useEdgesState([])

  const flowRef = useRef<any>(null)

  const [loading, setLoading] =
    useState(true)

  const [
    selectedResource,
    setSelectedResource,
  ] = useState<any>(null)

  const [
    resourceLoading,
    setResourceLoading,
  ] = useState(false)

  /*
   * Initial topology
   */
  useEffect(() => {
    let cancelled = false

    async function loadTopology() {
      try {
        setLoading(true)

        const response =
          await fetch(
            apiUrl(`/topology?namespace=${encodeURIComponent(namespace)}`)
          )

        if (!response.ok) {
          throw new Error(
            `Topology request failed: ${response.status}`
          )
        }

        const data: TopologyResponse =
          await response.json()

        if (cancelled) {
          return
        }

        setNodes(data.nodes ?? [])
        setEdges(data.edges ?? [])
      } catch (error) {
        if (!cancelled) {
          console.error(
            "Topology fetch failed:",
            error
          )

          setNodes([])
          setEdges([])
        }
      } finally {
        if (!cancelled) {
          setLoading(false)
        }
      }
    }

    loadTopology()

    return () => {
      cancelled = true
    }
  }, [
    namespace,
    setNodes,
    setEdges,
  ])

  /*
   * Live Kubernetes updates
   */
  useClusterWatch(
    useCallback(
      async (event) => {
        if (
          event.resource !== "Pod" &&
          event.resource !== "Service" &&
          event.resource !== "Deployment" &&
          event.resource !== "ReplicaSet"
        ) {
          return
        }

        try {
          const response =
            await fetch(
              apiUrl(`/topology?namespace=${encodeURIComponent(namespace)}`)
            )

          if (!response.ok) {
            return
          }

          const data =
            await response.json()

          setNodes(data.nodes ?? [])
          setEdges(data.edges ?? [])
        } catch (error) {
          console.error(
            "Live topology refresh failed:",
            error
          )
        }
      },
      [
        namespace,
        setNodes,
        setEdges,
      ]
    )
  )

  /*
   * Highlight + center resource selected
   * from the global "/" search.
   */
  useEffect(() => {
    if (!selectedSearchResource) {
      setNodes((current) =>
        current.map((node) => ({
          ...node,
          data: {
            ...node.data,
            highlight: false,
          },
        }))
      )

      return
    }

    const matchingNodeIds: string[] = []

    setNodes((current) =>
      current.map((node) => {
        const matches =
          node.data?.kind ===
            selectedSearchResource.kind &&
          node.data?.name ===
            selectedSearchResource.name &&
          node.data?.namespace ===
            selectedSearchResource.namespace

        if (matches) {
          matchingNodeIds.push(node.id)
        }

        return {
          ...node,
          data: {
            ...node.data,
            highlight: matches,
          },
        }
      })
    )

    /*
     * Center on the matching node after
     * React Flow receives the updated nodes.
     */
    setTimeout(() => {
      if (
        matchingNodeIds.length > 0 &&
        flowRef.current
      ) {
        flowRef.current.fitView({
          nodes: matchingNodeIds.map(
            (id) => ({
              id,
            })
          ),
          duration: 600,
          padding: 0.5,
        })
      }
    }, 50)
  }, [
    selectedSearchResource,
    setNodes,
  ])

  /*
   * Search filter
   */
  const filteredNodes =
    nodes.filter((node) => {
      if (!search) {
        return true
      }

      return node.data?.label
        ?.toLowerCase()
        .includes(
          search.toLowerCase()
        )
    })

  /*
   * Click resource inside topology.
   */
  async function handleNodeClick(
    _: React.MouseEvent,
    node: Node
  ) {
    const kind =
      node.data?.kind

    const resourceNamespace =
      node.data?.namespace ?? ""

    const name =
      node.data?.name ??
      node.data?.label

    if (!kind || !name) {
      return
    }

    try {
      setResourceLoading(true)

      const resource =
        await getResource(
          kind,
          resourceNamespace,
          name
        )

      setSelectedResource(resource)
    } catch (error) {
      console.error(
        "Resource fetch failed:",
        error
      )
    } finally {
      setResourceLoading(false)
    }
  }

  if (loading) {
    return (
      <div
        className="
          h-[600px]
          flex
          items-center
          justify-center
          rounded-xl
          border
          border-zinc-800
          bg-zinc-950
          text-zinc-400
        "
      >
        Loading topology...
      </div>
    )
  }

  return (
    <>
      <div
        className="
          h-[600px]
          w-full
          overflow-hidden
          rounded-xl
          border
          border-zinc-800
          bg-zinc-950
        "
      >
        <ReactFlow
          nodes={filteredNodes}
          edges={edges}
          onNodesChange={
            onNodesChange
          }
          onEdgesChange={
            onEdgesChange
          }
          nodeTypes={nodeTypes}
          onNodeClick={
            handleNodeClick
          }
          onInit={(instance) => {
            flowRef.current = instance
          }}
          fitView
        >
          <Background
            gap={20}
            size={1}
          />

          <Controls />

          <MiniMap />
        </ReactFlow>
      </div>

      <ResourceDetails
        resource={selectedResource}
        loading={resourceLoading}
        onClose={() =>
          setSelectedResource(null)
        }
      />
    </>
  )
}