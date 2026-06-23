//go:build hai

package main

import "embed"

//go:embed web/hai/dist
var buildFS embed.FS

//go:embed web/hai/dist/index.html
var indexPage []byte

const buildTheme = "hai"

const buildDistPath = "web/hai/dist"
