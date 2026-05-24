// Package web holds the embedded UI templates and static assets.
package web

import "embed"

//go:embed templates static
var Files embed.FS
