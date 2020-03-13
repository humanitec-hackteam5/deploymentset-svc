package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

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
	_, err = db.Exec(`DO $$
	  BEGIN
	    IF NOT EXISTS (
	      SELECT 1 FROM information_schema.tables WHERE table_name = 'set_owners'
	    )
	    THEN
	      DROP TABLE IF EXISTS sets;
				DROP TABLE IF EXISTS deltas;
	    END IF;
	  END
	$$;`)
	if err != nil {
		log.Println("Unable to perform migration.")
		log.Fatal(err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS sets (
	    id          TEXT NOT NULL PRIMARY KEY,
			set         JSONB NOT NULL
	)`)
	if err != nil {
		log.Println("Unable to create sets table.")
		log.Fatal(err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS set_owners (
      org_id      TEXT NOT NULL,
      app_id      TEXT NOT NULL,
			set_id      TEXT NOT NULL,
			UNIQUE (org_id, app_id, set_id)
	)`)
	if err != nil {
		log.Println("Unable to create sets table.")
		log.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS deltas (
	    id          TEXT NOT NULL,
      org_id      TEXT NOT NULL,
      app_id      TEXT NOT NULL,
			locked      BOOLEAN NOT NULL,
			metadata    JSONB NOT NULL,
      delta       JSONB NOT NULL,
			UNIQUE (org_id, app_id, id)
	)`)
	if err != nil {
		log.Println("Unable to create deltas table.")
		log.Fatal(err)
	}
	return nil
}

func (s *server) setupModel() {
	log.Println("Connecting to Database.")
	db, err := sql.Open("postgres", buildConnStr())
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Initializing Database.")
	initDb(db)

	s.model = model{db}
}
