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

// getSets returns a handler which returns a specific sets in the specified app.
//
// The handler expects the organization to be defined by a parameter "orgId", the app by "appId" and the set by "setId"
func (s *server) getSet() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		module, err := s.model.selectSet(params.ByName("orgId"), params.ByName("appId"), params.ByName("setId"))
		if err != nil {
			w.WriteHeader(500)
			return
		}

		jsonModule, err := json.Marshal(module)
		if err != nil {
			log.Println(err)
			w.WriteHeader(500)
		}
		w.Write(jsonModule)
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
