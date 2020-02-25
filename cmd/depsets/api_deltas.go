package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"humanitec.io/deploymentset-svc/pkg/depset"
)

// DeltaWrapper represents the "over-the-wire" structure of a Deployment Delta
type DeltaWrapper struct {
	ID       string        `json:"id"`
	Metadata DeltaMetadata `json:"metadata"`
	Content  depset.Delta  `json:"content"`
}

// DeltaMetadata contains things like first creation date and who created it
type DeltaMetadata struct {
	CreatedBy      string    `json:"createdBy"`
	CreatedAt      time.Time `json:"createdAt"`
	LastModifiedAt time.Time `json:"lastModifiedAt"`
	Contributers   []string  `json:"contributers,omitempty"`
}

func isInSlice(slice []string, str string) bool {
	for i := range slice {
		if slice[i] == str {
			return true
		}
	}
	return false
}

// listDeltas returns a handler which returns a list of all the deltas in the specified app.
//
// The handler expects the organization to be defined by a parameter "orgId" and app by "appId"
func (s *server) listDeltas() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		params := mux.Vars(r)
		sets, err := s.model.selectAllDeltas(params["orgId"], params["appId"])
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

// getDelta returns a handler which returns a specific delta in the specified app.
//
// The handler expects the organization to be defined by a parameter "orgId", the app by "appId" and the set by "deltaId"
func (s *server) getDelta() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		params := mux.Vars(r)
		set, err := s.model.selectDelta(params["orgId"], params["appId"], params["deltaId"])
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

// createDelta returns a handler which adds a delta to the specified app.
//
// The handler expects the organization to be defined by a parameter "orgId", the app by "appId".
//
// The Delta should be provided in the body.
//
// The handler returns the following status codes:
//
// 201 Delta created; body of response is new set ID
//
// 422 Delta was malformed
func (s *server) createDelta() http.HandlerFunc {
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

		createdTime := time.Now().UTC()
		metadata := DeltaMetadata{
			CreatedBy:      getUser(r),
			CreatedAt:      createdTime,
			LastModifiedAt: createdTime,
		}

		id, err := s.model.insertDelta(params["orgId"], params["appId"], false, metadata, delta)
		if err != nil {
			w.WriteHeader(500)
			return
		}
		fmt.Fprintf(w, `"%s"`, id)
	}
}

// replaceDelta returns a handler which replaces a delta with an ne delta.
//
// The handler expects the organization to be defined by a parameter "orgId", the app by "appId" and deltaId by "deltaId".
//
// The new Delta should be provided in the body.
//
// The handler returns the following status codes:
//
// 200 Delta sucessfully replaced.
//
// 404 The deltaId was not found.
//
// 422 Delta was malformed
func (s *server) replaceDelta() http.HandlerFunc {
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

		currentDeltaWrapper, err := s.model.selectDelta(params["orgId"], params["appId"], params["deltaId"])
		if errors.Is(err, ErrNotFound) {
			w.WriteHeader(404)
			return
		}

		metadata := currentDeltaWrapper.Metadata
		metadata.LastModifiedAt = time.Now().UTC()
		currentUser := getUser(r)

		if currentUser != metadata.CreatedBy && !isInSlice(metadata.Contributers, currentUser) {
			newContributers := make([]string, len(metadata.Contributers), len(metadata.Contributers)+1)
			copy(newContributers, metadata.Contributers)
			metadata.Contributers = append(newContributers, currentUser)
		}

		err = s.model.updateDelta(params["orgId"], params["appId"], params["deltaId"], false, metadata, delta)
		if err != nil {
			w.WriteHeader(500)
			return
		}
	}
}
