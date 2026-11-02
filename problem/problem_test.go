package problem_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mdemin729/go-openapi-contract-first/problem"
)

func TestWriteFillsDefaults(t *testing.T) {
	req := httptest.NewRequest("POST", "/greetings?x=1", nil)
	rec := httptest.NewRecorder()

	problem.Write(rec, req, problem.New(http.StatusBadRequest, "bad body").
		WithField("/language", "not one of en, es, de, fr"))

	if rec.Code != 400 {
		t.Fatalf("status = %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != problem.ContentType {
		t.Fatalf("content-type = %q", ct)
	}
	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	want := map[string]any{
		"type": "about:blank", "title": "Bad Request", "status": float64(400),
		"detail": "bad body", "instance": "/greetings",
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %v, want %v", k, got[k], v)
		}
	}
	errs, _ := got["errors"].([]any)
	if len(errs) != 1 {
		t.Fatalf("errors = %v", got["errors"])
	}
}

func TestOmitsEmptyMembers(t *testing.T) {
	b, _ := json.Marshal(problem.New(http.StatusNotFound, ""))
	if string(b) != `{"type":"about:blank","title":"Not Found","status":404}` {
		t.Fatalf("unexpected JSON: %s", b)
	}
}

func TestMethodNotAllowedSetsAllow(t *testing.T) {
	rec := httptest.NewRecorder()
	problem.MethodNotAllowed(rec, httptest.NewRequest("DELETE", "/hello", nil), "GET")
	if rec.Code != 405 || rec.Header().Get("Allow") != "GET" {
		t.Fatalf("code=%d allow=%q", rec.Code, rec.Header().Get("Allow"))
	}
}

func TestProblemIsAnError(t *testing.T) {
	var err error = problem.New(http.StatusConflict, "already shipped")
	if err.Error() != "Conflict: already shipped" {
		t.Fatal(err.Error())
	}
}
