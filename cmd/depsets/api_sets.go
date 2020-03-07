package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"humanitec.io/deploymentset-svc/pkg/depset"
)

// SetWrapper represents the "over-the-wire" structure of a Deployment Set
type SetWrapper struct {
	ID       string      `json:"id"`
	Metadata SetMetadata `json:"metadata"`
	Content  depset.Set  `json:"content"`
}

// SetMetadata contains things like first creation date and who created it
type SetMetadata struct {
	CreatedAt time.Time `json:"createdAt"`
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

		jsonSets, err := json.Marshal(sets)
		if err != nil {
			log.Println(err)
			w.WriteHeader(500)
			return
		}
		w.Write(jsonSets)
	}
}

// getUnscopedRawSet returns a handler which returns a specific set without checking org or app scope.
//
// The handler expects the set to be defined by a parameter "setId"
func (s *server) getUnscopedRawSet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		params := mux.Vars(r)
		set, err := s.model.selectUnscopedRawSet(params["setId"])
		if err != nil {
			w.WriteHeader(500)
			return
		}

		jsonSet, err := json.Marshal(set)
		if err != nil {
			log.Println(err)
			w.WriteHeader(500)
		}
		w.Write(jsonSet)
	}
}

// getSets returns a handler which returns a specific set in the specified app.
//
// The handler expects the organization to be defined by a parameter "orgId", the app by "appId" and the set by "setId"
func (s *server) getSet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		params := mux.Vars(r)
		set, err := s.model.selectSet(params["orgId"], params["appId"], params["setId"])
		if err != nil {
			w.WriteHeader(500)
			return
		}

		jsonSet, err := json.Marshal(set)
		if err != nil {
			log.Println(err)
			w.WriteHeader(500)
			return
		}
		w.Write(jsonSet)
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
				w.WriteHeader(404)
				fmt.Fprintf(w, `"Set with ID \"%s\" does not exist."`, params["leftSetId"])
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
				w.WriteHeader(404)
				fmt.Fprintf(w, `"Set with ID \"%s\" does not exist."`, params["rightSetId"])
				return
			} else if err != nil {
				w.WriteHeader(500)
				return
			}
		}

		delta := leftSet.Diff(rightSet)

		jsonDelta, err := json.Marshal(delta)
		if err != nil {
			log.Println(err)
			w.WriteHeader(500)
			return
		}
		w.WriteHeader(200)
		w.Write(jsonDelta)
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
		var delta depset.Delta
		if r.Body == nil {
			w.WriteHeader(422)
			log.Printf("Body of request was nil")
			return
		}
		err := json.NewDecoder(r.Body).Decode(&delta)
		if nil != err {
			w.WriteHeader(422)
			log.Println(err)
			return
		}

		var set depset.Set
		if !isZeroHash(params["setId"]) {
			set, err = s.model.selectRawSet(params["orgId"], params["appId"], params["setId"])
			if err == ErrNotFound {
				w.WriteHeader(404)
				fmt.Fprintf(w, `"Set with ID \"%s\" does not exist."`, params["setId"])
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
		newSw.Content, err = set.Apply(delta)
		if err != nil {
			w.WriteHeader(400)
			fmt.Fprintf(w, `"Delta is not compatible with Set"`)
			return
		}
		newSw.ID = newSw.Content.Hash()

		err = s.model.insertSet(params["orgId"], params["appId"], newSw)
		if err != nil && err != ErrAlreadyExists {
			w.WriteHeader(500)
			return
		}

		w.WriteHeader(http.StatusOK)
		out, _ := json.Marshal(newSw.ID)
		w.Write(out)
	}
}
