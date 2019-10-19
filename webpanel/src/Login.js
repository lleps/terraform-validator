import React from 'react';
import Avatar from '@material-ui/core/Avatar';
import Button from '@material-ui/core/Button';
import CssBaseline from '@material-ui/core/CssBaseline';
import TextField from '@material-ui/core/TextField';
import LockOutlinedIcon from '@material-ui/icons/LockOutlined';
import Typography from '@material-ui/core/Typography';
import { makeStyles } from '@material-ui/core/styles';
import Container from '@material-ui/core/Container';
import Cookies from 'js-cookie'
import axios from 'axios';

export const getSession = () => {
    const jwtKey = Cookies.get('__session');
    if (jwtKey) {
        return jwtKey;
    } else {
        return null;
    }
};

export const logOut = () => {
    Cookies.remove('__session')
};

export function setSessionKey(key) {
    Cookies.set('__session', key)
}

export function removeSession() {
    Cookies.remove('__session');
}

const useStyles = makeStyles(theme => ({
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
    avatar: {
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

export default function Login({ onLogin }) {
    const classes = useStyles();
    const [username, setUsername] = React.useState("");
    const [password, setPassword] = React.useState("");
    const [loading, setLoading] = React.useState(false);
    const [error, setError] = React.useState("");

    function postLogin() {
        setLoading(true);
        axios.post("/login", {
            username: username,
            password: password
        }).then(resp => {
            setLoading(false);
            onLogin(resp.data.token);
        }).catch((err) => {
            setLoading(false);
            if (err.response && err.response.status === 401) {
                setError("Invalid username or password.");
                setUsername("");
                setPassword("");
            } else {
                setError("An unexpected error occurred.");
                alert("Unexpected error: " + err);
            }
        })
    }

    return (
        <Container component="main" maxWidth="xs">
            <CssBaseline />
            <div className={classes.paper}>
                <Avatar className={classes.avatar}>
                    <LockOutlinedIcon />
                </Avatar>
                <Typography component="h1" variant="h5">
                    Sign in
                </Typography>
                <TextField
                    variant="outlined"
                    value={username}
                    margin="normal"
                    required
                    fullWidth
                    id="email"
                    label="Username"
                    name="username"
                    error={error !== ""}
                    autoFocus
                    onChange={e => setUsername(e.target.value)}
                />
                <TextField
                    variant="outlined"
                    value={password}
                    margin="normal"
                    required
                    fullWidth
                    name="password"
                    label="Password"
                    type="password"
                    id="password"
                    autoComplete="current-password"
                    helperText={error}
                    error={error !== ""}
                    onChange={e => setPassword(e.target.value)}
                />
                <Button
                    type="submit"
                    fullWidth
                    variant="contained"
                    color="primary"
                    className={classes.submit}
                    onClick={() => { postLogin() }}
                >
                    { loading ? "..." : "Login" }
                </Button>
            </div>
        </Container>
    );
}
