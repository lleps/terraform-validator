import React from 'react';
import './App.css';
import {BrowserRouter as Router, Redirect, Route} from "react-router-dom";
import Navigation from "./Navigation";
import Grid from "@material-ui/core/Grid";
import Paper from "@material-ui/core/Paper";
import {StateLogsTable, ValidationLogsTable} from "./Logs";
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


function Logs() {
    const classes = useStyles();

    return (
        <Grid item xs={12}>
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

function TFStates() {
    const classes = useStyles();

    return (
        <Grid item xs={12}>
            <Paper className={classes.paper}>
                <TFStatesTable/>
            </Paper>
        </Grid>
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
            <Route path="/" exact component={Logs}/>
            <Route path="/features" component={Features}/>
            <Route path="/tfstates" component={TFStates}/>
            <Route path="/foreignresources" component={ForeignResources}/>
            <Redirect exact from="/" to="/"/>
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
