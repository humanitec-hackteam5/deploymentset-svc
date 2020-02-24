package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gorilla/handlers"
	_ "github.com/lib/pq"
	"humanitec.io/deploymentset-svc/pkg/depset"
)

type modeler interface {
	insertSet(orgID string, appID string, sw SetWrapper) error
	selectAllSets(orgID string, appID string) ([]SetWrapper, error)
	selectSet(orgID string, appID string, setID string) (SetWrapper, error)
	selectRawSet(orgID string, appID string, setID string) (depset.Set, error)
	selectAllDeltas(orgID string, appID string) ([]DeltaWrapper, error)
	insertDelta(orgID string, appID string, locked bool, metadata DeltaMetadata, content depset.Delta) (string, error)
	updateDelta(orgID, appID, deltaID string, locked bool, metadata DeltaMetadata, content depset.Delta) error
	selectDelta(orgID string, appID string, deltaID string) (DeltaWrapper, error)
}

type server struct {
	model  modeler
	router http.Handler
}

func main() {
	var s server

	log.Println("Setting up Model")
	s.setupModel()

	log.Println("Setting up Routes")
	s.setupRoutes()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Listening on Port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, handlers.LoggingHandler(os.Stdout, s.router)))
}
