// Package handler implements the business logic behind the generated
// oas.Handler. Decoding, validation, routing and encoding all live in
// ogen-generated code.
package handler

import (
	"context"
	"fmt"
	"time"

	"github.com/mdemin729/go-openapi-contract-first/ogen/internal/oas"
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

func (h *Hello) greet(lang oas.Language, name string) *oas.Greeting {
	return &oas.Greeting{
		Message:   fmt.Sprintf("%s, %s!", salutations[lang], name),
		CreatedAt: h.now().UTC(),
	}
}
