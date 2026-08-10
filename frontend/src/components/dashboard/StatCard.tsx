interface StatCardProps {
  label: string
  value: number | string
  detail?: string
}

export function StatCard({
  label,
  value,
  detail,
}: StatCardProps) {
  return (
    <div className="
      rounded-xl
      border border-zinc-800
      bg-zinc-900/60
      p-5
      transition
      hover:border-zinc-700
    ">
      <p className="text-sm text-zinc-500">
        {label}
      </p>

      <p className="
        mt-2
        text-3xl
        font-semibold
        tracking-tight
      ">
        {value}
      </p>

      {detail && (
        <p className="mt-1 text-xs text-zinc-600">
          {detail}
        </p>
      )}
    </div>
  )
}