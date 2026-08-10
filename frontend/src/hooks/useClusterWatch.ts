"use client"

import { useEffect } from "react"
import { getWsUrl } from "@/lib/config"

export function useClusterWatch(
  onEvent: (event: any) => void
) {
  useEffect(() => {
    console.log("HOOK MOUNTED")

    const ws = new WebSocket(getWsUrl())

    ws.onopen = () => {
      console.log("WS OPEN")
    }

    ws.onmessage = (event) => {
        console.log("RAW", event.data)

        const parsed = JSON.parse(event.data)

        onEvent(parsed)
    }

    ws.onerror = (e) => {
      console.error("WS ERROR", e)
    }

    ws.onclose = (e) => {
      console.log("WS CLOSED", e.code, e.reason)
    }

    return () => {
      console.log("WS CLEANUP")
      ws.close()
    }
  }, [])
}