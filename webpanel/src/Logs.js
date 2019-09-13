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
import TextField from "@material-ui/core/TextField";
import DialogActions from "@material-ui/core/DialogActions";
import Dialog from "@material-ui/core/Dialog";
import {Link} from "react-router-dom";

function ValidationText(errors, tests) {
    if (errors === 0) {
        return <Typography color="primary" component="body1">{tests}/{tests}</Typography>
    } else {
        return <Typography color="secondary" component="body1">{tests-errors}/{tests}</Typography>
    }
}

function ValidationState(data) {
    let prevErrors = data.compliance_errors_prev;
    let prevTests = data.compliance_tests_prev;
    if (prevTests === 0) {
        return ValidationText(data.compliance_errors, data.compliance_tests);
    } else {
        return (
            <div>
                {ValidationText(prevErrors, prevTests)}
                <TrendingFlat/>
                {ValidationText(data.compliance_errors, data.compliance_tests)}
            </div>
        )
    }
}

function Lines(data) {
    if (data.compliance_tests_prev === 0) {
        return <div>new</div>
    }

    return <div>change</div>
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

export class LogDetailsDialog extends React.Component {
    state = {
        details: null,
        diffHtml: ""
    };

    componentDidMount() {
        axios.get("http://localhost:8080/logs/json/" + this.props.id)
            .then(res => {
                const details = res.data;
                this.setState({ details: details, diffHtml: details.state_diff_html });
            })
    }

    render() {
        return (
            <Dialog
                fullWidth="md"
                maxWidth="md"
                open={true}
                onClose={() => this.props.onClose()} aria-labelledby="form-dialog-title">
                <DialogContent>
                    { this.state.details === null ? <div align="center"><CircularProgress/></div> : "" }
                    <div
                        className="code"
                        dangerouslySetInnerHTML={{__html: this.state.diffHtml} }>
                    </div>
                </DialogContent>
                <DialogActions>
                    { false ? this.loadingSpinner() : <div/> }
                    <Button onClick={() => this.props.onClose()} color="primary">
                        Close
                    </Button>
                </DialogActions>
            </Dialog>
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
                            <TableCell>Type</TableCell>
                            <TableCell>Compliance</TableCell>
                            <TableCell align="right"/>
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
                                    <TableCell>{ValidationState(l)}</TableCell>
                                    <TableCell align="right">
                                        <Link to={"/logs/" + l.id}><Info/></Link>
                                        <Link to={"/logs/" + l.id}><Delete/></Link>
                                    </TableCell>
                                </TableRow>
                            ))}
                    </TableBody>
                </Table>
            </React.Fragment>
        )
    }
}

/*
tarea: mostrar la diff decentemente.
no importa lo demas.

ok. que es una diff decente:

una diff html, pero que sea codigo.
* mostrar una fuente monospaced
* un fondo gris claro.
* sacar icono del parrafo.
* tabular correctamente el html
* mejorar los colores

ok,

 */