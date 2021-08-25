package robokache

import (
	"encoding/json"
	"errors"
	"net/http"

	jwtmiddleware "github.com/auth0/go-jwt-middleware"
	"github.com/form3tech-oss/jwt-go"
	"github.com/gin-gonic/gin"
)

type Response struct {
	Message string `json:"message"`
}

type Jwks struct {
	Keys []JSONWebKeys `json:"keys"`
}

type JSONWebKeys struct {
	Kty string   `json:"kty"`
	Kid string   `json:"kid"`
	Use string   `json:"use"`
	N   string   `json:"n"`
	E   string   `json:"e"`
	X5c []string `json:"x5c"`
}

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

func validateUser(c *gin.Context) {
	jwtMiddleware := jwtmiddleware.New(jwtmiddleware.Options{
		ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
			// Verify 'aud' claim
			audience := "https://qgraph.org/api"

			// bug in form3tech-oss/jwt-go that doesn't accept list of audiences
			// need to convert to list because auth0 sends one
			// copied from https://github.com/leoromanovsky/golang-gin/pull/1/commits/eab87202b4a38471ee5744a879cd342a636d7990
			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				return token, errors.New("invalid claims type")
			}

			if audienceList, ok := claims["aud"].([]interface{}); ok {
				auds := make([]string, len(audienceList))
				for _, aud := range audienceList {
					audStr, ok := aud.(string)
					if !ok {
						return token, errors.New("invalid audience type")
					}
					auds = append(auds, audStr)
				}
				claims["aud"] = auds
			}

			checkAudience := token.Claims.(jwt.MapClaims).VerifyAudience(audience, false)
			if !checkAudience {
				return token, errors.New("invalid audience")
			}
			// Verify 'iss' claim
			iss := "https://qgraph.us.auth0.com/"
			checkIss := token.Claims.(jwt.MapClaims).VerifyIssuer(iss, false)
			if !checkIss {
				return token, errors.New("invalid issuer")
			}

			cert, err := getPemCert(token)
			if err != nil {
				panic(err.Error())
			}

			// set user email for document permissions
			userEmail := claims["https://qgraph.org/email"].(string)
			c.Set("userEmail", &userEmail)

			result, _ := jwt.ParseRSAPublicKeyFromPEM([]byte(cert))
			return result, nil
		},
		// Requests don't need a JWT
		CredentialsOptional: true,
		SigningMethod:       jwt.SigningMethodRS256,
	})
	if err := jwtMiddleware.CheckJWT(c.Writer, c.Request); err != nil {
		c.AbortWithStatus(401)
	}
	c.Next()
}

func getPemCert(token *jwt.Token) (string, error) {
	cert := ""
	resp, err := Client.Get("https://qgraph.us.auth0.com/.well-known/jwks.json")

	if err != nil {
		return cert, err
	}
	defer resp.Body.Close()

	var jwks = Jwks{}
	err = json.NewDecoder(resp.Body).Decode(&jwks)

	if err != nil {
		return cert, err
	}

	for k := range jwks.Keys {
		if token.Header["kid"] == jwks.Keys[k].Kid {
			cert = "-----BEGIN CERTIFICATE-----\n" + jwks.Keys[k].X5c[0] + "\n-----END CERTIFICATE-----"
		}
	}

	if cert == "" {
		err := errors.New("unable to find appropriate key")
		return cert, err
	}

	return cert, nil
}
