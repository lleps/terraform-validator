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

const axios = require('axios');

function ValidationState(data) {
    if (data.compliance_errors === 0) {
        return <Typography color="primary" component="body1">{data.compliance_tests}/{data.compliance_tests} tests</Typography>
    } else {
        return <Typography color="secondary" component="body1">{data.compliance_tests-data.compliance_errors}/{data.compliance_tests} tests</Typography>
    }
}

function Lines(data) {
    const added = data.lines_added;
    const removed = data.lines_removed;
    if (added > 0 && removed === 0) { // +x;
        return (
            <div>
                <Typography component="body1">+{added};</Typography>
            </div>
        )
    } else if (added === 0 && removed > 0) {// -y;
        return (
            <div>
                <Typography component="body1">-{removed};</Typography>
            </div>
        )
    } else { // +x; -y
        return (
            <div>
                <Typography component="body1">+{added}; </Typography>
                <Typography component="body1">-{removed};</Typography>
            </div>
        )
    }
}

export class ValidationLogsTable extends React.Component {
    state = {
        logs: []
    };

    componentDidMount() {
        axios.get(`http://localhost:8080/logs/json`)
            .then(res => {
                const logs = res.data;
                this.setState({ logs });
            })
    }

    render() {
        if (this.state.logs.length === 0) {
            return <div align="center"><CircularProgress/></div>
        }

        return (
            <React.Fragment>
                <Title>Latest Validations</Title>
                <Table size="small">
                    <TableHead>
                        <TableRow>
                            <TableCell>Date</TableCell>
                            <TableCell>Result</TableCell>
                            <TableCell align="right">Actions</TableCell>
                        </TableRow>
                    </TableHead>
                    <TableBody>
                        { this.state.logs
                            .filter(l => l.kind === "validation")
                            .map(l => (
                                <TableRow key={l.id}>
                                    <TableCell>{l.date_time}</TableCell>
                                    <TableCell>{ValidationState(l)}</TableCell>
                                    <TableCell align="right">
                                        <Button>Details</Button>
                                        <Button>Delete</Button>
                                    </TableCell>
                                </TableRow>
                            ))}
                    </TableBody>
                </Table>
            </React.Fragment>
        )
    }
}

export class StateLogsTable extends React.Component {
    state = {
        logs: []
    };

    componentDidMount() {
        axios.get(`http://localhost:8080/logs/json`)
            .then(res => {
                const logs = res.data;
                this.setState({ logs });
            })
    }

    render() {
        if (this.state.logs.length === 0) {
            return <div align="center"><CircularProgress/></div>
        }

        return (
            <React.Fragment>
                <Title>Latest State Changes</Title>
                <Table size="small">
                    <TableHead>
                        <TableRow>
                            <TableCell>Date</TableCell>
                            <TableCell>Bucket:Path</TableCell>
                            <TableCell>Lines Changed</TableCell>
                            <TableCell>Compliance Change</TableCell>
                            <TableCell align="right">Actions</TableCell>
                        </TableRow>
                    </TableHead>
                    <TableBody>
                        { this.state.logs
                            .filter(l => l.kind === "tfstate")
                            .map(l => (
                                <TableRow key={l.id}>
                                    <TableCell>{l.date_time}</TableCell>
                                    <TableCell>{l.details}</TableCell>
                                    <TableCell>{Lines(l)}</TableCell>
                                    <TableCell>{ValidationState(l)} -> {ValidationState(l)}</TableCell>
                                    <TableCell align="right">
                                        <Button>Details</Button>
                                        <Button>Delete</Button>
                                    </TableCell>
                                </TableRow>
                            ))}
                    </TableBody>
                </Table>
            </React.Fragment>
        )
    }
}