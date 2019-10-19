import React from "react";
import Title from "./Title";
import Table from "@material-ui/core/Table";
import TableHead from "@material-ui/core/TableHead";
import TableRow from "@material-ui/core/TableRow";
import TableCell from "@material-ui/core/TableCell";
import TableBody from "@material-ui/core/TableBody";
import {Button} from "@material-ui/core";
import CircularProgress from "@material-ui/core/CircularProgress";
import {Delete, Edit, Error, Info, Sync} from "@material-ui/icons";
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
import {ComplianceResult} from "./Compliance";
import {handledGet, handledPost, handledPut} from "./Requests";

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
            handledGet("/tfstates/" + id, data => {
                setAccount(data.account);
                setPath(data.path);
                setBucket(data.bucket);
                setTags(data.tags || []);
                setLoading(false);
            });
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
            handledGet(`/tfstates`, data => {
                if (anyTFStateMatchingBucketPath(data)) {
                    setInputError("That bucket:path combination already exists.");
                } else {
                    handledPost(`/tfstates`, body, () => onAdd());
                }
            })
        } else { // just PUT
            handledPut(`/tfstates/` + id, body, () => onAdd());
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

    return <ComplianceResult result={data.compliance_result}/>
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

        handledGet(`/tfstates`, data => {
            this.setState({ tfstates: data, updating: false });
        });
    }

    mutateTFState(id, mutator) {
        let listCopy = [...this.state.tfstates];
        let current = listCopy.filter(tfs => tfs.id === id);
        if (current.length !== 1) return;
        let mutated = mutator(current[0]);
        listCopy = listCopy.map(tfs => tfs.id === id ? mutated : tfs);
        this.setState({ tfstates: listCopy });
    }

    onSync(id) {
        // edit the entry locally and set the flag update_validation_locally.
        // just to show feedback while the POST below is going
        this.mutateTFState(id, old => {
            old.force_validation_locally = true;
            return old;
        });

        // do the request
        handledPost(`/tfstates/` + id + `/validate`, {}, () => {
            this.mutateTFState(id, old => {
                old.force_validation = true;
                return old;
            });
        });
    }

    syncTimer() {
        // For every entity that's syncing, refetch from db
        this.state.tfstates.forEach(tfstate => {
            if (tfstate.force_validation === true) {
                let id = tfstate.id;
                if (!this.state.updatingIds.has(id)) {
                    this.setState({ updatingIds: new Set(this.state.updatingIds).add(id) });

                    handledGet(`/tfstates/` + id,
                        data => this.mutateTFState(id, () => data),
                        () => {
                            // remove from ongoing set
                            let newUpdatingIds = new Set(this.state.updatingIds);
                            newUpdatingIds.delete(id);
                            this.setState({ updatingIds: newUpdatingIds });
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
                                        { (!l.force_validation && !l.force_validation_locally) ?
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