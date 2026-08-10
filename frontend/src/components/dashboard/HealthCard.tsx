export function HealthCard({
  score,
}: {
  score: number
}) {
  return (
    <div className="border rounded-xl p-4">
      <h2>Cluster Health</h2>

      <div className="text-5xl font-bold">
        {score}%
      </div>
    </div>
  )
}