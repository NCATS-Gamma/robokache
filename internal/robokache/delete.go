package robokache

import (
	"fmt"
)

// DeleteDocument deletes the document that matches the ID and Owner
func DeleteDocument(doc Document) error {
	result, err := db.Exec(`
		DELETE FROM document WHERE
		id=? AND owner=?;
	`, doc.ID, doc.Owner)

	rowsDeleted, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsDeleted == 0 {
		return fmt.Errorf("Bad Request: Check that the document exists and belongs to you")
	}
	return nil
}
