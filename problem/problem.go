// Package problem implements RFC 9457 "Problem Details for HTTP APIs".
//
// Every error a service returns, whether it comes from contract validation,
// routing, business logic or a panic, is encoded as one JSON shape with
// Content-Type application/problem+json. Clients therefore need exactly one
// error parser, and the shape is declared once in the OpenAPI contract as the
// Problem schema. The package has no dependencies beyond the standard library
// so that every service, whatever generator it uses, can share it.
//
//	{
//	  "type": "about:blank",
//	  "title": "Bad Request",
//	  "status": 400,
//	  "detail": "request body did not match the contract",
//	  "instance": "/greetings",
//	  "errors": [ { "pointer": "/language", "detail": "value is not one of the allowed values" } ]
//	}
package problem

import (
	"encoding/json"
	"net/http"
)

// ContentType is the media type RFC 9457 assigns to problem details.
const ContentType = "application/problem+json"

// TypeBlank is the RFC 9457 default: the problem is fully described by the
// HTTP status code and title.
const TypeBlank = "about:blank"

// Problem is the RFC 9457 document. Only title and status are required.
// The errors member is an extension (RFC 9457 section 3.2) used to report
// several field-level validation failures in one response.
type Problem struct {
	Type     string       `json:"type,omitempty"`
	Title    string       `json:"title"`
	Status   int          `json:"status"`
	Detail   string       `json:"detail,omitempty"`
	Instance string       `json:"instance,omitempty"`
	Errors   []FieldError `json:"errors,omitempty"`
}

// FieldError points at one invalid field. Pointer is a JSON Pointer
// (RFC 6901) into the request body, or a parameter name prefixed by its
// location, for example "query:name".
type FieldError struct {
	Pointer string `json:"pointer,omitempty"`
	Detail  string `json:"detail"`
}

// New builds a problem for an HTTP status. The title is the standard reason
// phrase, and the type is about:blank, which RFC 9457 defines as "this
// status code says it all".
func New(status int, detail string) *Problem {
	return &Problem{
		Type:   TypeBlank,
		Title:  http.StatusText(status),
		Status: status,
		Detail: detail,
	}
}

// WithType sets a problem type URI for errors that need more than a status
// code to be understood by clients, for example "/problems/insufficient-stock".
// Keep the set of types small and document each one next to the contract.
func (p *Problem) WithType(uri, title string) *Problem {
	p.Type = uri
	if title != "" {
		p.Title = title
	}
	return p
}

// WithField appends a field-level error.
func (p *Problem) WithField(pointer, detail string) *Problem {
	p.Errors = append(p.Errors, FieldError{Pointer: pointer, Detail: detail})
	return p
}

// Error lets a *Problem travel through code that works with error values.
func (p *Problem) Error() string {
	if p.Detail != "" {
		return p.Title + ": " + p.Detail
	}
	return p.Title
}

// Write serialises the problem with the correct Content-Type and status. If
// Instance is unset it is filled with the request path, so a client always
// knows which call failed. Write never writes a body for 1xx/2xx/3xx codes
// because those are not errors; callers should not do that.
func Write(w http.ResponseWriter, r *http.Request, p *Problem) {
	if p.Instance == "" && r != nil && r.URL != nil {
		p.Instance = r.URL.Path
	}
	w.Header().Set("Content-Type", ContentType)
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(p.Status)
	_ = json.NewEncoder(w).Encode(p)
}

// Convenience writers for the cases every service hits.

func BadRequest(w http.ResponseWriter, r *http.Request, detail string) {
	Write(w, r, New(http.StatusBadRequest, detail))
}

func NotFound(w http.ResponseWriter, r *http.Request) {
	Write(w, r, New(http.StatusNotFound, "no such route"))
}

func MethodNotAllowed(w http.ResponseWriter, r *http.Request, allow string) {
	if allow != "" {
		w.Header().Set("Allow", allow)
	}
	Write(w, r, New(http.StatusMethodNotAllowed, "method not allowed"))
}

// Internal hides the cause from the client; log it before calling this.
func Internal(w http.ResponseWriter, r *http.Request) {
	Write(w, r, New(http.StatusInternalServerError, "internal error"))
}
