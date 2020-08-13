package robokache

import (
	"database/sql"
	"errors"
	"fmt"
	"io/ioutil"
	"log"

	_ "github.com/mattn/go-sqlite3" // makes database/sql point to SQLite
)

// PostQuestion stores a question. It fails if question.owner != user.
func PostQuestion(userEmail string, question Question) error {
	// Check that the question is to be owned by the posting user
	if userEmail != question.Owner {
		return errors.New("Unauthorized: You do not own this question and may not add it")
	}

	// Connect to SQLite
	db, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// Add question to DB
	sqlStmt := fmt.Sprintf(`
	INSERT INTO questions(id, owner, visibility) VALUES
	('%s', '%s', %d);
	`, question.ID, question.Owner, question.Visibility)
	_, err = db.Exec(sqlStmt)
	if err != nil {
		log.Printf("%q: %s\n", err, sqlStmt)
		return err
	}

	// Create question file
	err = ioutil.WriteFile("data/"+question.ID+".json", []byte(question.Data), 0644)
	return err
}

// PostAnswer adds an answer. If fails if answer.question.owner != user.
func PostAnswer(userEmail string, answer Answer) error {
	// Confirm that such a question exists and we can add answers
	question, err := GetQuestion(userEmail, answer.Question)
	if err != nil {
		return err
	}
	if question["owner"] != userEmail {
		return errors.New("Unauthorized: You do not own this question and may not add answers")
	}

	// Open SQLite connection
	db, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Add answer to DB
	sqlStmt := fmt.Sprintf(`
	INSERT INTO answers (id, question, visibility)
	VALUES ('%s', '%s', %d);
	`, answer.ID, answer.Question, answer.Visibility)
	_, err = db.Exec(sqlStmt)
	if err != nil {
		log.Printf("%q: %s\n", err, sqlStmt)
		return err
	}

	// Add answer file
	err = ioutil.WriteFile("data/"+answer.ID+".json", []byte(answer.Data), 0644)
	return err
}
