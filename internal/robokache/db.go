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

	// Omit in json to prevent exposing user emails
	Owner      string         `db:"owner" json:"-"`
	// Replaces owner in JSON
	Owned      bool         `db:"-"     json:"owned"`
	Visibility *visibility     `db:"visibility" json:"visibility"`
}

type visibility int

const (
	invisible visibility = 0
	private   visibility = 1
	shareable visibility = 2
	public    visibility = 3
)

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

// Set Owned field for JSON responses
func (doc *Document) addOwned(owner string) {
	if doc.Owner == owner {
		doc.Owned = true
	} else {
		doc.Owned = false
	}
}

func clearDB() error {
	_, err := db.Exec(`DELETE FROM document`)
	return err
}

func loadSampleData() error {
	// Helper function to make an address from a constant
	// The parent field needs to have a pointer to an int
	// So this helps with that
	i := func(s int) *int { return &s }
	v := func(s visibility) *visibility { return &s }

	sampleDocuments := []Document{
		{ID: 0, Parent: nil,  Owner: "me@robokache.com",  Visibility: v(private)},

		{ID: 1, Parent: nil,  Owner: "me@robokache.com",  Visibility: v(shareable)},
		{ID: 2, Parent: i(1), Owner: "me@robokache.com",  Visibility: v(shareable)},
		{ID: 3, Parent: i(1), Owner: "me@robokache.com",  Visibility: v(public)},

		{ID: 4, Parent: nil,  Owner: "you@robokache.com", Visibility: v(public)},
		{ID: 5, Parent: nil,  Owner: "you@robokache.com", Visibility: v(shareable)},
		{ID: 6, Parent: nil,  Owner: "you@robokache.com", Visibility: v(private)},

		{ID: 7, Parent: i(5),  Owner: "you@robokache.com", Visibility: v(shareable)},
		{ID: 8, Parent: i(5),  Owner: "you@robokache.com", Visibility: v(public)},
	}

	for _, doc := range sampleDocuments {
		_, err := db.Exec(
			`INSERT INTO document(id, parent, owner, visibility) VALUES
			(?, ?, ?, ?)`, doc.ID, doc.Parent, doc.Owner, doc.Visibility)
		if err != nil {
			return err
		}
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

