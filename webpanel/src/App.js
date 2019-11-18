import React from 'react';
import './App.css';
import { BrowserRouter, Route, Redirect } from 'react-router-dom';
import { Security, ImplicitCallback } from '@okta/okta-react';
import OktaLoginPage from './OktaLoginPage';
import Navigation from "./Navigation";
import Grid from "@material-ui/core/Grid";
import Paper from "@material-ui/core/Paper";
import {LogDetailsDialog, LogsTable} from "./Logs";
import {FeatureAddDialog, FeatureEditDialog, FeaturesTable} from "./Features";
import {TFStateDialog, TFStatesTable} from "./TFStates";
import {ForeignResourceDetailsDialog, ForeignResourcesTable} from "./ForeignResources";
import {makeStyles} from "@material-ui/core";
import FloatingActionButtons from "./Fab";
import {getSession, setSessionKey} from "./Login";
import axios from "axios";
import CircularProgress from "@material-ui/core/CircularProgress";

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
                <LogsTable
                    kind="tfstate"
                    onSelectInfo={(id) => props.history.push("/logs/" + id) }
                />
            </Paper>
            <li/>
            <Paper className={classes.paper}>
                <LogsTable
                    kind="validation"
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
            onSave={() => pushRefresh(props.history, "/features")}
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

function TFStateAdd(props) {
    return (
        <TFStateDialog
            editMode={false}
            onAdd={() => pushRefresh(props.history, "/tfstates")}
            onCancel={() => props.history.push("/tfstates")}
        />
    );
}

function TFStateEdit(props) {
    let id = props.match.params.id;
    return (
        <TFStateDialog
            id={id}
            editMode={true}
            onAdd={() => pushRefresh(props.history, "/tfstates")}
            onCancel={() => props.history.push("/tfstates")}
        />
    );
}

function TFStates(props) {
    const classes = useStyles();

    return (
        <div>
            <Route path={`${props.match.url}/add`} component={TFStateAdd} />
            <Route path={`${props.match.url}/edit/:id`} component={TFStateEdit} />
            <Grid item xs={12}>
                <Paper className={classes.paper}>
                    <TFStatesTable onEdit={(id) => props.history.push("/tfstates/edit/" + id)}/>
                </Paper>
            </Grid>
            <FloatingActionButtons onClick={() => props.history.push("/tfstates/add")} />
        </div>
    );
}

function ForeignResourceDetails(props) {
    let id = props.match.params.id;
    return (
        <ForeignResourceDetailsDialog
            id={id}
            onClose={() => {
                props.history.push(`/foreignresources`)
            }}
        />
    );
}

function ForeignResources(props) {
    const classes = useStyles();

    return (
        <Grid item xs={12}>
            <Route path={`${props.match.url}/details/:id`} component={ForeignResourceDetails} />
            <Paper className={classes.paper}>
                <ForeignResourcesTable
                    onSelect={id => props.history.push(`/foreignresources/details/` + id)}
                />
            </Paper>
        </Grid>
    );
}

function Routes() {
    return (
        <div>
            <Route exact path="/" render={() => (
                <Redirect to="/logs"/>
            )}/>
            <Route path="/logs" component={Logs}/>
            <Route path="/features" component={Features}/>
            <Route path="/tfstates" component={TFStates}/>
            <Route path="/login" render={() => <Redirect to={"/logs"}/>}/>
        </div>
    );
}

function App() {
    let sess = getSession();
    const [error, setError] = React.useState("");
    const [config, setConfig] = React.useState(null);

    React.useEffect(() => {
        axios.get("/login-details")
            .then(res => {
                setConfig({
                    issuer: res.data.okta_issuer_url,
                    redirectUri: window.location.origin + '/implicit/callback',
                    clientId: res.data.okta_client_id,
                    pkce: true
                })
            })
            .catch(err => {
                setError("Can't fetch okta login details. Is the API alive?")
            });
    }, []);

    if (error !== "") {
        return <b>{error}</b>
    }

    if (!config) {
        return <CircularProgress/>
    }

    if (sess == null) {
        return <BrowserRouter>
            <Security {...config}>
                <Route exact path="/" render={() => (<Redirect to="/login"/>)}/>
                <Route path="/login" render={() =>
                    <OktaLoginPage
                        onLogin={key => {
                            setSessionKey(key);
                            window.location.reload();
                        }}
                    />
                }/>
                <Route path='/implicit/callback' component={ImplicitCallback}/>
            </Security>
        </BrowserRouter>;
    } else {
        return <BrowserRouter>
            <Security {...config}>
                <Navigation title={"Terraform Monitor"} content={Routes}/>
            </Security>
        </BrowserRouter>
    }
}

export default App;