// This file provides functions to handle JWT authentication.
package main

import (
	verifier "github.com/okta/okta-jwt-verifier-golang"
	"net/http"
	"strings"
)

func IsAuthenticated(r *http.Request) bool {
	authHeader := r.Header.Get("Authorization")

	if authHeader == "" {
		return false
	}
	tokenParts := strings.Split(authHeader, "Bearer ")
	bearerToken := tokenParts[1]

	tv := map[string]string{}
	tv["aud"] = "api://default"
	tv["cid"] = "0oa1ut3cyjyVyMJdP357"
	jv := verifier.JwtVerifier{
		Issuer:           "https://linuxtest.okta.com/oauth2/default",
		ClaimsToValidate: tv,
	}

	_, err := jv.New().VerifyAccessToken(bearerToken)

	if err != nil {
		return false
	}

	return true
}
