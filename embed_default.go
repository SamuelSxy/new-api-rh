//go:build !classic && !hai

package main

import "embed"

//go:embed web/default/dist
var buildFS embed.FS

//go:embed web/default/dist/index.html
var indexPage []byte

const buildTheme = "default"

const buildDistPath = "web/default/dist"
