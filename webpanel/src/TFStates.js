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

function ComplianceText(data) {
    if (data.compliance_present === true) {
        if (data.compliance_errors === 0) {
            return <Typography color="primary">yes ({data.compliance_tests} passing)</Typography>
        } else {
            return <Typography color="secondary">no ({data.compliance_errors}/{data.compliance_tests} failing)</Typography>
        }
    } else {
        return <Typography>unchecked</Typography>
    }
}

export class TFStatesTable extends React.Component {
    state = {
        tfstates: []
    };

    componentDidMount() {
        axios.get(`http://localhost:8080/tfstates/json`)
            .then(res => {
                const tfstates = res.data;
                this.setState({ tfstates });
            })
    }

    render() {
        return (
            <React.Fragment>
                <Title>Terraform States</Title>
                <Table size="small">
                    <TableHead>
                        <TableRow>
                            <TableCell>Bucket</TableCell>
                            <TableCell>Path</TableCell>
                            <TableCell>Last Update</TableCell>
                            <TableCell>Compliant</TableCell>
                            <TableCell>Actions</TableCell>
                        </TableRow>
                    </TableHead>
                    <TableBody>
                        { this.state.tfstates
                            .map(l => (
                                <TableRow key={l.id}>
                                    <TableCell>{l.bucket}</TableCell>
                                    <TableCell>{l.path}</TableCell>
                                    <TableCell>{l.last_update}</TableCell>
                                    <TableCell>{ComplianceText(l)}</TableCell>
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