package router

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/gin-contrib/gzip"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
)

// ThemeAssets holds the embedded frontend assets for the theme baked into this
// build. Only one frontend is embedded per binary (selected via build tags).
type ThemeAssets struct {
	BuildFS   embed.FS
	IndexPage []byte
	DistPath  string
}

func SetWebRouter(router *gin.Engine, assets ThemeAssets, canvasAssets *ThemeAssets) {
	themeFS := common.EmbedFolder(assets.BuildFS, assets.DistPath)

	router.Use(gzip.Gzip(gzip.DefaultCompression))
	router.Use(middleware.GlobalWebRateLimit())
	router.Use(middleware.Cache())
	router.Use(static.Serve("/", themeFS))

	if canvasAssets != nil && canvasAssets.DistPath != "" {
		sub, err := fs.Sub(canvasAssets.BuildFS, canvasAssets.DistPath)
		if err != nil {
			panic(err)
		}
		httpFS := http.FS(sub)
		fileServer := http.StripPrefix("/canvas", http.FileServer(httpFS))
		serveCanvasIndex := func(c *gin.Context) {
			c.Header("Cache-Control", "no-cache")
			c.Data(http.StatusOK, "text/html; charset=utf-8", canvasAssets.IndexPage)
		}
		// 仅注册通配路由；/canvas 由 gin 的 RedirectTrailingSlash 重定向到 /canvas/。
		router.GET("/canvas/*filepath", func(c *gin.Context) {
			fp := c.Param("filepath")
			if fp != "/" && fp != "" {
				if f, err := httpFS.Open(fp); err == nil {
					f.Close()
					fileServer.ServeHTTP(c.Writer, c.Request)
					return
				}
			}
			serveCanvasIndex(c)
		})
	}

	router.NoRoute(func(c *gin.Context) {
		c.Set(middleware.RouteTagKey, "web")
		if strings.HasPrefix(c.Request.RequestURI, "/v1") || strings.HasPrefix(c.Request.RequestURI, "/api") || strings.HasPrefix(c.Request.RequestURI, "/assets") {
			controller.RelayNotFound(c)
			return
		}
		if canvasAssets != nil && strings.HasPrefix(c.Request.RequestURI, "/canvas") {
			c.Header("Cache-Control", "no-cache")
			c.Data(http.StatusOK, "text/html; charset=utf-8", canvasAssets.IndexPage)
			return
		}
		c.Header("Cache-Control", "no-cache")
		c.Data(http.StatusOK, "text/html; charset=utf-8", assets.IndexPage)
	})
}
