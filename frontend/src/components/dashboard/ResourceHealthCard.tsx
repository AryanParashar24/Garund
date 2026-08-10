"use client"

interface ResourceHealthCardProps {
  title: string
  total: number
  healthy: number
  unhealthy: number
}

export function ResourceHealthCard({
  title,
  total,
  healthy,
  unhealthy,
}: ResourceHealthCardProps) {
  const healthyPercentage =
    total > 0 ? (healthy / total) * 100 : 0

  const healthyDegrees =
    healthyPercentage * 3.6

  return (
    <div className="rounded-xl border border-zinc-800 bg-zinc-900/50 p-4">
      <div className="flex items-center justify-between">
        <div>
          <div className="text-xs font-medium text-zinc-500">
            {title}
          </div>

          <div className="mt-1 text-xs text-zinc-600">
            Resource health
          </div>
        </div>

        <div className="text-xs text-zinc-500">
          {total} total
        </div>
      </div>

      <div className="mt-5 flex items-center gap-5">
        {/* Donut */}

        <div
          className="relative h-28 w-28 shrink-0 rounded-full"
          style={{
            background:
              total === 0
                ? "conic-gradient(#27272a 0deg 360deg)"
                : `conic-gradient(
                    #22c55e 0deg ${healthyDegrees}deg,
                    #ef4444 ${healthyDegrees}deg 360deg
                  )`,
          }}
        >
          {/* Donut hole */}

          <div className="absolute inset-[10px] flex flex-col items-center justify-center rounded-full bg-zinc-950">
            <span className="text-2xl font-semibold text-zinc-100">
              {total}
            </span>

            <span className="text-[10px] text-zinc-600">
              total
            </span>
          </div>
        </div>

        {/* Health breakdown */}

        <div className="min-w-0 space-y-3">
          {/* Healthy */}

          <div className="flex items-center gap-2">
            <span className="h-2 w-2 rounded-full bg-green-500" />

            <div>
              <div className="text-sm font-medium text-zinc-200">
                {healthy}
              </div>

              <div className="text-[11px] text-zinc-600">
                Healthy
              </div>
            </div>
          </div>

          {/* Unhealthy */}

          <div className="flex items-center gap-2">
            <span className="h-2 w-2 rounded-full bg-red-500" />

            <div>
              <div className="text-sm font-medium text-zinc-200">
                {unhealthy}
              </div>

              <div className="text-[11px] text-zinc-600">
                Unhealthy
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Percentage */}

      <div className="mt-4 border-t border-zinc-800 pt-3">
        <div className="flex items-center justify-between text-xs">
          <span className="text-zinc-600">
            Healthy
          </span>

          <span className="font-medium text-green-500">
            {healthyPercentage.toFixed(1)}%
          </span>
        </div>
      </div>
    </div>
  )
}