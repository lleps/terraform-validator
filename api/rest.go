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
	"sort"
	"time"
)

// registerEndpoint registers in the router an HTTP request
// with a clean handler that does proper error handling.
func registerEndpoint(
	router *mux.Router,
	db *database,
	endpoint string,
	handler func(*database, string, map[string]string) (string, int, error),
	method string,
) {
	router.HandleFunc(endpoint, func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

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
	}).Methods(method)
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
	return a[i].timestamp() < a[j].timestamp()
}

// restObjectHandler contains the handlers that effectively perform the operations.
type restObjectHandler struct {
	loadAllFunc   func(db *database) ([]restObject, error)
	deleteHandler func(db *database, id string) error
	postHandler   func(db *database, body string) (restObject, error)
	putHandler    func(db *database, obj restObject, body string) error
}

// registerObjEndpoints registers automatically GET/POST/DELETE/PUT methods
// (depending on the handlers passed) for the given endpoint, without
// having to duplicate database logic for every persistent object.
func registerObjEndpoints(router *mux.Router, endpoint string, db *database, handlers restObjectHandler) {

	// GET /endpoint and GET /endpoint/{id}.
	if handlers.loadAllFunc != nil {

		getAllHandler := func(_ *database, body string, _ map[string]string) (string, int, error) {
			objs, err := handlers.loadAllFunc(db)
			if err != nil {
				return "", 0, fmt.Errorf("GET: can't fetch object: %v", err)
			}

			// Returns a list of json objects.
			result := make([]interface{}, 0)
			for _, o := range objs {
				dst := make(map[string]interface{})
				dst["id"] = o.id()
				dst["timestamp"] = o.timestamp()
				o.writeBasic(dst)
				result = append(result, dst)
			}

			// sort the list by timestamp
			sort.Sort(ByRestObject(objs))

			// jsonify and return
			asJSON, err := json.MarshalIndent(result, "", "\t")
			if err != nil {
				return "", 0, err
			}

			return string(asJSON), http.StatusOK, nil
		}

		getSpecificHandler := func(_ *database, body string, urlVars map[string]string) (string, int, error) {
			id := urlVars["id"]
			objs, err := handlers.loadAllFunc(db)
			if err != nil {
				return "", 0, fmt.Errorf("GET: can't fetch object: %v", err)
			}
			for _, elem := range objs {
				if elem.id() == id {
					dst := make(map[string]interface{})
					dst["id"] = elem.id()
					elem.writeDetailed(dst)
					asJSON, err := json.MarshalIndent(dst, "", "\t")
					if err != nil {
						return "", 0, err
					}

					return string(asJSON), http.StatusOK, nil
				}
			}
			return "can't find object: " + id, http.StatusNotFound, nil
		}

		registerEndpoint(router, db, endpoint, getAllHandler, "GET")
		registerEndpoint(router, db, endpoint+"/{id}", getSpecificHandler, "GET")
	}

	// DELETE /endpoint/{id}
	if handlers.deleteHandler != nil {

		handler := func(_ *database, body string, vars map[string]string) (string, int, error) {
			id := vars["id"]
			objs, err := handlers.loadAllFunc(db)
			if err != nil {
				return "", 0, fmt.Errorf("DELETE: can't fetch object: %v", err)
			}

			for _, o := range objs {
				if o.id() == id {
					err := handlers.deleteHandler(db, id)
					if err != nil {
						return "", 0, fmt.Errorf("DELETE: can't delete object: %v", err)
					}

					return "", http.StatusOK, nil
				}
			}

			return "can't find object: " + id, http.StatusNotFound, nil
		}

		registerEndpoint(router, db, endpoint+"/{id}", handler, "DELETE")
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

		registerEndpoint(router, db, endpoint, handler, "POST")
	}

	// PUT /endpoint/{id}.
	if handlers.putHandler != nil {
		handler := func(_ *database, body string, vars map[string]string) (string, int, error) {
			id := vars["id"]
			objs, err := handlers.loadAllFunc(db)
			if err != nil {
				return "", 0, fmt.Errorf("PUT: can't fetch object: %v", err)
			}

			for _, o := range objs {
				if o.id() == id {
					err := handlers.putHandler(db, o, body)
					if err != nil {
						return "", 0, fmt.Errorf("PUT: can't put object: %v", err)
					}

					return "", http.StatusOK, nil
				}
			}

			return "", http.StatusNotFound, nil
		}

		registerEndpoint(router, db, endpoint+"/{id}", handler, "PUT")
	}
}
