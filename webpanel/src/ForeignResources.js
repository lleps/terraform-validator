import React from "react";
import Title from "./Title";
import Table from "@material-ui/core/Table";
import TableHead from "@material-ui/core/TableHead";
import TableRow from "@material-ui/core/TableRow";
import TableCell from "@material-ui/core/TableCell";
import TableBody from "@material-ui/core/TableBody";
import CircularProgress from "@material-ui/core/CircularProgress";
import axios from 'axios';
import {TimeAgo} from "./Time";
import {Info} from "@material-ui/icons";
import Tooltip from "@material-ui/core/Tooltip";
import {Button, Typography} from "@material-ui/core";
import Dialog from "@material-ui/core/Dialog";
import DialogTitle from "@material-ui/core/DialogTitle";
import DialogContent from "@material-ui/core/DialogContent";
import DialogActions from "@material-ui/core/DialogActions";
import IconButton from "@material-ui/core/IconButton";

export function ForeignResourceDetailsDialog({ id, onClose }) {
    const [details, setDetails] = React.useState("");
    const [loading, setLoading] = React.useState(true);
    const [type, setType] = React.useState("");

    React.useEffect(() => {
        axios.get("/foreignresources/" + id).then(res => {
            setLoading(false);
            setType(res.data.resource_type);
            setDetails(res.data.resource_details);
        }).catch(err => {
            console.log(err);
            setLoading(false)
        })
    }, []);

    return <Dialog
        fullWidth="md"
        maxWidth="md"
        open={true}
        onClose={() => onClose()} aria-labelledby="form-dialog-title">
        <DialogTitle id="customized-dialog-title" onClose={() => {}}>
            {type} Resource Details
        </DialogTitle>
        <DialogContent>
            { loading ? <div align="center"><CircularProgress/></div> : "" }
            <div className="code">
                {details}
            </div>
        </DialogContent>
        <DialogActions>
            <Button onClick={() => onClose()} color="primary">
                Close
            </Button>
        </DialogActions>
    </Dialog>
}

export class ForeignResourcesTable extends React.Component {
    state = {
        foreignresources: []
    };

    componentDidMount() {
        axios.get(`/foreignresources`)
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
                            <TableCell>Discovered</TableCell>
                            <TableCell>Type</TableCell>
                            <TableCell>Resource</TableCell>
                            <TableCell align="right"/>
                        </TableRow>
                    </TableHead>
                    <TableBody>
                        { this.state.foreignresources
                            .map(l => (
                                <TableRow key={l.id}>
                                    <TableCell><TimeAgo timestamp={l.timestamp}/></TableCell>
                                    <TableCell>{l.resource_type}</TableCell>
                                    <TableCell>
                                        {l.resource_id}
                                    </TableCell>
                                    <TableCell align={"right"}>
                                        <IconButton onClick={() => this.props.onSelect(l.id)}>
                                            <Info/>
                                        </IconButton>
                                    </TableCell>
                                </TableRow>
                            ))}
                    </TableBody>
                </Table>
            </React.Fragment>
        )
    }
}