package robokache

import (
	"errors"
	"io/ioutil"
	_ "github.com/mattn/go-sqlite3" // makes database/sql point to SQLite
)

// PutQuestion modifies an existing question. It fails if question.owner != user.
func PutQuestion(question Question) error {
	// Confirm that such a question exists and we can add answers
	questionDB, err := GetQuestion(question.Owner, question.ID)
	if err != nil {
		return err
	}
	if !(questionDB["owned"].(bool)) {
		return errors.New("Unauthorized: You do not own this question and may not add answers")
	}

	// Update question file (WriteFile will clear the file before modifying)
	err = ioutil.WriteFile(dataDir+"/"+question.ID+".json", []byte(question.Data), 0644)
	return err
}

// PutAnswer changes the answer with an existing question. If fails if answer.question.owner != user.
func PutAnswer(userEmail string, answer Answer) error {
	// Confirm that such a question exists and we can add answers
	question, err := GetQuestion(userEmail, answer.Question)
	if err != nil {
		return err
	}
	if !(question["owned"].(bool)) {
		return errors.New("Unauthorized: You do not own this question and may not add answers")
	}

	// Update answer file (WriteFile will clear the file before modifying)
	err = ioutil.WriteFile(dataDir+"/"+answer.ID+".json", []byte(answer.Data), 0644)
	return err
}
