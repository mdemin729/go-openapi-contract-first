// Package handler implements the business logic behind the generated
// oas.Handler. Decoding, validation, routing and encoding all live in
// ogen-generated code.
package handler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/mdemin729/go-openapi-contract-first/ogen/internal/oas"
	"github.com/mdemin729/go-openapi-contract-first/problem"
)

// Compile-time check that Hello satisfies the contract.
var _ oas.Handler = (*Hello)(nil)

var salutations = map[oas.Language]string{
	oas.LanguageEn: "Hello",
	oas.LanguageEs: "Hola",
	oas.LanguageDe: "Hallo",
	oas.LanguageFr: "Bonjour",
}

type Hello struct {
	now func() time.Time
}

func NewHello(now func() time.Time) *Hello {
	if now == nil {
		now = time.Now
	}
	return &Hello{now: now}
}

func (h *Hello) GetHello(_ context.Context, params oas.GetHelloParams) (oas.GetHelloRes, error) {
	return h.greet(oas.LanguageEn, params.Name.Or("World")), nil
}

func (h *Hello) CreateGreeting(_ context.Context, req *oas.GreetingRequest) (oas.CreateGreetingRes, error) {
	return h.greet(req.Language.Or(oas.LanguageEn), req.Name), nil
}

func (h *Hello) GetHealth(context.Context) (*oas.Health, error) {
	return &oas.Health{Status: oas.HealthStatusOk}, nil
}

// NewError maps an error returned by any handler method onto the contract's
// `default` response (ogen "convenient errors"). A *problem.Problem keeps
// its status and detail; anything else is logged and becomes a 500 whose
// cause is not sent to the client.
func (h *Hello) NewError(ctx context.Context, err error) *oas.ErrorStatusCode {
	var p *problem.Problem
	if !errors.As(err, &p) {
		slog.ErrorContext(ctx, "handler failed", "err", err)
		p = problem.New(http.StatusInternalServerError, "internal error")
	}
	res := oas.Problem{
		Title:  p.Title,
		Status: int32(p.Status),
		Detail: oas.NewOptString(p.Detail),
	}
	if u, err := url.Parse(p.Type); err == nil {
		res.Type = oas.NewOptURI(*u)
	}
	for _, fe := range p.Errors {
		res.Errors = append(res.Errors, oas.FieldError{Pointer: oas.NewOptString(fe.Pointer), Detail: fe.Detail})
	}
	return &oas.ErrorStatusCode{StatusCode: p.Status, Response: res}
}

func (h *Hello) greet(lang oas.Language, name string) *oas.Greeting {
	return &oas.Greeting{
		Message:   fmt.Sprintf("%s, %s!", salutations[lang], name),
		CreatedAt: h.now().UTC(),
	}
}
