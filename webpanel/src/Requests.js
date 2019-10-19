import axios from 'axios';
import {getSession, removeSession} from "./Login";

export function handledGet(endpoint, onResponse, onFinish) {
    handledGeneric("get", endpoint, {}, onResponse, onFinish);
}

export function handledDelete(endpoint, onResponse, onFinish) {
    handledGeneric("delete", endpoint, {}, onResponse, onFinish);
}

export function handledPost(endpoint, data, onResponse, onFinish) {
    handledGeneric("post", endpoint, data, onResponse, onFinish);
}

export function handledPut(endpoint, data, onResponse, onFinish) {
    handledGeneric("put", endpoint, data, onResponse, onFinish);
}

function handledGeneric(method, endpoint, data, onResponse, onFinish) {
    let jwtKey = getSession();
    if (!jwtKey) { // doesn't have the session cookie, reload.
        window.location.reload();
        return;
    }

    let config = {
        headers: {
            Authorization: "Bearer " + jwtKey
        }
    };

    let promise = null;
    if (method === "get") {
        promise = axios.get(endpoint, config);
    } else if (method === "post") {
        promise = axios.post(endpoint, data, config);
    } else if (method === "put") {
        promise = axios.put(endpoint, data, config);
    } else if (method === "delete") {
        promise = axios.delete(endpoint, config);
    } else {
        throw "Invalid method: " + method;
    }

    promise
        .then(response => {
            onResponse(response.data)
        })
        .catch(err => {
            if (err.response && err.response.status === 401) {
                // token not valid anymore
                console.log("401 received, reload page and wipe key.");
                removeSession();
                window.location.reload();
            } else {
                console.log("error in get request: " + err);
            }
        })
        .then(() => {
            if (onFinish) onFinish()
        });
}