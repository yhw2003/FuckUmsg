package server

import "embed"

//go:embed web_dist
var embeddedStatic embed.FS
