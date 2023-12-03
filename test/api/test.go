//go:build itest

// Package test contains integration tests for the package internal/api. Each
// Go file except this one and main_test.go corresponds to a endpoint.
package api

import (
	"database/sql"
	"net/http"
)

// db is the database connection pool used during integration testing.
// It is set in main_test.go/TestMain.
var db *sql.DB

const (
	// jwtKey is the JWT key used for signing and validating JWTs during
	// integration testing.
	jwtKey = "itest-jwt-key-0123456789qwerty"

	// JWTs to be used for testing purposes
	jwtTeam1Admin = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ0ZWFtMUF" +
		"kbWluIn0.hdiH2HHc8QFT9VbkpfXKubtV5-mMIT__tmMmYZHMVeA"
	jwtTeam1Member = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ0ZWFtMU" +
		"1lbWJlciJ9.uJbS6vSFZzH1Nfbbto3ega9COg9dMuo63iYHmMYJ6bc"
	jwtTeam2Admin = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ0ZWFtMkF" +
		"kbWluIn0.vjQ93bx9-LK7SZEmhuzISf-Mcf_-A2bZ6VbLn27THPY"
	jwtTeam2Member = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ0ZWFtMk" +
		"1lbWJlciJ9.g4FxHf1WupHGzzlvvi-8my1shFhpNuaWZKfJSV-Edxs"
	jwtTeam3Admin = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ0ZWFtM0F" +
		"kbWluIn0.QHFI2okGYug7GNwMwwpwYyTtZkx53I-R-uNjlodCwTU"
	jwtTeam4Admin = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ0ZWFtNEF" +
		"kbWluIn0.BxguaMUSynY33m3CB3jsV-l4ZC0bTE8_8XJJ8VFNo3o"
)

// addCookieAuth is used in various test cases to authenticate the request
// being sent to a handler.
func addCookieAuth(token string) func(*http.Request) {
	return func(req *http.Request) {
		req.AddCookie(&http.Cookie{Name: "auth-token", Value: token})
	}
}