// Package docs publishes the OpenAPI contract and an interactive viewer
// next to the API itself:
//
//	GET /openapi.yaml  the contract exactly as written
//	GET /openapi.json  the same contract as JSON (for tools that want JSON)
//	GET /docs          Swagger UI
//	GET /redoc         ReDoc (read-only reference docs)
//
// Unlike oapi-codegen, ogen does not embed a parsed spec in generated code,
// so the JSON form is converted from the embedded YAML at startup.
package docs

import (
	_ "embed"
	"encoding/json"
	"net/http"

	"github.com/go-faster/yaml"
)

//go:embed swagger.html
var swaggerHTML []byte

//go:embed redoc.html
var redocHTML []byte

func Register(mux *http.ServeMux, rawYAML []byte) error {
	var doc any
	if err := yaml.Unmarshal(rawYAML, &doc); err != nil {
		return err
	}
	specJSON, err := json.Marshal(doc)
	if err != nil {
		return err
	}

	mux.HandleFunc("GET /openapi.yaml", static("application/yaml", rawYAML))
	mux.HandleFunc("GET /openapi.json", static("application/json", specJSON))
	mux.HandleFunc("GET /docs", static("text/html; charset=utf-8", swaggerHTML))
	mux.HandleFunc("GET /redoc", static("text/html; charset=utf-8", redocHTML))
	mux.Handle("GET /{$}", http.RedirectHandler("/docs", http.StatusFound))
	return nil
}

func static(contentType string, body []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = w.Write(body)
	}
}
