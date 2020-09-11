package robokache

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type visibility int

const (
	invisible visibility = 0
	private   visibility = 1
	shareable visibility = 2
	public    visibility = 3
)

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

			// Relace the ID with a hashed ID for each document
			for i := range documents {
				documents[i].addHash()
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

			document.addHash()

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

			// Return as binary data
			c.Header("Content-Type", "application/octet-stream")
			_, err = c.Writer.Write(data)
			if err != nil {
				handleErr(c, err)
				return
			}
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
				documents[i].addHash()
			}

			// Return
			c.JSON(200, documents)
		})
		authorized.POST("/document", func(c *gin.Context) {
			// Get user
			userEmail := c.GetString("userEmail")

			// Parse the document from JSON
			var doc Document
			err := c.ShouldBindJSON(&doc)
			if err != nil {
				handleErr(c, err)
				return
			}
			// Set the document owner from the user's Google Auth
			doc.Owner = userEmail

			// Convert user given hashes to IDs
			err = doc.addID()
			if err != nil {
				handleErr(c, err)
				return
			}

			// Add document to DB
			newID, err := PostDocument(doc)
			if err != nil {
				handleErr(c, err)
				return
			}

			// Convert new ID to hash
			hashedID, err := idToHash(newID)
			if err != nil {
				handleErr(c, err)
				return
			}

			// Return hashed ID as application/text
			c.String(201, hashedID)
		})
		authorized.PUT("/document/:id", func(c *gin.Context) {
			// Get user
			userEmail := c.GetString("userEmail")

			// Get document id
			id, err := hashToID(c.Param("id"))
			if err != nil {
				handleErr(c, err)
				return
			}

			// Parse the document from JSON
			var doc Document
			err = c.ShouldBindJSON(&doc)
			if err != nil {
				handleErr(c, err)
				return
			}
			// Convert user given hashes to IDs
			err = doc.addID()
			if err != nil {
				handleErr(c, err)
				return
			}

			// Set the document owner from the user's Google Auth
			doc.Owner = userEmail
			// Set the document based on the URL param
			doc.ID = id

			// Add document to DB
			err = EditDocument(doc)
			if err != nil {
				handleErr(c, err)
				return
			}

			// Return hashed ID as application/text
			c.String(201, "ok")
		})
		authorized.PUT("/document/:id/data", func(c *gin.Context) {
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

			// Get raw data from HTTP request body
			data, err := c.GetRawData()
			if err != nil {
				handleErr(c, err)
				return
			}

			// Write data to disk
			err = SetData(id, data)
			if err != nil {
				handleErr(c, err)
				return
			}

			// Return
			c.String(201, "ok")
		})
		authorized.DELETE("/document/:id", func(c *gin.Context) {
			// Get user
			userEmail := c.GetString("userEmail")

			// Get document id
			id, err := hashToID(c.Param("id"))
			if err != nil {
				handleErr(c, err)
				return
			}

			// Build a document object that is used to query the database for
			// the delete request
			doc := Document{ID: id, Owner: userEmail}

			err = DeleteDocument(doc)
			if err != nil {
				handleErr(c, err)
				return
			}

			c.String(200, "ok")
		})
	}
	return r

}
