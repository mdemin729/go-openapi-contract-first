// Package server wires the ogen-generated server, the business handlers and
// the published documentation into one http.Handler.
package server

import (
	"context"
	"net/http"

	"github.com/ogen-go/ogen/ogenerrors"

	"github.com/mdemin729/go-openapi-contract-first/ogen/api"
	"github.com/mdemin729/go-openapi-contract-first/ogen/internal/docs"
	"github.com/mdemin729/go-openapi-contract-first/ogen/internal/handler"
	"github.com/mdemin729/go-openapi-contract-first/ogen/internal/oas"
	"github.com/mdemin729/go-openapi-contract-first/problem"
)

func New(h *handler.Hello) (http.Handler, error) {
	// ogen validates parameters and bodies against the schema in generated
	// code, so no separate validation middleware is needed. Errors returned
	// by handlers go through handler.NewError (ogen "convenient errors");
	// this hook only sees transport-level failures: decode, validation,
	// unsupported content type.
	apiServer, err := oas.NewServer(h,
		oas.WithErrorHandler(func(_ context.Context, w http.ResponseWriter, r *http.Request, err error) {
			problem.Write(w, r, problem.New(ogenerrors.ErrorCode(err), err.Error()))
		}),
		oas.WithNotFound(func(w http.ResponseWriter, r *http.Request) {
			problem.NotFound(w, r)
		}),
		oas.WithMethodNotAllowed(func(w http.ResponseWriter, r *http.Request, allowed string) {
			problem.MethodNotAllowed(w, r, allowed)
		}),
	)
	if err != nil {
		return nil, err
	}

	root := http.NewServeMux()
	if err := docs.Register(root, api.Spec); err != nil {
		return nil, err
	}
	root.Handle("/", apiServer)
	return root, nil
}
