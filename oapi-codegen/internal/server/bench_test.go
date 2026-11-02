package server_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mdemin729/openapi-go-template/oapi-codegen/internal/handler"
	"github.com/mdemin729/openapi-go-template/oapi-codegen/internal/server"
)

// Identical in both implementations so results are comparable.
func benchmark(b *testing.B, method, target, body string) {
	h, err := server.New(handler.NewHello(fixedNow))
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for b.Loop() {
		req := httptest.NewRequest(method, target, strings.NewReader(body))
		if body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code >= http.StatusInternalServerError {
			b.Fatal(rec.Code)
		}
	}
}

func BenchmarkGetHello(b *testing.B) { benchmark(b, "GET", "/hello?name=Gopher", "") }
func BenchmarkCreateGreeting(b *testing.B) {
	benchmark(b, "POST", "/greetings", `{"name":"Ana","language":"es"}`)
}
func BenchmarkRejectInvalid(b *testing.B) {
	benchmark(b, "POST", "/greetings", `{"name":"Ana","language":"xx"}`)
}
