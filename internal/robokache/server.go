package robokache

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3" // makes database/sql point to SQLite
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func fatal(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

const (
	dbFile = "./data/q&a.db"
)

// Question represents a user's question
type Question struct {
	ID         string
	Owner      string
	Visibility visibility
	Data       string
}

// Answer represents the answer to a question
type Answer struct {
	ID         string
	Question   string
	Visibility visibility
	Data       string
}

type visibility int

const (
	invisible visibility = 0
	private   visibility = 1
	shareable visibility = 2
	public    visibility = 3
)

var visibilityToInt = map[string]visibility{
	"invisible": invisible,
	"private":   private,
	"shareable": shareable,
	"public":    public,
}
var intToVisibility = []string{
	"invisible",
	"private",
	"shareable",
	"public",
}

func handleErr(c *gin.Context, err error) {
	errorMsg := err.Error()
	if strings.HasPrefix(errorMsg, "Bad Request") {
		c.JSON(400, errorMsg)
	} else if strings.HasPrefix(errorMsg, "Unauthorized") {
		c.JSON(401, errorMsg)
	} else if strings.HasPrefix(errorMsg, "Not Found") {
		c.JSON(404, errorMsg)
	} else {
		c.JSON(500, errorMsg)
	}
}

// AddGUI adds the GUI endpoints
func AddGUI(r *gin.Engine) {
	// Serve HTML
	r.LoadHTMLGlob("./web/index.html")
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})

	// Serve static files (openapi.yml)
	r.Static("/docs", "./api")
}

// SetupRouter sets up the router
func SetupRouter() *gin.Engine {
	r := gin.Default()

	// Serve secured endpoints
	authorized := r.Group("/api")
	authorized.Use(GetUser)
	{
		authorized.GET("/questions", func(c *gin.Context) {
			// Get user
			userEmail := c.GetString("userEmail")

			// Get user's documents from database
			questions, err := GetQuestions(userEmail)
			if err != nil {
				handleErr(c, err)
				return
			}

			// Return
			c.JSON(200, questions)
		})
		authorized.GET("/questions/:id/answers", func(c *gin.Context) {
			// Get user
			userEmail := c.GetString("userEmail")

			// Get question id
			questionID := c.Param("id")

			// Get user's documents from database
			answers, err := GetAnswers(userEmail, questionID)
			if err != nil {
				handleErr(c, err)
				return
			}

			// Return
			c.JSON(200, answers)
		})
		authorized.GET("/questions/:id", func(c *gin.Context) {
			// Get user
			userEmail := c.GetString("userEmail")

			// Get question id
			id := c.Param("id")

			// Get user's documents from database
			question, err := GetQuestion(userEmail, id)
			if err != nil {
				handleErr(c, err)
				return
			}

			// Return
			c.JSON(200, question)
		})
		authorized.GET("/answers/:id", func(c *gin.Context) {
			// Get user
			userEmail := c.GetString("userEmail")

			// Get answer id
			id := c.Param("id")

			// Get user's documents from database
			answer, err := GetAnswer(userEmail, id)
			if err != nil {
				handleErr(c, err)
				return
			}

			// Return
			c.JSON(200, answer)
		})
		authorized.POST("/questions", func(c *gin.Context) {
			// Get user
			userEmail := c.GetString("userEmail")

			// Get request body
			data, err := c.GetRawData()
			fatal(err)

			// Get visibility query parameter
			visibility := visibilityToInt[c.DefaultQuery("visibility", "shareable")]

			// Generate uuid
			id := uuid.New().String()

			// Add question to DB
			doc := Question{id, userEmail, visibility, string(data)}
			err = PostQuestion(userEmail, doc)
			if err != nil {
				handleErr(c, err)
				return
			}

			// Return
			c.JSON(201, id)
		})
		authorized.POST("/questions/:id/answers", func(c *gin.Context) {
			// Get user
			userEmail := c.GetString("userEmail")

			// Get request body
			data, err := c.GetRawData()
			fatal(err)

			// Get visibility query parameter
			visibility := visibilityToInt[c.DefaultQuery("visibility", "shareable")]

			// Get question
			questionID := c.Param("id")

			// Generate uuid
			id := uuid.New().String()

			// Add answer to database
			doc := Answer{id, questionID, visibility, string(data)}
			err = PostAnswer(userEmail, doc)
			if err != nil {
				handleErr(c, err)
				return
			}

			// Return
			c.JSON(201, id)
		})
	}
	return r
}

// SetupDB sets up the SQLite database
func SetupDB() {
	os.RemoveAll("data/")
	os.Mkdir("data/", 0755)

	db, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	sqlStmt := `
	CREATE TABLE questions (id TEXT NOT NULL PRIMARY KEY, owner TEXT, visibility INTEGER);
	DELETE FROM questions;
	CREATE TABLE answers (id TEXT NOT NULL PRIMARY KEY, question TEXT, visibility INTEGER);
	DELETE FROM answers;
	`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		log.Printf("%q: %s\n", err, sqlStmt)
		return
	}
}
