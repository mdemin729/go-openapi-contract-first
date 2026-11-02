// Package server wires the generated router, contract validation, the
// business handlers and the published documentation into one http.Handler.
package server

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/getkin/kin-openapi/routers"
	middleware "github.com/oapi-codegen/nethttp-middleware"

	"github.com/mdemin729/openapi-go-template/oapi-codegen/api"
	"github.com/mdemin729/openapi-go-template/oapi-codegen/internal/docs"
	"github.com/mdemin729/openapi-go-template/oapi-codegen/internal/gen"
	"github.com/mdemin729/openapi-go-template/oapi-codegen/internal/handler"
)

func New(h *handler.Hello) (http.Handler, error) {
	spec, err := gen.GetSwagger()
	if err != nil {
		return nil, err
	}

	strict := gen.NewStrictHandlerWithOptions(h, nil, gen.StrictHTTPServerOptions{
		RequestErrorHandlerFunc: func(w http.ResponseWriter, _ *http.Request, err error) {
			writeProblem(w, http.StatusBadRequest, err.Error())
		},
		ResponseErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			slog.ErrorContext(r.Context(), "handler failed", "err", err)
			writeProblem(w, http.StatusInternalServerError, "internal error")
		},
	})

	// Reject anything that does not match the contract (types, enums,
	// lengths, patterns, unknown fields) before it reaches handler code.
	validate := middleware.OapiRequestValidatorWithOptions(spec, &middleware.Options{
		DoNotValidateServers:  true,
		SilenceServersWarning: true,
		ErrorHandlerWithOpts: func(_ context.Context, err error, w http.ResponseWriter, _ *http.Request, opts middleware.ErrorHandlerOpts) {
			switch {
			case errors.Is(err, routers.ErrPathNotFound):
				writeProblem(w, http.StatusNotFound, "no such route")
			case errors.Is(err, routers.ErrMethodNotAllowed):
				writeProblem(w, http.StatusMethodNotAllowed, "method not allowed")
			default:
				// kin-openapi appends the full schema after the first line; keep
				// the human-readable part only.
				msg, _, _ := strings.Cut(err.Error(), "\n")
				writeProblem(w, opts.StatusCode, msg)
			}
		},
	})

	apiMux := http.NewServeMux()
	apiHandler := validate(gen.HandlerFromMux(strict, apiMux))

	root := http.NewServeMux()
	docs.Register(root, api.Spec, spec)
	root.Handle("/", apiHandler)
	return root, nil
}

func writeProblem(w http.ResponseWriter, status int, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(gen.Problem{
		Title:  http.StatusText(status),
		Status: int32(status),
		Detail: &detail,
	})
}
