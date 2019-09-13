import React from 'react';
import ListItem from '@material-ui/core/ListItem';
import ListItemIcon from '@material-ui/core/ListItemIcon';
import ListItemText from '@material-ui/core/ListItemText';
import DashboardIcon from '@material-ui/icons/Dashboard';
import BarChartIcon from '@material-ui/icons/BarChart';
import LayersIcon from '@material-ui/icons/Layers';
import AssignmentIcon from '@material-ui/icons/Assignment';
import {Link} from "react-router-dom";

export const mainListItems = (
    <div>
        <ListItem component={Link} to={'/logs'} button>
            <ListItemIcon>
                <DashboardIcon />
            </ListItemIcon>
            <ListItemText primary="Events" />
        </ListItem>

        <ListItem component={Link} to={'/features'} button>
            <ListItemIcon>
                <AssignmentIcon />
            </ListItemIcon>
            <ListItemText primary="Features" />
        </ListItem>

        <ListItem component={Link} to={'/tfstates'} button>
            <ListItemIcon>
                <BarChartIcon />
            </ListItemIcon>
            <ListItemText primary="States" />
        </ListItem>

        <ListItem component={Link} to={'/foreignresources'} button>
            <ListItemIcon>
                <LayersIcon />
            </ListItemIcon>
            <ListItemText primary="Foreign Resources" />
        </ListItem>
    </div>
);

export const secondaryListItems = (
  <div>
  </div>
);