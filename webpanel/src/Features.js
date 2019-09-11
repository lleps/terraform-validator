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
import Dialog from "@material-ui/core/Dialog";
import DialogContent from "@material-ui/core/DialogContent";
import TextField from "@material-ui/core/TextField";
import DialogActions from "@material-ui/core/DialogActions";
import axios from 'axios';

function EnabledState(data) {
    if (data.enabled === true) {
        return <Typography color="primary">enabled</Typography>
    } else {
        return <Typography color="secondary">disabled</Typography>
    }
}

export class FeaturesTable extends React.Component {
    state = {
        features: [],
        editing: false,
        currentEditingFeature: "none",
        currentEditingSource: "none"
    };

    onSourceChange(e) {
        this.setState({ currentEditingSource: e.target.value })
    }

    onEditClick(e, feature, source) {
        console.log(e);
        this.setState({ editing: true, currentEditingFeature: feature, currentEditingSource: source });
    }

    save() {
        if (this.state.currentEditingFeature === "none") return
        let config = {
            headers: {'Access-Control-Allow-Origin': '*'}
        };
        axios.post(`http://localhost:8080/features/json`, {
            name: this.state.currentEditingFeature,
            source: this.state.currentEditingSource,
        }, config)
    }

    close() {
        this.setState({ editing: false })
    }

    componentDidMount() {
        axios.get(`http://localhost:8080/features/json`)
            .then(res => {
                const features = res.data;
                this.setState({ features: features });
            })
    }

    render() {
        if (this.state.features.length === 0) {
            return <div align="center"><CircularProgress/></div>
        }

        return (
            <div>
                { /* table */ }
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
                                            <Button onClick={e => this.onEditClick(e, f.id, f.source)}>Edit</Button>
                                            <Button>Delete</Button>
                                        </TableCell>
                                    </TableRow>
                                ))}
                        </TableBody>
                    </Table>
                </React.Fragment>

                { /* edit dialog */ }
                <Dialog
                    fullWidth="md"
                    maxWidth="md"
                    open={this.state.editing}
                    onClose={() => this.close()} aria-labelledby="form-dialog-title">
                    <DialogContent>
                        <TextField
                            id="filled-full-width"
                            multiline
                            label={ this.state.currentEditingFeature }
                            rowsMax="20"
                            inputProps={{
                                style: {fontSize: 15, fontFamily: "Monospace" }
                            }}
                            value={this.state.currentEditingSource}
                            onChange={e => this.onSourceChange(e)}
                            fullWidth
                            margin="normal"
                            variant="filled"
                        />
                    </DialogContent>
                    <DialogActions>
                        <Button onClick={() => this.close()} color="primary">
                            Cancel
                        </Button>
                        <Button onClick={() => this.save()} color="primary">
                            Save
                        </Button>
                    </DialogActions>
                </Dialog>
            </div>
        )
    }
}