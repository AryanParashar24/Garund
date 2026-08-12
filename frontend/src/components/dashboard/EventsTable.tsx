"use client"

import { useMemo, useState } from "react"

import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"

interface KubernetesEvent {
  type: string
  reason: string
  namespace: string
  message: string

  name?: string
  count?: number

  firstSeen?: string
  lastSeen?: string
  eventTime?: string

  source?: {
    component?: string
    host?: string
  }

  involvedObject?: {
    kind?: string
    name?: string
    namespace?: string
    uid?: string
    apiVersion?: string
  }
}

interface EventsTableProps {
  events: KubernetesEvent[]

  onResourceSelect?: (resource: {
    kind: string
    name: string
    namespace: string
  }) => void
}

type EventFilter =
  | "all"
  | "warning"
  | "normal"

type SortMode =
  | "latest"
  | "warnings"
  | "frequent"

function getEventTime(
  event: KubernetesEvent
): string | null {
  return (
    event.eventTime ||
    event.lastSeen ||
    event.firstSeen ||
    null
  )
}

function timestampValue(
  event: KubernetesEvent
) {
  const timestamp = getEventTime(event)

  if (!timestamp) {
    return 0
  }

  const value = new Date(timestamp).getTime()

  return Number.isNaN(value)
    ? 0
    : value
}

function formatRelativeTime(
  timestamp?: string | null
) {
  if (!timestamp) {
    return "Unknown"
  }

  const date = new Date(timestamp)

  if (Number.isNaN(date.getTime())) {
    return "Unknown"
  }

  const diff = Math.max(
    0,
    Date.now() - date.getTime()
  )

  const seconds = Math.floor(
    diff / 1000
  )

  if (seconds < 60) {
    return `${seconds}s ago`
  }

  const minutes = Math.floor(
    seconds / 60
  )

  if (minutes < 60) {
    return `${minutes}m ago`
  }

  const hours = Math.floor(
    minutes / 60
  )

  if (hours < 24) {
    return `${hours}h ago`
  }

  const days = Math.floor(
    hours / 24
  )

  if (days < 30) {
    return `${days}d ago`
  }

  const months = Math.floor(
    days / 30
  )

  return `${months}mo ago`
}

function formatExactTime(
  timestamp?: string | null
) {
  if (!timestamp) {
    return "Unknown"
  }

  const date = new Date(timestamp)

  if (Number.isNaN(date.getTime())) {
    return "Unknown"
  }

  return new Intl.DateTimeFormat(
    undefined,
    {
      dateStyle: "medium",
      timeStyle: "medium",
    }
  ).format(date)
}

function eventSeverity(
  event: KubernetesEvent
) {
  if (event.type === "Warning") {
    return {
      icon: "⚠",
      label: "Warning",
      badge:
        "border-yellow-500/30 bg-yellow-500/10 text-yellow-400",
    }
  }

  return {
    icon: "✓",
    label: "Normal",
    badge:
      "border-emerald-500/30 bg-emerald-500/10 text-emerald-400",
  }
}

export function EventsTable({
  events,
  onResourceSelect,
}: EventsTableProps) {
  const [filter, setFilter] =
    useState<EventFilter>("all")

  const [namespaceFilter, setNamespaceFilter] =
    useState("all")

  const [resourceFilter, setResourceFilter] =
    useState("all")

  const [search, setSearch] =
    useState("")

  const [sortMode, setSortMode] =
    useState<SortMode>("latest")

  const namespaces = useMemo(() => {
    return Array.from(
      new Set(
        events
          .map(
            (event) =>
              event.involvedObject
                ?.namespace ||
              event.namespace
          )
          .filter(Boolean)
      )
    ).sort()
  }, [events])

  const resourceKinds = useMemo(() => {
    return Array.from(
      new Set(
        events
          .map(
            (event) =>
              event.involvedObject?.kind
          )
          .filter(Boolean)
      )
    ).sort()
  }, [events])

  const filteredEvents = useMemo(() => {
    const query =
      search.trim().toLowerCase()

    const result = events.filter(
      (event) => {
        const namespace =
          event.involvedObject
            ?.namespace ||
          event.namespace ||
          ""

        const kind =
          event.involvedObject?.kind ||
          ""

        const resourceName =
          event.involvedObject?.name ||
          event.name ||
          ""

        const matchesFilter =
          filter === "all" ||
          (filter === "warning" &&
            event.type === "Warning") ||
          (filter === "normal" &&
            event.type !== "Warning")

        const matchesNamespace =
          namespaceFilter === "all" ||
          namespace === namespaceFilter

        const matchesResource =
          resourceFilter === "all" ||
          kind === resourceFilter

        const matchesSearch =
          !query ||
          event.reason
            ?.toLowerCase()
            .includes(query) ||
          event.message
            ?.toLowerCase()
            .includes(query) ||
          resourceName
            .toLowerCase()
            .includes(query) ||
          kind
            .toLowerCase()
            .includes(query) ||
          namespace
            .toLowerCase()
            .includes(query)

        return (
          matchesFilter &&
          matchesNamespace &&
          matchesResource &&
          matchesSearch
        )
      }
    )

    return result.sort(
      (a, b) => {
        if (sortMode === "warnings") {
          if (
            a.type === "Warning" &&
            b.type !== "Warning"
          ) {
            return -1
          }

          if (
            a.type !== "Warning" &&
            b.type === "Warning"
          ) {
            return 1
          }
        }

        if (sortMode === "frequent") {
          return (
            (b.count ?? 1) -
            (a.count ?? 1)
          )
        }

        return (
          timestampValue(b) -
          timestampValue(a)
        )
      }
    )
  }, [
    events,
    filter,
    namespaceFilter,
    resourceFilter,
    search,
    sortMode,
  ])

  return (
    <div className="w-full">

      {/* Header */}

      <div className="border-b border-zinc-800 px-4 py-3">

        <div className="flex items-center justify-between gap-4">

          <div>
            <h2 className="text-sm font-medium text-zinc-200">
              Event History
            </h2>

            <p className="mt-0.5 text-xs text-zinc-500">
              Kubernetes cluster activity
            </p>
          </div>

          <div className="text-xs text-zinc-600">
            {filteredEvents.length} / {events.length}
          </div>

        </div>

        {/* Severity filters */}

        <div className="mt-4 flex flex-wrap items-center gap-2">

          {(
            [
              ["all", "All"],
              ["warning", "Warning"],
              ["normal", "Normal"],
            ] as const
          ).map(([value, label]) => (

            <button
              key={value}
              type="button"
              onClick={() =>
                setFilter(value)
              }
              className={`rounded-md border px-3 py-1.5 text-xs transition-colors ${
                filter === value
                  ? "border-zinc-600 bg-zinc-800 text-zinc-100"
                  : "border-zinc-800 bg-zinc-950 text-zinc-500 hover:bg-zinc-900 hover:text-zinc-300"
              }`}
            >
              {label}
            </button>

          ))}

        </div>

        {/* Search + filters */}

        <div className="mt-3 grid grid-cols-1 gap-2 md:grid-cols-4">

          <input
            value={search}
            onChange={(event) =>
              setSearch(event.target.value)
            }
            placeholder="Search events..."
            className="rounded-md border border-zinc-800 bg-zinc-950 px-3 py-2 text-xs text-zinc-200 outline-none placeholder:text-zinc-600 focus:border-zinc-600"
          />

          <select
            value={namespaceFilter}
            onChange={(event) =>
              setNamespaceFilter(
                event.target.value
              )
            }
            className="rounded-md border border-zinc-800 bg-zinc-950 px-3 py-2 text-xs text-zinc-400 outline-none"
          >
            <option value="all">
              All namespaces
            </option>

            {namespaces.map(
              (namespace) => (
                <option
                  key={namespace}
                  value={namespace}
                >
                  {namespace}
                </option>
              )
            )}
          </select>

          <select
            value={resourceFilter}
            onChange={(event) =>
              setResourceFilter(
                event.target.value
              )
            }
            className="rounded-md border border-zinc-800 bg-zinc-950 px-3 py-2 text-xs text-zinc-400 outline-none"
          >
            <option value="all">
              All resources
            </option>

            {resourceKinds.map(
              (kind) => (
                <option
                  key={kind}
                  value={kind}
                >
                  {kind}
                </option>
              )
            )}
          </select>

          <select
            value={sortMode}
            onChange={(event) =>
              setSortMode(
                event.target.value as SortMode
              )
            }
            className="rounded-md border border-zinc-800 bg-zinc-950 px-3 py-2 text-xs text-zinc-400 outline-none"
          >
            <option value="latest">
              Latest
            </option>

            <option value="warnings">
              Warnings first
            </option>

            <option value="frequent">
              Most frequent
            </option>
          </select>

        </div>

      </div>

      {/* Event table */}

      <div className="max-h-[560px] overflow-auto">

        <Table>

          <TableHeader className="sticky top-0 z-10 bg-zinc-950">

            <TableRow className="border-zinc-800 hover:bg-transparent">

              <TableHead>
                Severity
              </TableHead>

              <TableHead>
                Reason
              </TableHead>

              <TableHead>
                Resource
              </TableHead>

              <TableHead>
                Namespace
              </TableHead>

              <TableHead>
                Message
              </TableHead>

              <TableHead className="text-right">
                Count
              </TableHead>

              <TableHead className="text-right">
                Time
              </TableHead>

            </TableRow>

          </TableHeader>

          <TableBody>

            {filteredEvents.map(
              (event, index) => {
                const severity =
                  eventSeverity(event)

                const timestamp =
                  getEventTime(event)

                const resourceKind =
                  event.involvedObject
                    ?.kind ||
                  "Unknown"

                const resourceName =
                  event.involvedObject
                    ?.name ||
                  event.name ||
                  "Unknown"

                const resourceNamespace =
                  event.involvedObject
                    ?.namespace ||
                  event.namespace ||
                  "default"

                const canOpenResource =
                  Boolean(
                    event.involvedObject
                      ?.kind &&
                    event.involvedObject
                      ?.name
                  )

                return (
                  <TableRow
                    key={`${event.name}-${index}`}
                    className="border-zinc-800 transition-colors hover:bg-zinc-900/60"
                  >

                    <TableCell>

                      <span
                        className={`inline-flex items-center gap-1.5 rounded-md border px-2 py-1 text-[11px] font-medium ${severity.badge}`}
                      >
                        {severity.icon}
                        {severity.label}
                      </span>

                    </TableCell>

                    <TableCell>
                      <span className="font-medium text-zinc-200">
                        {event.reason ||
                          "Unknown"}
                      </span>
                    </TableCell>

                    <TableCell>

                      {canOpenResource ? (

                        <button
                          type="button"
                          onClick={() =>
                            onResourceSelect?.({
                              kind:
                                resourceKind,
                              name:
                                resourceName,
                              namespace:
                                resourceNamespace,
                            })
                          }
                          className="text-left transition-colors hover:text-blue-400"
                        >

                          <div className="font-medium text-zinc-200">
                            {resourceKind}
                            <span className="mx-1 text-zinc-600">
                              /
                            </span>
                            {resourceName}
                          </div>

                          <div className="mt-0.5 text-[10px] text-zinc-600">
                            Open resource
                          </div>

                        </button>

                      ) : (

                        <div className="font-medium text-zinc-400">
                          {resourceKind}
                          <span className="mx-1 text-zinc-600">
                            /
                          </span>
                          {resourceName}
                        </div>

                      )}

                    </TableCell>

                    <TableCell>

                      <span className="rounded-md bg-zinc-900 px-2 py-1 text-xs text-zinc-400">
                        {resourceNamespace}
                      </span>

                    </TableCell>

                    <TableCell>

                      <div
                        className="max-w-[360px] truncate text-xs text-zinc-400"
                        title={event.message}
                      >
                        {event.message}
                      </div>

                    </TableCell>

                    <TableCell className="text-right">

                      <span className="text-xs text-zinc-400">
                        {event.count ?? 1}
                      </span>

                    </TableCell>

                    <TableCell className="text-right">

                      <div
                        title={formatExactTime(
                          timestamp
                        )}
                      >

                        <div className="text-xs font-medium text-zinc-300">
                          {formatRelativeTime(
                            timestamp
                          )}
                        </div>

                        <div className="mt-0.5 text-[10px] text-zinc-600">
                          {formatExactTime(
                            timestamp
                          )}
                        </div>

                      </div>

                    </TableCell>

                  </TableRow>
                )
              }
            )}

          </TableBody>

        </Table>

        {filteredEvents.length === 0 && (

          <div className="flex h-32 items-center justify-center text-sm text-zinc-600">
            No events match the current filters.
          </div>

        )}

      </div>

    </div>
  )
}