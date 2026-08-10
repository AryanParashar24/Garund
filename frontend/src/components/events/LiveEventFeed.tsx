"use client"

import { useState } from "react"
import { useClusterWatch } from "@/hooks/useClusterWatch"

export function LiveEventFeed() {

    const [events, setEvents] = useState<any[]>([])

    useClusterWatch((event) => {
        setEvents(prev => [
            {
            ...event,
            time: new Date().toLocaleTimeString(),
            },
            ...prev,
        ])
    })

    return (

        <div className="border rounded-xl p-4 h-[600px] overflow-y-auto bg-zinc-950 text-white">
            <h2 className="text-xl font-bold mb-4">
                Live Cluster Events
            </h2>

            <div className="space-y-3">

                {events.map((e, i) => (

                    <div
                        key={i}
                        className="
                        rounded-lg
                        bg-zinc-900
                        border
                        border-zinc-800
                        p-3
                        hover:bg-zinc-800
                        transition
                        "
                    >

                        <div className="font-bold text-green-400">

                            {e.resource} {e.action}

                        </div>

                        <div className="text-sm text-zinc-400">

                            {e.namespace}

                        </div>

                        {e.name &&

                            <div>{e.name}</div>

                        }

                        {e.reason &&

                            <div>{e.reason}</div>

                        }

                        {e.message &&

                            <div className="text-sm">

                                {e.message}

                            </div>

                        }

                        <div className="text-xs mt-2">

                            {e.time}

                        </div>

                    </div>

                ))}

            </div>

        </div>

    )

}