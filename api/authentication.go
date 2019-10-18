// This file provides functions to handle JWT authentication.
package main

import (
	"encoding/json"
	"github.com/auth0/go-jwt-middleware"
	"github.com/dgrijalva/jwt-go"
	"net/http"
	"time"
)

var loginUsername = "" // passed through the cli
var loginPassword = ""

// authenticateHandler is the handler to bind to the login endpoint.
func authenticateHandler(_ *database, body string, _ map[string]string) (string, int, error) {
	var fields struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if json.Unmarshal([]byte(body), &fields) != nil {
		return "", http.StatusBadRequest, nil
	}

	if fields.Username != loginUsername || fields.Password != loginPassword {
		return "", http.StatusUnauthorized, nil
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"name": fields.Username,
		"exp":  time.Now().Add(time.Minute * 120).Unix(),
	})
	tokenString, err := token.SignedString([]byte(loginUsername + loginPassword))
	if err != nil {
		return "", 0, err
	}

	response := map[string]string{"token": tokenString}
	marshaled, err := json.Marshal(response)
	if err != nil {
		return "", 0, err
	}

	return string(marshaled), http.StatusOK, nil
}

// authMiddleware is used verify on endpoints that require authorization.
var authMiddleware = jwtmiddleware.New(
	jwtmiddleware.Options{
		ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
			return []byte(loginUsername + loginPassword), nil
		},
		SigningMethod: jwt.SigningMethodHS256,
	})
