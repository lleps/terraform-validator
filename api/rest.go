// This file provides methods to register REST requests (for object
// collections or single endpoints) without too much boilerplate.

package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"io/ioutil"
	"log"
	"net/http"
	"sort"
	"strconv"
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

		code := http.StatusInternalServerError
		defer func() {
			log.Printf("%s %s [from %s]: HTTP %d", r.Method, r.URL, r.RemoteAddr, code)
		}()

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
	}).Methods(method)
}

// restObject groups the info required for all DB objects by the REST API.
type restObject interface {
	id() string                                     // the obj id. To be matched in in GET,DELETE /type/{id)
	topLevel() string                               // for CLI. what to show in GET /type
	details() string                                // for CLI. what to show in GET /type/{id}
	writeTopLevelFields(dst map[string]interface{}) // for json responses.  GET /type/json.
	writeDetailedFields(dst map[string]interface{}) // for json responses.  GET /type/json/{id}.
}

// ByRestObject wraps the type to sort by id
type ByRestObject []restObject

func (a ByRestObject) Len() int      { return len(a) }
func (a ByRestObject) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByRestObject) Less(i, j int) bool {
	id1, err1 := strconv.ParseInt(a[i].id(), 10, 64)
	id2, err2 := strconv.ParseInt(a[j].id(), 10, 64)
	if err1 != nil || err2 != nil {
		return false
	}
	return id2 < id1
}

// collectionEndpointBuilder contains the parameters to pass to registerCollectionEndpoint
type collectionEndpointBuilder struct {
	router       *mux.Router
	endpoint     string
	dbFetchFunc  func(db *database) ([]restObject, error)
	dbRemoveFunc func(db *database, id string) error
	dbInsertFunc func(db *database, body string) error
}

// registerCollectionEndpoint register automatically GET/POST/DELETE
// methods for the given endpoint, without having to duplicate logic
// for every persistent object in the database.
func registerCollectionEndpoint(db *database, builder collectionEndpointBuilder) {
	// GET /endpoint and GET /endpoint/{id}
	if builder.dbFetchFunc != nil {
		allHandler := func(body string, _ map[string]string) (string, int, error) {
			objs, err := builder.dbFetchFunc(db)
			if err != nil {
				return "", 0, fmt.Errorf("GET: can't fetch object: %v", err)
			}
			sort.Sort(ByRestObject(objs))
			sb := strings.Builder{}
			for _, o := range objs {
				sb.WriteString(o.topLevel())
				sb.WriteRune('\n')
			}
			return sb.String(), http.StatusOK, nil
		}
		detailsHandler := func(body string, urlVars map[string]string) (string, int, error) {
			id := urlVars["id"]
			objs, err := builder.dbFetchFunc(db)
			if err != nil {
				return "", 0, fmt.Errorf("GET: can't fetch object: %v", err)
			}
			for _, elem := range objs {
				if elem.id() == id {
					return elem.details(), http.StatusOK, nil
				}
			}
			return "can't find object: " + id, http.StatusNotFound, nil
		}
		allHandlerJSON := func(body string, _ map[string]string) (string, int, error) {
			objs, err := builder.dbFetchFunc(db)
			if err != nil {
				return "", 0, fmt.Errorf("GET: can't fetch object: %v", err)
			}
			sort.Sort(ByRestObject(objs))
			result := make([]interface{}, 0)
			for _, o := range objs {
				dst := make(map[string]interface{})
				dst["id"] = o.id()
				o.writeTopLevelFields(dst)
				result = append(result, dst)
			}
			asJSON, err := json.Marshal(result)
			if err != nil {
				return "", 0, err
			}

			return string(asJSON), http.StatusOK, nil
		}
		detailsHandlerJSON := func(body string, urlVars map[string]string) (string, int, error) {
			id := urlVars["id"]
			objs, err := builder.dbFetchFunc(db)
			if err != nil {
				return "", 0, fmt.Errorf("GET: can't fetch object: %v", err)
			}
			for _, elem := range objs {
				if elem.id() == id {
					dst := make(map[string]interface{})
					dst["id"] = elem.id()
					elem.writeDetailedFields(dst)
					asJSON, err := json.Marshal(dst)
					if err != nil {
						return "", 0, err
					}

					return string(asJSON), http.StatusOK, nil
				}
			}
			return "can't find object: " + id, http.StatusNotFound, nil
		}
		registerEndpoint(builder.router, builder.endpoint+"/json", allHandlerJSON, "GET")
		registerEndpoint(builder.router, builder.endpoint+"/json/{id}", detailsHandlerJSON, "GET")
		registerEndpoint(builder.router, builder.endpoint, allHandler, "GET")
		registerEndpoint(builder.router, builder.endpoint+"/{id}", detailsHandler, "GET")
	}

	// DELETE /endpoint/{id}
	if builder.dbRemoveFunc != nil {
		handler := func(body string, vars map[string]string) (string, int, error) {
			id := vars["id"]
			objs, err := builder.dbFetchFunc(db)
			if err != nil {
				return "", 0, fmt.Errorf("DELETE: can't fetch object: %v", err)
			}

			for _, o := range objs {
				if o.id() == id {
					err := builder.dbRemoveFunc(db, id)
					if err != nil {
						return "", 0, fmt.Errorf("DELETE: can't delete object: %v", err)
					}

					return "", http.StatusOK, nil
				}
			}

			return "can't find object: " + id, http.StatusNotFound, nil
		}
		registerEndpoint(builder.router, builder.endpoint+"/{id}", handler, "DELETE")
	}

	// POST /endpoint
	if builder.dbInsertFunc != nil {
		handler := func(body string, _ map[string]string) (string, int, error) {
			if err := builder.dbInsertFunc(db, body); err != nil {
				return "", 0, fmt.Errorf("POST: can't insert object: %v", err)
			}
			return "", http.StatusOK, nil
		}
		registerEndpoint(builder.router, builder.endpoint, handler, "POST")
	}
}
