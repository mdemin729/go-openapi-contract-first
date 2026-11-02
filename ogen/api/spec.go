// Package api holds the OpenAPI contract. The YAML is embedded so the binary
// can publish the exact contract it was built from.
package api

import _ "embed"

//go:embed openapi.yaml
var Spec []byte
