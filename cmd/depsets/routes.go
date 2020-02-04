package main

import "github.com/gorilla/mux"

func (s *server) setupRoutes() {
	r := mux.NewRouter()
	r.Methods("GET").Path("/orgs/{orgId}/apps/{appId}/sets/{leftSetId}").Queries("diff", "{rightSetId}").Handler(s.diffSets())
	r.Methods("POST").Path("/orgs/{orgId}/apps/{appId}/sets/{setId}").Handler(s.applyDelta())
	r.Methods("GET").Path("/orgs/{orgId}/apps/{appId}/sets/{setId}").Handler(s.getSet())
	r.Methods("GET").Path("/orgs/{orgId}/apps/{appId}/sets").Handler(s.listSets())

	s.router = r
}
