package robokache

import (
	"os"
	"io"
	"database/sql"
	"fmt"
	"strconv"

	_ "github.com/mattn/go-sqlite3" // makes database/sql point to SQLite
)

// GetDocument gets all documents where owner = user OR visibility >= public.
func GetDocuments(userEmail *string, hasParent *bool) ([]Document, error) {
	docs := make([]Document, 0)
	var err error

	queryString := `
		SELECT * FROM document
		WHERE (owner=? OR visibility>=?)`
	// If we are given hasParent add that to the query
	if hasParent != nil {
		if *hasParent {
			queryString += "AND parent IS NOT NULL"
		} else {
			queryString += "AND parent IS NULL"
		}
	}

	err = db.Select(&docs, queryString, userEmail, public)

	if err != nil {
		return nil, err
	}
	return docs, nil
}

// Getdocument gets a document by ID.
// It fails if its owner != user AND visibility < shareable.
func GetDocument(userEmail *string, id int) (Document, error) {
	var doc Document

	// Get rows user is allowed to see
	err := db.Get(&doc, `
		SELECT * FROM document
		WHERE id=? AND (owner=? OR visibility>=?)
	`, id, userEmail, shareable)

	if err == sql.ErrNoRows {
		return doc, fmt.Errorf("Not Found: Check that the document exists and that you have permission to view it.")
	} else if err != nil {
		return doc, err
	}


	return doc, nil
}

// Get all the documents with given id as the parent
func GetDocumentChildren(userEmail *string, id int) ([]Document, error) {
	docs := make([]Document, 0)

	err := db.Select(&docs, `
		SELECT * FROM document
		WHERE parent=? AND (owner=? OR visibility>=?)`,
	id, userEmail, shareable)

	if err != nil {
		return docs, err
	}

	return docs, nil
}

func GetData(id int, w io.Writer) error {
	filename := dataDir + "/files/" + strconv.Itoa(id)

	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		// If the file does not exist write nothing and just return
	    return nil
	} else if err != nil {
		return err
	}

	// Open data file
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Use io.Copy to write without a buffer
	_, err = io.Copy(w, file)
	if err != nil {
		return err
	}

	return nil
}
