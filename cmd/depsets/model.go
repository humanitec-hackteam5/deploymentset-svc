package main

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"log"

	"humanitec.io/deploymentset-svc/pkg/depset"
)

type model struct {
	*sql.DB
}

type persistableSet depset.Set
type persistableSetMetadata SetMetadata

type persistableDelta depset.Delta

// ErrNotFound indicates that the resource could not be found
var ErrNotFound = errors.New("not found")

// ErrAlreadyExists indicates that this resource already exists
var ErrAlreadyExists = errors.New("already exists")

// Provide a way for depset.Set to implement the driver.Valuer interface.
func (s persistableSet) Value() (driver.Value, error) {
	return json.Marshal(s)
}

// Provide a way for depset.Set implement the sql.Scanner interface.
func (s *persistableSet) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(b, &s)
}

// Provide a way for depset.SetMetadata to implement the driver.Valuer interface.
func (s persistableSetMetadata) Value() (driver.Value, error) {
	return json.Marshal(s)
}

// Provide a way for depset.SetMetadata implement the sql.Scanner interface.
func (s *persistableSetMetadata) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(b, &s)
}

// Provide a way for depset.Delta to implement the driver.Valuer interface.
func (d persistableDelta) Value() (driver.Value, error) {
	return json.Marshal(d)
}

// Provide a way for depset.Delta implement the sql.Scanner interface.
func (d *persistableDelta) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(b, &d)
}

func (db model) selectAllSets(orgID string, appID string) ([]SetWrapper, error) {
	rows, err := db.Query(`SELECT id, metadata, content FROM sets WHERE org_id = $1 AND app_id = $2`, orgID, appID)
	defer rows.Close()
	if err != nil {
		log.Printf("Database error fetching sets in org `%s` and app `%s`.", orgID, appID)
		log.Println(err)
		return nil, err
	}

	var sets []SetWrapper
	for rows.Next() {
		var sw SetWrapper
		rows.Scan(&sw.ID, (*persistableSetMetadata)(&sw.Metadata), (*persistableSet)(&sw.Content))
		sets = append(sets, sw)
	}
	return sets, nil
}

func (db model) selectSet(orgID string, appID string, setID string) (SetWrapper, error) {
	row := db.QueryRow(`SELECT id, metadata, content FROM sets WHERE org_id = $1 AND app_id = $2 AND id = $3`, orgID, appID, setID)
	var sw SetWrapper
	err := row.Scan(&sw.ID, (*persistableSetMetadata)(&sw.Metadata), (*persistableSet)(&sw.Content))
	if err == sql.ErrNoRows {
		return SetWrapper{}, ErrNotFound
	} else if err != nil {
		log.Printf("Database error fetching set in org `%s` and app `%s` with Id `%s`.", orgID, appID, setID)
		log.Println(err)
		return SetWrapper{}, err
	}
	return sw, nil
}

func (db model) selectRawSet(orgID string, appID string, setID string) (depset.Set, error) {
	row := db.QueryRow(`SELECT content FROM sets WHERE org_id = $1 AND app_id = $2 AND id = $3`, orgID, appID, setID)
	var set depset.Set
	err := row.Scan((*persistableSet)(&set))
	if err == sql.ErrNoRows {
		return depset.Set{}, ErrNotFound
	} else if err != nil {
		log.Printf("Database error fetching set in org `%s` and app `%s` with Id `%s`.", orgID, appID, setID)
		log.Println(err)
		return depset.Set{}, err
	}
	return set, nil
}

func (db model) insertSet(orgID string, appID string, sw SetWrapper) error {
	result, err := db.Exec(`INSERT INTO sets (org_id, app_id, id, metadata, content ) VALUES ($1, $2, $3, $4, $5) ON CONFLICT DO NOTHING`, orgID, appID, sw.ID, (*persistableSetMetadata)(&sw.Metadata), (*persistableSet)(&sw.Content))
	if err != nil {
		log.Printf("Database error inserting set in org `%s` and app `%s` with Id `%s`.", orgID, appID, sw.ID)
		log.Println(err)
		return err
	}
	numRows, err := result.RowsAffected()
	if numRows == 0 {
		log.Printf("Set with Id `%s` already exists in org `%s` and app `%s`.", sw.ID, orgID, appID)
		return ErrAlreadyExists
	}
	return nil
}
