package main

import "github.com/julienschmidt/httprouter"

func (s *server) setupRoutes() {
	s.router = httprouter.New()
	s.router.GET("/orgs/:orgId/apps/:appId/sets", s.listSets())
	s.router.GET("/orgs/:orgId/apps/:appId/sets/:setId", s.getSet())
	s.router.POST("/orgs/:orgId/apps/:appId/sets/:setId", s.applyDelta())
	/*
		s.router.GET("/orgs/:orgId/apps/:appId/deltas", s.listDeltas())
		s.router.GET("/orgs/:orgId/apps/:appId/deltas/:deltaName", s.getDelta())
		s.router.POST("/orgs/:orgId/apps/:appId/sets/:deltaName", s.addDelta())
	*/
}
