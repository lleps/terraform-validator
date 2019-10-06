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
import {Delete, Edit, Error, Info, Sync} from "@material-ui/icons";
import Tooltip from "@material-ui/core/Tooltip";
import IconButton from "@material-ui/core/IconButton";
import Dialog from "@material-ui/core/Dialog";
import DialogTitle from "@material-ui/core/DialogTitle";
import DialogContent from "@material-ui/core/DialogContent";
import TextField from "@material-ui/core/TextField";
import DialogActions from "@material-ui/core/DialogActions";
import {DeleteDialog} from "./DeleteDialog";
import {Account, TagList, TagListField} from "./TagList";
import {SelectAccount} from "./Account";
import LinearProgress from "@material-ui/core/LinearProgress";

export function TFStateDialog({ editMode, onAdd, onCancel, id }) {
    const [loading, setLoading] = React.useState(false);
    const [account, setAccount] = React.useState("");
    const [bucket, setBucket] = React.useState("");
    const [path, setPath] = React.useState("");
    const [inputError, setInputError] = React.useState("");
    const [tags, setTags] = React.useState(!editMode ? ["default"] : []);

    React.useEffect(() => {
        if (editMode) {
            setLoading(true);
            axios.get("/tfstates/" + id)
                .then(res => {
                    setAccount(res.data.account);
                    setPath(res.data.path);
                    setBucket(res.data.bucket);
                    setTags(res.data.tags || []);
                    setLoading(false);
                })
                .catch(err => console.log("error getting details: " + err));
        }
    }, [editMode, id]);

    function onClickOk() {
        let body = {
            account: account,
            bucket: bucket,
            path: path,
            tags: tags
        };
        function anyTFStateMatchingBucketPath(list) {
            return list.findIndex(obj => (obj.path === path && obj.bucket === bucket)) !== -1
        }

        if (!editMode) { // POST (with tfstate duplication check)
            axios.get(`/tfstates`).then(res => {
                if (anyTFStateMatchingBucketPath(res.data)) {
                    setInputError("The bucket:path combination already exists.");
                } else {
                    axios.post(`/tfstates`, body).then(() => {
                        onAdd()
                    })
                }
            })
        } else { // just PUT
            axios.put(`/tfstates/` + id, body).then(() => {
                onAdd()
            })
        }
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
        return account.length > 0 && bucket.length > 0 && path.length > 0;
    }

    return <Dialog open={true} onClose={() => onCancel()}>
            <DialogTitle>{editMode ? "Edit" : "Add"} TFState</DialogTitle>
            <DialogContent>
                { loading ? <div align={"center"}><CircularProgress/></div> : "" }
                <TextField
                    autoComplete="off"
                    autoFocus
                    value={account}
                    margin="dense"
                    id="account"
                    label="Account Alias"
                    onChange={e => setAccount(e.target.value)}
                    type="text"
                    fullWidth
                    disabled={loading}
                />
                <TextField
                    autoComplete="off"
                    value={bucket}
                    margin="dense"
                    id="bucket"
                    label="S3 Bucket"
                    onChange={handleBucketChange}
                    type="text"
                    fullWidth
                    disabled={loading}
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
                    disabled={loading}
                />
                <TagListField tags={tags} onChange={(t) => setTags(t)}/>
            </DialogContent>
            <DialogActions>
                <Button onClick={() => onCancel()} color="primary">
                    Cancel
                </Button>
                <Button onClick={() => onClickOk()} color="primary" disabled={loading || !inputNotEmpty() || inputError !== ""}>
                    { editMode ? "Save" : "Add" }
                </Button>
            </DialogActions>
        </Dialog>
}

function LastUpdateLabel(data) {
    if (data.last_update === "") {
        return <span>-</span>;
    }
    return <span>{data.last_update}</span>;
}

function TableEntryCompliance(data) {
    // states currently in validation
    if (data.force_validation === true) {
        return <LinearProgress color="primary"/>
    }

    // this data is set locally, to display something while the sync post request is in progress.
    if (data.force_validation_locally === true) {
        return <CircularProgress/>
    }

    // state never validated
    if (data.compliance_present !== true) {
        return <Typography>-</Typography>
    }

    // some error in compliance
    if (data.compliance_error !== undefined) {
        return <Tooltip
            title={
                <React.Fragment>
                    {data.compliance_error}
                </React.Fragment>
            }>
            <Error color={"error"}/>
        </Tooltip>
    }

    // full pair state and tooltip
    return <span>{TableEntryComplianceLabel(data)}{TableEntryComplianceTooltip(data)}</span>
}

function TableEntryComplianceLabel(data) {
    if (data.compliance_present === true) {
        if (data.compliance_errors === 0) {
            return <Typography color="primary" component="body1">{data.compliance_tests}/{data.compliance_tests}</Typography>
        } else {
            return <Typography color="secondary" component="body1">{data.compliance_tests-data.compliance_errors}/{data.compliance_tests}</Typography>
        }
    } else {
        return <Typography>-</Typography>
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
        updating: false,
        account: "All",
        updatingIds: new Set(),
    };

    fetchData() {
        this.setState({ updating: true });

        axios.get(`/tfstates`)
            .then(res => {
                const tfstates = res.data;
                this.setState({ tfstates });
                this.setState({ updating: false });
            })
    }

    onSync(id) {
        // edit the entry locally and set the flag update_validation_locally.
        // just to show feedback while the POST below is going
        let newData = this.state.tfstates.filter(tfs => tfs.id === id)[0];
        newData.force_validation_locally = true;
        let newTFStates = this.state.tfstates.map(tfs => tfs.id === id ? newData : tfs);
        this.setState({ tfstates: newTFStates });

        // do the request
        axios.post(`/tfstates/` + id + `/validate`).then(() => this.fetchData());
    }

    syncTimer() {
        // For every entity that's syncing, refetch from db
        this.state.tfstates.forEach(tfstate => {
            if (tfstate.force_validation === true) {
                let id = tfstate.id;
                if (!this.state.updatingIds.has(id)) {
                    this.setState({ updatingIds: new Set(this.state.updatingIds).add(id) });

                    axios.get(`/tfstates/` + id)
                        .then(res => {
                            let newData = res.data;
                            let newTFStates = this.state.tfstates.map(tfs => tfs.id === id ? newData : tfs);
                            let newUpdatingIds = new Set(this.state.updatingIds);
                            newUpdatingIds.delete(id);
                            this.setState({
                                tfstates: newTFStates,
                                updatingIds: newUpdatingIds
                            });
                        }).catch(() => {
                            let newUpdatingIds = new Set(this.state.updatingIds);
                            newUpdatingIds.delete(id);
                            this.setState({
                                updatingIds: newUpdatingIds
                            });
                        });
                }
            }
        })
    }

    componentDidMount() {
        this.fetchData();
        let interval = setInterval(() => this.syncTimer(), 1000);
        this.setState({ syncInterval: interval });
    }

    componentWillUnmount() {
        clearInterval(this.state.syncInterval);
    }

    render() {
        return (
            <React.Fragment>
                { this.state.deleting != null ? <DeleteDialog
                    deleteUrl={"/tfstates/" + this.state.deleting}
                    message={"Delete TFState?"}
                    onCancel={() => this.setState({ deleting: null })}
                    onDelete={() => {
                        this.setState({ deleting: null });
                        this.fetchData();
                    }}/> : ""
                }

                <div>
                    <Title>Terraform States</Title>
                    <div/>
                    <SelectAccount
                        objs={this.state.tfstates}
                        selected={this.state.account}
                        onSelect={v => this.setState({ account: v })}
                    />
                    <div/>
                </div>
                <Table size="small">
                    <TableHead>
                        <TableRow>
                            <TableCell>Account</TableCell>
                            <TableCell>Bucket@Path</TableCell>
                            <TableCell>Tags</TableCell>
                            <TableCell>Last Update</TableCell>
                            <TableCell>Compliant</TableCell>
                            <TableCell align="right"/>
                        </TableRow>
                    </TableHead>
                    <TableBody>
                        { this.state.tfstates
                            .filter(s => this.state.account === "All" || this.state.account === s.account)
                            .map(l => (
                                <TableRow key={l.id}>
                                    <TableCell><Account account={l.account}/></TableCell>
                                    <TableCell>{l.bucket}<b>@</b>{l.path}</TableCell>
                                    <TableCell>
                                        <TagList tags={l.tags}/>
                                    </TableCell>
                                    <TableCell>{LastUpdateLabel(l)}</TableCell>
                                    <TableCell>{TableEntryCompliance(l)}</TableCell>
                                    <TableCell align="right">
                                        { !l.force_validation ?
                                            <IconButton
                                                onClick={() => this.onSync(l.id)}>
                                                <Sync/>
                                            </IconButton>
                                            : ""
                                        }

                                        <IconButton onClick={() => this.props.onEdit(l.id)}>
                                            <Edit/>
                                        </IconButton>
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