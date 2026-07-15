//go:build !canvas

package main

import "embed"

var canvasBuildFS embed.FS

var canvasIndexPage []byte

const canvasDistPath = ""
