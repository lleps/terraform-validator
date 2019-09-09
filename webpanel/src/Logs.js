import Typography from "@material-ui/core/Typography";
import React from "react";
import Title from "./Title";
import Table from "@material-ui/core/Table";
import TableHead from "@material-ui/core/TableHead";
import TableRow from "@material-ui/core/TableRow";
import TableCell from "@material-ui/core/TableCell";
import TableBody from "@material-ui/core/TableBody";
import {Button} from "@material-ui/core";

const axios = require('axios');

function ValidationState(data) {
    if (data.compliance_errors === 0) {
        return <Typography color="primary"><b>PASSING {data.compliance_tests}/{data.compliance_tests}</b></Typography>
    } else {
        return <Typography color="secondary"><b>FAILING {data.compliance_tests-data.compliance_errors}/{data.compliance_tests}</b></Typography>
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
        return (
            <React.Fragment>
                <Title>Latest Validations</Title>
                <Table size="small">
                    <TableHead>
                        <TableRow>
                            <TableCell>Date</TableCell>
                            <TableCell>Result</TableCell>
                            <TableCell>Actions</TableCell>
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
                            <TableCell>Actions</TableCell>
                        </TableRow>
                    </TableHead>
                    <TableBody>
                        { this.state.logs
                            .filter(l => l.kind === "tfstate")
                            .map(l => (
                                <TableRow key={l.id}>
                                    <TableCell>{l.date_time}</TableCell>
                                    <TableCell>{l.details}</TableCell>
                                    <TableCell><Typography color="primary">+25;</Typography> <Typography color="secondary">-4;</Typography></TableCell>
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