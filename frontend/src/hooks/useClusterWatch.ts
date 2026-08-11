"use client"

import { useEffect, useRef } from "react"
import { getWsUrl } from "@/lib/config"

export function useClusterWatch(
  onEvent: (event: any) => void
) {
  const onEventRef = useRef(onEvent)

  useEffect(() => {
    onEventRef.current = onEvent
  }, [onEvent])

  useEffect(() => {
    const ws = new WebSocket(getWsUrl())

    ws.onmessage = (event) => {
      try {
        onEventRef.current(JSON.parse(event.data))
      } catch {
        console.error("Received an invalid cluster-watch message")
      }
    }

    ws.onerror = (e) => {
      console.error("WS ERROR", e)
    }

    return () => {
      ws.close()
    }
  }, [])
}
