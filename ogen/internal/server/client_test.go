package server_test

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/mdemin729/go-openapi-contract-first/ogen/internal/handler"
	"github.com/mdemin729/go-openapi-contract-first/ogen/internal/oas"
	"github.com/mdemin729/go-openapi-contract-first/ogen/internal/server"
)

// TestGeneratedClient is how another Go service is expected to call this
// API: through the client generated from the same contract, never through
// hand-written http.NewRequest calls.
func TestGeneratedClient(t *testing.T) {
	h, err := server.New(handler.NewHello(fixedNow))
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(h)
	defer srv.Close()

	c, err := oas.NewClient(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	t.Run("get hello", func(t *testing.T) {
		res, err := c.GetHello(ctx, oas.GetHelloParams{Name: oas.NewOptString("Gopher")})
		if err != nil {
			t.Fatal(err)
		}
		g, ok := res.(*oas.Greeting)
		if !ok || g.Message != "Hello, Gopher!" {
			t.Fatalf("res = %#v", res)
		}
	})

	t.Run("create greeting", func(t *testing.T) {
		res, err := c.CreateGreeting(ctx, &oas.GreetingRequest{Name: "Ana", Language: oas.NewOptLanguage(oas.LanguageEs)})
		if err != nil {
			t.Fatal(err)
		}
		g, ok := res.(*oas.Greeting)
		if !ok || g.Message != "Hola, Ana!" {
			t.Fatalf("res = %#v", res)
		}
	})

	t.Run("validation error is a typed Problem", func(t *testing.T) {
		// The 400 declared in the contract is one of the CreateGreetingRes
		// sum-type members, so the caller switches on the type instead of
		// parsing a status code.
		res, err := c.CreateGreeting(ctx, &oas.GreetingRequest{Name: ""})
		if err != nil {
			var esc *oas.ErrorStatusCode
			if errors.As(err, &esc) {
				t.Fatalf("unexpected default response %d: %+v", esc.StatusCode, esc.Response)
			}
			t.Fatal(err)
		}
		p, ok := res.(*oas.Problem)
		if !ok || p.Status != 400 || p.Title != "Bad Request" || p.Instance.Or("") != "/greetings" {
			t.Fatalf("res = %#v", res)
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
