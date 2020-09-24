package robokache

import (
	"net/http"
	"strings"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gin-contrib/cors"
	log "github.com/sirupsen/logrus"
)

// Initialize logging
func init() {
	if gin.Mode() == gin.DebugMode {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.WarnLevel)
	}
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

// Query parameters for Document get request
type GetDocumentQuery struct {
	HasParent *bool `form:"has_parent"`
}

// SetupRouter sets up the router
func SetupRouter() *gin.Engine {
	r := gin.Default()

	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = []string{"http://lvh.me", "http://lvh.me:8080", "https://robokop.renci.org"}
	corsConfig.AllowCredentials = true
	corsConfig.AllowHeaders =  []string{"Authorization", "Content-Type", "Accept"}

	r.Use(cors.New(corsConfig))

	api := r.Group("/api")

	// GET endpoints don't necessarily require auth
	{
		api.GET("/document", func(c *gin.Context) {
			var userEmail *string
			// Check if we have been provided authorization
			reqToken, err := GetRequestBearerToken(c)
			// If we are given an authorization token
			// get the user email associated
			if err == nil {
				userEmail, err = GetUser(reqToken)
				// If the token is invalid, always abort
				if err != nil {
					handleErr(c, err)
					return
				}
			} // userEmail will be nil here if the user is not logged in

			// Parse query parameters into queryParams struct
			var queryParams GetDocumentQuery
			err = c.ShouldBindQuery(&queryParams)
			if err != nil {
				handleErr(c, fmt.Errorf("Bad Request: Error parsing query parameters"))
			}

			// Get documents from database
			documents, err := GetDocuments(userEmail, queryParams.HasParent)
			if err != nil {
				handleErr(c, err)
				return
			}

			// Relace the ID with a hashed ID for each document
			for i := range documents {
				documents[i].addHash()
				// If userEmail == nil, none of the documents will have the
				// "owned" flag set to true
				if userEmail != nil {
					documents[i].addOwned(*userEmail)
				}
			}

			// Return
			c.JSON(http.StatusOK, documents)
		})
		api.GET("/document/:id", func(c *gin.Context) {
			var userEmail *string
			reqToken, err := GetRequestBearerToken(c)
			if err == nil {
				userEmail, err = GetUser(reqToken)
				if err != nil {
					handleErr(c, err)
					return
				}
			}

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
			if userEmail != nil {
				document.addOwned(*userEmail)
			}

			// Return
			c.JSON(http.StatusOK, document)
		})
		api.GET("/document/:id/data", func(c *gin.Context) {
			var userEmail *string
			reqToken, err := GetRequestBearerToken(c)
			if err == nil {
				userEmail, err = GetUser(reqToken)
				if err != nil {
					handleErr(c, err)
					return
				}
			}

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

			c.Header("Content-Type", "application/octet-stream")
			// Get data from disk and write it to HTTP response
			err = GetData(id, c.Writer)
			if err != nil {
				handleErr(c, err)
				return
			}
		})
		api.GET("/document/:id/children", func(c *gin.Context) {
			var userEmail *string
			reqToken, err := GetRequestBearerToken(c)
			if err == nil {
				userEmail, err = GetUser(reqToken)
				if err != nil {
					handleErr(c, err)
					return
				}
			}

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

			// Get documents that have this as a parent
			documents, err := GetDocumentChildren(userEmail, id)
			if err != nil {
				handleErr(c, err)
				return
			}

			// Convert IDs to hashes
			for i := range documents {
				documents[i].addHash()
				if userEmail != nil {
					documents[i].addOwned(*userEmail)
				}
			}

			// Return
			c.JSON(http.StatusOK, documents)
		})
	}

	// Serve secured endpoints for all routes that can modify data
	authorized := api.Group("")
	authorized.Use(AddUserToContext)
	{
		authorized.POST("/document/:id/children", func(c *gin.Context) {
			// Get user
			userEmail := c.GetString("userEmail")

			// Get document id
			parentID, err := hashToID(c.Param("id"))
			if err != nil {
				handleErr(c, err)
				return
			}

			// Get the parent so we can set the default visibility
			// to the visibility of the parent
			parent, err := GetDocument(&userEmail, parentID)
			if err != nil {
				handleErr(c, err)
			}
			newDoc := Document{
				Parent: &parent.ID,
				Visibility: parent.Visibility,
				Owner: userEmail,
			}

			// Add the document to the database
			newDocID, err := PostDocument(newDoc)
			if err != nil {
				handleErr(c, err)
			}

			// Write data to disk
			err = SetData(newDocID, c.Request.Body)
			if err != nil {
				handleErr(c, err)
				return
			}

			// Convert ID to hash
			newDocIDHash, err := idToHash(newDocID)
			if err != nil {
				handleErr(c, err)
			}

			// Return
			c.String(http.StatusOK, newDocIDHash)
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
			c.String(http.StatusCreated, hashedID)
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
			// Convert user-given parent hash and ID hash to IDs
			err = doc.addID()
			if err != nil {
				handleErr(c, err)
				return
			}

			log.WithFields(
				log.Fields{"doc" : fmt.Sprintf("%+v", doc)}).Debug("Updating document")

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
			c.String(http.StatusOK, "ok")
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
			_, err = GetDocument(&userEmail, id)
			if err != nil {
				handleErr(c, err)
				return
			}

			// Write data to disk
			err = SetData(id, c.Request.Body)
			if err != nil {
				handleErr(c, err)
				return
			}

			// Return
			c.String(http.StatusOK, "ok")
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

			c.String(http.StatusOK, "ok")
		})
	}
	return r

}
