package robokache

import (
	"database/sql"
	"fmt"
	"os"
	"log"

	"errors"
	_ "github.com/mattn/go-sqlite3" // makes database/sql point to SQLite
)

// DeleteQuestion deletes an existing question. It fails if question.owner != user.
func DeleteQuestion(question Question) error {
	// Confirm that such a question exists that we have permissions for
	questionDB, err := GetQuestion(question.Owner, question.ID)
	if err != nil {
		return err
	}
	if !(questionDB["owned"].(bool)) {
		return errors.New("Unauthorized: You do not own this question and may not add answers")
	}

	// Open SQLite database
	db, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Delete question
	stmt := fmt.Sprintf(`
		DELETE FROM questions
		WHERE id='%s' AND owner='%s'
	`, question.ID, question.Owner)
	_, err = db.Exec(stmt)
	if err != nil {
		log.Fatal(err)
	}

	// Delete question file
	err = os.Remove(dataDir+"/"+question.ID+".json")
	return err
}

// DeleteAnswer removes the specified answer. If fails if answer.question.owner != user.
func DeleteAnswer(userEmail string, answer Answer) error {
	// Confirm that such a question exists and we can add answers
	question, err := GetQuestion(userEmail, answer.Question)
	if err != nil {
		return err
	}
	if !(question["owned"].(bool)) {
		return errors.New("Unauthorized: You do not own this question and may not add answers")
	}

	// Open SQLite database
	db, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Delete answer
	stmt := fmt.Sprintf(`
		DELETE FROM answers
		WHERE id='%s' AND question='%s'
	`, answer.ID, question["id"])
	res, err := db.Exec(stmt)
	if err != nil {
		log.Fatal(err)
	}
	ra, _ := res.RowsAffected()
	fmt.Printf("Rows affected: %d", ra)

	// Delete question file
	err = os.Remove(dataDir+"/"+answer.ID+".json")
	return err
}
