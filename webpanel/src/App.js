import React from 'react';
import './App.css';
import {BrowserRouter as Router, Route} from "react-router-dom";
import Navigation from "./Navigation";
import Grid from "@material-ui/core/Grid";
import Paper from "@material-ui/core/Paper";
import {LogDetailsDialog, StateLogsTable, ValidationLogsTable} from "./Logs";
import {FeatureAddDialog, FeatureEditDialog, FeaturesTable} from "./Features";
import {TFStatesTable} from "./TFStates";
import {ForeignResourcesTable} from "./ForeignResources";
import {makeStyles} from "@material-ui/core";
import FloatingActionButtons from "./Fab";

const useStyles = makeStyles(theme => ({
    paper: {
        padding: theme.spacing(2),
        display: 'flex',
        overflow: 'auto',
        flexDirection: 'column',
    },
}));


function Logs(props) {
    const classes = useStyles();

    return (
        <Grid item xs={12}>
            <Route path={`${props.match.url}/:id`} component={LogDetails}/>
            <Paper className={classes.paper}>
                <StateLogsTable
                    onSelectInfo={(id) => props.history.push("/logs/" + id) }
                />
            </Paper>
            <li></li>
            <Paper className={classes.paper}>
                <ValidationLogsTable
                    onSelectInfo={(id) => props.history.push("/logs/" + id) }
                />
            </Paper>
        </Grid>
    );
}


function FeatureAdd(props) {
    return (
        <FeatureAddDialog
            onAdd={(name) => pushRefresh(props.history, "/features/edit/" + name)}
            onCancel={() => props.history.push("/features")}
        />
    );
}

function pushRefresh(history, url) {
    history.push("/");
    history.push(url);
}

function FeatureEdit(props) {
    let id = props.match.params.id;
    return (
        <FeatureEditDialog
            id={id}
            onCancel={() => props.history.push("/features")}
            onSave={() => props.history.push("/features")}
        />
    );
}

function Features(props) {
    const classes = useStyles();

    return (
        <Grid item xs={12}>
            <Route path={`${props.match.url}/add`} component={FeatureAdd} />
            <Route path={`${props.match.url}/edit/:id`} component={FeatureEdit} />
            <Paper className={classes.paper}>
                <FeaturesTable onSelect={(id) => props.history.push("/features/edit/" + id)} />
            </Paper>
            <FloatingActionButtons onClick={() => props.history.push("/features/add")} />
        </Grid>
    );
}

function LogDetails(props) {
    let id = props.match.params.id;
    return (
        <LogDetailsDialog
            id={id}
            onClose={() => {
                props.history.push(`/logs`)
            }}
        />
    );
}

//

function TFStates() {
    const classes = useStyles();

    return (
        <div>
            <Grid item xs={12}>
                <Paper className={classes.paper}>
                    <TFStatesTable/>
                </Paper>
            </Grid>
            <FloatingActionButtons/>
        </div>
    );
}

function ForeignResources() {
    const classes = useStyles();

    return (
        <Grid item xs={12}>
            <Paper className={classes.paper}>
                <ForeignResourcesTable/>
            </Paper>
        </Grid>
    );
}

function Routes() {
    return (
        <div>
            <Route path="/logs" component={Logs}/>
            <Route path="/features" component={Features}/>
            <Route path="/tfstates" component={TFStates}/>
            <Route path="/foreignresources" component={ForeignResources}/>
        </div>
    );
}

function App() {
    return (
        <Router>
            <Navigation title={"Terraform Monitor"} content={Routes}/>
        </Router>
    );
}

export default App;
