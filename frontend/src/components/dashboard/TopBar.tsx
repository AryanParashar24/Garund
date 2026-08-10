"use client"

interface Props {
  namespaces: string[]
  namespace: string
  search: string
  onNamespaceChange: (ns: string) => void
  onSearchChange: (value: string) => void
}

export function TopBar({
  namespaces,
  namespace,
  search,
  onNamespaceChange,
  onSearchChange,
}: Props) {
  return (
    <div className="flex items-center justify-between gap-4 mb-8">

      <div className="text-4xl font-bold">
        GARUND
      </div>

      <input
        className="border rounded-lg px-4 py-2 w-[350px]"
        placeholder="Search pods, services..."
        value={search}
        onChange={(e) => onSearchChange(e.target.value)}
      />

      <select
        className="border rounded-lg px-3 py-2"
        value={namespace}
        onChange={(e) =>
          onNamespaceChange(e.target.value)
        }
      >
        <option value="">
          All Namespaces
        </option>

        {namespaces.map(ns => (
          <option
            key={ns}
            value={ns}
          >
            {ns}
          </option>
        ))}
      </select>

    </div>
  )
}