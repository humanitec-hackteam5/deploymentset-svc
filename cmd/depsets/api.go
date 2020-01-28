package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
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

// DeltaWrapper represents the "over-the-wire" structure of a Deployment Delta
type DeltaWrapper struct {
	Name     string        `json:"name"`
	Metadata DeltaMetadata `json:"metadata"`
	Content  depset.Delta  `json:"content"`
}

// DeltaMetadata contains things like first creation date and who created it
type DeltaMetadata struct {
	CreateAt       time.Time `json:"createdAt"`
	LastModifiedAt time.Time `json:"lastModifiedAt"`
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
func (s *server) listSets() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		sets, err := s.model.selectAllSets(params.ByName("orgId"), params.ByName("appId"))
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

// getSets returns a handler which returns either:
// - a specific set in the specified app.
// - the difference between the specified set and another set
//
// The handler expects the organization to be defined by a parameter "orgId", the app by "appId" and the set by "setId"
func (s *server) getSetOrDiff() httprouter.Handle {
	getSet := func(w http.ResponseWriter, orgId, appId, setId string) {
		set, err := s.model.selectSet(orgId, appId, setId)
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

	// diff returns a Delta describing how to generate setId from paramSetId
	//
	// The handler expects the organization to be defined by a parameter "orgId", the app by "appId", the left set by "setId" and right set by "paramSetId"
	//
	//
	// The handler returns the following status codes:
	//
	// 200 Delta was sucessfully calculated, will be in body
	//
	// 404 Set was not found one or other of the setIds is not valid or present in the app.
	diff := func(w http.ResponseWriter, orgId, appId, setId, paramSetId string) {
		var leftSet depset.Set
		var err error
		if !isZeroHash(setId) {
			leftSet, err = s.model.selectRawSet(orgId, appId, setId)
			if err == ErrNotFound {
				w.WriteHeader(404)
				fmt.Fprintf(w, `"Set with ID \"%s\" does not exist."`, setId)
				return
			} else if err != nil {
				w.WriteHeader(500)
				return
			}
		}

		var rightSet depset.Set
		if !isZeroHash(paramSetId) {
			rightSet, err = s.model.selectRawSet(orgId, appId, paramSetId)
			if err == ErrNotFound {
				w.WriteHeader(404)
				fmt.Fprintf(w, `"Set with ID \"%s\" does not exist."`, paramSetId)
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
		}
		w.WriteHeader(200)
		w.Write(jsonDelta)
	}
	return func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		diffSetId, exists := r.URL.Query()["diff"]
		if exists {
			diff(w, params.ByName("orgId"), params.ByName("appId"), params.ByName("setId"), diffSetId[0])
		} else {
			getSet(w, params.ByName("orgId"), params.ByName("appId"), params.ByName("setId"))
		}
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
// 200 Delta applied, set already exists; body of response is new set ID
//
// 201 Delta applied, set was created for first time; body of response is new set ID
//
// 400 Delta is not compatible with set
//
// 404 Set was not found
//
// 422 Delta was malformed
func (s *server) applyDelta() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
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
		if !isZeroHash(params.ByName("setId")) {
			set, err = s.model.selectRawSet(params.ByName("orgId"), params.ByName("appId"), params.ByName("setId"))
			if err == ErrNotFound {
				w.WriteHeader(404)
				fmt.Fprintf(w, `"Set with ID \"%s\" does not exist."`, params.ByName("setId"))
				return
			} else if err != nil {
				w.WriteHeader(500)
				return
			}
		}

		newSw := SetWrapper{}
		newSw.Content, err = set.Apply(delta)
		if err != nil {
			w.WriteHeader(400)
			fmt.Fprintf(w, `"Delta is not compatible with Set"`)
			return
		}
		newSw.ID = newSw.Content.Hash()

		err = s.model.insertSet(params.ByName("orgId"), params.ByName("appId"), newSw)
		if err == ErrAlreadyExists {
			w.WriteHeader(200)
		} else if err != nil {
			w.WriteHeader(500)
			return
		} else {
			w.WriteHeader(201)
		}
		fmt.Fprintf(w, `"%s"`, newSw.ID)
	}
}
