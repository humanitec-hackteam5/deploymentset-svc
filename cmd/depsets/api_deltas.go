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
	CreatedBy      string    `json:"created_by"`
	CreatedAt      time.Time `json:"created_at"`
	LastModifiedAt time.Time `json:"last_modified_at"`
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
		deltas, err := s.model.selectAllDeltas(params["orgId"], params["appId"])
		if err != nil {
			w.WriteHeader(500)
			return
		}

		// Handle special case of empty list as it could just be nil.
		if len(deltas) == 0 {
			fmt.Fprintf(w, `[]`)
			return
		}

		writeAsJSON(w, http.StatusOK, deltas)
	}
}

// getDelta returns a handler which returns a specific delta in the specified app.
//
// The handler expects the organization to be defined by a parameter "orgId", the app by "appId" and the set by "deltaId"
func (s *server) getDelta() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		params := mux.Vars(r)
		deltaWrapper, err := s.model.selectDelta(params["orgId"], params["appId"], params["deltaId"])
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				w.WriteHeader(http.StatusNotFound)
			} else {
				w.WriteHeader(500)
			}
			return
		}

		writeAsJSON(w, http.StatusOK, deltaWrapper)
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
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		}
		err := json.NewDecoder(r.Body).Decode(&delta)
		if nil != err {
			w.WriteHeader(http.StatusUnprocessableEntity)
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
		writeAsJSON(w, http.StatusOK, id)
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
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		}
		err := json.NewDecoder(r.Body).Decode(&delta)
		if nil != err {
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		}

		currentDeltaWrapper, err := s.model.selectDelta(params["orgId"], params["appId"], params["deltaId"])
		if errors.Is(err, ErrNotFound) {
			writeAsJSON(w, http.StatusNotFound, fmt.Sprintf(`Delta with ID "%s" not available in Application "%s/%s".`, params["deltaId"], params["orgId"], params["appId"]))
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
		w.WriteHeader(http.StatusNoContent)
	}
}

// updateDelta returns a handler which updates a delte in place
//
// The handler expects the organization to be defined by a parameter "orgId", the app by "appId" and deltaId by "deltaId".
//
// The new Delta should be provided in the body.
//
// The handler returns the following status codes:
//
// 200 Delta sucessfully replaced.
//
// 400 The deltas cannot be merged as they are not compatible.
//
// 404 The deltaId was not found.
//
// 422 Delta was malformed
func (s *server) updateDelta() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		params := mux.Vars(r)
		var deltas []depset.Delta
		if r.Body == nil {
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		}
		err := json.NewDecoder(r.Body).Decode(&deltas)
		if nil != err {
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		}

		currentDeltaWrapper, err := s.model.selectDelta(params["orgId"], params["appId"], params["deltaId"])
		if errors.Is(err, ErrNotFound) {
			writeAsJSON(w, http.StatusNotFound, fmt.Sprintf(`Delta with ID "%s" not available in Application "%s/%s".`, params["deltaId"], params["orgId"], params["appId"]))
			return
		}

		if len(deltas) == 0 {
			jsonDeltaWrapper, err := json.Marshal(currentDeltaWrapper)
			if err != nil {
				log.Println(err)
				w.WriteHeader(500)
				return
			}
			w.Write(jsonDeltaWrapper)
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

		newDelta, err := depset.MergeDeltas(currentDeltaWrapper.Content, deltas...)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		err = s.model.updateDelta(params["orgId"], params["appId"], params["deltaId"], false, metadata, newDelta)
		if err != nil {
			w.WriteHeader(500)
			return
		}

		writeAsJSON(w, http.StatusOK, DeltaWrapper{
			ID:       params["deltaId"],
			Metadata: metadata,
			Content:  newDelta,
		})
	}
}
