// Package handler implements the business logic behind the generated
// gen.StrictServerInterface. It only deals with typed request/response
// objects; HTTP decoding, routing and encoding live in generated code.
package handler

import (
	"context"
	"fmt"
	"time"

	"github.com/mdemin729/go-openapi-contract-first/oapi-codegen/internal/gen"
)

// Compile-time check that Hello satisfies the contract. If an operation is
// added to openapi.yaml and code is regenerated, the build fails here until
// it is implemented.
var _ gen.StrictServerInterface = (*Hello)(nil)

var salutations = map[gen.Language]string{
	gen.En: "Hello",
	gen.Es: "Hola",
	gen.De: "Hallo",
	gen.Fr: "Bonjour",
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

func (h *Hello) GetHello(_ context.Context, req gen.GetHelloRequestObject) (gen.GetHelloResponseObject, error) {
	name := "World"
	if req.Params.Name != nil {
		name = *req.Params.Name
	}
	return gen.GetHello200JSONResponse(h.greet(gen.En, name)), nil
}

func (h *Hello) CreateGreeting(_ context.Context, req gen.CreateGreetingRequestObject) (gen.CreateGreetingResponseObject, error) {
	lang := gen.En
	if req.Body.Language != nil {
		lang = *req.Body.Language
	}
	return gen.CreateGreeting201JSONResponse(h.greet(lang, req.Body.Name)), nil
}

func (h *Hello) GetHealth(context.Context, gen.GetHealthRequestObject) (gen.GetHealthResponseObject, error) {
	return gen.GetHealth200JSONResponse{Status: gen.Ok}, nil
}

func (h *Hello) greet(lang gen.Language, name string) gen.Greeting {
	return gen.Greeting{
		Message:   fmt.Sprintf("%s, %s!", salutations[lang], name),
		CreatedAt: h.now().UTC(),
	}
}
