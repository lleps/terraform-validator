import Typography from "@material-ui/core/Typography";
import React from "react";
import Title from "./Title";
import Table from "@material-ui/core/Table";
import TableHead from "@material-ui/core/TableHead";
import TableRow from "@material-ui/core/TableRow";
import TableCell from "@material-ui/core/TableCell";
import TableBody from "@material-ui/core/TableBody";
import {Button} from "@material-ui/core";
import CircularProgress from "@material-ui/core/CircularProgress";
import {AccessTime, Info, TrendingFlat} from "@material-ui/icons";
import axios from 'axios';
import DialogContent from "@material-ui/core/DialogContent";
import DialogActions from "@material-ui/core/DialogActions";
import Dialog from "@material-ui/core/Dialog";
import DialogTitle from "@material-ui/core/DialogTitle";
import Tooltip from "@material-ui/core/Tooltip";
import IconButton from "@material-ui/core/IconButton";
import {TimeAgo} from "./Time";
import {Account} from "./TagList";
import {SelectAccount} from "./Account";
import {ComplianceResult} from "./Compliance";

export class LogDetailsDialog extends React.Component {
    state = {
        details: null,
        diffHtml: "",
    };

    componentDidMount() {
        axios.get("/logs/" + this.props.id)
            .then(res => {
                const details = res.data;
                this.setState({ details: details, diffHtml: details.state_diff_html });
            })
    }

    render() {
        // Add all features, prev and current, to this list.
        let allFeatures = [];
        let prevResult = [];
        let currResult = [];
        if (this.state.details !== null) {
            let featuresNow = this.state.details.compliance_result.FeaturesResult;
            let featuresPrev = this.state.details.prev_compliance_result.FeaturesResult;

            if (featuresNow != null) {
                for (const feature in featuresNow) {
                    if (allFeatures.indexOf(feature) === -1) allFeatures.push(feature);
                }
            }

            if (featuresPrev != null) {
                for (const feature in featuresPrev) {
                    if (allFeatures.indexOf(feature) === -1) allFeatures.push(feature);
                }
            }
            prevResult = this.state.details.prev_compliance_result;
            currResult = this.state.details.compliance_result;
        }

        return (
            <Dialog
                fullWidth="md"
                maxWidth="md"
                open={true}
                onClose={() => this.props.onClose()} aria-labelledby="form-dialog-title">
                <DialogTitle id="customized-dialog-title" onClose={() => {}}>
                    Event Details
                </DialogTitle>
                <DialogContent>
                    { this.state.details === null ? <div align="center"><CircularProgress/></div> : "" }
                    <Title>Features</Title>
                    <Table size="small">
                        <TableHead>
                        </TableHead>
                        <TableBody>
                            { allFeatures.map((f) =>
                                <TableRow key={f}>
                                    <TableCell>{f}</TableCell>
                                    <TableCell align="right">
                                        <TableCell>
                                            <div/>
                                            <FeaturePassingChange
                                                oldPassing={(prevResult.FeaturesResult || {})[f]}
                                                newPassing={currResult.FeaturesResult[f]}
                                                oldErrors={(prevResult.FeaturesFailures || {})[f]}
                                                newErrors={currResult.FeaturesFailures[f]}
                                            />
                                        </TableCell>
                                    </TableCell>
                                </TableRow>
                            ) }
                        </TableBody>
                    </Table>

                    <Title>State Diff</Title>
                    <div
                        className="code"
                        dangerouslySetInnerHTML={{__html: this.state.diffHtml} }>
                    </div>
                </DialogContent>
                <DialogActions>
                    <Button onClick={() => this.props.onClose()} color="primary">
                        Close
                    </Button>
                </DialogActions>
            </Dialog>
        )
    }
}

function ValidationState(l) {
    return <span>
        <ComplianceResult result={l.prev_compliance_result}/>
         <TrendingFlat/>
        <ComplianceResult result={l.compliance_result}/>
    </span>
}

function LinesChangedLabel(l) {
    if (l.prev_compliance_result.Initialized === false) {
        return <div>State added</div>
    }

    return <div>State changed</div>
}

function FeaturePassing({ passing, errors }) {
    let msg = "none";
    let color = "action";
    if (passing === true) {
        msg = "passing";
        color = "primary";
    } else if (passing === false) {
        msg = "failing";
        color = "error";
    }

    return <span>
        <Typography color={color} component="body1">{msg}</Typography>
        { (errors != null && !passing)
            ? <Tooltip title={<ul>{errors.map((err) => <li>{err}</li>)}</ul>}><Info/></Tooltip>
            : ""
        }
    </span>;
}

function FeaturePassingChange({ oldPassing, newPassing, oldErrors, newErrors}) {
    return (
        <span>
            <FeaturePassing passing={oldPassing} errors={oldErrors}/>
            <TrendingFlat/>
            <FeaturePassing passing={newPassing} errors={newErrors}/>
        </span>
    )
}

function LogTableColumns(kind) {
    if (kind === "tfstate") {
        return (
            <React.Fragment>
                <TableCell>Account</TableCell>
                <TableCell><AccessTime/></TableCell>
                <TableCell>Bucket:Path</TableCell>
                <TableCell>Event</TableCell>
                <TableCell>Compliance</TableCell>
            </React.Fragment>
        );
    } else {
        return (
            <React.Fragment>
                <TableCell><AccessTime/></TableCell>
                <TableCell>Compliance</TableCell>
            </React.Fragment>
        );
    }
}

function LogTableCells(l) {
    if (l.kind === "tfstate") {
        return (
            <React.Fragment>
                <TableCell><Account account={l.account}/></TableCell>
                <TableCell><TimeAgo timestamp={l.timestamp}/></TableCell>
                <TableCell>{l.details}</TableCell>
                <TableCell>{LinesChangedLabel(l)}</TableCell>
                <TableCell>{ValidationState(l)}</TableCell>
            </React.Fragment>
        );
    } else {
        return (
            <React.Fragment>
                <TableCell><TimeAgo timestamp={l.timestamp}/></TableCell>
                <TableCell>{ValidationState(l)}</TableCell>
            </React.Fragment>
        );
    }
}

export class LogsTable extends React.Component {
    state = {
        logs: [],
        updating: false,
        account: "All",
    };

    fetchData() {
        this.setState({ updating: true });
        axios.get(`/logs`)
            .then(res => {
                const logs = res.data;
                this.setState({ logs });
                this.setState({ updating: false });
            })
    }

    componentDidMount() {
        this.fetchData();
    }

    render() {
        return (
            <React.Fragment>
                <div>
                    <Title>Latest {this.props.kind === "tfstate" ? "State Changes" : "Validations"}</Title>
                    { this.props.kind === "tfstate" ?
                        <SelectAccount
                            objs={this.state.logs}
                            selected={this.state.account}
                            onSelect={v => this.setState({ account: v })}
                        />
                        : ""}
                </div>
                <Table size="small">
                    <TableHead>
                        <TableRow>
                            {LogTableColumns(this.props.kind)}
                            <TableCell align="right"/>
                        </TableRow>
                    </TableHead>
                    <TableBody>
                        { this.state.logs
                            .filter(l => this.state.account === "All" || this.state.account === l.account)
                            .filter(l => l.kind === this.props.kind)
                            .map(l =>
                                <TableRow key={l.id}>
                                    {LogTableCells(l)}
                                    <TableCell align="right">
                                        <IconButton onClick={() => this.props.onSelectInfo(l.id)} >
                                            <Info/>
                                        </IconButton>
                                    </TableCell>
                                </TableRow>
                            )
                        }
                    </TableBody>
                </Table>
                { this.state.updating ? <div align="center"><CircularProgress/></div> : "" }
            </React.Fragment>
        )
    }
}