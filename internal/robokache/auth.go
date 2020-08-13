package robokache

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3" // makes database/sql point to SQLite
)

// HTTPClient implements Get()
type HTTPClient interface {
	Get(url string) (*http.Response, error)
}

var (
	// Client is used to get the authentication certificates
	Client HTTPClient
)

func init() {
	Client = &http.Client{}
}

func issuedByGoogle(claims *jwt.MapClaims) bool {
	return claims.VerifyIssuer("accounts.google.com", true) ||
		claims.VerifyIssuer("https://accounts.google.com", true)
}

// GetUser verifies authorization and sets the userEmail context
func GetUser(c *gin.Context) {
	// Get bearer (JWT) token from header
	header := c.Request.Header
	reqToken := header.Get("Authorization")
	splitToken := strings.Split(reqToken, "Bearer ")
	if len(splitToken) != 2 {
		c.AbortWithStatusJSON(401, "No Authorization header provided")
		return
	}
	reqToken = splitToken[1]

	// Verify token authenticity
	token, err := jwt.ParseWithClaims(reqToken, &jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
		resp, err := Client.Get("https://www.googleapis.com/oauth2/v1/certs")
		fatal(err)
		if resp.StatusCode != 200 {
			return nil, errors.New("Failed to contact certification authority")
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		fatal(err)
		var certs map[string]string
		json.Unmarshal(body, &certs)
		pem := certs[token.Header["kid"].(string)]
		verifyKey, err := jwt.ParseRSAPublicKeyFromPEM([]byte(pem))
		fatal(err)
		return verifyKey, nil
	})
	if err != nil {
		c.AbortWithStatusJSON(401, err.Error())
		return
	}

	// Verify claims
	claims, ok := token.Claims.(*jwt.MapClaims)
	if !ok {
		panic(errors.New("token.Claims -> *jwt.MapClaims assertion failed"))
	}
	if !token.Valid {
		c.AbortWithStatusJSON(401, "INVALID iat/nbt/exp")
		return
	}
	if !claims.VerifyAudience("297705140796-41v2ra13t7mm8uvu2dp554ov1btt80dg.apps.googleusercontent.com", true) {
		c.AbortWithStatusJSON(401, fmt.Sprintf("INVALID aud: %s", (*claims)["aud"]))
		return
	}
	if !issuedByGoogle(claims) {
		c.AbortWithStatusJSON(401, fmt.Sprintf("INVALID iss: %s", (*claims)["iss"]))
		return
	}

	// Return user email
	c.Set("userEmail", (*claims)["email"].(string))
	c.Next()
}
