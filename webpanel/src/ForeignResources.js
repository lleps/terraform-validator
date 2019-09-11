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

export class ForeignResourcesTable extends React.Component {
    state = {
        foreignresources: []
    };

    componentDidMount() {
        axios.get(`http://localhost:8080/foreignresources/json`)
            .then(res => {
                const foreignresources = res.data;
                this.setState({ foreignresources });
            })
    }

    render() {
        if (this.state.foreignresources.length === 0) {
            return <div align="center"><CircularProgress/></div>
        }

        return (
            <React.Fragment>
                <Title>Foreign Resources</Title>
                <Table size="small">
                    <TableHead>
                        <TableRow>
                            <TableCell>Date</TableCell>
                            <TableCell>Type</TableCell>
                            <TableCell>ID</TableCell>
                            <TableCell align="right">Actions</TableCell>
                        </TableRow>
                    </TableHead>
                    <TableBody>
                        { this.state.foreignresources
                            .map(l => (
                                <TableRow key={l.id}>
                                    <TableCell>{l.date_time}</TableCell>
                                    <TableCell>{l.resource_type}</TableCell>
                                    <TableCell>{l.resource_id}</TableCell>
                                    <TableCell align="right">
                                        <Button>Details</Button>
                                        <Button>Exclude</Button>
                                    </TableCell>
                                </TableRow>
                            ))}
                    </TableBody>
                </Table>
            </React.Fragment>
        )
    }
}