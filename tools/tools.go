//go:build tools

// Package tools pins build-time tooling (documentation generation) as module
// dependencies so `go generate` uses a reproducible version.
package tools

import (
	_ "github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs"
)
