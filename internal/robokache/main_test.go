package robokache

import (
	"bytes"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"log"

	"github.com/gin-gonic/gin"

	"github.com/dgrijalva/jwt-go"
	"github.com/stretchr/testify/assert"
)

type MockClient struct{}

func fatal(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

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

var (
	privKeyPath = "../../test/certs/test.key"
	pubKeyPath  = "../../test/certs/test.cert"
)

var (
	verifyKey    *rsa.PublicKey
	signKey      *rsa.PrivateKey
	router       *gin.Engine
	signedString string
)

func init() {
	Client = &MockClient{}

	signBytes, err := ioutil.ReadFile(privKeyPath)
	fatal(err)
	signKey, err = jwt.ParseRSAPrivateKeyFromPEM(signBytes)
	fatal(err)
	verifyBytes, err := ioutil.ReadFile(pubKeyPath)
	fatal(err)
	verifyKey, err = jwt.ParseRSAPublicKeyFromPEM(verifyBytes)
	fatal(err)

	// Grab our router
	router = SetupRouter()

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

}

func TestGetDocuments(t *testing.T) {
	clearDB(); loadSampleData()

	w := performRequest(router, "GET", "/api/document", signedString, nil)
	assert.Equal(t, http.StatusOK, w.Code)

	var response []map[string]interface{}
	err := json.Unmarshal([]byte(w.Body.String()), &response)
	assert.Nil(t, err)

	// Should be able to see my documents (4) + you public documents (2)
	assert.Equal(t, 6, len(response))
}

func TestGetDocumentsNoParent(t *testing.T) {
	clearDB(); loadSampleData()

  // Gets root documents
	w := performRequest(router, "GET", "/api/document?has_parent=false", signedString, nil)
	assert.Equal(t, http.StatusOK, w.Code)

	var response []map[string]interface{}
	err := json.Unmarshal([]byte(w.Body.String()), &response)
	assert.Nil(t, err)

	// Should be able to see my root documents (2) + you root public documents (1)
	assert.Equal(t, 3, len(response))
	for _, doc := range response {
		assert.Equal(t, "", doc["parent"])
	}
}

func TestGetDocumentsHasParent(t *testing.T) {
	clearDB(); loadSampleData()

    // Gets root documents
	w := performRequest(router, "GET", "/api/document?has_parent=true", signedString, nil)
	assert.Equal(t, http.StatusOK, w.Code)

	var response []map[string]interface{}
	err := json.Unmarshal([]byte(w.Body.String()), &response)
	assert.Nil(t, err)

	// Should be able to see my child documents (2) + you child public documents (1)
	assert.Equal(t, 3, len(response))
	for _, doc := range response {
		t.Log(doc)
		assert.NotEqual(t, "", doc["parent"])
	}
}

func TestGetMePrivateDocument(t *testing.T) {
	clearDB(); loadSampleData()

	// Can get my own private document
	hashedID, _ := idToHash(0)
	w := performRequest(router, "GET", "/api/document/" + hashedID, signedString, nil)
	assert.Equal(t, http.StatusOK, w.Code)

	// Check that the response looks ok
	var response map[string]interface{}
	err := json.Unmarshal([]byte(w.Body.String()), &response)
	assert.Nil(t, err)
	assert.Equal(t, hashedID, response["id"])
	assert.Equal(t, true, response["owned"])
}

func TestGetYouPrivateDocument(t *testing.T) {
	clearDB(); loadSampleData()

	hashedID, err := idToHash(6)
	assert.Nil(t, err)
	w := performRequest(router, "GET", "/api/document/" + hashedID, signedString, nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetYouShareableDocument(t *testing.T) {
	clearDB(); loadSampleData()

	// Can get other's shareable documents
	hashedID, err := idToHash(5)
	assert.Nil(t, err)
	w := performRequest(router, "GET", "/api/document/" + hashedID, signedString, nil)
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal([]byte(w.Body.String()), &response)
	assert.Nil(t, err)
	assert.Equal(t, hashedID, response["id"])
	assert.Equal(t, false, response["owned"])
}

func TestGetChildren(t *testing.T) {
	clearDB(); loadSampleData()

	// Can see all of my own child documents
	hashedID, _ := idToHash(1)
	w := performRequest(router, "GET", "/api/document/" + hashedID + "/children", signedString, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var response []map[string]interface{}
	err := json.Unmarshal([]byte(w.Body.String()), &response)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(response))

	// Can see only public child documents if the document is not owned by me
	hashedID, _ = idToHash(5)
	assert.Nil(t, err)
	w = performRequest(router, "GET", "/api/document/" + hashedID + "/children", signedString, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	err = json.Unmarshal([]byte(w.Body.String()), &response)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(response))
}

func TestGetPutData(t *testing.T) {
	clearDB(); loadSampleData()

	id, _ := idToHash(1)
	requestBody := "This is a string to test the data saving functionality"
	w := performRequest(router, "PUT",
			fmt.Sprintf(`/api/document/%s/data`, id),
			signedString, &requestBody)
	assert.Equal(t, http.StatusOK, w.Code)

	w = performRequest(router, "GET",
			fmt.Sprintf(`/api/document/%s/data`, id),
			signedString, &requestBody)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, requestBody, w.Body.String())
}

func TestGetNoData(t *testing.T) {
	clearDB(); loadSampleData()

	id, _ := idToHash(1)
	w := performRequest(router, "GET",
			fmt.Sprintf(`/api/document/%s/data`, id),
			signedString, nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "", w.Body.String())
}

// Test the shortcut route to add a child with data
func TestPostChildWithData(t *testing.T) {
	clearDB(); loadSampleData()

	id, _ := idToHash(1)
	requestBody := "This is a string to test the data saving functionality"
	w := performRequest(router, "POST",
			fmt.Sprintf(`/api/document/%s/children`, id),
			signedString, &requestBody)
	assert.Equal(t, http.StatusOK, w.Code)

	newDocumentIDHash := w.Body.String()
	log.Println(newDocumentIDHash)

	newDocumentID, err := hashToID(newDocumentIDHash)
	assert.Nil(t, err)
	assert.Greater(t, newDocumentID, 8)

	// Check that the document was created with the same visibility as parent
	w = performRequest(router, "GET",
			fmt.Sprintf(`/api/document/%s`, newDocumentIDHash),
			signedString, &requestBody)
	var response map[string]interface{}
	err = json.Unmarshal([]byte(w.Body.String()), &response)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Nil(t, err)
	// JSON response numbers are parsed as floats
	assert.Equal(t, float64(shareable), response["visibility"])

	// Check that the document data was saved correctly
	w = performRequest(router, "GET",
			fmt.Sprintf(`/api/document/%s/data`, newDocumentIDHash),
			signedString, &requestBody)
	assert.Equal(t, requestBody, w.Body.String())
}


func TestPostDocument(t *testing.T) {
	clearDB(); loadSampleData()
	requestBody := `{ "visibility" : 4 }`
	w := performRequest(router, "POST", "/api/document", signedString, &requestBody)
	assert.Equal(t, http.StatusCreated, w.Code)

	fmt.Println(w.Body.String())

	// Check that the ID was returned
	createdID, err := hashToID(w.Body.String())
	assert.Nil(t, err)
	assert.Greater(t, createdID, 8)
}

func TestPostDocumentWithParent(t *testing.T) {
	clearDB(); loadSampleData()

	parentID, _ := idToHash(1)
	requestBody := fmt.Sprintf(
			`{ "parent" : "%s", "visibility" : %d }`,
		parentID, shareable)
	w := performRequest(router, "POST", "/api/document", signedString, &requestBody)
	assert.Equal(t, http.StatusCreated, w.Code)

	// Check that the ID was returned
	createdID, err := hashToID(w.Body.String())
	assert.Nil(t, err)
	assert.Greater(t, createdID, 8)
}

func TestPostDocumentInvalidParent(t *testing.T) {
	clearDB(); loadSampleData()

	// Parent document is not owned by me
	parentID, _ := idToHash(4)
	requestBody := fmt.Sprintf(
			`{ "parent" : "%s", "visibility" : %d }`,
		parentID, shareable)
	w := performRequest(router, "POST", "/api/document", signedString, &requestBody)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Parent document has less visibility
	parentID, _ = idToHash(0)
	requestBody = fmt.Sprintf(
			`{ "parent" : "%s", "visibility" : %d }`,
		parentID, shareable)
	w = performRequest(router, "POST", "/api/document", signedString, &requestBody)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPutDocument(t *testing.T) {
	clearDB(); loadSampleData()

	requestBody := fmt.Sprintf(`{ "visibility" : %d }`, private)
	idHash, _ := idToHash(1)
	w := performRequest(router, "PUT",
			fmt.Sprintf(`/api/document/%s`, idHash),
			signedString, &requestBody)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPutDocumentModifyParent(t *testing.T) {
	clearDB(); loadSampleData()

	newParentID, _ := idToHash(0)
	requestBody := fmt.Sprintf(`{ "parent" : "%s", "visibility" : %d }`,
							   newParentID, private)
	id, _ := idToHash(1)
	w := performRequest(router, "PUT",
			fmt.Sprintf(`/api/document/%s`, id),
			signedString, &requestBody)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPutDocumentInvalidParent(t *testing.T) {
	clearDB(); loadSampleData()

	// Not enough visibility on parent
	newParentID, _ := idToHash(0)
	requestBody := fmt.Sprintf(`{ "parent" : "%s" }`, newParentID)
	id, _ := idToHash(1)
	w := performRequest(router, "PUT",
			fmt.Sprintf(`/api/document/%s`, id),
			signedString, &requestBody)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Not owned by current user
	newParentID, _ = idToHash(5)
	requestBody = fmt.Sprintf(`{ "parent" : "%s" }`, newParentID)
	id, _ = idToHash(1)
	w = performRequest(router, "PUT",
			fmt.Sprintf(`/api/document/%s`, id),
			signedString, &requestBody)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Doesn't exist
	newParentID, _ = idToHash(45)
	requestBody = fmt.Sprintf(`{ "parent" : "%s" }`, newParentID)
	id, _ = idToHash(1)
	w = performRequest(router, "PUT",
			fmt.Sprintf(`/api/document/%s`, id),
			signedString, &requestBody)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeleteDocument(t *testing.T) {
	clearDB(); loadSampleData()

	id, _ := idToHash(1)
	w := performRequest(router, "DELETE",
			fmt.Sprintf(`/api/document/%s`, id),
			signedString, nil)
	assert.Equal(t, http.StatusOK, w.Code)

	// Can't delete other user's document
	id, _ = idToHash(4)
	w = performRequest(router, "DELETE",
			fmt.Sprintf(`/api/document/%s`, id),
			signedString, nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestBadToken(t *testing.T) {
	w := performRequest(router, "POST", "/api/document", "abc", nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// Benchmark to test how the application handles large files

func BenchmarkGetPutLargeData(b *testing.B) {
	var testBytes []byte
	var testString string

	// File size in MB
	testFileSize := 1024 * 1024 * 100

	testBytes = make([]byte, testFileSize)
	for i := 0; i < testFileSize; i++ {
		testBytes[i] = 'a' + byte(i%26)
	}
	testString = string(testBytes)

	id, _ := idToHash(1)

	// Repeat benchmark to get accurate timing data
	for i := 0; i < b.N; i++ {
		loadSampleData()
		b.StartTimer()

		performRequest(router, "PUT",
				fmt.Sprintf(`/api/document/%s/data`, id),
				signedString, &testString)
		performRequest(router, "GET",
				fmt.Sprintf(`/api/document/%s/data`, id),
				signedString, nil)

		b.StopTimer()
		clearDB()
	}
}
