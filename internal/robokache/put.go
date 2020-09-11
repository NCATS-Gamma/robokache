package robokache

import (
	"fmt"
	"database/sql"
	"io/ioutil"
	"strconv"
)

// EditDocument modifies the document with the given ID and updates the rest of the fields.
func EditDocument(doc Document) error {

	// Since this is a selective update request we may not be given the parent ID
	// So if we are not, we have to pull it from the database
	if doc.Parent == nil {
		// We have to check if the visibilty is less than the parent
		row := db.QueryRowx(
			`SELECT parent FROM document WHERE
			 id=? AND owner=?
			 `, doc.ID, doc.Owner)
		// Save parent ID to document
		err := row.Scan(&doc.Parent)
		if err == sql.ErrNoRows {
			return fmt.Errorf("Bad Request: Check that the document exists and belongs to you.")
		} else if err != nil {
			return err
		}
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
		WHERE
		id=? AND owner=?;
	`, doc.Visibility, doc.Parent, doc.ID, doc.Owner)

	rowsUpdated, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsUpdated == 0 {
		return fmt.Errorf("Bad Request: Check that the document exists and belongs to you")
	}
	return nil
}

func SetData(id int, data []byte) error {
	filename := dataDir + "/" + strconv.Itoa(id)
	err := ioutil.WriteFile(filename, data, 0644)
	if err != nil {
		return err
	}
	return nil
}
