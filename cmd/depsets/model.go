package main

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"humanitec.io/deploymentset-svc/pkg/depset"
)

// model is the underlying type for the entire model.
type model struct {
	*sql.DB
}

// ErrNotFound indicates that the resource could not be found
var ErrNotFound = errors.New("not found")

// ErrAlreadyExists indicates that this resource already exists
var ErrAlreadyExists = errors.New("already exists")

// A persistable version of a depset.Set
type persistableSet depset.Set

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

// persistableSetMetadata is a persistable version of SetMetadata
type persistableSetMetadata SetMetadata

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

// persistableDelta is a persistable version of depset.Delta
type persistableDelta depset.Delta

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

// selectAllSets fetches a list of all the sets created in a particular app
func (db model) selectAllSets(orgID string, appID string) ([]SetWrapper, error) {
	rows, err := db.Query(`SELECT id, metadata, content FROM sets WHERE org_id = $1 AND app_id = $2`, orgID, appID)
	defer rows.Close()
	if err != nil {
		log.Printf("Database error fetching sets in org `%s` and app `%s`. (%v)", orgID, appID, err)
		return nil, fmt.Errorf("select all sets: %v", err)
	}

	var sets []SetWrapper
	for rows.Next() {
		var sw SetWrapper
		rows.Scan(&sw.ID, (*persistableSetMetadata)(&sw.Metadata), (*persistableSet)(&sw.Content))
		sets = append(sets, sw)
	}
	return sets, nil
}

// selecteSet fetches a particular set from an app.
// The ErrNotFound sential error is returned if the specific set could not be found.
func (db model) selectSet(orgID string, appID string, setID string) (SetWrapper, error) {
	row := db.QueryRow(`SELECT id, metadata, content FROM sets WHERE org_id = $1 AND app_id = $2 AND id = $3`, orgID, appID, setID)
	var sw SetWrapper
	err := row.Scan(&sw.ID, (*persistableSetMetadata)(&sw.Metadata), (*persistableSet)(&sw.Content))
	if err == sql.ErrNoRows {
		return SetWrapper{}, ErrNotFound
	} else if err != nil {
		log.Printf("Database error fetching set in org `%s` and app `%s` with Id `%s`. (%v)", orgID, appID, setID, err)
		return SetWrapper{}, fmt.Errorf("select set: %v", err)
	}
	return sw, nil
}

// selectRawSet returns a depset.Set rather than SetWrapper version of a set.
func (db model) selectRawSet(orgID string, appID string, setID string) (depset.Set, error) {
	row := db.QueryRow(`SELECT content FROM sets WHERE org_id = $1 AND app_id = $2 AND id = $3`, orgID, appID, setID)
	var set depset.Set
	err := row.Scan((*persistableSet)(&set))
	if err == sql.ErrNoRows {
		return depset.Set{}, ErrNotFound
	} else if err != nil {
		log.Printf("Database error fetching set in org `%s` and app `%s` with Id `%s`. (%v)", orgID, appID, setID, err)
		return depset.Set{}, fmt.Errorf("select set: %v", err)
	}
	return set, nil
}

// insertSet stores a set for a particular app.
// The sentinal error ErrAlreadyExists is returened if that set already exists.
func (db model) insertSet(orgID string, appID string, sw SetWrapper) error {
	result, err := db.Exec(`INSERT INTO sets (org_id, app_id, id, metadata, content ) VALUES ($1, $2, $3, $4, $5) ON CONFLICT DO NOTHING`, orgID, appID, sw.ID, (*persistableSetMetadata)(&sw.Metadata), (*persistableSet)(&sw.Content))
	if err != nil {
		log.Printf("Database error inserting set in org `%s` and app `%s` with Id `%s`. (%v)", orgID, appID, sw.ID, err)
		return fmt.Errorf("insert set: %v", err)
	}
	numRows, err := result.RowsAffected()
	if err != nil {
		log.Printf("Database error requesting rows-affected inserting set in org `%s` and app `%s` with Id `%s`. (%v)", orgID, appID, sw.ID, err)
		return fmt.Errorf("rows affected, insert set: %v", err)
	}
	if numRows == 0 {
		log.Printf("Set with Id `%s` already exists in org `%s` and app `%s`.", sw.ID, orgID, appID)
		return ErrAlreadyExists
	}
	return nil
}
