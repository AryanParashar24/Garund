"use client"

import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"

import { Badge } from "@/components/ui/badge"

interface Pod {
  name: string
  namespace: string
  status: string
  node: string
}

interface PodTableProps {
  pods: Pod[]

  onResourceSelect?: (resource: {
    kind: string
    name: string
    namespace: string
  }) => void
}

export function PodTable({
  pods,
  onResourceSelect,
}: PodTableProps) {
  return (
    <div>
      <div className="
        border-b
        border-zinc-800
        px-4
        py-3
      ">
        <h2 className="
          text-sm
          font-medium
          text-zinc-200
        ">
          Pods
        </h2>

        <p className="
          mt-0.5
          text-xs
          text-zinc-500
        ">
          Running workloads in the cluster
        </p>
      </div>

      <Table>

        <TableHeader>
          <TableRow>

            <TableHead>
              Name
            </TableHead>

            <TableHead>
              Namespace
            </TableHead>

            <TableHead>
              Status
            </TableHead>

            <TableHead>
              Node
            </TableHead>

          </TableRow>
        </TableHeader>

        <TableBody>

          {pods.map((pod) => (

            <TableRow
              key={`${pod.namespace}-${pod.name}`}
              onClick={() =>
                onResourceSelect?.({
                  kind: "Pod",
                  name: pod.name,
                  namespace: pod.namespace,
                })
              }
              title="Open pod details"
              className="
                cursor-pointer
                transition-colors
                hover:bg-zinc-900
              "
            >

              <TableCell className="
                font-medium
                text-zinc-200
              ">
                {pod.name}
              </TableCell>

              <TableCell className="
                text-zinc-400
              ">
                {pod.namespace}
              </TableCell>

              <TableCell>

                <Badge>
                  {pod.status}
                </Badge>

              </TableCell>

              <TableCell className="
                text-zinc-400
              ">
                {pod.node}
              </TableCell>

            </TableRow>

          ))}

        </TableBody>

      </Table>

    </div>
  )
}