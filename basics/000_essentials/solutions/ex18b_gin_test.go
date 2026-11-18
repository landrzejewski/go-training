package solutions

import "github.com/gin-gonic/gin"

// The examples use call from ex18_http_test.go, so copy both files.

func init() { gin.SetMode(gin.TestMode) } // no debug listing of the routes

func ExampleNewGreetRouter() {
	r := NewGreetRouter()
	call(r, "GET", "/hello/Ann", "")
	call(r, "GET", "/hello/Ann?lang=pl", "")
	call(r, "POST", "/hello/Ann", "")
	// Output:
	// 200 {"greeting":"hello, Ann"}
	// 200 {"greeting":"cześć, Ann"}
	// 405 405 method not allowed
}

func ExampleNewSignupRouter() {
	r := NewSignupRouter()
	call(r, "POST", "/signup", `{"email":"ann@example.com","password":"long enough","age":30}`)
	call(r, "POST", "/signup", `{"email":"ann@example.com","password":"short","age":30}`)
	call(r, "POST", "/signup", `{"email":"not-an-email","password":"long enough"}`)
	call(r, "POST", "/signup", `{}`)
	call(r, "POST", "/signup", `{"email":`)
	// Output:
	// 201 {"email":"ann@example.com"}
	// 400 {"invalid":["password"]}
	// 400 {"invalid":["age","email"]}
	// 400 {"invalid":["age","email","password"]}
	// 400 {"error":"invalid JSON"}
}

func ExampleNewTodoRouter() {
	store := NewTodoStore()
	r := NewTodoRouter(store, "secret")
	call(r, "POST", "/api/admin/todos", `{"title":"buy milk"}`)
	call(r, "POST", "/api/admin/todos", `{"title":"buy milk"}`, "X-API-Key", "secret")
	call(r, "POST", "/api/admin/todos", `{}`, "X-API-Key", "secret")
	call(r, "GET", "/api/todos", "")
	call(r, "GET", "/api/todos/1", "")
	call(r, "GET", "/api/todos/2", "")
	// Output:
	// 401 {"error":"unauthorized"}
	// 201 {"id":1,"title":"buy milk","done":false}
	// 400 {"error":"title is required"}
	// 200 [{"id":1,"title":"buy milk","done":false}]
	// 200 {"id":1,"title":"buy milk","done":false}
	// 404 {"error":"todo not found"}
}
