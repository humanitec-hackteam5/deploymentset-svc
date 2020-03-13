package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"humanitec.io/deploymentset-svc/pkg/depset"
)

// SetWrapper represents the "over-the-wire" structure of a Deployment Set
type SetWrapper struct {
	ID string `json:"id"`
	depset.Set
}

// SetMetadata contains things like first creation date and who created it
type SetMetadata struct {
}

// isZeroHash returns true if the string is entirely made of zeros
func isZeroHash(h string) bool {
	for _, c := range h {
		if c != '0' {
			return false
		}
	}
	return true
}

// listSets returns a handler which returns a list of all the sets in the specified app.
//
// The handler expects the organization to be defined by a parameter "orgId" and app by "appId"
func (s *server) listSets() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		params := mux.Vars(r)
		sets, err := s.model.selectAllSets(params["orgId"], params["appId"])
		if err != nil {
			w.WriteHeader(500)
			return
		}

		// Handle special case of empty list as it could just be nil.
		if len(sets) == 0 {
			fmt.Fprintf(w, `[]`)
			return
		}

		writeAsJSON(w, http.StatusOK, sets)
	}
}

// getUnscopedRawSet returns a handler which returns a specific set without checking org or app scope.
//
// The handler expects the set to be defined by a parameter "setId"
func (s *server) getUnscopedRawSet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		params := mux.Vars(r)

		// Short circit for the null set
		if isZeroHash(params["setId"]) {
			writeAsJSON(w, http.StatusOK, depset.Set{
				Modules: map[string]map[string]interface{}{},
			})
			return
		}

		set, err := s.model.selectUnscopedRawSet(params["setId"])
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				writeAsJSON(w, http.StatusNotFound, fmt.Sprintf(`Set with ID "%s" not available in Application "%s/%s".`, params["setId"], params["orgId"], params["appId"]))
				return
			}
			w.WriteHeader(500)
			return
		}

		writeAsJSON(w, http.StatusOK, set)
	}
}

// getSets returns a handler which returns a specific set in the specified app.
//
// The handler expects the organization to be defined by a parameter "orgId", the app by "appId" and the set by "setId"
func (s *server) getSet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		params := mux.Vars(r)

		// Short circit for the null set
		if isZeroHash(params["setId"]) {
			writeAsJSON(w, http.StatusOK, depset.Set{
				Modules: map[string]map[string]interface{}{},
			})
			return
		}

		set, err := s.model.selectSet(params["orgId"], params["appId"], params["setId"])
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				writeAsJSON(w, http.StatusNotFound, fmt.Sprintf(`Set with ID "%s" not available in Application "%s/%s".`, params["setId"], params["orgId"], params["appId"]))
				return
			}
			w.WriteHeader(500)
			return
		}

		writeAsJSON(w, http.StatusOK, set)
	}
}

// diff returns a Delta describing how to generate leftSetId from rightSetId
//
// The handler expects the organization to be defined by a parameter "orgId", the app by "appId", the left set by "leftSetId" and right set by "rightSetId"
//
//
// The handler returns the following status codes:
//
// 200 Delta was sucessfully calculated, will be in body
//
// 404 Set was not found one or other of the setIds is not valid or present in the app.
func (s *server) diffSets() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		params := mux.Vars(r)
		var leftSet depset.Set
		var err error
		if !isZeroHash(params["leftSetId"]) {
			leftSet, err = s.model.selectRawSet(params["orgId"], params["appId"], params["leftSetId"])
			if err == ErrNotFound {
				writeAsJSON(w, http.StatusNotFound, fmt.Sprintf(`Set with ID "%s" not available in Application "%s/%s".`, params["leftSetId"], params["orgId"], params["appId"]))
				return
			} else if err != nil {
				w.WriteHeader(500)
				return
			}
		}

		var rightSet depset.Set
		if !isZeroHash(params["rightSetId"]) {
			rightSet, err = s.model.selectRawSet(params["orgId"], params["appId"], params["rightSetId"])
			if err == ErrNotFound {
				writeAsJSON(w, http.StatusNotFound, fmt.Sprintf(`Set with ID "%s" not available in Application "%s/%s".`, params["rightSetId"], params["orgId"], params["appId"]))
				return
			} else if err != nil {
				w.WriteHeader(500)
				return
			}
		}

		delta := leftSet.Diff(rightSet)

		writeAsJSON(w, http.StatusOK, delta)
	}
}

// applyDelta returns a handler which applies a delta to a specified set.
//
// The handler expects the organization to be defined by a parameter "orgId", the app by "appId" and the set by "setId"
//
// The Delta should be provided in the body.
//
// The handler returns the following status codes:
//
// 200 Delta applied; body of response is new set ID
//
// 400 Delta is not compatible with set
//
// 404 Set was not found
//
// 422 Delta was malformed
func (s *server) applyDelta() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		params := mux.Vars(r)
		if r.Body == nil {
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		}
		var delta depset.Delta
		err := json.NewDecoder(r.Body).Decode(&delta)
		if nil != err {
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		}

		var set depset.Set
		if !isZeroHash(params["setId"]) {
			set, err = s.model.selectRawSet(params["orgId"], params["appId"], params["setId"])
			if err == ErrNotFound {
				writeAsJSON(w, http.StatusNotFound, fmt.Sprintf(`"Set with ID \"%s\" does not exist."`, params["setId"]))
				return
			} else if err != nil {
				w.WriteHeader(500)
				return
			}
		}

		if len(delta.Modules.Add) == 0 && len(delta.Modules.Remove) == 0 && len(delta.Modules.Update) == 0 {
			// Short circuit for the empty delta
			w.WriteHeader(200)
			if isZeroHash(params["setId"]) {
				fmt.Fprintf(w, `"0000000000000000000000000000000000000000"`)
			} else {
				fmt.Fprintf(w, `"%s"`, params["setId"])
			}
			return
		}

		newSw := SetWrapper{}
		newSw.Set, err = set.Apply(delta)
		if err != nil {
			writeAsJSON(w, http.StatusBadRequest, "Delta is not compatible with Set")
			return
		}
		newSw.ID = newSw.Set.Hash()

		err = s.model.insertSet(params["orgId"], params["appId"], newSw)
		if err != nil && err != ErrAlreadyExists {
			w.WriteHeader(500)
			return
		}

		writeAsJSON(w, http.StatusOK, newSw.ID)
	}
}
