package solutions

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
)

// call sends one request through h with httptest - no network, no server - and prints
// "status body". header holds optional name/value pairs. The 18b, 18c and 18d tests use it too.
func call(h http.Handler, method, target, body string, header ...string) {
	rec := recordBody(h, method, target, body, header...)
	fmt.Println(strings.TrimSpace(fmt.Sprint(rec.Code, " ", rec.Body.String())))
}

// record sends one request without a body and returns the recorder, to look at the headers.
func record(h http.Handler, method, target string, header ...string) *httptest.ResponseRecorder {
	return recordBody(h, method, target, "", header...)
}

func recordBody(h http.Handler, method, target, body string, header ...string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	for i := 0; i+1 < len(header); i += 2 {
		req.Header.Set(header[i], header[i+1])
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func ExampleNewGreetMux() {
	mux := NewGreetMux()
	call(mux, "GET", "/hello/Ann", "")
	call(mux, "GET", "/hello/Ann?lang=pl", "")
	call(mux, "GET", "/hello/Ann?lang=de", "")
	call(mux, "POST", "/hello/Ann", "")
	call(mux, "GET", "/hello/", "")
	// Output:
	// 200 {"greeting":"hello, Ann"}
	// 200 {"greeting":"cześć, Ann"}
	// 200 {"greeting":"hallo, Ann"}
	// 405 Method Not Allowed
	// 404 404 page not found
}

func ExampleNewTodoMux() {
	mux := NewTodoMux(NewTodoStore())
	call(mux, "GET", "/todos", "")
	call(mux, "POST", "/todos", `{"title":"  buy milk "}`)
	call(mux, "POST", "/todos", `{"title":"   "}`)
	call(mux, "POST", "/todos", `{"title":`)
	call(mux, "POST", "/todos", `{"title":"`+strings.Repeat("x", 2000)+`"}`)
	call(mux, "GET", "/todos/1", "")
	call(mux, "GET", "/todos/7", "")
	call(mux, "GET", "/todos/abc", "")
	call(mux, "GET", "/todos", "")
	// Output:
	// 200 []
	// 201 {"id":1,"title":"buy milk","done":false}
	// 400 {"error":"title is required"}
	// 400 {"error":"invalid JSON"}
	// 413 {"error":"body too large"}
	// 200 {"id":1,"title":"buy milk","done":false}
	// 404 {"error":"todo not found"}
	// 400 {"error":"invalid id"}
	// 200 [{"id":1,"title":"buy milk","done":false}]
}

func ExampleChain() {
	store := NewTodoStore()
	store.Add("write tests")
	// LogRequests is listed first, so it is the outermost: it also logs the 401s.
	h := Chain(NewTodoMux(store), LogRequests(os.Stdout), RequireAPIKey("secret"))

	call(h, "GET", "/todos", "")
	call(h, "GET", "/todos", "", "X-API-Key", "wrong")
	call(h, "GET", "/todos", "", "X-API-Key", "secret")
	// Output:
	// GET /todos 401
	// 401 {"error":"unauthorized"}
	// GET /todos 401
	// 401 {"error":"unauthorized"}
	// GET /todos 200
	// 200 [{"id":1,"title":"write tests","done":false}]
}
