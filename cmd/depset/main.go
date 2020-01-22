package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/handlers"
	"github.com/julienschmidt/httprouter"
	_ "github.com/lib/pq"
)

type server struct {
	db     *sql.DB
	router *httprouter.Router
}

func twoToPow(i int) int {
	return int(1 << uint(i))
}

func processDbEnvVar(varName string) string {
	value := os.Getenv(varName)
	if value == "" {
		log.Printf("Variable `%s` not set.", varName)
	}
	// The connection string requires that single quotes are escaped
	return strings.ReplaceAll(value, "'", "\\'")
}

func buildConnStr() string {
	dbName := processDbEnvVar("DATABASE_NAME")
	dbUser := processDbEnvVar("DATABASE_USER")
	dbPassword := processDbEnvVar("DATABASE_PASSWORD")
	dbHost := processDbEnvVar("DATABASE_HOST")

	return fmt.Sprintf("dbname='%s' user='%s' password='%s' host='%s' connect_timeout=1 sslmode=disable", dbName, dbUser, dbPassword, dbHost)
}

func initDb(db *sql.DB) error {
	attempt := 1
	_, err := db.Query("SET timezone = 'utc'")
	for err != nil && attempt < 6 {
		log.Printf("Cannot connect to DB, backing off and trying again in %d seconds.", twoToPow(attempt))
		log.Println(err)
		time.Sleep(time.Duration(twoToPow(attempt)) * time.Second)
		attempt++
		_, err = db.Query("SET timezone = 'utc'")
	}
	if attempt >= 6 {
		log.Fatal("Unable to connect to Database. ")
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS sets (
	    id          VARCHAR(40) NOT NULL PRIMARY KEY,
      org_id      TEXT NOT NULL,
      app_id      TEXT NOT NULL,
			metadata    JSONB NOT NULL,
      content     JSONB NOT NULL
	)`)
	if err != nil {
		log.Println("Unable to create sets table.")
		log.Fatal(err)
	}
	/*
			_, err = db.Exec(`CREATE TABLE IF NOT EXISTS deltas (
			    id          SERIAL PRIMARY KEY,
		      org_id      TEXT NOT NULL,
		      app_id      TEXT NOT NULL,
		      name        TEXT NOT NULL,
					metadata    JSONB NOT NULL,
		      content     JSONB NOT NULL
		      UNIQUE(org_id,app_id,name)`)
			if err != nil {
				log.Println("Unable to create deltas table.")
				log.Fatal(err)
			}
	*/
	return nil
}

func main() {
	var s server
	var err error

	log.Println("Connecting to Database.")
	s.db, err = sql.Open("postgres", buildConnStr())
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Initializing Database.")
	initDb(s.db)

	log.Println("Setting up Routes")
	s.setupRoutes()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Listening on Port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, handlers.LoggingHandler(os.Stdout, s.router)))
}
