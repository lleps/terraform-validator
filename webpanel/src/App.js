import React from 'react';
import './App.css';
import {BrowserRouter as Router, Redirect, Route} from "react-router-dom";
import Navigation from "./Navigation";
import Grid from "@material-ui/core/Grid";
import Paper from "@material-ui/core/Paper";
import {LogDetailsDialog, StateLogsTable, ValidationLogsTable} from "./Logs";
import {FeaturesTable} from "./Features";
import {TFStatesTable} from "./TFStates";
import {ForeignResourcesTable} from "./ForeignResources";
import {makeStyles} from "@material-ui/core";

const useStyles = makeStyles(theme => ({
    paper: {
        padding: theme.spacing(2),
        display: 'flex',
        overflow: 'auto',
        flexDirection: 'column',
    },
}));


function Logs({ match }) {
    const classes = useStyles();

    return (
        <Grid item xs={12}>
            <Route path={`${match.url}/:id`} component={LogDetails}/>
            <Paper className={classes.paper}>
                <StateLogsTable/>
            </Paper>
            <li></li>
            <Paper className={classes.paper}>
                <ValidationLogsTable/>
            </Paper>
        </Grid>
    );
}

function Features() {
    const classes = useStyles();

    return (
        <Grid item xs={12}>
            <Paper className={classes.paper}>
                <FeaturesTable/>
            </Paper>
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

function TFStates({ match }) {
    const classes = useStyles();

    return (
        <div>
            <Grid item xs={12}>
                <Paper className={classes.paper}>
                    <TFStatesTable/>
                </Paper>
            </Grid>
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
/*

log details.
for 2 types.
only focus on tfstate.
you have:

last state (nullable), state.
last json (nullable), json.

should show both differences.

small table:

Features
  other        passing
  s3_buckets   passing
  none         passing => failing
    error1 (red small)
    error2

Differences:

 */
