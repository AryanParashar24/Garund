/**
 * API base URL for Garund backend.
 * Set NEXT_PUBLIC_API_BASE_URL to an empty string for a same-origin proxy.
 * Development defaults to the separately runnable Go API.
 */
export function getApiBase(): string {
  if (typeof window !== "undefined") {
    return process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080"
  }
  return (
    process.env.API_BASE_URL ??
    process.env.NEXT_PUBLIC_API_BASE_URL ??
    "http://localhost:8080"
  )
}

export function apiUrl(path: string): string {
  const base = getApiBase()
  const normalized = path.startsWith("/") ? path : `/${path}`
  return `${base}${normalized}`
}

export function getWsUrl(): string {
  if (typeof window === "undefined") {
    const base = process.env.API_BASE ?? "http://localhost:8080"
    return base.replace(/^http/, "ws") + "/ws"
  }

  const base = getApiBase()
  if (base) {
    return base.replace(/^http/, "ws") + "/ws"
  }

  const protocol = window.location.protocol === "https:" ? "wss:" : "ws:"
  return `${protocol}//${window.location.host}/ws`
}
