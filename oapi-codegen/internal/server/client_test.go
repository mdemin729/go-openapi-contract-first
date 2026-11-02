package server_test

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/mdemin729/go-openapi-contract-first/oapi-codegen/internal/gen"
	"github.com/mdemin729/go-openapi-contract-first/oapi-codegen/internal/handler"
	"github.com/mdemin729/go-openapi-contract-first/oapi-codegen/internal/server"
)

// TestGeneratedClient is how another Go service is expected to call this
// API: through the client generated from the same contract, never through
// hand-written http.NewRequest calls. The client is typed, so a renamed
// field or a removed operation breaks the caller's build, not production.
func TestGeneratedClient(t *testing.T) {
	h, err := server.New(handler.NewHello(fixedNow))
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(h)
	defer srv.Close()

	c, err := gen.NewClientWithResponses(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	t.Run("get hello", func(t *testing.T) {
		name := "Gopher"
		res, err := c.GetHelloWithResponse(ctx, &gen.GetHelloParams{Name: &name})
		if err != nil {
			t.Fatal(err)
		}
		if res.JSON200 == nil || res.JSON200.Message != "Hello, Gopher!" {
			t.Fatalf("status=%d body=%s", res.StatusCode(), res.Body)
		}
	})

	t.Run("create greeting", func(t *testing.T) {
		lang := gen.Es
		res, err := c.CreateGreetingWithResponse(ctx, gen.CreateGreetingJSONRequestBody{Name: "Ana", Language: &lang})
		if err != nil {
			t.Fatal(err)
		}
		if res.JSON201 == nil || res.JSON201.Message != "Hola, Ana!" {
			t.Fatalf("status=%d body=%s", res.StatusCode(), res.Body)
		}
	})

	t.Run("validation error is a typed Problem", func(t *testing.T) {
		res, err := c.CreateGreetingWithResponse(ctx, gen.CreateGreetingJSONRequestBody{Name: ""})
		if err != nil {
			t.Fatal(err)
		}
		if res.ApplicationproblemJSON400 == nil {
			t.Fatalf("expected 400 problem, got status=%d body=%s", res.StatusCode(), res.Body)
		}
		p := res.ApplicationproblemJSON400
		if p.Status != 400 || p.Title != "Bad Request" || p.Instance == nil || *p.Instance != "/greetings" {
			t.Fatalf("problem = %+v", p)
		}
	})
}

// A 405 must list the supported methods (RFC 9110 section 15.5.6).
func TestMethodNotAllowedCarriesAllow(t *testing.T) {
	h, _ := server.New(handler.NewHello(fixedNow))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("DELETE", "/hello", nil))
	if rec.Code != 405 || rec.Header().Get("Allow") != "GET" {
		t.Fatalf("code=%d allow=%q body=%s", rec.Code, rec.Header().Get("Allow"), rec.Body)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Fatalf("content-type=%q", ct)
	}
}
