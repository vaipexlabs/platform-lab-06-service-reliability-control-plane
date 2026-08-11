// Package contracts exposes the canonical service reliability schema without
// requiring a runtime filesystem path.
package contracts

import "embed"

// Schemas contains the versioned service reliability schemas.
//
//go:embed *.schema.json
var Schemas embed.FS
