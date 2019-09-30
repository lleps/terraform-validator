import React from 'react';
import { makeStyles } from '@material-ui/core/styles';
import Fab from '@material-ui/core/Fab';
import AddIcon from '@material-ui/icons/Add';

const useStyles = makeStyles(theme => ({
    fab: {
        position: 'absolute',
        bottom: theme.spacing(4),
        right: theme.spacing(4),
    },
    extendedIcon: {
        marginRight: theme.spacing(1),
    },
}));

export default function FloatingActionButtons({ onClick }) {
    const classes = useStyles();

    return (
        <div>
            <Fab color="secondary" aria-label="add" className={classes.fab} onClick={(e) => onClick(e)}>
                <AddIcon />
            </Fab>
        </div>
    );
}