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
	errorResponse := map[string]string{
		"message" : errorMsg,
	}
	if strings.HasPrefix(errorMsg, "Bad Request") {
		c.JSON(400, errorResponse)
	} else if strings.HasPrefix(errorMsg, "Unauthorized") {
		c.JSON(401, errorResponse)
	} else if strings.HasPrefix(errorMsg, "Forbidden") {
		c.JSON(403, errorResponse)
	} else if strings.HasPrefix(errorMsg, "Not Found") {
		c.JSON(404, errorResponse)
	} else {
		log.WithFields(log.Fields{"error" : err}).
		WithContext(c).
		Error("Internal Server Error")
		// Rewrite error message so that we don't expose it to the user
		errorResponse["message"] = "Internal Server Error"
		c.JSON(500, errorResponse)
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

func GetUserEmail(c *gin.Context) (*string) {
	val, ok := c.Get("userEmail")
	if ok {
		email, _ := val.(*string)
		return email
	}
	return nil
}

// SetupRouter sets up the router
func SetupRouter() *gin.Engine {
	r := gin.Default()

	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = []string{"http://lvh.me", "http://lvh.me:8080", "https://robokop.renci.org"}
	corsConfig.AllowCredentials = true
	corsConfig.AllowHeaders =  []string{"Authorization", "Content-Type", "Accept"}

	r.Use(cors.New(corsConfig))
	r.Use(AddUserToContext)

	api := r.Group("/api")

	// GET endpoints don't necessarily require auth
	{
		api.GET("/document", func(c *gin.Context) {
			userEmail := GetUserEmail(c)
			// userEmail will be nil here if the user is not logged in

			// Parse query parameters into queryParams struct
			var queryParams GetDocumentQuery
			err := c.ShouldBindQuery(&queryParams)
			if err != nil {
				handleErr(c, fmt.Errorf("Bad Request: Error parsing query parameters"))
				return
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
			userEmail := GetUserEmail(c)

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
			userEmail := GetUserEmail(c)

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
			userEmail := GetUserEmail(c)

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

	api.POST("/document/:id/children", func(c *gin.Context) {
		userEmail := GetUserEmail(c)
		if userEmail == nil {
			handleErr(c,
			fmt.Errorf("Unauthorized: You must be logged in to add a document."))
			return
		}

		// Get document id
		parentID, err := hashToID(c.Param("id"))
		if err != nil {
			handleErr(c, err)
			return
		}

		// Get the parent so we can set the default visibility
		// to the visibility of the parent
		parent, err := GetDocument(userEmail, parentID)
		if err != nil {
			handleErr(c, err)
			return
		}
		newDoc := Document{
			Parent: &parent.ID,
			Visibility: parent.Visibility,
			Owner: *userEmail,
		}

		// Add the document to the database
		newDocID, err := PostDocument(newDoc)
		if err != nil {
			handleErr(c, err)
			return
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
			return
		}

		// Return
		response := make(map[string]string)
		response["id"] = newDocIDHash
		c.JSON(http.StatusOK, response)
	})

	api.POST("/document", func(c *gin.Context) {
		userEmail := GetUserEmail(c)
		if userEmail == nil {
			handleErr(c,
			fmt.Errorf("Unauthorized: You must be logged in to add a document."))
			return
		}

		// Parse the document from JSON
		var doc Document
		err := c.ShouldBindJSON(&doc)
		if err != nil {
			handleErr(c, err)
			return
		}
		// Set the document owner from the user's Google Auth
		doc.Owner = *userEmail

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

		// Return
		response := make(map[string]string)
		response["id"] = hashedID
		c.JSON(http.StatusCreated, response)
	})
	api.PUT("/document/:id", func(c *gin.Context) {
		userEmail := GetUserEmail(c)
		if userEmail == nil {
			handleErr(c,
			fmt.Errorf("Unauthorized: You must be logged in to edit a document."))
			return
		}

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

		// Check we have permission to update this document
		existingDoc, err := GetDocumentForEditing(*userEmail, id)
		if err != nil {
			handleErr(c, err)
			return
		}

		// Set the document owner from the user's Google Auth
		doc.Owner = *userEmail
		// Set the document ID based on the URL param
		doc.ID = id

		// Add document to DB
		err = EditDocument(doc, existingDoc)
		if err != nil {
			handleErr(c, err)
			return
		}

		log.WithFields(
			log.Fields{"doc" : fmt.Sprintf("%+v", doc)}).Debug("Updating document")

			response := make(map[string]string)
			c.JSON(http.StatusOK, response)
		})
		api.PUT("/document/:id/data", func(c *gin.Context) {
			userEmail := GetUserEmail(c)
			if userEmail == nil {
				handleErr(c,
				fmt.Errorf("Unauthorized: You must be logged in to edit a document."))
				return
			}

			// Get document id
			id, err := hashToID(c.Param("id"))
			if err != nil {
				handleErr(c, err)
				return
			}

			// Check we have permission to update this document
			_, err = GetDocumentForEditing(*userEmail, id)
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

			response := make(map[string]string)
			c.JSON(http.StatusOK, response)
		})
		api.DELETE("/document/:id", func(c *gin.Context) {
			userEmail := GetUserEmail(c)
			if userEmail == nil {
				handleErr(c,
				fmt.Errorf("Unauthorized: You must be logged in to delete a document."))
				return
			}

			// Get document id
			id, err := hashToID(c.Param("id"))
			if err != nil {
				handleErr(c, err)
				return
			}

			// Check we have permission to delete this document
			existingDoc, err := GetDocumentForEditing(*userEmail, id)
			if err != nil {
				handleErr(c, err)
				return
			}

			err = DeleteDocument(existingDoc)
			if err != nil {
				handleErr(c, err)
				return
			}

			response := make(map[string]string)
			c.JSON(http.StatusOK, response)
		})
		return r

}
