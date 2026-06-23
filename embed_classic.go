//go:build classic

package main

import "embed"

//go:embed web/classic/dist
var buildFS embed.FS

//go:embed web/classic/dist/index.html
var indexPage []byte

const buildTheme = "classic"

const buildDistPath = "web/classic/dist"
