"use client"

import { Handle, Position } from "reactflow"

interface ResourceNodeProps {
  data: {
    label: string
    kind: string
    namespace: string

    status?: string
    health?: string

    replicas?: number
    readyReplicas?: number
    updatedReplicas?: number
    unavailableReplicas?: number
    availableReplicas?: number

    restarts?: number

    svcType?: string
    backends?: number

    node?: string
    podIP?: string
    clusterIP?: string

    age?: string

    selected?: boolean
    highlight?: boolean
  }
}

export function ResourceNode({
  data,
}: ResourceNodeProps) {
  const health = data.health

  const borderStyle =
    health === "critical"
      ? "border-red-500/80 shadow-red-500/20 shadow-lg"
      : health === "warning"
      ? "border-yellow-500/80 shadow-yellow-500/20 shadow-md"
      : "border-emerald-500/80 shadow-emerald-500/20 shadow-md"
      
  const highlight =
    data.highlight
      ? "ring-4 ring-blue-400 ring-offset-2"
      : ""

  return (
    <div
      className={`
        min-w-[220px]
        rounded-xl
        border-2
        ${borderStyle}
        ${highlight}
        bg-white
        p-4
        text-gray-900
        shadow-sm
        transition-all
        duration-300
        hover:scale-105
        hover:shadow-xl
        cursor-pointer
      `}
    >
      <Handle
        type="target"
        position={Position.Left}
      />

      {/* Header */}
      <div className="flex items-center justify-between gap-3">
        <div className="min-w-0">
          <div className="truncate font-bold">
            {data.label}
          </div>

          <div className="mt-1 text-xs text-gray-500">
            {data.kind}
          </div>
        </div>

        {health && (
          <div
            className={`
              h-2.5
              w-2.5
              shrink-0
              rounded-full
              ${
                health === "critical"
                  ? "bg-red-500"
                  : health === "warning"
                  ? "bg-yellow-500"
                  : "bg-green-500"
              }
            `}
          />
        )}
      </div>

      {/* Namespace */}
      {data.namespace && (
        <div className="mt-2 truncate text-xs text-gray-500">
          {data.namespace}
        </div>
      )}

      {/* Status */}
      {data.status && (
        <div className="mt-3 flex justify-between gap-3 text-xs">
          <span className="text-gray-500">
            Status
          </span>

          <span className="truncate font-medium">
            {data.status}
          </span>
        </div>
      )}

      {/* Replicas */}
      {data.replicas !== undefined && (
        <div className="mt-2 flex justify-between text-xs">
          <span className="text-gray-500">
            Replicas
          </span>

          <span className="font-medium">
            {data.readyReplicas ?? 0}/{data.replicas}
          </span>
        </div>
      )}

      {/* Updated replicas */}
      {data.updatedReplicas !== undefined && (
        <div className="mt-2 flex justify-between text-xs">
          <span className="text-gray-500">
            Updated
          </span>

          <span className="font-medium">
            {data.updatedReplicas}
          </span>
        </div>
      )}

      {/* Available replicas */}
      {data.availableReplicas !== undefined && (
        <div className="mt-2 flex justify-between text-xs">
          <span className="text-gray-500">
            Available
          </span>

          <span className="font-medium">
            {data.availableReplicas}
          </span>
        </div>
      )}

      {/* Unavailable replicas */}
      {data.unavailableReplicas !== undefined && (
        <div className="mt-2 flex justify-between text-xs">
          <span className="text-gray-500">
            Unavailable
          </span>

          <span
            className={
              data.unavailableReplicas > 0
                ? "font-bold text-red-500"
                : "font-medium"
            }
          >
            {data.unavailableReplicas}
          </span>
        </div>
      )}

      {/* Restarts */}
      {data.restarts !== undefined && (
        <div className="mt-2 flex justify-between text-xs">
          <span className="text-gray-500">
            Restarts
          </span>

          <span
            className={
              data.restarts > 0
                ? "font-bold text-red-500"
                : "font-medium"
            }
          >
            {data.restarts}
          </span>
        </div>
      )}

      {/* Service type */}
      {data.svcType && (
        <div className="mt-2 flex justify-between text-xs">
          <span className="text-gray-500">
            Type
          </span>

          <span className="font-medium">
            {data.svcType}
          </span>
        </div>
      )}

      {/* Service backends */}
      {data.backends !== undefined && (
        <div className="mt-2 flex justify-between text-xs">
          <span className="text-gray-500">
            Backends
          </span>

          <span className="font-medium">
            {data.backends}
          </span>
        </div>
      )}

      {/* Pod node */}
      {data.node && (
        <div className="mt-2 flex justify-between gap-3 text-xs">
          <span className="shrink-0 text-gray-500">
            Node
          </span>

          <span className="truncate text-right font-medium">
            {data.node}
          </span>
        </div>
      )}

      {/* Pod IP */}
      {data.podIP && (
        <div className="mt-2 flex justify-between gap-3 text-xs">
          <span className="shrink-0 text-gray-500">
            IP
          </span>

          <span className="truncate text-right font-medium">
            {data.podIP}
          </span>
        </div>
      )}

      {/* Cluster IP */}
      {data.clusterIP && (
        <div className="mt-2 flex justify-between gap-3 text-xs">
          <span className="shrink-0 text-gray-500">
            Cluster IP
          </span>

          <span className="truncate text-right font-medium">
            {data.clusterIP}
          </span>
        </div>
      )}

      {/* Age */}
      {data.age && (
        <div className="mt-2 flex justify-between text-xs">
          <span className="text-gray-500">
            Age
          </span>

          <span className="font-medium">
            {data.age}
          </span>
        </div>
      )}

      <Handle
        type="source"
        position={Position.Right}
      />
    </div>
  )
}