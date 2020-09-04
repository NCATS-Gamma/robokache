package robokache

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"

	_ "github.com/mattn/go-sqlite3" // makes database/sql point to SQLite
)

// GetQuestions gets all questions where owner = user OR visibility >= public.
func GetQuestions(userEmail string) ([]map[string]interface{}, error) {
	// Open SQLite database
	db, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Get rows user is allowed to see
	stmt := fmt.Sprintf(`
	SELECT id, owner, visibility FROM questions
	WHERE owner='%s' OR visibility>=%d
	`, userEmail, public)
	rows, err := db.Query(stmt)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	// Read rows
	var docs []map[string]interface{}
	for rows.Next() {
		var id string
		var owner string
		var visibility visibility

		// Copy row into variables
		err = rows.Scan(&id, &owner, &visibility)
		fatal(err)

		// Compile map of questions
		docs = append(docs, map[string]interface{}{
			"id":         id,
			"owner":      owner,
			"visibility": intToVisibility[visibility],
		})
	}
	return docs, nil
}

// GetQuestion gets a question by ID.
// It fails if its owner != user AND visibility < shareable.
func GetQuestion(userEmail string, id string) (map[string]interface{}, error) {

	// Open SQLite database
	db, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Get rows user is allowed to see
	stmt := fmt.Sprintf(`
	SELECT id, owner, visibility FROM questions
	WHERE id='%s' AND (owner='%s' OR visibility>=%d)
	`, id, userEmail, shareable)
	rows, err := db.Query(stmt)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	// Read rows
	for rows.Next() {
		var id string
		var owner string
		var visibility visibility

		// Copy row into variables
		err = rows.Scan(&id, &owner, &visibility)
		fatal(err)

		// Read associated JSON file
		data, err := ioutil.ReadFile(dataDir + "/" + id + ".json")
		check(err)

		// Return question
		question := map[string]interface{}{
			"id":         id,
			"owner":      owner,
			"visibility": intToVisibility[visibility],
			"data":       string(data),
		}
		return question, nil
	}
	return nil, fmt.Errorf("Not Found: Question %s", id)
}

// GetAnswers get all answers where the question's owner = user
// OR question.visibility >= public AND answer.visibility >= public.
func GetAnswers(userEmail string, question string) ([]map[string]interface{}, error) {

	// Open SQLite database
	db, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Get rows user is allowed to see
	stmt := fmt.Sprintf(`
	SELECT answers.id, questions.id, answers.visibility, questions.owner FROM answers
	LEFT OUTER JOIN questions ON answers.question=questions.id
	WHERE answers.question='%s' AND (questions.owner='%s' OR (questions.visibility>=%d AND answers.visibility>=%d))
	`, question, userEmail, public, public)
	rows, err := db.Query(stmt)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	// Read rows
	var docs []map[string]interface{}
	for rows.Next() {
		var id string
		var question string
		var visibility visibility
		var owner string

		// Copy row into variables
		err = rows.Scan(&id, &question, &visibility, &owner)
		fatal(err)

		// Compile map of answers
		docs = append(docs, map[string]interface{}{
			"id":         id,
			"question":   question,
			"visibility": intToVisibility[visibility],
		})
	}
	return docs, nil
}

// GetAnswer gets an answer by ID.
// It fail if answer.question.owner != user AND
// (answer.question.visibility < shareable OR anwer.visibility < shareable).
func GetAnswer(userEmail string, id string) (map[string]interface{}, error) {

	// Open SQLite database
	db, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Get rows user is allowed to see
	stmt := fmt.Sprintf(`
	SELECT answers.id, questions.id, answers.visibility FROM answers
	LEFT OUTER JOIN questions ON answers.question=questions.id
	WHERE answers.id='%s' AND (questions.owner='%s' OR (questions.visibility>=%d AND answers.visibility>=%d))
	`, id, userEmail, shareable, shareable)
	rows, err := db.Query(stmt)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	// Read rows
	for rows.Next() {
		var id string
		var question string
		var visibility visibility

		// Copy row into variables
		err = rows.Scan(&id, &question, &visibility)
		fatal(err)

		// Read associated JSON file
		data, err := ioutil.ReadFile(dataDir + "/" + id + ".json")
		check(err)

		answer := map[string]interface{}{
			"id":         id,
			"question":   question,
			"visibility": intToVisibility[visibility],
			"data":       string(data),
		}
		return answer, nil
	}
	return nil, fmt.Errorf("Not Found: Answer %s", id)
}
