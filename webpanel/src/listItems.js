import React from 'react';
import ListItem from '@material-ui/core/ListItem';
import ListItemIcon from '@material-ui/core/ListItemIcon';
import ListItemText from '@material-ui/core/ListItemText';
import DashboardIcon from '@material-ui/icons/Dashboard';
import BarChartIcon from '@material-ui/icons/BarChart';
import LayersIcon from '@material-ui/icons/Layers';
import AssignmentIcon from '@material-ui/icons/Assignment';
import {Link} from "react-router-dom";

// TODO: every item should redirect to the corresponding URL...
export const mainListItems = (
    <div>
        <Link color="inherit" to="/">
            <ListItem button>
                <ListItemIcon>
                    <DashboardIcon />
                </ListItemIcon>
                <ListItemText primary="Dashboard" />
            </ListItem>
        </Link>

        <Link color="inherit" to="/features">
            <ListItem button>
                <ListItemIcon>
                    <AssignmentIcon />
                </ListItemIcon>
                <ListItemText primary="Features" />
            </ListItem>
        </Link>

        <Link color="inherit" to="/tfstates">
            <ListItem button>
                <ListItemIcon>
                    <BarChartIcon />
                </ListItemIcon>
                <ListItemText primary="States" />
            </ListItem>
        </Link>

        <Link color="inherit" to="/foreignresources">
            <ListItem button>
                <ListItemIcon>
                    <LayersIcon />
                </ListItemIcon>
                <ListItemText primary="Foreign Resources" />
            </ListItem>
        </Link>
    </div>
);

export const secondaryListItems = (
  <div>
  </div>
);