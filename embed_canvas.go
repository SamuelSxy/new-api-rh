//go:build canvas

package main

import (
	"embed"

	"github.com/QuantumNous/new-api/common"
)

//go:embed infinite-canvas/web/dist
var canvasBuildFS embed.FS

//go:embed infinite-canvas/web/dist/index.html
var canvasIndexPage []byte

const canvasDistPath = "infinite-canvas/web/dist"

func init() {
	common.CanvasEnabled = true
}
