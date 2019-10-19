import * as React from "react";
import {Button, Dialog} from "@material-ui/core";
import DialogTitle from "@material-ui/core/DialogTitle";
import DialogActions from "@material-ui/core/DialogActions";
import CircularProgress from "@material-ui/core/CircularProgress";
import {handledDelete} from "./Requests";

export function DeleteDialog({ message, deleteUrl, onCancel, onDelete }) {
    const [deleting, setDeleting] = React.useState(false);

    function onClickOk() {
        setDeleting(true);
        handledDelete(deleteUrl, () => {
            onDelete();
            setDeleting(false);
        });
    }

    return (
        <Dialog open={true} onClose={() => onCancel()}>
            <DialogTitle>
                {message}
            </DialogTitle>
            <DialogActions>
                <Button onClick={() => onCancel()} color="primary">
                    No
                </Button>
                {deleting ? <CircularProgress/> : ""}
                <Button onClick={() => onClickOk()} color="primary" disabled={deleting}>
                    Yes, Delete
                </Button>
            </DialogActions>
        </Dialog>
    )
}