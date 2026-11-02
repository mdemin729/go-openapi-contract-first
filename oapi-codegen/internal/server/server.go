// Package server wires the generated router, contract validation, the
// business handlers and the published documentation into one http.Handler.
package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/routers"
	middleware "github.com/oapi-codegen/nethttp-middleware"

	"github.com/mdemin729/go-openapi-contract-first/oapi-codegen/api"
	"github.com/mdemin729/go-openapi-contract-first/oapi-codegen/internal/docs"
	"github.com/mdemin729/go-openapi-contract-first/oapi-codegen/internal/gen"
	"github.com/mdemin729/go-openapi-contract-first/oapi-codegen/internal/handler"
	"github.com/mdemin729/go-openapi-contract-first/problem"
)

func New(h *handler.Hello) (http.Handler, error) {
	spec, err := gen.GetSwagger()
	if err != nil {
		return nil, err
	}

	strict := gen.NewStrictHandlerWithOptions(h, nil, gen.StrictHTTPServerOptions{
		RequestErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			problem.BadRequest(w, r, err.Error())
		},
		ResponseErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			// A handler may return a *problem.Problem to send a specific
			// status (409, 422, ...). Anything else is an internal error
			// whose cause is logged but never sent to the client.
			var p *problem.Problem
			if errors.As(err, &p) {
				problem.Write(w, r, p)
				return
			}
			slog.ErrorContext(r.Context(), "handler failed", "err", err)
			problem.Internal(w, r)
		},
	})

	// Reject anything that does not match the contract (types, enums,
	// lengths, patterns, unknown fields) before it reaches handler code.
	validate := middleware.OapiRequestValidatorWithOptions(spec, &middleware.Options{
		DoNotValidateServers:  true,
		SilenceServersWarning: true,
		ErrorHandlerWithOpts: func(_ context.Context, err error, w http.ResponseWriter, r *http.Request, opts middleware.ErrorHandlerOpts) {
			switch {
			case errors.Is(err, routers.ErrPathNotFound):
				problem.NotFound(w, r)
			case errors.Is(err, routers.ErrMethodNotAllowed):
				problem.MethodNotAllowed(w, r, allowedMethods(spec, r.URL.Path))
			default:
				// kin-openapi appends the full schema after the first line; keep
				// the human-readable part only.
				msg, _, _ := strings.Cut(err.Error(), "\n")
				problem.Write(w, r, problem.New(opts.StatusCode, msg))
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

// allowedMethods lists the methods the contract defines for a literal path,
// so a 405 can carry the Allow header required by RFC 9110 section 15.5.6.
func allowedMethods(spec *openapi3.T, path string) string {
	item := spec.Paths.Find(path)
	if item == nil {
		return ""
	}
	var methods []string
	for m := range item.Operations() {
		methods = append(methods, m)
	}
	sort.Strings(methods)
	return strings.Join(methods, ", ")
}
