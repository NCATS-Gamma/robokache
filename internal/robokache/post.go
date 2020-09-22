package robokache

import (
	"fmt"
	"database/sql"
)

// PostDocument stores a document in the DB. It fails if question.owner != user.
func PostDocument(doc Document) (int, error) {
	if doc.Parent != nil {
		// Check that the parent:
		// 1. Exists
		// 2. Has the same owner
		// 3. Has more or the same visibility than the child
		var parent Document
		row := db.QueryRowx(
			`SELECT * FROM document WHERE
			 id=? AND owner=? AND visibility>=?
			 `, doc.Parent, doc.Owner, doc.Visibility)
		err := row.StructScan(&parent)
		if err == sql.ErrNoRows {
			return -1, fmt.Errorf("Bad Request: Check that the parent exists and does not have less visibility than the child you are trying to add.")
		} else if err != nil {
			return -1, err
		}
	}
	// Add question to DB
	result, err := db.Exec(`
		INSERT INTO document(owner, parent, visibility, metadata) VALUES
    (?, ?, ?, ?);
	`, doc.Owner, doc.Parent, doc.Visibility, doc.Metadata)

	if err != nil {
		return -1, err
	}
	newId, err := result.LastInsertId()
	if err != nil {
		return -1, err
	}
	return int(newId), nil
}
