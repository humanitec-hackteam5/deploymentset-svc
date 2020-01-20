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
	CreateAt time.Time `json:"createdAt"`
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

func isZeroHash(h string) bool {
	for _, c := range h {
		if c != '0' {
			return false
		}
	}
	return true
}

func (s *server) listSets() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		modules, err := s.selectAllSets(params.ByName("orgId"), params.ByName("appId"))
		if err != nil {
			w.WriteHeader(500)
			return
		}

		jsonModules, err := json.Marshal(modules)
		if err != nil {
			log.Println(err)
			w.WriteHeader(500)
			return
		}
		w.Write(jsonModules)
	}
}

func (s *server) getSet() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		module, err := s.selectSet(params.ByName("orgId"), params.ByName("appId"), params.ByName("setId"))
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
		log.Printf("`%s`", params.ByName("setId"))
		if !isZeroHash(params.ByName("setId")) {
			set, err = s.selectRawSet(params.ByName("orgId"), params.ByName("appId"), params.ByName("setId"))
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
		}
		newSw.ID = newSw.Content.Hash()

		err = s.insertSet(params.ByName("orgId"), params.ByName("appId"), newSw)
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
