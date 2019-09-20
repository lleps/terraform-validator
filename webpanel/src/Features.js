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
import IconButton from "@material-ui/core/IconButton";
import DialogTitle from "@material-ui/core/DialogTitle";
import {DeleteDialog} from "./DeleteDialog";

export function FeatureAddDialog({ onAdd, onCancel }) {
    const [name, setName] = React.useState("");
    const [inputError, setInputError] = React.useState("");

    function onClickOk() {
        axios.get(`http://localhost:8080/features/json`).then(res => {
            if (res.data.findIndex(obj => obj.id === name) === -1) {
                if (!nameIsValid(name)) {
                    setInputError("Invalid name.");
                    return;
                }

                axios.post(`http://localhost:8080/features`, {
                    name: name,
                    source: "Feature: " + name + "\n\n",
                }).then(() => {
                    onAdd(name);
                }).catch(error => {
                    console.log(error);
                })
            } else {
                setInputError("Feature '" + name + "' already exists.");
            }
        }).catch(error => {
            console.log(error);
        })
    }

    function handleChange(e) {
        setName(e.target.value);
        if (!nameIsValid(e.target.value)) {
            setInputError("Must match regex 'a-zA-Z0-9_'.");
        } else {
            setInputError("");
        }
    }

    function nameIsValid(name) {
        let regex = new RegExp("^[a-zA-Z0-9_]*$");
        return name.length > 0 && name.length < 30 && regex.test(name);
    }

    return <div>
        <Dialog open={true} onClose={() => onCancel()}>
            <DialogTitle>Add Feature</DialogTitle>
            <DialogContent>
                <TextField
                    autoComplete="off"
                    autoFocus
                    value={name}
                    margin="dense"
                    id="name"
                    label="Feature name"
                    onChange={handleChange}
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
                <Button onClick={() => onClickOk()} color="primary" disabled={!nameIsValid(name) || inputError !== ""}>
                    Create
                </Button>
            </DialogActions>
        </Dialog>
    </div>
}

export function FeatureEditDialog({ id, onSave, onCancel }) {
    const [source, setSource] = React.useState("");
    const [loading, setLoading] = React.useState(true);
    const [saving, setSaving] = React.useState(false);

    React.useEffect(() => {
        axios.get("http://localhost:8080/features/json/" + id)
            .then(res => {
                setSource(res.data.source);
                setLoading(false);
            })
            .catch(err => console.log("error getting details: " + err));
    }, []);

    function save() {
        setSaving(true);
        axios.post(`http://localhost:8080/features`, {
            name: id,
            source: source,
        }).then(() => {
            setSaving(false);
            onSave();
        }).catch(error => {
            console.log(error);
        })
    }

    let body;
    if (loading) {
        body = <div align={"center"}><CircularProgress/></div>;
    } else {
        body = <TextField
            id="filled-full-width"
            multiline
            label={id}
            rowsMax="20"
            inputProps={{
                style: {fontSize: 15, fontFamily: "Monospace" }
            }}
            value={source}
            onChange={e => setSource(e.target.value)}
            fullWidth
            margin="normal"
            variant="filled"
        />
    }

    return (
        <Dialog
            fullWidth="md"
            maxWidth="md"
            open={true}
            onClose={() => onCancel()} aria-labelledby="form-dialog-title">
            <DialogTitle>Edit Feature</DialogTitle>
            <DialogContent>
                {body}
            </DialogContent>
            <DialogActions>
                <Button onClick={() => onCancel()} color="primary">
                    Cancel
                </Button>
                <Button onClick={() => save()} color="primary">
                    Save
                </Button>
                { saving ? <CircularProgress/> : "" }
            </DialogActions>
        </Dialog>
    )
}

function FeatureEnabledLabel(data) {
    if (data.enabled === true) {
        return <Typography color="primary">enabled</Typography>
    } else {
        return <Typography color="secondary">disabled</Typography>
    }
}

export class FeaturesTable extends React.Component {
    state = {
        features: [],
        loadingTable: false,
        deleteSelect: null,
    };

    fetchData() {
        this.setState({ loadingTable: true });

        axios.get(`http://localhost:8080/features/json`).then(res => {
            const features = res.data;
            this.setState({ features: features, loadingTable: false });
        }).catch(error => {
            this.setState({ loadingTable: false });
            console.log(error);
        })
    }

    componentDidMount() {
        this.fetchData();
    }

    render() {
        return  (
            <React.Fragment>
                { this.state.deleteSelect != null
                    ? <DeleteDialog
                        message={"Delete feature '" + this.state.deleteSelect + "'?"}
                        deleteUrl={"http://localhost:8080/features/" + this.state.deleteSelect}
                        onDelete={() => {
                            this.setState({ deleteSelect: null});
                            this.fetchData()
                        }}
                        onCancel={() => this.setState({ deleteSelect: null })}
                    />
                    : ""
                }

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
                                    <TableCell>{FeatureEnabledLabel(f)}</TableCell>
                                    <TableCell align="right">
                                        <IconButton onClick={() => this.props.onSelect(f.id)}>
                                            <Edit/>
                                        </IconButton>
                                        <IconButton onClick={() => this.setState({ deleteSelect: f.id })}>
                                            <Delete/>
                                        </IconButton>
                                    </TableCell>
                                </TableRow>
                            ))}
                    </TableBody>
                </Table>
                { this.state.loadingTable ? <div align="center"><CircularProgress/></div> : "" }
            </React.Fragment>
        );
    }
}