import React from 'react';
import './App.css';
import Paper from '@material-ui/core/Paper';
import { BrowserRouter as Router, Route, Link } from "react-router-dom";
import {TFStatesTable} from "./TFStates";
import List from "@material-ui/core/List";
import ListItem from "@material-ui/core/ListItem";
import ListItemText from "@material-ui/core/ListItemText";
import CssBaseline from "@material-ui/core/CssBaseline";
import AppBar from "@material-ui/core/AppBar";
import Drawer from '@material-ui/core/Drawer';
import Hidden from '@material-ui/core/Hidden';
import IconButton from '@material-ui/core/IconButton';
import Toolbar from '@material-ui/core/Toolbar';
import Typography from '@material-ui/core/Typography';
import { makeStyles, useTheme } from '@material-ui/core/styles';

const drawerWidth = 240;

const useStyles = makeStyles(theme => ({
  root: {
    display: 'flex',
  },
  drawer: {
    [theme.breakpoints.up('sm')]: {
      width: drawerWidth,
      flexShrink: 0,
    },
  },
  appBar: {
    marginLeft: drawerWidth,
    [theme.breakpoints.up('sm')]: {
      width: `calc(100% - ${drawerWidth}px)`,
    },
  },
  menuButton: {
    marginRight: theme.spacing(2),
    [theme.breakpoints.up('sm')]: {
      display: 'none',
    },
  },
  toolbar: theme.mixins.toolbar,
  drawerPaper: {
    width: drawerWidth,
  },
  content: {
    flexGrow: 1,
    padding: theme.spacing(3),
  },
  paper: {
    padding: theme.spacing(2),
    display: 'flex',
    overflow: 'auto',
    flexDirection: 'column',
  },
}));

function ResponsiveDrawer(props) {
  const { container } = props;
  const classes = useStyles();
  const theme = useTheme();
  const [mobileOpen, setMobileOpen] = React.useState(false);

  function handleDrawerToggle() {
    setMobileOpen(!mobileOpen);
  }

  const drawer = (
      <div>
        <div className={classes.toolbar} />
        <List>
          <Link to="/">
            <ListItem button key="Logs">
              <ListItemText primary="Logs" />
            </ListItem>
          </Link>
          <Link to="/features">
            <ListItem button key="Features">
              <ListItemText primary="Features" />
            </ListItem>
          </Link>
          <Link to="/tfstates">
            <ListItem button key="Terraform States">
              <ListItemText primary="Terraform States" />
            </ListItem>
          </Link>
          <Link to="/foreignresources">
            <ListItem button key="Foreign Resources">
              <ListItemText primary="Foreign Resources" />
            </ListItem>
          </Link>
        </List>
      </div>
  );

  return (
      <div className={classes.root}>
        <CssBaseline />
        <AppBar position="fixed" className={classes.appBar}>
          <Toolbar>
            <IconButton
                color="inherit"
                aria-label="open drawer"
                edge="start"
                onClick={handleDrawerToggle}
                className={classes.menuButton}
            >
            </IconButton>
            <Typography variant="h6" noWrap>
              Terraform Monitor for AWS
            </Typography>
          </Toolbar>
        </AppBar>
        <nav className={classes.drawer} aria-label="mailbox folders">
          {/* The implementation can be swapped with js to avoid SEO duplication of links. */}
          <Hidden smUp implementation="css">
            <Drawer
                container={container}
                variant="temporary"
                anchor={theme.direction === 'rtl' ? 'right' : 'left'}
                open={mobileOpen}
                onClose={handleDrawerToggle}
                classes={{
                  paper: classes.drawerPaper,
                }}
                ModalProps={{
                  keepMounted: true, // Better open performance on mobile.
                }}
            >
              {drawer}
            </Drawer>
          </Hidden>
          <Hidden xsDown implementation="css">
            <Drawer
                classes={{
                  paper: classes.drawerPaper,
                }}
                variant="permanent"
                open
            >
              {drawer}
            </Drawer>
          </Hidden>
        </nav>
        <main className={classes.content}>
          <div className={classes.toolbar} />
            <Route path="/" exact component={Logs} />
            <Route path="/tfstates/" component={TFStates} />
            <Route path="/features/" component={Features} />
            <Route path="/foreignresources/" component={ForeignResources} />
        </main>
      </div>
  );
}

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
        <TFStatesTable/>
      </Paper>
  )
}

function App() {
  return (
    <div className="App">
      <header className="App-header">
        <Router>
          <ResponsiveDrawer/>
        </Router>
      </header>
    </div>
  );
}

export default App;
