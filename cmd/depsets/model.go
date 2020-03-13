package main

import (
	"database/sql"
	"database/sql/driver"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"time"

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

// Provide a way for SetMetadata to implement the driver.Valuer interface.
func (s persistableSetMetadata) Value() (driver.Value, error) {
	return json.Marshal(s)
}

// Provide a way for SetMetadata implement the sql.Scanner interface.
func (s *persistableSetMetadata) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(b, &s)
}

// A persistable version of a depset.Set
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

// persistableSetMetadata is a persistable version of SetMetadata
type persistableDeltaMetadata DeltaMetadata

// Provide a way for DeltaMetadata to implement the driver.Valuer interface.
func (d persistableDeltaMetadata) Value() (driver.Value, error) {
	return json.Marshal(d)
}

// Provide a way for DeltaMetadata implement the sql.Scanner interface.
func (d *persistableDeltaMetadata) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(b, &d)
}

// selectAllSets fetches a list of all the sets created in a particular app
func (db model) selectAllSets(orgID string, appID string) ([]SetWrapper, error) {
	rows, err := db.Query(`
		SELECT sets.id, sets.set
		FROM sets
		LEFT JOIN set_owners
		ON sets.id = set_id
		WHERE org_id = $1 AND app_id = $2`, orgID, appID)
	defer rows.Close()
	if err != nil {
		log.Printf("Database error fetching sets in org `%s` and app `%s`. (%v)", orgID, appID, err)
		return nil, fmt.Errorf("select all sets: %w", err)
	}

	var sets []SetWrapper
	for rows.Next() {
		var sw SetWrapper
		rows.Scan(&sw.ID, (*persistableSet)(&sw.Set))
		sets = append(sets, sw)
	}
	return sets, nil
}

// selecteSet fetches a particular set from an app.
// The ErrNotFound sential error is returned if the specific set could not be found.
func (db model) selectSet(orgID string, appID string, setID string) (SetWrapper, error) {
	row := db.QueryRow(`SELECT sets.id, sets.set
		FROM sets
		LEFT JOIN set_owners
		ON sets.id = set_id
		WHERE org_id = $1 AND app_id = $2 AND sets.id = $3`, orgID, appID, setID)
	var sw SetWrapper
	err := row.Scan(&sw.ID, (*persistableSet)(&sw.Set))
	if err == sql.ErrNoRows {
		return SetWrapper{}, ErrNotFound
	} else if err != nil {
		log.Printf("Database error fetching set in org `%s` and app `%s` with Id `%s`. (%v)", orgID, appID, setID, err)
		return SetWrapper{}, fmt.Errorf("select set: %w", err)
	}
	return sw, nil
}

// selectUnscopedRawSet fetches a particular set.
// The ErrNotFound sential error is returned if the specific set could not be found.
func (db model) selectUnscopedRawSet(setID string) (depset.Set, error) {
	row := db.QueryRow(`SELECT set FROM sets WHERE id = $1`, setID)
	var set depset.Set
	err := row.Scan((*persistableSet)(&set))
	if err == sql.ErrNoRows {
		return depset.Set{}, ErrNotFound
	} else if err != nil {
		log.Printf("Database error fetching set with Id `%s`. (%v)", setID, err)
		return depset.Set{}, fmt.Errorf("select set: %w", err)
	}
	return set, nil
}

// selectRawSet returns a depset.Set rather than SetWrapper version of a set.
func (db model) selectRawSet(orgID string, appID string, setID string) (depset.Set, error) {
	row := db.QueryRow(`SELECT sets.set
		FROM set_owners
		LEFT JOIN sets
		ON id = set_id
		WHERE org_id = $1 AND app_id = $2 AND set_id = $3`,
		orgID, appID, setID)
	var set depset.Set
	err := row.Scan((*persistableSet)(&set))
	if err == sql.ErrNoRows {
		return depset.Set{}, ErrNotFound
	} else if err != nil {
		log.Printf("Database error fetching set in org `%s` and app `%s` with Id `%s`. (%v)", orgID, appID, setID, err)
		return depset.Set{}, fmt.Errorf("select set: %w", err)
	}
	return set, nil
}

// insertSet stores a set for a particular app.
// The sentinal error ErrAlreadyExists is returened if that set already exists.
func (db model) insertSet(orgID string, appID string, sw SetWrapper) error {
	_, err := db.Exec(`INSERT INTO sets (id, set) VALUES ($1, $2) ON CONFLICT DO NOTHING`, sw.ID, (*persistableSet)(&sw.Set))
	if err != nil {
		log.Printf("Database error inserting set with Id `%s`. (%v)", sw.ID, err)
		return fmt.Errorf("insert set: %w", err)
	}

	result, err := db.Exec(`INSERT INTO set_owners (org_id, app_id, set_id) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`, orgID, appID, sw.ID)
	if err != nil {
		log.Printf("Database error inserting set_owners with ID `%s` in app %s/%s. (%v)", sw.ID, orgID, appID, err)
		return fmt.Errorf("insert set_owners: %w", err)
	}
	numRows, err := result.RowsAffected()
	if err != nil {
		log.Printf("Database error requesting rows-affected inserting set_owners ID `%s` in app %s/%s. (%v)", sw.ID, orgID, appID, err)
		return fmt.Errorf("rows affected, insert set_owners: %w", err)
	}
	if numRows == 0 {
		log.Printf("Set with ID `%s` already exists in app %s/%s. (%v)", sw.ID, orgID, appID, err)
		return ErrAlreadyExists
	}
	return nil
}

// selectAllDeltas fetches a list of all the deltas created in a particular app
func (db model) selectAllDeltas(orgID string, appID string) ([]DeltaWrapper, error) {
	rows, err := db.Query(`SELECT id, metadata, delta FROM deltas WHERE org_id = $1 AND app_id = $2`, orgID, appID)
	defer rows.Close()
	if err != nil {
		log.Printf("Database error fetching deltas in org `%s` and app `%s`. (%v)", orgID, appID, err)
		return nil, fmt.Errorf("select all sets (%s, %s): %w", orgID, appID, err)
	}

	var deltas []DeltaWrapper
	for rows.Next() {
		var dw DeltaWrapper
		rows.Scan(&dw.ID, (*persistableDeltaMetadata)(&dw.Metadata), (*persistableDelta)(&dw.Delta))
		deltas = append(deltas, dw)
	}
	return deltas, nil
}

// insertDelta stores a delta for a particular app.
func (db model) insertDelta(orgID, appID string, locked bool, metadata DeltaMetadata, content depset.Delta) (string, error) {

	// We just need a unique ID here. Does not need to be cryptographically unguessable - just unique.
	rand.Seed(time.Now().UnixNano())
	randomValue := make([]byte, 20, 20)
	notUnique := true
	var id string

	for notUnique {
		rand.Read(randomValue)
		id = hex.EncodeToString(randomValue)
		result, err := db.Exec(`INSERT INTO deltas (org_id, app_id, id, locked, metadata, delta ) VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT DO NOTHING`, orgID, appID, id, locked, (*persistableDeltaMetadata)(&metadata), (*persistableDelta)(&content))
		if err != nil {
			log.Printf("Database error inserting delta in org `%s` and app `%s` and ID `%s`. (%v)", orgID, appID, id, err)
			return "", fmt.Errorf("insert delta: %w", err)
		}
		numRows, err := result.RowsAffected()
		if err != nil {
			log.Printf("Database error requesting rows-affected inserting delta in org `%s` and app `%s` and ID `%s`. (%v)", orgID, appID, id, err)
			return "", fmt.Errorf("rows affected, insert delta: %w", err)
		}
		notUnique = numRows == 0
	}
	return id, nil
}

// updateDelta stores a delta for a particular app.
func (db model) updateDelta(orgID, appID, deltaID string, locked bool, metadata DeltaMetadata, delta depset.Delta) error {
	result, err := db.Exec(`UPDATE deltas SET metadata = $4, delta = $5 WHERE org_id = $1 AND app_id = $2 AND id = $3`, orgID, appID, deltaID, (*persistableDeltaMetadata)(&metadata), (*persistableDelta)(&delta))
	if err != nil {
		log.Printf("Database error updating delta `%s`. (%v)", deltaID, err)
		return fmt.Errorf("update delta (%s): %w", deltaID, err)
	}
	numRows, err := result.RowsAffected()
	if err != nil {
		log.Printf("Database error requesting rows-affected updating delta in org `%s` and app `%s` and ID `%s`. (%v)", orgID, appID, deltaID, err)
		return fmt.Errorf("rows affected, update delta: %w", err)
	}
	if numRows == 0 {
		return ErrNotFound
	}
	return nil
}

// selecteSet fetches a particular set from an app.
// The ErrNotFound sential error is returned if the specific set could not be found.
func (db model) selectDelta(orgID string, appID string, deltaID string) (DeltaWrapper, error) {
	row := db.QueryRow(`SELECT id, metadata, delta FROM deltas WHERE org_id = $1 AND app_id = $2 AND id = $3`, orgID, appID, deltaID)
	var dw DeltaWrapper
	err := row.Scan(&dw.ID, (*persistableDeltaMetadata)(&dw.Metadata), (*persistableDelta)(&dw.Delta))
	if err == sql.ErrNoRows {
		return DeltaWrapper{}, ErrNotFound
	} else if err != nil {
		log.Printf("Database error fetching delta in org `%s` and app `%s` with Id `%s`. (%v)", orgID, appID, deltaID, err)
		return DeltaWrapper{}, fmt.Errorf("select delta (%s, %s, %s): %w", orgID, appID, deltaID, err)
	}
	return dw, nil
}
