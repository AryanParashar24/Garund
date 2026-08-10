"use client"

interface NamespaceSelectorProps {
  namespaces: string[]
  value: string
  onChange: (namespace: string) => void
}

export function NamespaceSelector({
  namespaces,
  value,
  onChange,
}: NamespaceSelectorProps) {
  return (
    <select
      value={value}
      onChange={(event) =>
        onChange(event.target.value)
      }
      className="
        rounded-lg
        border border-zinc-800
        bg-zinc-900
        px-3 py-2
        text-sm
        text-zinc-200
        outline-none
        transition
        hover:border-zinc-700
        focus:border-orange-500
      "
    >
      <option value="">
        All namespaces
      </option>

      {namespaces.map((namespace) => (
        <option
          key={namespace}
          value={namespace}
        >
          {namespace}
        </option>
      ))}
    </select>
  )
}