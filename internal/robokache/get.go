package robokache

import (
	"os"
	"io/ioutil"
	"database/sql"
	"fmt"
	"strconv"

	_ "github.com/mattn/go-sqlite3" // makes database/sql point to SQLite
)

// GetDocument gets all documents where owner = user OR visibility >= public.
func GetDocuments(userEmail string) ([]Document, error) {
	// Slice of rows
	var docs []Document

	// Get rows user is allowed to see
	err := db.Select(&docs, `
		SELECT * FROM document
		WHERE owner=? OR visibility>=?
	`, userEmail, public)

	if err != nil {
		return nil, err
	}
	return docs, nil
}

// Getdocument gets a document by ID.
// It fails if its owner != user AND visibility < shareable.
func GetDocument(userEmail string, id int) (Document, error) {
	var doc Document

	// Get rows user is allowed to see
	err := db.Get(&doc, `
		SELECT * FROM document
		WHERE id=? AND (owner=? OR visibility>=?)
	`, id, userEmail, shareable)

	if err == sql.ErrNoRows {
		return doc, fmt.Errorf("Not Found: document %d", id)
	} else if err != nil {
		return doc, err
	}


	return doc, nil
}

// Get all the documents with given id as the parent
func GetDocumentChildren(userEmail string, id int) ([]Document, error) {
	var docs []Document

	err := db.Select(&docs, `
		SELECT * FROM document
		WHERE parent=? AND (owner=? OR visibility>=?)`,
	id, userEmail, public)

	if err != nil {
		return docs, err
	}

	return docs, nil
}

func GetData(id int) ([]byte, error) {
	filename := dataDir + "/" + strconv.Itoa(id)

	// If the file does not exist, return empty data
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
	    return []byte{}, nil
	} else if err != nil {
		return nil, err
	}
	// Read associated JSON file
	data, err := ioutil.ReadFile(filename)

	if err != nil {
		return nil, err
	}

	return data, nil
}
