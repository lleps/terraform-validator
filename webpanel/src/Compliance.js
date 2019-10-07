import React from "react";
import {Typography} from "@material-ui/core";
import Tooltip from "@material-ui/core/Tooltip";
import {Error, Info} from "@material-ui/icons";

export function ComplianceResult( { result, showTooltip }) {
    if (result.Initialized === false) {
        return <Typography>-</Typography>
    }

    if (result.Error === true) {
        return <span>
            <Typography component={"body1"} color={"error"}>error</Typography>
            <Tooltip
                title={
                    <React.Fragment>
                        {result.ErrorMessage.split("\n").map(l => <span>{l}<br/></span>)}
                    </React.Fragment>
                }>
                <Error color={"error"}/>
            </Tooltip>
        </span>
    }

    return <span>
        <ComplianceLabel passing={result.PassCount} failing={result.FailCount}/>
        <Tooltip
            title={
                <React.Fragment>
                    <ResultList resultsMap={result.FeaturesResult} failuresMap={result.FeaturesFailures}/>
                </React.Fragment>
            }>
            <Info/>
        </Tooltip>
    </span>
}

function ComplianceLabel({ passing, failing }) {
    return <Typography color={failing === 0 ? "primary" : "secondary"} component={"body1"}>
        {passing}/{passing+failing}
    </Typography>
}

function ResultList({ resultsMap, failuresMap }) {
    let passing = [];
    let failing = [];
    let errors = [];
    for (let f in resultsMap) {
        let result = resultsMap[f] === true;
        if (result) {
            passing.push(f);
        } else {
            failing.push(f);
        }
        let errorList = failuresMap[f];
        if (errorList != null && errorList.length > 0) {
            errorList.forEach((errName) => {
                errors.push(f + ": " + errName);
            });
        }
    }

    return (
        <div>
            { passing.length > 0 ? "Passing:" : <div/> }
            <ul>
                {passing.map((k) =>
                    <li>{k}</li>
                )}
            </ul>
            { failing.length > 0 ? "Failing:" : <div/> }
            <ul>
                {failing.map((k) =>
                    <li>{k}</li>
                )}
            </ul>
            { errors.length > 0 ? "Errors:" : <div/> }
            <ul>
                {errors.map((k) =>
                    <li>{k}</li>
                )}
            </ul>
        </div>
    );
}