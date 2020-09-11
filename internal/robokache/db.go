package robokache

import (
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3" // makes database/sql point to SQLite
)

type Document struct {
    // Omit in JSON to prevent exposing primary key
	ID         int            `db:"id"     json:"-"`
	// Replaces ID in JSON, not stored in db
	Hash       string         `db:"-"      json:"id"`
	// Allow parent to be null using a pointer
	Parent     *int           `db:"parent" json:"-"`
	// Replaces parent field in JSON, not stored in db
	ParentHash string         `db:"-"      json:"parent"`

	Owner      string         `db:"owner" json:"owner"`
	Visibility visibility     `db:"visibility" json:"visibility"`
}

var visibilityToInt = map[string]visibility{
	"invisible": invisible,
	"private":   private,
	"shareable": shareable,
	"public":    public,
}
var intToVisibility = []string{
	"invisible",
	"private",
	"shareable",
	"public",
}

// Change IDs in document to Hashes
func (doc *Document) addHash() error {
	var err error
	doc.Hash, err = idToHash(doc.ID)
	if err != nil {
		return err
	}
	// Change parent ID to hash
	if doc.Parent != nil {
		doc.ParentHash, err = idToHash(*doc.Parent)
		if err != nil {
			return err
		}
	}
	return nil
}

// Change Hashes in document to IDs
func (doc *Document) addID() error {
	if doc.Hash != "" {
		newID, err := hashToID(doc.Hash)
		if err != nil {
			return err
		}
		doc.ID = newID
	}

	// Change parent ID to hash
	if doc.ParentHash != "" {
		newParentID, err := hashToID(doc.ParentHash)
		if err != nil {
			return err
		}
		doc.Parent = &newParentID
	}
	return nil
}

func clearDB() error {
	_, err := db.Exec(`DELETE FROM document`)
	return err
}

func loadSampleData() error {
	_, err := db.Exec(
		`INSERT INTO document(id, parent, owner, visibility) VALUES
		(0, NULL, 'user1@robokache.com', 3)`)
	if err != nil {
		return err
	}

	_, err = db.Exec(
		`INSERT INTO document(id, parent, owner, visibility) VALUES
		(1, 0, 'user1@robokache.com', 3)`)
	if err != nil {
		return err
	}

	return nil
}

var db *sqlx.DB

// SetupDB sets up the SQLite database if it does not exist
func init() {
	db = sqlx.MustConnect("sqlite3", dbFile)

	sqlStmt := `
		CREATE TABLE IF NOT EXISTS document (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			parent INTEGER,
			owner TEXT,
			visibility INTEGER
		);`

	db.MustExec(sqlStmt)
}

