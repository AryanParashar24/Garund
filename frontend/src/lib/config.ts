/**
 * API base URL for Garund backend.
 * Empty string = same origin (embedded in garund binary or proxied).
 */
export function getApiBase(): string {
  if (typeof window !== "undefined") {
    return process.env.NEXT_PUBLIC_API_BASE ?? ""
  }
  return process.env.API_BASE ?? process.env.NEXT_PUBLIC_API_BASE ?? ""
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
