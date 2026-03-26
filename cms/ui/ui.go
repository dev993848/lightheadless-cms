// Package ui embeds the web assets (HTML templates and static files) into the binary.
// This allows the CMS to be distributed as a single self-contained executable.
package ui

import "embed"

// Templates holds all HTML template files embedded at compile time.
//
//go:embed templates
var Templates embed.FS

// Static holds all static assets (JS, CSS) embedded at compile time.
//
//go:embed static
var Static embed.FS
