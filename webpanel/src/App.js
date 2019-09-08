import React from 'react';
import './App.css';
import { makeStyles } from '@material-ui/core/styles';
import { Button } from '@material-ui/core';
import Table from '@material-ui/core/Table';
import TableBody from '@material-ui/core/TableBody';
import TableCell from '@material-ui/core/TableCell';
import TableHead from '@material-ui/core/TableHead';
import TableRow from '@material-ui/core/TableRow';
import Paper from '@material-ui/core/Paper';
import Title from './Title';
import Typography from '@material-ui/core/Typography';
import { BrowserRouter as Router, Route, Link } from "react-router-dom";

const axios = require('axios');

function ComplianceText(data) {
  if (data.compliance_present === true) {
    if (data.compliance_errors === 0) {
      return <Typography color="primary">yes ({data.compliance_tests} passing)</Typography>
    } else {
      return <Typography color="secondary">no ({data.compliance_errors}/{data.compliance_tests} failing)</Typography>
    }
  } else {
    return <Typography>unchecked</Typography>
  }
}

class TFStateList extends React.Component {
  state = {
    tfstates: []
  };

  componentDidMount() {
    axios.get(`http://localhost:8080/tfstates/json`)
      .then(res => {
        const tfstates = res.data;
        this.setState({ tfstates });
      })
  }

  render() {
    return (
      <React.Fragment>
      <Title>Latest state changes</Title>
      <Table size="small">
        <TableHead>
          <TableRow>
            <TableCell>Bucket</TableCell>
            <TableCell>Path</TableCell>
            <TableCell>Last Update</TableCell>
            <TableCell>Compliant</TableCell>
            <TableCell>Actions</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
        { this.state.tfstates
          .map(l => (
            <TableRow key={l.id}>
              <TableCell>{l.bucket}</TableCell>
              <TableCell>{l.path}</TableCell>
              <TableCell>{l.last_update}</TableCell>
              <TableCell>{ComplianceText(l)}</TableCell>
              <TableCell align="right">
                <Button>Details</Button>
                <Button>Delete</Button>
              </TableCell>
          </TableRow>
        ))}
        </TableBody>
      </Table>
      </React.Fragment>
    )
  }
}

const drawerWidth = 240;

const useStyles = makeStyles(theme => ({
  root: {
    display: 'flex',
  },
  toolbar: {
    paddingRight: 24, // keep right padding when drawer closed
  },
  toolbarIcon: {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'flex-end',
    padding: '0 8px',
    ...theme.mixins.toolbar,
  },
  appBar: {
    zIndex: theme.zIndex.drawer + 1,
    transition: theme.transitions.create(['width', 'margin'], {
      easing: theme.transitions.easing.sharp,
      duration: theme.transitions.duration.leavingScreen,
    }),
  },
  appBarShift: {
    marginLeft: drawerWidth,
    width: `calc(100% - ${drawerWidth}px)`,
    transition: theme.transitions.create(['width', 'margin'], {
      easing: theme.transitions.easing.sharp,
      duration: theme.transitions.duration.enteringScreen,
    }),
  },
  menuButton: {
    marginRight: 36,
  },
  menuButtonHidden: {
    display: 'none',
  },
  title: {
    flexGrow: 1,
  },
  drawerPaper: {
    position: 'relative',
    whiteSpace: 'nowrap',
    width: drawerWidth,
    transition: theme.transitions.create('width', {
      easing: theme.transitions.easing.sharp,
      duration: theme.transitions.duration.enteringScreen,
    }),
  },
  drawerPaperClose: {
    overflowX: 'hidden',
    transition: theme.transitions.create('width', {
      easing: theme.transitions.easing.sharp,
      duration: theme.transitions.duration.leavingScreen,
    }),
    width: theme.spacing(7),
    [theme.breakpoints.up('sm')]: {
      width: theme.spacing(9),
    },
  },
  appBarSpacer: theme.mixins.toolbar,
  content: {
    flexGrow: 1,
    height: '100vh',
    overflow: 'auto',
  },
  container: {
    paddingTop: theme.spacing(4),
    paddingBottom: theme.spacing(4),
  },
  paper: {
    padding: theme.spacing(2),
    display: 'flex',
    overflow: 'auto',
    flexDirection: 'column',
  },
  fixedHeight: {
    height: 240,
  },
}));

function Features() {
  const classes = useStyles();

  return (
      <Paper className={classes.paper}>
        <b>Feature list!</b>
      </Paper>
  )
}

function Logs() {
  const classes = useStyles();

  return (
      <Paper className={classes.paper}>
        <b>Log list!</b>
      </Paper>
  )
}

function ForeignResources() {
  const classes = useStyles();

  return (
      <Paper className={classes.paper}>
        <b>Log list!</b>
      </Paper>
  )
}

function TFStates() {
  const classes = useStyles();

  return (
      <Paper className={classes.paper}>
        <TFStateList/>
      </Paper>
  )
}

function App() {
  return (
    <div className="App">
      <header className="App-header">
        <Router>
          <nav>
            <ul>
              <li>
                <Link to="/">Logs</Link>
              </li>
              <li>
                <Link to="/tfstates/">Registered states</Link>
              </li>
              <li>
                <Link to="/features/">Features</Link>
              </li>
            </ul>
          </nav>
          <Route path="/" exact component={Logs} />
          <Route path="/tfstates/" component={TFStates} />
          <Route path="/features/" component={Features} />
          <Route path="/foreignresources/" component={ForeignResources} />
        </Router>
      </header>
    </div>
  );
}

export default App;
