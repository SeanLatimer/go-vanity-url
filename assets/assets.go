package assets

import "embed"

//go:embed templates/*.tmpl
var Content embed.FS
