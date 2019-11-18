import React from 'react';
import Cookies from 'js-cookie'

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