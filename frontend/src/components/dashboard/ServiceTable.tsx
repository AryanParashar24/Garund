"use client"

import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"

interface Service {
  name: string
  namespace: string
  type: string
}

interface ServiceTableProps {
  services: Service[]

  onResourceSelect?: (resource: {
    kind: string
    name: string
    namespace: string
  }) => void
}

export function ServiceTable({
  services,
  onResourceSelect,
}: ServiceTableProps) {
  return (
    <div>

      <div className="border-b border-zinc-800 px-4 py-3">
        <h2 className="text-sm font-medium text-zinc-200">
          Services
        </h2>

        <p className="mt-0.5 text-xs text-zinc-500">
          Kubernetes services in the cluster
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
              Type
            </TableHead>
          </TableRow>
        </TableHeader>

        <TableBody>

          {services.map((svc) => (

            <TableRow
              key={`${svc.namespace}-${svc.name}`}
              onClick={() =>
                onResourceSelect?.({
                  kind: "Service",
                  name: svc.name,
                  namespace: svc.namespace,
                })
              }
              title="Open service details"
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
                {svc.name}
              </TableCell>

              <TableCell className="
                text-zinc-400
              ">
                {svc.namespace}
              </TableCell>

              <TableCell className="
                text-zinc-400
              ">
                {svc.type}
              </TableCell>

            </TableRow>

          ))}

        </TableBody>

      </Table>

    </div>
  )
}