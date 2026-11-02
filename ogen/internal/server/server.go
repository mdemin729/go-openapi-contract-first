// Package server wires the ogen-generated server, the business handlers and
// the published documentation into one http.Handler.
package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/ogen-go/ogen/ogenerrors"

	"github.com/mdemin729/go-openapi-contract-first/ogen/api"
	"github.com/mdemin729/go-openapi-contract-first/ogen/internal/docs"
	"github.com/mdemin729/go-openapi-contract-first/ogen/internal/handler"
	"github.com/mdemin729/go-openapi-contract-first/ogen/internal/oas"
)

func New(h *handler.Hello) (http.Handler, error) {
	// ogen validates parameters and bodies against the schema in generated
	// code, so no separate validation middleware is needed.
	apiServer, err := oas.NewServer(h,
		oas.WithErrorHandler(func(ctx context.Context, w http.ResponseWriter, _ *http.Request, err error) {
			code := ogenerrors.ErrorCode(err)
			if code >= http.StatusInternalServerError {
				slog.ErrorContext(ctx, "handler failed", "err", err)
				writeProblem(w, code, "internal error")
				return
			}
			writeProblem(w, code, err.Error())
		}),
		oas.WithNotFound(func(w http.ResponseWriter, _ *http.Request) {
			writeProblem(w, http.StatusNotFound, "no such route")
		}),
		oas.WithMethodNotAllowed(func(w http.ResponseWriter, _ *http.Request, allowed string) {
			w.Header().Set("Allow", allowed)
			writeProblem(w, http.StatusMethodNotAllowed, "method not allowed")
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

func writeProblem(w http.ResponseWriter, status int, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	// Must be a pointer: ogen's MarshalJSON has a pointer receiver, and
	// encoding/json cannot serialise an unset Opt* field by reflection.
	_ = json.NewEncoder(w).Encode(&oas.Problem{
		Title:  http.StatusText(status),
		Status: int32(status),
		Detail: oas.NewOptString(detail),
	})
}
