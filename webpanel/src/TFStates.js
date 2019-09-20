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
import axios from 'axios';
import {Delete, Info} from "@material-ui/icons";
import Tooltip from "@material-ui/core/Tooltip";
import IconButton from "@material-ui/core/IconButton";
import Dialog from "@material-ui/core/Dialog";
import DialogTitle from "@material-ui/core/DialogTitle";
import DialogContent from "@material-ui/core/DialogContent";
import TextField from "@material-ui/core/TextField";
import DialogActions from "@material-ui/core/DialogActions";
import {DeleteDialog} from "./DeleteDialog";

export function TFStateAddDialog({ onAdd, onCancel }) {
    const [bucket, setBucket] = React.useState("");
    const [path, setPath] = React.useState("");
    const [inputError, setInputError] = React.useState("");

    function onClickOk() {
        axios.get(`http://localhost:8080/tfstates/json`).then(res => {
            if (res.data.findIndex(obj => (obj.path === path && obj.bucket === bucket)) === -1) {
                axios.post(`http://localhost:8080/tfstates`, {
                    bucket: bucket,
                    path: path,
                }).then(() => {
                    onAdd();
                }).catch(error => {
                    console.log(error);
                })
            } else {
                setInputError("That bucket:path is already registered.");
            }
        }).catch(error => {
            console.log(error);
        })
    }

    function handlePathChange(e) {
        setPath(e.target.value);
        setInputError("");
    }

    function handleBucketChange(e) {
        setBucket(e.target.value);
        setInputError("");
    }

    function inputNotEmpty() {
        return bucket.length > 0 && path.length > 0;
    }

    return <div>
        <Dialog open={true} onClose={() => onCancel()}>
            <DialogTitle>Add TFState</DialogTitle>
            <DialogContent>
                <TextField
                    autoComplete="off"
                    autoFocus
                    value={bucket}
                    margin="dense"
                    id="bucket"
                    label="S3 Bucket"
                    onChange={handleBucketChange}
                    type="text"
                    fullWidth
                />
                <TextField
                    autoComplete="off"
                    value={path}
                    margin="dense"
                    id="path"
                    label="Path to TFState"
                    onChange={handlePathChange}
                    error={inputError !== ""}
                    helperText={inputError}
                    type="text"
                    fullWidth
                />
            </DialogContent>
            <DialogActions>
                <Button onClick={() => onCancel()} color="primary">
                    Cancel
                </Button>
                <Button onClick={() => onClickOk()} color="primary" disabled={!inputNotEmpty() || inputError !== ""}>
                    Add
                </Button>
            </DialogActions>
        </Dialog>
    </div>
}

function LastUpdateLabel(data) {
    if (data.last_update === "") {
        return <span>never</span>;
    }
    return <span>{data.last_update}</span>;
}

function TableEntryComplianceLabel(data) {
    if (data.compliance_present === true) {
        if (data.compliance_errors === 0) {
            return <Typography color="primary" component="body1">yes {data.compliance_tests}/{data.compliance_tests}</Typography>
        } else {
            return <Typography color="secondary" component="body1">no {data.compliance_tests-data.compliance_errors}/{data.compliance_tests}</Typography>
        }
    } else {
        return <Typography>unchecked</Typography>
    }
}

function TableEntryComplianceDetails(data) {
    if (data.compliance_present !== true) {
        return <div/>
    }
    let passing = [];
    let failing = [];
    let errors = [];
    for (let f in data.compliance_features) {
        let result = data.compliance_features[f] === true;
        if (result) {
            passing.push(f);
        } else {
            failing.push(f);
        }
        let errorList = data.compliance_fail_messages[f];
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

function TableEntryComplianceTooltip(data) {
    if (data.last_update === "") {
        return <div/>
    }

    return (
        <Tooltip
            title={
                <React.Fragment>
                    {TableEntryComplianceDetails(data)}
                </React.Fragment>
            }>
            <Info/>
        </Tooltip>
    );
}

export class TFStatesTable extends React.Component {
    state = {
        tfstates: [],
        deleting: null,
        updating: false
    };

    fetchData() {
        this.setState({ updating: true });
        axios.get(`http://localhost:8080/tfstates/json`)
            .then(res => {
                const tfstates = res.data;
                this.setState({ tfstates });
                this.setState({ updating: false });
            })
    }

    componentDidMount() {
        this.fetchData()
    }

    render() {
        return (
            <React.Fragment>
                { this.state.deleting != null ? <DeleteDialog
                    deleteUrl={"http://localhost:8080/tfstates/" + this.state.deleting}
                    message={"Delete TFState #" + this.state.deleting + "?"}
                    onCancel={() => this.setState({ deleting: null })}
                    onDelete={() => {
                        this.setState({ deleting: null });
                        this.fetchData();
                    }}/> : ""
                }

                <Title>Terraform States</Title>
                <Table size="small">
                    <TableHead>
                        <TableRow>
                            <TableCell>Bucket</TableCell>
                            <TableCell>Path</TableCell>
                            <TableCell>Last Update</TableCell>
                            <TableCell>Compliant</TableCell>
                            <TableCell align="right"/>
                        </TableRow>
                    </TableHead>
                    <TableBody>
                        { this.state.tfstates
                            .map(l => (
                                <TableRow key={l.id}>
                                    <TableCell>{l.bucket}</TableCell>
                                    <TableCell>{l.path}</TableCell>
                                    <TableCell>{LastUpdateLabel(l)}</TableCell>
                                    <TableCell>{TableEntryComplianceLabel(l)} {TableEntryComplianceTooltip(l)}</TableCell>
                                    <TableCell align="right">
                                        <IconButton onClick={() => this.setState({ deleting: l.id })}>
                                            <Delete/>
                                        </IconButton>
                                    </TableCell>
                                </TableRow>
                            ))}
                    </TableBody>
                </Table>
                { this.state.updating ? <div align="center"><CircularProgress/></div> : "" }
            </React.Fragment>
        )
    }
}