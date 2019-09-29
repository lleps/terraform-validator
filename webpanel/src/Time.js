import React from "react";
import ReactTimeAgoWithTooltip from "react-time-ago/modules/ReactTimeAgoWithTooltip";

export function TimeAgo({ timestamp }) {
    return <ReactTimeAgoWithTooltip date={timestamp/1000}/>
}