package robokache

import (
	"fmt"
	"database/sql"
	"io"
	"os"
	"strconv"
)

// EditDocument modifies the document with the given ID and updates the rest of the fields.
func EditDocument(doc Document) error {

	// Since this is a selective update request we may not be given fields
	// So we have to pull it from the database
	var existingDoc Document
	err := db.Get(&existingDoc,
		`SELECT * FROM document WHERE
		 id=? AND owner=?`, doc.ID, doc.Owner)
	if err == sql.ErrNoRows {
		return fmt.Errorf("Bad Request: Check that the document exists and belongs to you.")
	} else if err != nil {
		return err
	}

	// Fill in parent and visibility fields if not given
	if doc.Parent == nil {
		doc.Parent = existingDoc.Parent
	}
	if doc.Visibility == nil {
		doc.Visibility = existingDoc.Visibility
	}

	// If the parent is still null the document has no parent
	if doc.Parent != nil {
		var parent Document
		// If a parent exists, we have to check that the parent fits these criteria:
		// 1. Exists in the DB
		// 2. Has the same owner
		// 3. Has more or the same visibility than the child
		row := db.QueryRowx(
			`SELECT * FROM document WHERE
			 id=? AND owner=? AND visibility>=?
			 `, doc.Parent, doc.Owner, doc.Visibility)
		err := row.StructScan(&parent)
		if err == sql.ErrNoRows {
			return fmt.Errorf("Bad Request: Check that the parent exists and that you are not changing this document to be more visible than the parent")
		} else if err != nil {
			return err
		}
	}

	// Update document
	result, err := db.Exec(`
		UPDATE document SET
		visibility=?, parent=?
		WHERE id=?;
	`, doc.Visibility, doc.Parent, doc.ID)

	_, err = result.RowsAffected()
	if err != nil {
		return err
	}
	return nil
}

func SetData(id int) (io.WriteCloser, error) {
	filename := dataDir + "/files/" + strconv.Itoa(id)
	f, err := os.Create(filename)
	if err != nil {
		return nil, err
	}
	return f, nil
}
