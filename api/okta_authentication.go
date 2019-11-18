// This file provides functions to handle okta authentication.
package main

import (
	"encoding/json"
	verifier "github.com/okta/okta-jwt-verifier-golang"
	"net/http"
	"strings"
)

var (
	OktaClientId  = ""
	OktaIssuerUrl = ""
)

// InitOktaLoginCredentials sets the credentials used
// in LoginDetailsHandler and IsAuthenticated.
func InitOktaLoginCredentials(clientId, issuerUrl string) {
	OktaClientId = clientId
	OktaIssuerUrl = issuerUrl
}

// LoginDetailsHandler will return a json containing the okta
// client id and url (to use for oauth2 login).
func LoginDetailsHandler(_ *database, _ string, _ map[string]string) (string, int, error) {
	m := make(map[string]string)
	m["okta_client_id"] = OktaClientId
	m["okta_issuer_url"] = OktaIssuerUrl
	result, _ := json.Marshal(m)
	return string(result), http.StatusOK, nil
}

// IsAuthenticated returns true if the given request has
// an 'Authorization' header with a valid okta access token.
func IsAuthenticated(r *http.Request) bool {
	if OktaClientId == "" || OktaIssuerUrl == "" {
		panic("okta credentials not initialized. Init them using InitOktaLoginCredentials")
	}

	authHeader := r.Header.Get("Authorization")

	if authHeader == "" {
		return false
	}
	tokenParts := strings.Split(authHeader, "Bearer ")
	bearerToken := tokenParts[1]

	tv := map[string]string{}
	tv["aud"] = "api://default"
	tv["cid"] = OktaClientId
	jv := verifier.JwtVerifier{
		Issuer:           OktaIssuerUrl,
		ClaimsToValidate: tv,
	}

	_, err := jv.New().VerifyAccessToken(bearerToken)

	if err != nil {
		return false
	}

	return true
}
