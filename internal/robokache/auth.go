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

// Gets bearer (JWT) token from header
func GetRequestBearerToken(c *gin.Context) (string, error) {
	header := c.Request.Header
	reqToken := header.Get("Authorization")
	splitToken := strings.Split(reqToken, "Bearer ")
	if len(splitToken) != 2 {
		return "", fmt.Errorf("Invalid Authorization header")
	}
	reqToken = splitToken[1]
	return reqToken, nil
}

// Verifies authorization and sets the userEmail context
func GetUser(reqToken string) (*string, error) {
	// Verify token authenticity
	token, err := jwt.ParseWithClaims(reqToken, &jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
		resp, err := Client.Get("https://www.googleapis.com/oauth2/v1/certs")
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != 200 {
			return nil, errors.New("Failed to contact certification authority")
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		var certs map[string]string
		json.Unmarshal(body, &certs)
		pem := certs[token.Header["kid"].(string)]
		verifyKey, err := jwt.ParseRSAPublicKeyFromPEM([]byte(pem))
		if err != nil {
			return nil, err
		}
		return verifyKey, nil
	})
	if err != nil {
		return nil, err
	}

	// Verify claims
	claims, ok := token.Claims.(*jwt.MapClaims)
	if !ok {
		return nil, errors.New("token.Claims -> *jwt.MapClaims assertion failed")
	}
	if !token.Valid {
		return nil, errors.New("INVALID iat/nbt/exp")
	}
	if !claims.VerifyAudience("297705140796-41v2ra13t7mm8uvu2dp554ov1btt80dg.apps.googleusercontent.com", true) {
		return nil, fmt.Errorf("INVALID aud: %s", (*claims)["aud"])
	}
	if !issuedByGoogle(claims) {
		return nil, fmt.Errorf("INVALID iss: %s", (*claims)["iss"])
	}

	userEmail := (*claims)["email"].(string)

	return &userEmail, nil
}

// Runs GetUser and GetRequestBearerToken and puts the results
// in the Gin context.
// Aborts with Unauthorized if there are any issues.
func AddUserToContext(c *gin.Context) {
	reqToken, err := GetRequestBearerToken(c)
	if err != nil {
    handleErr(c, fmt.Errorf("Unauthorized: %v", err))
    c.Abort()
		return
	}
	userEmail, err := GetUser(reqToken)
	if err != nil {
    handleErr(c, fmt.Errorf("Unauthorized: %v", err))
    c.Abort()
		return
	}

	// Set user email on context and continue middleware chain
	c.Set("userEmail", *userEmail)
	c.Next()
}
