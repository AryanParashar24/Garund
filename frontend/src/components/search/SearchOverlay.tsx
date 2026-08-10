"use client"

import { useEffect, useState } from "react"

import { searchResources } from "@/lib/api"
import { SearchResult } from "./types"

interface SearchOverlayProps {
  onSelect: (resource: SearchResult) => void
}

export function SearchOverlay({
  onSelect,
}: SearchOverlayProps) {
  const [open, setOpen] = useState(false)
  const [query, setQuery] = useState("")
  const [results, setResults] = useState<SearchResult[]>([])
  const [selectedIndex, setSelectedIndex] = useState(0)
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    function handleKeyDown(event: KeyboardEvent) {
      const target = event.target as HTMLElement

      const typing =
        target.tagName === "INPUT" ||
        target.tagName === "TEXTAREA" ||
        target.isContentEditable

      if (
        event.key === "/" &&
        !open &&
        !typing
      ) {
        event.preventDefault()

        setOpen(true)
        setQuery("")
        setResults([])
        setSelectedIndex(0)

        return
      }

      if (!open) {
        return
      }

      if (event.key === "Escape") {
        event.preventDefault()

        setOpen(false)
        setQuery("")
        setResults([])

        return
      }

      if (event.key === "ArrowDown") {
        event.preventDefault()

        setSelectedIndex((current) =>
          Math.min(
            current + 1,
            results.length - 1
          )
        )

        return
      }

      if (event.key === "ArrowUp") {
        event.preventDefault()

        setSelectedIndex((current) =>
          Math.max(current - 1, 0)
        )

        return
      }

      if (event.key === "Enter") {
        event.preventDefault()

        const selected =
          results[selectedIndex]

        if (selected) {
          onSelect(selected)

          setOpen(false)
          setQuery("")
          setResults([])
        }
      }
    }

    window.addEventListener(
      "keydown",
      handleKeyDown
    )

    return () => {
      window.removeEventListener(
        "keydown",
        handleKeyDown
      )
    }
  }, [
    open,
    results,
    selectedIndex,
    onSelect,
  ])

  useEffect(() => {
    if (!query.trim()) {
      setResults([])
      setSelectedIndex(0)
      return
    }

    let cancelled = false

    async function loadResults() {
      try {
        setLoading(true)

        const data =
          await searchResources(query)

        if (!cancelled) {
          setResults(data)
          setSelectedIndex(0)
        }
      } catch (error) {
        console.error(
          "Search failed:",
          error
        )

        if (!cancelled) {
          setResults([])
        }
      } finally {
        if (!cancelled) {
          setLoading(false)
        }
      }
    }

    loadResults()

    return () => {
      cancelled = true
    }
  }, [query])

  if (!open) {
    return null
  }

  return (
    <div
      className="
        fixed
        inset-0
        z-[100]
        flex
        items-start
        justify-center
        bg-black/60
        pt-24
        backdrop-blur-sm
      "
      onMouseDown={() => setOpen(false)}
    >
      <div
        className="
          w-[720px]
          overflow-hidden
          rounded-xl
          border
          border-zinc-700
          bg-zinc-950
          shadow-2xl
        "
        onMouseDown={(event) =>
          event.stopPropagation()
        }
      >
        <div className="border-b border-zinc-800">
          <input
            autoFocus
            value={query}
            onChange={(event) =>
              setQuery(event.target.value)
            }
            placeholder="Search Kubernetes resources..."
            className="
              w-full
              bg-transparent
              px-5
              py-4
              text-sm
              text-zinc-100
              outline-none
            "
          />
        </div>

        <div className="max-h-[500px] overflow-y-auto">
          {loading && (
            <div className="px-5 py-6 text-sm text-zinc-500">
              Searching...
            </div>
          )}

          {!loading &&
            query &&
            results.length === 0 && (
              <div className="px-5 py-6 text-sm text-zinc-500">
                No resources found
              </div>
            )}

          {results.map((item, index) => {
            const selected =
              index === selectedIndex

            return (
              <button
                key={`${item.kind}-${item.namespace}-${item.name}`}
                type="button"
                onMouseEnter={() =>
                  setSelectedIndex(index)
                }
                onClick={() => {
                  onSelect(item)

                  setOpen(false)
                  setQuery("")
                  setResults([])
                }}
                className={`
                  flex
                  w-full
                  items-center
                  justify-between
                  border-b
                  border-zinc-800
                  px-5
                  py-3
                  text-left
                  transition
                  ${
                    selected
                      ? "bg-zinc-800"
                      : "hover:bg-zinc-900"
                  }
                `}
              >
                <div>
                  <div className="text-xs text-zinc-500">
                    {item.kind}
                  </div>

                  <div className="font-medium text-zinc-100">
                    {item.name}
                  </div>

                  <div className="text-xs text-zinc-500">
                    {item.namespace}
                  </div>
                </div>

                {item.status && (
                  <div className="text-xs text-zinc-400">
                    {item.status}
                  </div>
                )}
              </button>
            )
          })}
        </div>

        <div className="flex gap-5 border-t border-zinc-800 px-5 py-2 text-xs text-zinc-600">
          <span>↑ ↓ Navigate</span>
          <span>Enter Open</span>
          <span>Esc Close</span>
        </div>
      </div>
    </div>
  )
}