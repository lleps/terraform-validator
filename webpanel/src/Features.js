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

function EnabledState(data) {
    if (data.enabled === true) {
        return <Typography color="primary">enabled</Typography>
    } else {
        return <Typography color="secondary">disabled</Typography>
    }
}

export class FeaturesTable extends React.Component {
    state = {
        features: []
    };

    componentDidMount() {
        axios.get(`http://localhost:8080/features/json`)
            .then(res => {
                const features = res.data;
                this.setState({ features });
            })
    }

    render() {
        if (this.state.features.length === 0) {
            return <div align="center"><CircularProgress/></div>
        }

        return (
            <React.Fragment>
                <Title>Features</Title>
                <Table size="small">
                    <TableHead>
                        <TableRow>
                            <TableCell>Name</TableCell>
                            <TableCell>State</TableCell>
                            <TableCell align="right">Actions</TableCell>
                        </TableRow>
                    </TableHead>
                    <TableBody>
                        { this.state.features
                            .map(f => (
                                <TableRow key={f.id}>
                                    <TableCell>{f.id}</TableCell>
                                    <TableCell>{EnabledState(f)}</TableCell>
                                    <TableCell align="right">
                                        <Button>Toggle</Button>
                                        <Button>Edit</Button>
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