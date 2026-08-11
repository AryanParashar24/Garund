"use client"

import { useEffect, useState } from "react"

import {
  getReliability,
  ReliabilityResponse,
} from "@/lib/api"

interface ReliabilityPanelProps {
  namespace?: string
  service?: string
  clusterId?: string
}

export function ReliabilityPanel({
  namespace = "",
  service = "",
  clusterId = "",
}: ReliabilityPanelProps) {
  const [
    reliability,
    setReliability,
  ] = useState<ReliabilityResponse | null>(null)

  const [loading, setLoading] =
    useState(true)

  useEffect(() => {
    let cancelled = false

    async function load() {
      try {
        setLoading(true)

        const data = await getReliability(
          namespace,
          service,
          clusterId
        )

        if (!cancelled) {
          setReliability(data)
        }
      } catch (error) {
        console.error(
          "Reliability fetch failed:",
          error
        )
      } finally {
        if (!cancelled) {
          setLoading(false)
        }
      }
    }

    load()

    return () => {
      cancelled = true
    }
  }, [namespace, service, clusterId])

  if (loading) {
    return (
      <section className="
        overflow-hidden
        rounded-xl
        border
        border-zinc-800
        bg-zinc-900/30
      ">
        <div className="
          px-4
          py-6
          text-sm
          text-zinc-500
        ">
          Loading reliability data...
        </div>
      </section>
    )
  }

  if (!reliability) {
    return null
  }

  return (
    <section className="
      overflow-hidden
      rounded-xl
      border
      border-zinc-800
      bg-zinc-900/30
    ">

      {/* Header */}

      <div className="
        flex
        items-center
        justify-between
        border-b
        border-zinc-800
        px-4
        py-3
      ">

        <div>
          <h2 className="
            text-sm
            font-medium
            text-zinc-200
          ">
            Reliability
          </h2>

          <p className="
            mt-0.5
            text-xs
            text-zinc-500
          ">
            SLI · SLO · SLA
          </p>
        </div>

        <div className="
          text-xs
          text-zinc-500
        ">
          {reliability.slo.window}
        </div>

      </div>

      {/* SLIs */}

      <div className="
        grid
        grid-cols-1
        gap-3
        p-4
        md:grid-cols-3
      ">

        {reliability.slis.map(
          (sli) => {

            const statusClass =
              sli.status === "critical"
                ? "text-red-400"
                : sli.status === "warning"
                ? "text-yellow-400"
                : sli.status === "unavailable"
                ? "text-zinc-500"
                : "text-emerald-400"

            return (
              <div
                key={sli.type}
                className="
                  rounded-lg
                  border
                  border-zinc-800
                  bg-zinc-950
                  p-4
                "
              >

                <div className="
                  text-xs
                  text-zinc-500
                ">
                  {sli.name}
                </div>

                <div className="
                  mt-2
                  flex
                  items-baseline
                  gap-1
                ">

                  <span
                    className={`
                      text-2xl
                      font-semibold
                      ${statusClass}
                    `}
                  >
                    {typeof sli.value === "number"
                      ? sli.value.toFixed(2)
                      : "N/A"}
                  </span>

                  <span className="
                    text-xs
                    text-zinc-600
                  ">
                    {sli.unit}
                  </span>

                </div>

                <div className="
                  mt-2
                  text-xs
                  text-zinc-600
                ">
                  Target{" "}
                  {sli.target}
                  {sli.unit}
                </div>

                <div className="
                  mt-1
                  text-[11px]
                  text-zinc-700
                ">
                  {sli.status === "unavailable"
                    ? "Telemetry unavailable"
                    : sli.status}
                </div>

              </div>
            )
          }
        )}

      </div>

      {/* SLO + SLA */}

      <div className="
        grid
        grid-cols-1
        gap-3
        border-t
        border-zinc-800
        p-4
        md:grid-cols-2
      ">

        {/* SLO */}

        <div className="
          rounded-lg
          border
          border-zinc-800
          bg-zinc-950
          p-4
        ">

          <div className="
            text-xs
            uppercase
            tracking-wide
            text-zinc-600
          ">
            SLO
          </div>

          <div className="
            mt-2
            flex
            items-center
            justify-between
          ">

            <span className="
              text-sm
              text-zinc-300
            ">
              Availability
            </span>

            <span className="
              font-semibold
              text-zinc-100
            ">
              {reliability.slo.target}%
            </span>

          </div>

          {/* Error budget */}

          {reliability.slo.status !==
            "unavailable" && (
            <div className="
              mt-3
              h-1.5
              overflow-hidden
              rounded-full
              bg-zinc-800
            ">
              <div
                className="
                  h-full
                  rounded-full
                  bg-emerald-500
                "
                style={{
                  width: `${Math.min(
                    reliability.slo
                      .errorBudgetRemaining,
                    100
                  )}%`,
                }}
              />
            </div>
          )}

          <div className="
            mt-2
            text-xs
            text-zinc-500
          ">
            {reliability.slo.status ===
            "unavailable"
              ? "N/A error budget remaining"
              : `${reliability.slo.errorBudgetRemaining.toFixed(
                  1
                )}% error budget remaining`}
          </div>

          {/* SLO status */}

          <div className="
            mt-3
            flex
            items-center
            justify-between
            border-t
            border-zinc-900
            pt-3
          ">

            <span className="
              text-xs
              text-zinc-600
            ">
              Status
            </span>

            <span
              className={`
                text-xs
                font-medium
                ${
                  reliability.slo.status ===
                  "critical"
                    ? "text-red-400"
                    : reliability.slo.status ===
                      "warning"
                    ? "text-yellow-400"
                    : reliability.slo.status ===
                      "unavailable"
                    ? "text-zinc-500"
                    : "text-emerald-400"
                }
              `}
            >
              {reliability.slo.status}
            </span>

          </div>

        </div>

        {/* SLA */}

        <div className="
          rounded-lg
          border
          border-zinc-800
          bg-zinc-950
          p-4
        ">

          <div className="
            text-xs
            uppercase
            tracking-wide
            text-zinc-600
          ">
            SLA
          </div>

          <div className="
            mt-2
            flex
            items-center
            justify-between
          ">

            <span className="
              text-sm
              text-zinc-300
            ">
              Service commitment
            </span>

            <span
              className={`
                text-sm
                font-semibold
                ${
                  reliability.sla.status ===
                  "breached"
                    ? "text-red-400"
                    : reliability.sla.status ===
                      "at_risk"
                    ? "text-yellow-400"
                    : reliability.sla.status ===
                      "unavailable"
                    ? "text-zinc-500"
                    : "text-emerald-400"
                }
              `}
            >
              {reliability.sla.status}
            </span>

          </div>

          <div className="
            mt-3
            text-xs
            text-zinc-500
          ">
            Availability commitment{" "}
            {reliability.sla
              .availabilityTarget}
            %
          </div>

          <div className="
            mt-1
            text-xs
            text-zinc-600
          ">
            Latency commitment{" "}
            {reliability.sla
              .latencyTargetMs ?? "N/A"}
            ms
          </div>

          {/* SLA status explanation */}

          <div className="
            mt-3
            border-t
            border-zinc-900
            pt-3
            text-[11px]
            text-zinc-600
          ">
            {reliability.sla.status ===
            "unavailable"
              ? "Insufficient telemetry to evaluate SLA"
              : reliability.sla.status ===
                "breached"
              ? "Service availability is below the SLA commitment"
              : reliability.sla.status ===
                "at_risk"
              ? "Service is approaching the SLA boundary"
              : "Service is currently meeting the SLA commitment"}
          </div>

        </div>

      </div>

    </section>
  )
}
