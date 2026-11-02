// Package docs publishes the OpenAPI contract and an interactive viewer
// next to the API itself:
//
//	GET /openapi.yaml  the contract exactly as written
//	GET /openapi.json  the same contract as JSON (for tools that want JSON)
//	GET /docs          Swagger UI
//	GET /redoc         ReDoc (read-only reference docs)
package docs

import (
	_ "embed"
	"net/http"

	"github.com/getkin/kin-openapi/openapi3"
)

//go:embed swagger.html
var swaggerHTML []byte

//go:embed redoc.html
var redocHTML []byte

func Register(mux *http.ServeMux, rawYAML []byte, spec *openapi3.T) {
	specJSON, err := spec.MarshalJSON()
	if err != nil {
		panic(err) // the spec was already loaded once; this cannot fail in practice
	}

	mux.HandleFunc("GET /openapi.yaml", static("application/yaml", rawYAML))
	mux.HandleFunc("GET /openapi.json", static("application/json", specJSON))
	mux.HandleFunc("GET /docs", static("text/html; charset=utf-8", swaggerHTML))
	mux.HandleFunc("GET /redoc", static("text/html; charset=utf-8", redocHTML))
	mux.Handle("GET /{$}", http.RedirectHandler("/docs", http.StatusFound))
}

func static(contentType string, body []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = w.Write(body)
	}
}
