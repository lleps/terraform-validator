// This file provides functions to handle JWT authentication.
package main

import (
	verifier "github.com/okta/okta-jwt-verifier-golang"
	"net/http"
	"strings"
)

var (
	OktaClientId  = ""
	OktaIssuerUrl = ""
)

func InitOktaLoginCredentials(clientId, issuerUrl string) {
	OktaClientId = clientId
	OktaIssuerUrl = issuerUrl
}

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
