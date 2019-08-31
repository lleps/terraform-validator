// This file provides methods to register REST requests (for object
// collections or single endpoints) without too much boilerplate.

package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

// registerEndpoint registers in the router an HTTP request with
// proper error handling and logging.
func registerEndpoint(
	router *mux.Router,
	endpoint string,
	handler func(string, map[string]string) (string, int, error),
	method string,
) {
	router.HandleFunc(endpoint, func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		log.Println()
		log.Printf("%s %s [from %s]", r.Method, r.URL, r.RemoteAddr)

		// parse body and vars
		vars := mux.Vars(r)
		bodyBytes, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Println("Can't read body:", err)
			return
		}

		// execute the handler
		response, code, err := handler(string(bodyBytes), vars)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = fmt.Fprint(w, err.Error())
			log.Println("Handler error:", err)
			return
		}

		// write response
		w.WriteHeader(code)
		_, err = fmt.Fprint(w, response)
		if err != nil {
			log.Println("Can't write response:", err)
		}

		// log request and response code
		log.Printf("HTTP Response: %d", code)
	}).Methods(method)
}

// restObject groups the info required for all DB objects by the REST API.
type restObject interface {
	id() string // the obj id. To be matched in in GET,DELETE /type/{id)
	topLevel() string // what to show in GET /type
	details() string // what to show in GET /type/{id}
}

// collectionEndpointBuilder contains the parameters to pass to registerCollectionEndpoint
type collectionEndpointBuilder struct {
	router *mux.Router
	endpoint string
	dbFetcher func(db *database) ([]restObject, error)
	dbRemover func(db *database, id string) error
	dbInserter func(db *database, body string) error
}

// registerCollectionEndpoint register automatically GET/POST/DELETE
// methods for the given endpoint, without having to duplicate logic
// for every persistent object in the database.
func registerCollectionEndpoint(db *database, builder collectionEndpointBuilder) {
	// GET /endpoint and GET /endpoint/{id}
	if builder.dbFetcher != nil {
		allHandler := func(body string, _ map[string]string) (string, int, error) {
			objs, err := builder.dbFetcher(db)
			if err != nil {
				return "", 0, fmt.Errorf("GET: can't fetch object: %v", err)
			}
			sb := strings.Builder{}
			for _, o := range objs {
				sb.WriteString(o.topLevel())
				sb.WriteRune('\n')
			}
			return sb.String(), http.StatusOK, nil
		}
		detailsHandler := func(body string, urlVars map[string]string) (string, int, error) {
			id := urlVars["id"]
			objs, err := builder.dbFetcher(db)
			if err != nil {
				return "", 0, fmt.Errorf("GET: can't fetch object: %v", err)
			}
			for _, elem := range objs {
				if elem.id() == id {
					return elem.details(), http.StatusOK, nil
				}
			}
			return "", http.StatusNotFound, nil
		}
		registerEndpoint(builder.router, builder.endpoint, allHandler, "GET")
		registerEndpoint(builder.router, builder.endpoint + "/{id}", detailsHandler, "GET")
	}

	// GET /endpoint/{id}
	if builder.dbFetcher != nil {
		handler := func(body string, _ map[string]string) (string, int, error) {
			objs, err := builder.dbFetcher(db)
			if err != nil {
				return "", 0, fmt.Errorf("GET: can't fetch object: %v", err)
			}
			sb := strings.Builder{}
			for _, o := range objs {
				sb.WriteString(o.topLevel())
				sb.WriteRune('\n')
			}
			return sb.String(), http.StatusOK, nil
		}
		registerEndpoint(builder.router, builder.endpoint, handler, "GET")
	}

	// DELETE /endpoint/{id}
	if builder.dbRemover != nil {
		handler := func(body string, vars map[string]string) (string, int, error) {
			id := vars["id"]
			objs, err := builder.dbFetcher(db)
			if err != nil {
				return "", 0, fmt.Errorf("DELETE: can't fetch object: %v", err)
			}

			for _, o := range objs {
				if o.id() == id {
					err := builder.dbRemover(db, id)
					if err != nil {
						return "", 0, fmt.Errorf("DELETE: can't delete object: %v", err)
					}

					return "", http.StatusOK, nil
				}
			}

			return "can't find object", http.StatusNotFound, nil
		}
		registerEndpoint(builder.router, builder.endpoint + "/{id}", handler, "DELETE")
	}

	// POST /endpoint
	if builder.dbInserter != nil {
		handler := func(body string, _ map[string]string) (string, int, error) {
			if err := builder.dbInserter(db, body); err != nil {
				return "", 0, fmt.Errorf("POST: can't insert object: %v", err)
			}
			return "", http.StatusOK, nil
		}
		registerEndpoint(builder.router, builder.endpoint, handler, "POST")
	}
}

