package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	jwt "github.com/dgrijalva/jwt-go"
)

// HumanitecClaims represents the claims in the Humanitec provided JWT
type HumanitecClaims struct {
	jwt.StandardClaims
	UserUUID string   `json:"user_uuid,omitempty"`
	OrgUUIDs []string `json:"organization_uuids,omitempty"`
	Username string   `json:"username,omitempty"`
	Scope    string   `json:"scope,omitempty"`
}

// claimsFromJWT parses a JWT for the claims
func claimsFromJWT(JWT string) (HumanitecClaims, error) {
	parser := jwt.Parser{nil, false, true}
	var claims HumanitecClaims
	_, _, err := (&parser).ParseUnverified(JWT, &claims)
	if err != nil {
		return HumanitecClaims{}, fmt.Errorf("extracting humanitec claims from JWT: %w", err)
	}
	return claims, nil
}

// getUser gets the user from the supplied JWT. For testing without JWT, the "From" header can be used to supply the username.
func getUser(r *http.Request) string {
	if auth := r.Header.Get("authorization"); auth != "" {
		if strings.HasPrefix(auth, "JWT") {
			claims, err := claimsFromJWT(auth[4:])
			if err == nil {
				return claims.Username
			}
			log.Printf("getUser: %v", err)
		}
	} else if userName := r.Header.Get("from"); userName != "" {
		return userName
	}
	return "UNKNOWN"
}

// writeAsJSON writes the supplied object to a response along with the status code.
func writeAsJSON(w http.ResponseWriter, statusCode int, obj interface{}) {
	jsonObj, err := json.Marshal(obj)
	if err != nil {
		log.Println(err)
		w.WriteHeader(500)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(jsonObj)
}

func (s *server) isAlive() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}
}

func (s *server) isReady() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}
}
