package main

import (
	"bytes"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/NCATS-Gamma/robokache/internal/robokache"
	"github.com/dgrijalva/jwt-go"
	"github.com/stretchr/testify/assert"
)

func fatal(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

type MockClient struct{}

func (m *MockClient) Get(url string) (*http.Response, error) {
	if url != "https://www.googleapis.com/oauth2/v1/certs" {
		return &http.Response{
			StatusCode: 404,
			Body:       ioutil.NopCloser(bytes.NewReader([]byte("Page not found"))),
		}, nil
	}
	cert, err := ioutil.ReadFile(pubKeyPath)
	fatal(err)
	json := fmt.Sprintf(`{"default":"%s"}`, strings.ReplaceAll(string(cert), "\n", `\n`))
	r := ioutil.NopCloser(bytes.NewReader([]byte(json)))
	return &http.Response{
		StatusCode: 200,
		Body:       r,
	}, nil
}

func performRequest(r http.Handler, method, path string, jwt string, body *string) *httptest.ResponseRecorder {
	var req *http.Request
	if body == nil {
		req, _ = http.NewRequest(method, path, nil)
	} else {
		req, _ = http.NewRequest(method, path, strings.NewReader(*body))
	}
	req.Header.Add("Authorization", "Bearer "+jwt)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

const (
	privKeyPath = "certs/test.key"
	pubKeyPath  = "certs/test.cert"
)

var (
	verifyKey    *rsa.PublicKey
	signKey      *rsa.PrivateKey
	router       *gin.Engine
	signedString string
)

// func fatal(err error) {
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// }

func init() {
	robokache.Client = &MockClient{}

	signBytes, err := ioutil.ReadFile(privKeyPath)
	fatal(err)
	signKey, err = jwt.ParseRSAPrivateKeyFromPEM(signBytes)
	fatal(err)
	verifyBytes, err := ioutil.ReadFile(pubKeyPath)
	fatal(err)
	verifyKey, err = jwt.ParseRSAPublicKeyFromPEM(verifyBytes)
	fatal(err)

	// Grab our router
	router = robokache.SetupRouter()

	type MyCustomClaims struct {
		Email string `json:"email,omitempty"`
		jwt.StandardClaims
	}

	// Create the Claims
	claims := MyCustomClaims{
		"me@robokache.com",
		jwt.StandardClaims{
			Issuer:   "accounts.google.com",
			Audience: "297705140796-41v2ra13t7mm8uvu2dp554ov1btt80dg.apps.googleusercontent.com",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = "default"
	signedString, _ = token.SignedString(signKey)

	robokache.SetupDB()
	question := robokache.Question{"0", "me@robokache.com", 1, "{\n    \"hello\": \"world\"\n}"}
	robokache.PostQuestion("me@robokache.com", question)
	question = robokache.Question{"1", "you@robokache.com", 3, ""}
	robokache.PostQuestion("you@robokache.com", question)
	question = robokache.Question{"2", "you@robokache.com", 1, ""}
	robokache.PostQuestion("you@robokache.com", question)
	answer := robokache.Answer{"0a", "0", 1, "42"}
	robokache.PostAnswer("me@robokache.com", answer)
	answer = robokache.Answer{"1a", "1", 1, ""}
	robokache.PostAnswer("you@robokache.com", answer)
}

func TestGetQuestions(t *testing.T) {
	w := performRequest(router, "GET", "/api/questions", signedString, nil)
	if !assert.Equal(t, http.StatusOK, w.Code) {
		return
	}
	var response []map[string]string
	err2 := json.Unmarshal([]byte(w.Body.String()), &response)
	assert.Nil(t, err2)
}

func TestGetQuestion(t *testing.T) {
	w := performRequest(router, "GET", "/api/questions/0", signedString, nil)
	if !assert.Equal(t, http.StatusOK, w.Code) {
		return
	}
	var response map[string]string
	err2 := json.Unmarshal([]byte(w.Body.String()), &response)
	assert.Nil(t, err2)
}

func TestPostQuestion(t *testing.T) {
	requestBody := "test question"
	w := performRequest(router, "POST", "/api/questions", signedString, &requestBody)
	if !assert.Equal(t, http.StatusCreated, w.Code) {
		return
	}
	var response string
	err2 := json.Unmarshal([]byte(w.Body.String()), &response)
	assert.Nil(t, err2)
}

func TestGetAnswers(t *testing.T) {
	w := performRequest(router, "GET", "/api/questions/0/answers", signedString, nil)
	if !assert.Equal(t, http.StatusOK, w.Code) {
		return
	}
	var response []map[string]string
	err2 := json.Unmarshal([]byte(w.Body.String()), &response)
	assert.Nil(t, err2)
}

func TestGetAnswer(t *testing.T) {
	w := performRequest(router, "GET", "/api/answers/0a", signedString, nil)
	if !assert.Equal(t, http.StatusOK, w.Code) {
		return
	}
	var response map[string]string
	err2 := json.Unmarshal([]byte(w.Body.String()), &response)
	assert.Nil(t, err2)
}

func TestPostAnswer(t *testing.T) {
	requestBody := "test answer"
	w := performRequest(router, "POST", "/api/questions/0/answers", signedString, &requestBody)
	if !assert.Equal(t, http.StatusCreated, w.Code) {
		return
	}
	var response string
	err2 := json.Unmarshal([]byte(w.Body.String()), &response)
	assert.Nil(t, err2)
}

func TestBadToken(t *testing.T) {
	w := performRequest(router, "POST", "/api/questions", "abc", nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestNoSuchQuestion(t *testing.T) {
	requestBody := "test answer"
	w := performRequest(router, "POST", "/api/questions/404/answers", signedString, &requestBody)
	assert.Equal(t, http.StatusNotFound, w.Code)
}
