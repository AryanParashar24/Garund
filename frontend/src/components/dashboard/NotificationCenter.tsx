"use client"

import { useState } from "react"
import { useClusterWatch } from "@/hooks/useClusterWatch"

export function NotificationCenter() {

    const [notifications, setNotifications] =
        useState<any[]>([])

    useClusterWatch((event) => {

        if (
            event.reason === "Scheduled" ||
            event.reason === "Pulled" ||
            event.reason === "Started"
        ) {
            return
        }

        setNotifications(prev => [
            event,
            ...prev,
        ].slice(0, 20))
    })

    return (

        <div className="border rounded-xl p-4">

            <h2 className="font-bold mb-4">

                Notifications

            </h2>

            {notifications.map((n, i) => (

                <div
                    key={i}
                    className="border-b py-2"
                >

                    <div className="font-semibold">

                        {n.resource}

                    </div>

                    <div className="text-xs">

                        {n.reason || n.action}

                    </div>

                    <div className="text-xs text-gray-500">

                        {n.namespace}

                    </div>

                </div>

            ))}

        </div>

    )

}