package server_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers/gorillamux"

	"github.com/mdemin729/openapi-go-template/oapi-codegen/internal/gen"
	"github.com/mdemin729/openapi-go-template/oapi-codegen/internal/handler"
	"github.com/mdemin729/openapi-go-template/oapi-codegen/internal/server"
)

var fixedNow = func() time.Time { return time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC) }

func TestAPI(t *testing.T) {
	h, err := server.New(handler.NewHello(fixedNow))
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(h)
	defer srv.Close()

	tests := []struct {
		name, method, path, body string
		wantStatus               int
		wantBody                 string
	}{
		{"default greeting", "GET", "/hello", "", 200, `"message":"Hello, World!"`},
		{"named greeting", "GET", "/hello?name=Gopher", "", 200, `"message":"Hello, Gopher!"`},
		{"name violates pattern", "GET", "/hello?name=%3Cscript%3E", "", 400, `"status":400`},
		{"name too long", "GET", "/hello?name=" + strings.Repeat("a", 51), "", 400, `"status":400`},
		{"post greeting", "POST", "/greetings", `{"name":"Ana","language":"es"}`, 201, `"message":"Hola, Ana!"`},
		{"unknown enum", "POST", "/greetings", `{"name":"Ana","language":"xx"}`, 400, `"status":400`},
		{"unknown field", "POST", "/greetings", `{"name":"Ana","extra":1}`, 400, `"status":400`},
		{"missing required", "POST", "/greetings", `{}`, 400, `"status":400`},
		{"health", "GET", "/healthz", "", 200, `"status":"ok"`},
		{"unknown route", "GET", "/nope", "", 404, `"status":404`},
		{"wrong method", "DELETE", "/hello", "", 405, `"status":405`},
		{"spec yaml", "GET", "/openapi.yaml", "", 200, "openapi: 3.0.3"},
		{"spec json", "GET", "/openapi.json", "", 200, `"openapi":"3.0.3"`},
		{"swagger ui", "GET", "/docs", "", 200, "swagger-ui"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest(tc.method, srv.URL+tc.path, strings.NewReader(tc.body))
			if tc.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			if resp.StatusCode != tc.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", resp.StatusCode, tc.wantStatus, body)
			}
			if !strings.Contains(string(body), tc.wantBody) {
				t.Fatalf("body %q does not contain %q", body, tc.wantBody)
			}
		})
	}
}

// TestResponsesMatchContract validates real responses against the spec, so
// the implementation cannot drift from what openapi.yaml promises.
func TestResponsesMatchContract(t *testing.T) {
	spec, err := gen.GetSwagger()
	if err != nil {
		t.Fatal(err)
	}
	spec.Servers = nil
	router, err := gorillamux.NewRouter(spec)
	if err != nil {
		t.Fatal(err)
	}
	h, _ := server.New(handler.NewHello(fixedNow))

	cases := []struct{ method, target, body string }{
		{"GET", "/hello?name=Gopher", ""},
		{"GET", "/hello?name=!!", ""},
		{"POST", "/greetings", `{"name":"Ana","language":"fr"}`},
		{"POST", "/greetings", `{"name":""}`},
		{"GET", "/healthz", ""},
	}
	for _, c := range cases {
		req := httptest.NewRequest(c.method, c.target, strings.NewReader(c.body))
		if c.body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		route, params, err := router.FindRoute(req)
		if err != nil {
			t.Fatal(err)
		}
		in := &openapi3filter.ResponseValidationInput{
			RequestValidationInput: &openapi3filter.RequestValidationInput{Request: req, PathParams: params, Route: route},
			Status:                 rec.Code,
			Header:                 rec.Header(),
			Body:                   io.NopCloser(bytes.NewReader(rec.Body.Bytes())),
		}
		if err := openapi3filter.ValidateResponse(context.Background(), in); err != nil {
			t.Errorf("%s %s -> %d does not match contract: %v", c.method, c.target, rec.Code, err)
		}
	}
}
