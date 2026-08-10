"use client"

import { useMemo, useState } from "react"

interface ResourceDetailsProps {
  resource: any
  loading: boolean
  onClose: () => void
}

type Tab = "overview" | "events" | "raw"

export function ResourceDetails({
  resource,
  loading,
  onClose,
}: ResourceDetailsProps) {
  const [activeTab, setActiveTab] =
    useState<Tab>("overview")

  const events = useMemo(() => {
    if (!resource) return []

    if (Array.isArray(resource.events)) {
      return resource.events
    }

    return []
  }, [resource])

  if (!resource && !loading) {
    return null
  }

  const kind =
    resource?.kind ||
    resource?.metadata?.kind ||
    "Resource"

  const name =
    resource?.name ||
    resource?.metadata?.name ||
    "Unknown resource"

  const namespace =
    resource?.namespace ||
    resource?.metadata?.namespace ||
    "Cluster scoped"

  const created =
    resource?.creationTimestamp ||
    resource?.metadata?.creationTimestamp

  return (
    <div
      className="
        fixed
        inset-y-0
        right-0
        z-50
        flex
        w-full
        max-w-xl
        flex-col
        border-l
        border-zinc-800
        bg-zinc-950
        text-zinc-100
        shadow-2xl
      "
    >
      {/* Header */}

      <div
        className="
          flex
          items-start
          justify-between
          border-b
          border-zinc-800
          px-5
          py-4
        "
      >
        <div className="min-w-0">
          <div className="flex items-center gap-2">
            <span
              className="
                rounded-md
                border
                border-zinc-700
                bg-zinc-900
                px-2
                py-1
                text-[11px]
                font-medium
                text-zinc-400
              "
            >
              {kind}
            </span>

            <span
              className="
                text-xs
                text-zinc-600
              "
            >
              {namespace}
            </span>
          </div>

          <h2
            className="
              mt-2
              truncate
              text-base
              font-semibold
              text-zinc-100
            "
          >
            {name}
          </h2>

          <p
            className="
              mt-1
              text-xs
              text-zinc-500
            "
          >
            Kubernetes resource
          </p>
        </div>

        <button
          type="button"
          onClick={onClose}
          className="
            ml-4
            flex
            h-8
            w-8
            shrink-0
            items-center
            justify-center
            rounded-md
            border
            border-zinc-800
            bg-zinc-900
            text-zinc-500
            transition
            hover:border-zinc-700
            hover:bg-zinc-800
            hover:text-zinc-200
          "
          aria-label="Close resource details"
        >
          ×
        </button>
      </div>

      {/* Tabs */}

      <div
        className="
          border-b
          border-zinc-800
          px-5
        "
      >
        <div className="flex gap-1">
          {(
            [
              ["overview", "Overview"],
              ["events", "Events"],
              ["raw", "Raw"],
            ] as const
          ).map(([value, label]) => {
            const active =
              activeTab === value

            return (
              <button
                key={value}
                type="button"
                onClick={() =>
                  setActiveTab(value)
                }
                className={`
                  relative
                  px-3
                  py-3
                  text-xs
                  font-medium
                  transition
                  ${
                    active
                      ? "text-zinc-100"
                      : "text-zinc-500 hover:text-zinc-300"
                  }
                `}
              >
                {label}

                {active && (
                  <span
                    className="
                      absolute
                      inset-x-2
                      bottom-0
                      h-0.5
                      rounded-full
                      bg-zinc-100
                    "
                  />
                )}
              </button>
            )
          })}
        </div>
      </div>

      {/* Content */}

      <div className="min-h-0 flex-1 overflow-y-auto">
        {loading ? (
          <div className="flex h-full items-center justify-center">
            <div className="text-sm text-zinc-500">
              Loading resource...
            </div>
          </div>
        ) : (
          <>
            {/* Overview */}

            {activeTab === "overview" && (
              <div className="space-y-6 p-5">

                <section>
                  <SectionTitle>
                    Resource
                  </SectionTitle>

                  <div
                    className="
                      overflow-hidden
                      rounded-lg
                      border
                      border-zinc-800
                    "
                  >
                    <InfoRow
                      label="Kind"
                      value={kind}
                    />

                    <InfoRow
                      label="Name"
                      value={name}
                    />

                    <InfoRow
                      label="Namespace"
                      value={namespace}
                    />

                    {created && (
                      <InfoRow
                        label="Created"
                        value={formatDate(created)}
                      />
                    )}
                  </div>
                </section>

                {/* Status */}

                {resource?.status && (
                  <section>
                    <SectionTitle>
                      Status
                    </SectionTitle>

                    <StatusSummary
                      resource={resource}
                    />
                  </section>
                )}

                {/* Conditions */}

                {Array.isArray(
                  resource?.status?.conditions
                ) &&
                  resource.status.conditions.length >
                    0 && (
                    <section>
                      <SectionTitle>
                        Conditions
                      </SectionTitle>

                      <div className="space-y-2">
                        {resource.status.conditions.map(
                          (
                            condition: any,
                            index: number
                          ) => (
                            <div
                              key={`${condition.type}-${index}`}
                              className="
                                flex
                                items-center
                                justify-between
                                rounded-lg
                                border
                                border-zinc-800
                                bg-zinc-900/40
                                px-3
                                py-3
                              "
                            >
                              <div>
                                <div className="
                                  text-xs
                                  font-medium
                                  text-zinc-300
                                ">
                                  {condition.type}
                                </div>

                                {condition.reason && (
                                  <div className="
                                    mt-1
                                    text-[11px]
                                    text-zinc-600
                                  ">
                                    {condition.reason}
                                  </div>
                                )}
                              </div>

                              <span
                                className={`
                                  rounded-md
                                  px-2
                                  py-1
                                  text-[11px]
                                  font-medium
                                  ${
                                    condition.status ===
                                    "True"
                                      ? "bg-emerald-500/10 text-emerald-400"
                                      : condition.status ===
                                        "False"
                                      ? "bg-red-500/10 text-red-400"
                                      : "bg-zinc-800 text-zinc-400"
                                  }
                                `}
                              >
                                {condition.status}
                              </span>
                            </div>
                          )
                        )}
                      </div>
                    </section>
                  )}

                {/* Labels */}

                {resource?.metadata?.labels &&
                  Object.keys(
                    resource.metadata.labels
                  ).length > 0 && (
                    <section>
                      <SectionTitle>
                        Labels
                      </SectionTitle>

                      <div className="
                        rounded-lg
                        border
                        border-zinc-800
                        bg-zinc-900/40
                        p-3
                      ">
                        <div className="space-y-2">
                          {Object.entries(
                            resource.metadata.labels
                          ).map(
                            ([key, value]) => (
                              <div
                                key={key}
                                className="
                                  flex
                                  gap-3
                                  text-xs
                                "
                              >
                                <span className="
                                  min-w-0
                                  flex-1
                                  break-all
                                  text-zinc-500
                                ">
                                  {key}
                                </span>

                                <span className="
                                  min-w-0
                                  flex-1
                                  break-all
                                  text-right
                                  text-zinc-300
                                ">
                                  {String(value)}
                                </span>
                              </div>
                            )
                          )}
                        </div>
                      </div>
                    </section>
                  )}
              </div>
            )}

            {/* Events */}

            {activeTab === "events" && (
              <div className="p-5">
                <SectionTitle>
                  Resource events
                </SectionTitle>

                {events.length === 0 ? (
                  <div
                    className="
                      rounded-lg
                      border
                      border-zinc-800
                      bg-zinc-900/40
                      px-4
                      py-8
                      text-center
                    "
                  >
                    <div className="
                      text-sm
                      text-zinc-400
                    ">
                      No events found
                    </div>

                    <div className="
                      mt-1
                      text-xs
                      text-zinc-600
                    ">
                      No recorded events are associated
                      with this resource.
                    </div>
                  </div>
                ) : (
                  <div className="space-y-3">
                    {events.map(
                      (
                        event: any,
                        index: number
                      ) => {
                        const warning =
                          event.type ===
                          "Warning"

                        return (
                          <div
                            key={
                              event.uid ??
                              `${event.reason}-${index}`
                            }
                            className="
                              rounded-lg
                              border
                              border-zinc-800
                              bg-zinc-900/40
                              p-4
                            "
                          >
                            <div className="
                              flex
                              items-start
                              justify-between
                              gap-4
                            ">
                              <div>
                                <div className="
                                  text-xs
                                  font-medium
                                  text-zinc-200
                                ">
                                  {event.reason ||
                                    "Event"}
                                </div>

                                <div className="
                                  mt-1
                                  text-xs
                                  leading-5
                                  text-zinc-500
                                ">
                                  {event.message ||
                                    "No message"}
                                </div>
                              </div>

                              <span
                                className={`
                                  shrink-0
                                  rounded-md
                                  px-2
                                  py-1
                                  text-[10px]
                                  font-medium
                                  ${
                                    warning
                                      ? "bg-red-500/10 text-red-400"
                                      : "bg-emerald-500/10 text-emerald-400"
                                  }
                                `}
                              >
                                {warning
                                  ? "Warning"
                                  : "Normal"}
                              </span>
                            </div>

                            {event.count && (
                              <div className="
                                mt-3
                                text-[11px]
                                text-zinc-600
                              ">
                                Count: {event.count}
                              </div>
                            )}
                          </div>
                        )
                      }
                    )}
                  </div>
                )}
              </div>
            )}

            {/* Raw */}

            {activeTab === "raw" && (
              <div className="p-5">
                <SectionTitle>
                  Resource definition
                </SectionTitle>

                <pre
                  className="
                    max-h-[calc(100vh-180px)]
                    overflow-auto
                    rounded-lg
                    border
                    border-zinc-800
                    bg-zinc-900
                    p-4
                    font-mono
                    text-[11px]
                    leading-5
                    text-zinc-300
                  "
                >
                  {JSON.stringify(
                    resource,
                    null,
                    2
                  )}
                </pre>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  )
}

/* -------------------------------------------------- */
/* Small UI components                                */
/* -------------------------------------------------- */

function SectionTitle({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <h3
      className="
        mb-3
        text-[11px]
        font-semibold
        uppercase
        tracking-wider
        text-zinc-500
      "
    >
      {children}
    </h3>
  )
}

function InfoRow({
  label,
  value,
}: {
  label: string
  value: string
}) {
  return (
    <div
      className="
        flex
        items-center
        justify-between
        gap-4
        border-b
        border-zinc-800
        px-3
        py-3
        last:border-b-0
      "
    >
      <span className="text-xs text-zinc-500">
        {label}
      </span>

      <span
        className="
          max-w-[65%]
          truncate
          text-right
          text-xs
          text-zinc-200
        "
      >
        {value}
      </span>
    </div>
  )
}

function StatusSummary({
  resource,
}: {
  resource: any
}) {
  const status = resource?.status

  const values = [
    ["Phase", status?.phase],
    ["Ready", status?.readyReplicas],
    ["Available", status?.availableReplicas],
    ["Replicas", status?.replicas],
    ["Updated", status?.updatedReplicas],
  ].filter(([, value]) => value !== undefined)

  if (values.length === 0) {
    return (
      <div className="
        rounded-lg
        border
        border-zinc-800
        bg-zinc-900/40
        px-4
        py-4
        text-xs
        text-zinc-600
      ">
        No summarized status information available.
      </div>
    )
  }

  return (
    <div className="
      grid
      grid-cols-2
      gap-2
    ">
      {values.map(([label, value]) => (
        <div
          key={String(label)}
          className="
            rounded-lg
            border
            border-zinc-800
            bg-zinc-900/40
            px-3
            py-3
          "
        >
          <div className="
            text-[10px]
            uppercase
            tracking-wide
            text-zinc-600
          ">
            {label}
          </div>

          <div className="
            mt-1
            text-sm
            font-medium
            text-zinc-200
          ">
            {String(value)}
          </div>
        </div>
      ))}
    </div>
  )
}

function formatDate(value: string) {
  try {
    return new Date(value).toLocaleString()
  } catch {
    return value
  }
}