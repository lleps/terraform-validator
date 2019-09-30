import React from "react";
import ReactTimeAgo from "react-time-ago/modules/ReactTimeAgo";

export function TimeAgo({ timestamp }) {
    if (timestamp <= 0 || timestamp === undefined) {
        return <span>Not set</span>
    }

    return <ReactTimeAgo date={timestamp*1000}/>
}