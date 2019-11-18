import React, { Component } from 'react';
import { withAuth } from '@okta/okta-react';
import {makeStyles} from "@material-ui/core";
import Button from "@material-ui/core/Button";
import Container from "@material-ui/core/Container";

export default withAuth(class OktaLoginPage extends Component {
    constructor(props) {
        super(props);
        this.state = {
            loading: true,
            error: null
        };
        this.checkAuthentication = this.checkAuthentication.bind(this);
        this.checkAuthentication();
        this.login = this.login.bind(this);
        this.logout = this.logout.bind(this);
    }

    async checkAuthentication() {
        const authenticated = await this.props.auth.isAuthenticated();
        if (authenticated !== this.state.authenticated) {
            this.setState({ authenticated });
        }

        if (authenticated === true) {
            let t = await this.props.auth.getAccessToken();
            this.setState({ loading: false });
            this.props.onLogin(t);
        } else {
            this.setState({ loading: false });
        }
    }

    async login() {
        this.props.auth.login();
    }

    async logout() {
        this.props.auth.logout('/');
    }

    useStyles = makeStyles(theme => ({
        '@global': {
            body: {
                backgroundColor: theme.palette.common.white,
            },
        },
        paper: {
            marginTop: theme.spacing(8),
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
        },
        avatar:{
            margin: theme.spacing(1),
            backgroundColor: theme.palette.secondary.main,
        },
        form: {
            width: '100%', // Fix IE 11 issue.
            marginTop: theme.spacing(1),
        },
        submit: {
            margin: theme.spacing(3, 0, 2),
        },
    }));

    render() {
        let button = !this.state.authenticated ?
            <Button
                type="submit"
                fullWidth
                variant="contained"
                color="primary"
                onClick={this.login}>
                { this.state.loading ? "Loading..." : "Sign In with Okta" }
            </Button> :
            <Button
                type="submit"
                fullWidth
                variant="contained"
                color="secondary"
                onClick={this.logout}>
                { this.state.loading ? "Loading" : "Sign Out" }
            </Button>;

        return <Container component="main" maxWidth="xs">
            {button}
        </Container>;
    }
});