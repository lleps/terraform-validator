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
import {Delete, Edit} from "@material-ui/icons";
import Link from "@material-ui/core/Link";
import IconButton from "@material-ui/core/IconButton";

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
        editingFeature: "none",
        editingSource: "none",
        loadingTable: false,
        updatingFeature: false,
    };

    onSourceChange(e) {
        this.setState({ editingSource: e.target.value })
    }

    onClickEdit(e, feature, source) {
        if (this.state.loadingTable) return;

        this.setState({ editing: true, editingFeature: feature, editingSource: source });
    }

    fetchTable() {
        this.setState({ loadingTable: true });

        axios.get(`http://localhost:8080/features/json`).then(res => {
            const features = res.data;
            this.setState({ features: features, loadingTable: false });
        }).catch(error => {
            this.setState({ loadingTable: false });
            console.log(error);
        })
    }

    save() {
        if (this.state.editingFeature === "none") return;

        this.setState({ updatingFeature: true });

        axios.post(`http://localhost:8080/features`, {
            name: this.state.editingFeature,
            source: this.state.editingSource,
        }).then(() => {
            this.fetchTable();
            this.setState({ editing: false, editingFeature: "none", editingSource: "none", updatingFeature: false });
        }).catch(error => {
            console.log(error);
        })
    }

    close() {
        this.setState({ editing: false })
    }

    componentDidMount() {
        this.fetchTable()
    }

    loadingSpinner() {
        return <div align="center"><CircularProgress/></div>
    }

    render() {
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
                                <TableCell />
                            </TableRow>
                        </TableHead>
                        <TableBody>
                            { this.state.features
                                .map(f => (
                                    <TableRow key={f.id}>
                                        <TableCell>{f.id}</TableCell>
                                        <TableCell>{EnabledState(f)}</TableCell>
                                        <TableCell align="right">
                                            <IconButton onClick={e => this.onClickEdit(e, f.id, f.source)}>
                                                <Edit/>
                                            </IconButton>
                                            <IconButton><Delete/></IconButton>
                                        </TableCell>
                                    </TableRow>
                                ))}
                        </TableBody>
                    </Table>
                    { this.state.loadingTable ? this.loadingSpinner() : <div/> }

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
                            label={ this.state.editingFeature }
                            rowsMax="20"
                            inputProps={{
                                style: {fontSize: 15, fontFamily: "Monospace" }
                            }}
                            value={this.state.editingSource}
                            onChange={e => this.onSourceChange(e)}
                            fullWidth
                            margin="normal"
                            variant="filled"
                        />
                    </DialogContent>
                    <DialogActions>
                        { this.state.updatingFeature ? this.loadingSpinner() : <div/> }
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