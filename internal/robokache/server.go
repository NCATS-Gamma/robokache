package robokache

import (
	"log"
	"net/http"
	"os"
	"strings"
	"fmt"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3" // makes database/sql point to SQLite
	"github.com/jmoiron/sqlx"
	"github.com/speps/go-hashids"
)

var db *sqlx.DB
var hid *hashids.HashID

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

// Question represents a user's question
type Document struct {
    // Omit in JSON to prevent exposing primary key
	ID         int            `db:"id"     json:"-"`
	// Replaces ID in JSON, not stored in db
	Hash       string         `db:"-"      json:"id"`
	// Allow parent to be null using a pointer
	Parent     *int           `db:"parent" json:"-"`
	// Replaces parent field in JSON, not stored in db
	ParentHash string         `db:"-"      json:"parent"`

	Owner      string         `db:"owner"`
	Visibility visibility     `db:"visibility"`
}

func addHash(doc *Document) error {
	// Change document ID to hash
	var err error
	doc.Hash, err = idToHash(doc.ID)
	if err != nil {
		return err
	}
	// Change parent ID to hash
	if doc.Parent != nil {
		doc.ParentHash, err = idToHash(*doc.Parent)
		if err != nil {
			return err
		}
	}
	return nil
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
		authorized.GET("/document", func(c *gin.Context) {
			// Get user
			userEmail := c.GetString("userEmail")

			// Get user's documents from database
			documents, err := GetDocuments(userEmail)
			if err != nil {
				handleErr(c, err)
				return
			}

			for i := range documents {
				addHash(&documents[i])
			}

			// Return
			c.JSON(200, documents)
		})
		authorized.GET("/document/:id", func(c *gin.Context) {
			// Get user
			userEmail := c.GetString("userEmail")

			// Get document id
			id, err := hashToID(c.Param("id"))
			if err != nil {
				handleErr(c, err)
				return
			}

			// Get document from database
			document, err := GetDocument(userEmail, id)
			if err != nil {
				handleErr(c, err)
				return
			}

			addHash(&document)

			// Return
			c.JSON(200, document)
		})
		authorized.GET("/document/:id/data", func(c *gin.Context) {
			// Get user
			userEmail := c.GetString("userEmail")

			// Get document id
			id, err := hashToID(c.Param("id"))
			if err != nil {
				handleErr(c, err)
				return
			}

			// Get document from database to ensure we have permission
			// to access this endpoint
			_, err = GetDocument(userEmail, id)
			if err != nil {
				handleErr(c, err)
				return
			}

			// Get data from disk
			data, err := GetData(id)
			if err != nil {
				handleErr(c, err)
				return
			}

			// Return
			c.JSON(200, data)
		})
		authorized.GET("/document/:id/children", func(c *gin.Context) {
			// Get user
			userEmail := c.GetString("userEmail")

			// Get document id
			id, err := hashToID(c.Param("id"))
			if err != nil {
				handleErr(c, err)
				return
			}

			// Get documents that have this as a parent
			documents, err := GetDocumentChildren(userEmail, id)
			if err != nil {
				handleErr(c, err)
				return
			}

			// Convert IDs to hashes
			for i := range documents {
				addHash(&documents[i])
			}

			// Return
			c.JSON(200, documents)
		})
		/*
		authorized.POST("/questions", func(c *gin.Context) {
			// Get user
			userEmail := c.GetString("userEmail")

			var doc Question
			err := c.ShouldBindJSON(&doc)
			if err != nil {
				handleErr(c, err)
				return
			}

			doc.Owner = userEmail

			// Add question to DB
			err = PostQuestion(userEmail, doc)
			if err != nil {
				handleErr(c, err)
				return
			}

			// Return
			c.JSON(201, id)
		})
		authorized.POST("/answers", func(c *gin.Context) {
			// Get user
			userEmail := c.GetString("userEmail")

			// Get request body
			data, err := c.GetRawData()
			fatal(err)

			// Get visibility query parameter
			visibility := visibilityToInt[c.DefaultQuery("visibility", "shareable")]

			// Get question
			questionID := c.Query("question_id")

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
		*/
	}

	// The rest of these routes are only added if we have a
	// testing variable (these allow for easy db modification
	_, testingEnv := os.LookupEnv("TESTING")
	if testingEnv {
		r.GET("/db/clear", func(c *gin.Context) {
			_, err := db.Exec(`DROP table document`)
			if err != nil {
				handleErr(c, err)
				return
			}
			SetupDB()
			c.JSON(200, nil)
		})

		r.GET("/db/loadSample", func(c *gin.Context) {
			// Load sample data
			_, err := db.Exec(
				`INSERT INTO document(id, parent, owner, visibility) VALUES
					(0, NULL, 'user1@robokache.com', 3)`)
			if err != nil {
				handleErr(c, err)
				return
			}
			_, err = db.Exec(
				`INSERT INTO document(id, parent, owner, visibility) VALUES
					(1, 0, 'user1@robokache.com', 3)`)
			if err != nil {
				handleErr(c, err)
				return
			}
			c.JSON(200, nil)
		})
	}
	return r
}

func SetupHashids() {
	hd := hashids.NewData()
	hd.Salt = "This salt is unguessable. Don't even try"
	hd.MinLength = 8

	var err error
	hid, err = hashids.NewWithData(hd)
	if err != nil {
		log.Fatal(err)
	}
}

// Convert an API hash to an integer ID (database primary key)
func hashToID(hash string) (int, error) {
	ids, err := hid.DecodeWithError(hash)
	if err != nil || len(ids) != 1 {
		return -1, fmt.Errorf("Bad Request: Invalid document ID")
	}
	return ids[0], nil
}
// Convert an API hash to an integer ID (database primary key)
func idToHash(id int) (string, error) {
	hash, err := hid.Encode([]int{id})
	if err != nil {
		return "", err
	}
	return hash, nil
}

// SetupDB sets up the SQLite database
func SetupDB() {
	db = sqlx.MustConnect("sqlite3", dbFile)

	sqlStmt := `
		CREATE TABLE IF NOT EXISTS document (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			parent INTEGER,
			owner TEXT,
			visibility INTEGER
		);`

	db.MustExec(sqlStmt)
}
