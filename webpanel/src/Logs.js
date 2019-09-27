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
import {Delete, Info, TrendingFlat} from "@material-ui/icons";
import axios from 'axios';
import DialogContent from "@material-ui/core/DialogContent";
import DialogActions from "@material-ui/core/DialogActions";
import Dialog from "@material-ui/core/Dialog";
import DialogTitle from "@material-ui/core/DialogTitle";
import Tooltip from "@material-ui/core/Tooltip";
import IconButton from "@material-ui/core/IconButton";
import {DeleteDialog} from "./DeleteDialog";

export class LogDetailsDialog extends React.Component {
    constructor(props) {
        super(props);
    }

    state = {
        details: null,
        diffHtml: ""
    };

    componentDidMount() {
        axios.get("/logs/" + this.props.id)
            .then(res => {
                const details = res.data;
                this.setState({ details: details, diffHtml: details.state_diff_html });
            })
    }

    render() {
        let atDate = this.state.details !== null ? "at " + this.state.details.date_time : "";

        // Add all features, prev and current, to this list.
        let allFeatures = [];
        if (this.state.details !== null) {
            let featuresNow = this.state.details.compliance_features;
            let featuresPrev = this.state.details.compliance_features_prev;
            if (featuresNow != null) {
                for (let f in featuresNow) {
                    if (allFeatures.indexOf(f) === -1) allFeatures.push(f);
                }
            }
            if (featuresPrev != null) {
                for (let f in featuresPrev) {
                    if (allFeatures.indexOf(f) === -1) allFeatures.push(f);
                }
            }
        }

        return (
            <Dialog
                fullWidth="md"
                maxWidth="md"
                open={true}
                onClose={() => this.props.onClose()} aria-labelledby="form-dialog-title">
                <DialogTitle id="customized-dialog-title" onClose={() => {}}>
                    Details for Event #{this.props.id} {atDate}
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
                                            <FeaturePassingChange
                                                oldPassing={(this.state.details.compliance_features_prev || {})[f]}
                                                newPassing={this.state.details.compliance_features[f]}
                                                oldErrors={(this.state.details.compliance_fail_messages_prev || {})[f]}
                                                newErrors={this.state.details.compliance_fail_messages[f]}
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

function ValidationText(errors, tests) {
    if (errors === 0) {
        return <Typography color="primary" component="body1">{tests}/{tests}</Typography>
    } else {
        return <Typography color="secondary" component="body1">{tests-errors}/{tests}</Typography>
    }
}

function ValidationState(l) {
    let prevErrors = l.compliance_errors_prev;
    let prevTests = l.compliance_tests_prev;
    if (prevTests === 0) {
        return ValidationText(l.compliance_errors, l.compliance_tests);
    } else {
        return (
            <div>
                {ValidationText(prevErrors, prevTests)}
                <TrendingFlat/>
                {ValidationText(l.compliance_errors, l.compliance_tests)}
            </div>
        )
    }
}

function LinesChangedLabel(l) {
    if (l.compliance_tests_prev === 0) {
        return <div>new</div>
    }

    return <div>change</div>
}

function FeaturePassing({ passing, errors}) {
    return <span>
        <Typography color={passing ? "primary" : "secondary"} component="body1">
            {passing ? "passing" : "failing"}
        </Typography>
        { (errors != null && !passing)
            ? <Tooltip title={<ul>{errors.map((err) => <li>{err}</li>)}</ul>}><Info/></Tooltip>
            : ""
        }
    </span>;
}

function FeaturePassingChange({ oldPassing, newPassing, oldErrors, newErrors}) {
    if (oldPassing === newPassing || oldPassing == null) {
        return <FeaturePassing passing={newPassing} errors={newErrors}/>
    }

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
                <TableCell>Date</TableCell>
                <TableCell>Bucket:Path</TableCell>
                <TableCell>Type</TableCell>
                <TableCell>Compliance</TableCell>
            </React.Fragment>
        );
    } else {
        return (
            <React.Fragment>
                <TableCell>Date</TableCell>
                <TableCell>Compliance</TableCell>
            </React.Fragment>
        );
    }
}

function LogTableCells(l) {
    if (l.kind === "tfstate") {
        return (
            <React.Fragment>
                <TableCell>{l.date_time}</TableCell>
                <TableCell>{l.details}</TableCell>
                <TableCell>{LinesChangedLabel(l)}</TableCell>
                <TableCell>{ValidationState(l)}</TableCell>
            </React.Fragment>
        );
    } else {
        return (
            <React.Fragment>
                <TableCell>{l.date_time}</TableCell>
                <TableCell>{ValidationState(l)}</TableCell>
            </React.Fragment>
        );
    }
}

export class LogsTable extends React.Component {
    constructor(props) {
        super(props);
    }

    state = {
        logs: [],
        updating: false,
        deleting: null,
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
                {this.state.deleting != null
                    ? <DeleteDialog
                        message={"Delete Log Event #" + this.state.deleting + "?"}
                        deleteUrl={"/logs/" + this.state.deleting}
                        onCancel={() => this.setState({deleting: null})}
                        onDelete={() => {
                            this.fetchData();
                            this.setState({deleting: null});
                        }}
                    />
                    : ""
                }

                <Title>Latest {this.props.kind === "tfstate" ? "State Changes" : "Validations"}</Title>
                <Table size="small">
                    <TableHead>
                        <TableRow>
                            {LogTableColumns(this.props.kind)}
                            <TableCell align="right"/>
                        </TableRow>
                    </TableHead>
                    <TableBody>
                        { this.state.logs
                            .filter(l => l.kind === this.props.kind)
                            .map(l =>
                                <TableRow key={l.id}>
                                    {LogTableCells(l)}
                                    <TableCell align="right">
                                        <IconButton onClick={() => this.props.onSelectInfo(l.id)} >
                                            <Info/>
                                        </IconButton>
                                        <IconButton onClick={() => this.setState({ deleting: l.id })}>
                                            <Delete/>
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