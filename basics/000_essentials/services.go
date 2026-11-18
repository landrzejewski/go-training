package main

// Modules 018-020 of the essentials: how to build, secure and store a small web service.
// Every section lives in a file of its own. This file is the map, plus the three helpers that
// every section uses to show its results.
//
//	file                    section  what you build
//	s018_http.go            018      a JSON API for users, with the standard library (net/http)
//	s018b_gin.go            018b     the same API with Gin, the most popular Go web framework
//	s018c_security.go       018c     a protected API with net/http: security headers, password
//	                                 hashing, login with JWT tokens, CSRF, CORS, rate limits, TLS
//	s018d_gin_security.go   018d     the same protection in Gin, and the traps Gin adds
//	s019_database_sql.go    019      the users stored in PostgreSQL, with database/sql
//	s020_gorm.go            020      the users stored in PostgreSQL, with GORM
//
// ONE EXAMPLE RUNS THROUGH ALL OF THEM: a users API that can create, get, list and delete
// users. 018 builds it, 018b builds it again with Gin, and 019 and 020 move its data from
// memory into a database - without changing a single handler. 018c and 018d protect a separate
// small API with a login: POST /login, GET /me and an admin-only route.
//
// HOW TO READ A SECTION: its file starts with a numbered list of steps. Each step is one
// function, in that order, and the types and functions a step introduces come right after it.
// Run a section and its output shows the same numbers, so every line points back to its step
// (run these from the repository root):
//
//	go run ./basics/000_essentials 18c      # only section 018c
//	go run ./basics/000_essentials '18*'    # 018, 018b, 018c and 018d
//	go run ./basics/000_essentials list     # every section of the essentials
//
// Sections 019 and 020 need PostgreSQL: run `docker-compose up -d` in the repository root.
// Without it they say so, and run only what needs no database.

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/gin-gonic/gin"
)

// Gin's default "debug mode" prints every route as it is registered. Release mode keeps the
// output of a section the same whether it runs alone or with the others (018b explains it).
func init() { gin.SetMode(gin.ReleaseMode) }

// step prints the heading of one step of a section, such as "  2. the users API: ...".
func step(title string) { fmt.Println("  " + title) }

// show prints one result of the current step, indented under the step's heading.
func show(format string, args ...any) { fmt.Printf("    "+format+"\n", args...) }

// callAPI sends ONE request to a handler and returns the answer as "status body", for example
// `201 {"id":1}`. It needs no server and no network: httptest.NewRequest builds the request,
// httptest.NewRecorder catches the response, and the handler runs as a plain function call.
// All the demos call their APIs this way, and services_test.go tests them the same way.
// header holds optional name/value pairs, such as "Authorization", "Bearer <token>".
func callAPI(h http.Handler, method, path, body string, header ...string) string {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for i := 0; i+1 < len(header); i += 2 {
		req.Header.Set(header[i], header[i+1])
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return strings.TrimSpace(fmt.Sprintf("%d %s", rec.Code, rec.Body.String()))
}
