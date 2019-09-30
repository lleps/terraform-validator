import Select from "@material-ui/core/Select";
import MenuItem from "@material-ui/core/MenuItem";
import React from "react";

export function SelectAccount({ objs, selected, onSelect }) {
    let accounts = objs.map(o => o.account).filter(o => o !== "");
    let unique = [...new Set(accounts)];
    return <Select
        value={selected}
        onChange={e => onSelect(e.target.value)}
    >
        <MenuItem value={"All"}>All</MenuItem>
        {unique.map((e) =>
            <MenuItem value={e}>{e}</MenuItem>

        )}
    </Select>
}