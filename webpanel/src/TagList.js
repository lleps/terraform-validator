import React from "react";
import InputAdornment from "@material-ui/core/InputAdornment";
import {Label} from "@material-ui/icons";
import TextField from "@material-ui/core/TextField";

function strHash(s) {
    let hash = 0, i, chr;
    if (s.length === 0) return hash;
    for (i = 0; i < s.length; i++) {
        chr   = s.charCodeAt(i);
        hash  = ((hash << 5) - hash) + chr;
        hash |= 0; // Convert to 32bit integer
    }
    return hash;
}

export function TagListField({ tags, onChange }) {
    return (
        <TextField
            id="tags"
            label={"Tags"}
            value={tags.join(",")}
            autoComplete={"off"}
            InputProps={{
                style: {
                    fontFamily: "Monospace",
                    background: "#EEEEEE"
                },
                startAdornment: (
                    <InputAdornment position="start">
                        <Label />
                    </InputAdornment>
                ),
            }}
            onChange={e => onChange(e.target.value.split(","))}
            fullWidth
            margin="normal"
            variant={"outlined"}
        />
    )
}

export function Tag({ tag }) {
    let tagColorCount = 20; // colors defined in index.css as label-{0-19}
    return <span><span className={"label label-" + (strHash(tag) % tagColorCount)}>{tag}</span>&nbsp;</span>
}

export function TagList({ tags }) {
    return <div>
        {(tags || []).map((t,idx) => <span><Tag tag={t}/>{idx%7===6 ? <br/> : ""}</span>) }
    </div>
}