// This file provides methods to register REST requests (for object
// collections or single endpoints) without too much boilerplate.

package main

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"io/ioutil"
	"log"
	"net/http"
	"reflect"
	"sort"
	"time"
)

func registerAuthenticatedEndpoint(
	router *mux.Router,
	db *database,
	endpoint string,
	handler func(*database, string, map[string]string) (string, int, error),
	method string,
) {
	registerEndpoint(true, router, db, endpoint, handler, method)
}

func registerPublicEndpoint(
	router *mux.Router,
	db *database,
	endpoint string,
	handler func(*database, string, map[string]string) (string, int, error),
	method string,
) {
	registerEndpoint(false, router, db, endpoint, handler, method)
}

// registerEndpoint registers in the router an HTTP handler
// with a clean handler that does proper error handling and
// implements authentication if specified.
func registerEndpoint(
	requireAuthentication bool,
	router *mux.Router,
	db *database,
	endpoint string,
	handler func(*database, string, map[string]string) (string, int, error),
	method string,
) {
	handleFunc := func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		if requireAuthentication && !IsAuthenticated(r) {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("401 - Not authorized"))
			return
		}

		vars := mux.Vars(r)
		bodyBytes, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Println("Can't read body:", err)
			return
		}

		response, code, err := handler(db, string(bodyBytes), vars)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = fmt.Fprint(w, err.Error())
			log.Println("Handler error:", err)
			return
		}

		w.WriteHeader(code)
		_, err = fmt.Fprint(w, response)
		if err != nil {
			log.Println("Can't write response:", err)
		}
	}

	router.HandleFunc(endpoint, handleFunc).Methods(method)
}

// restObject defines some generic methods for objects that are accessible through the rest API.
type restObject interface {
	id() string                               // The object uuid
	timestamp() int64                         // When this object was created.
	writeBasic(dst map[string]interface{})    // Write short object fields (when getting all objects).
	writeDetailed(dst map[string]interface{}) // Write detailed fields (when getting this specific object).
}

// generateId generates a new UUID for a new rest object.
func generateId() string {
	return uuid.New().String()
}

// generateTimestamp generates the timestamp value required for rest objects.
func generateTimestamp() int64 {
	return time.Now().Unix()
}

func restObjectTime(obj restObject) time.Time {
	return time.Unix(obj.timestamp(), 0)
}

// ByRestObject wraps the type to sort by timestamp, since dynamo
// doesn't keep the insert order of entities.
type ByRestObject []restObject

func (a ByRestObject) Len() int      { return len(a) }
func (a ByRestObject) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByRestObject) Less(i, j int) bool {
	return a[i].timestamp() > a[j].timestamp()
}

// restObjectHandler contains the handlers that effectively perform the operations.
type restObjectHandler struct {
	loadAllFunc   func(db *database) ([]restObject, error)
	loadOneFunc   func(db *database, id string) (restObject, error)
	deleteHandler func(db *database, id string) error
	postHandler   func(db *database, body string) (restObject, error)
	putHandler    func(db *database, obj restObject, body string) error
}

// registerAuthenticatedObjEndpoints registers automatically GET/POST/DELETE/PUT methods
// (depending on the handlers passed) for the given endpoint, without
// having to duplicate database logic for every persistent object.
func registerAuthenticatedObjEndpoints(router *mux.Router, endpoint string, db *database, handlers restObjectHandler) {

	// GET /endpoint
	if handlers.loadAllFunc != nil {
		handler := func(_ *database, body string, _ map[string]string) (string, int, error) {
			objs, err := handlers.loadAllFunc(db)
			if err != nil {
				return "", 0, fmt.Errorf("GET: can't fetch object: %v", err)
			}

			// sort the list by timestamp
			sort.Sort(ByRestObject(objs))

			// Returns a list of json objects.
			result := make([]interface{}, 0)
			for _, o := range objs {
				dst := make(map[string]interface{})
				dst["id"] = o.id()
				dst["timestamp"] = o.timestamp()
				o.writeBasic(dst)
				result = append(result, dst)
			}

			// jsonify and return
			asJSON, err := json.MarshalIndent(result, "", "\t")
			if err != nil {
				return "", 0, err
			}

			return string(asJSON), http.StatusOK, nil
		}

		registerAuthenticatedEndpoint(router, db, endpoint, handler, "GET")
	}

	// GET /endpoint/{id}
	if handlers.loadOneFunc != nil {
		handler := func(_ *database, body string, urlVars map[string]string) (string, int, error) {
			id := urlVars["id"]
			obj, err := handlers.loadOneFunc(db, id)
			if err != nil {
				return "", 0, fmt.Errorf("GET: can't fetch object: %v", err)
			}

			if reflect.ValueOf(obj).IsNil() {
				return "can't find obj for id " + id, http.StatusNotFound, nil
			}

			result := make(map[string]interface{})
			result["id"] = obj.id()
			result["timestamp"] = obj.timestamp()
			obj.writeDetailed(result)
			asJSON, err := json.MarshalIndent(result, "", "\t")
			if err != nil {
				return "", 0, err
			}

			return string(asJSON), http.StatusOK, nil
		}

		registerAuthenticatedEndpoint(router, db, endpoint+"/{id}", handler, "GET")
	}

	// DELETE /endpoint/{id}
	if handlers.deleteHandler != nil {
		handler := func(_ *database, body string, vars map[string]string) (string, int, error) {
			id := vars["id"]
			obj, err := handlers.loadOneFunc(db, id)
			if err != nil {
				return "", 0, fmt.Errorf("GET: can't fetch object: %v", err)
			}

			if reflect.ValueOf(obj).IsNil() {
				return "can't find obj for id " + id, http.StatusNotFound, nil
			}

			if err := handlers.deleteHandler(db, id); err != nil {
				return "", 0, fmt.Errorf("DELETE: can't delete object: %v", err)
			}

			return "", http.StatusOK, nil
		}

		registerAuthenticatedEndpoint(router, db, endpoint+"/{id}", handler, "DELETE")
	}

	// POST /endpoint
	if handlers.postHandler != nil {
		handler := func(_ *database, body string, _ map[string]string) (string, int, error) {
			obj, err := handlers.postHandler(db, body)
			if err != nil {
				return "", 0, fmt.Errorf("POST: can't insert object: %v", err)
			}

			marshalled, err := json.Marshal(map[string]string{"id": obj.id()})
			if err != nil {
				return "", 0, fmt.Errorf("POST: can't marshall object: %v", err)
			}

			return string(marshalled), http.StatusOK, nil
		}

		registerAuthenticatedEndpoint(router, db, endpoint, handler, "POST")
	}

	// PUT /endpoint/{id}
	if handlers.putHandler != nil {
		handler := func(_ *database, body string, vars map[string]string) (string, int, error) {
			id := vars["id"]
			obj, err := handlers.loadOneFunc(db, id)
			if err != nil {
				return "", 0, fmt.Errorf("GET: can't fetch object: %v", err)
			}

			if reflect.ValueOf(obj).IsNil() {
				return "can't find obj for id " + id, http.StatusNotFound, nil
			}

			if err := handlers.putHandler(db, obj, body); err != nil {
				return "", 0, fmt.Errorf("PUT: can't put object: %v", err)
			}

			return "", http.StatusOK, nil
		}

		registerAuthenticatedEndpoint(router, db, endpoint+"/{id}", handler, "PUT")
	}
}
