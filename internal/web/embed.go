package web

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

//go:embed all:dist
var dist embed.FS

// RegisterRoutes serves the embedded frontend as a single-page application.
func RegisterRoutes(router *gin.Engine) {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		return
	}

	fileServer := http.FileServer(http.FS(sub))

	router.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path

		// API routes are registered before NoRoute; never serve HTML for them.
		if strings.HasPrefix(path, "/api/") {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}

		// Try static file first.
		if path != "/" && !strings.HasSuffix(path, "/") {
			if f, err := sub.Open(strings.TrimPrefix(path, "/")); err == nil {
				f.Close()
				fileServer.ServeHTTP(c.Writer, c.Request)
				return
			}
		}

		// SPA fallback.
		data, err := fs.ReadFile(sub, "index.html")
		if err != nil {
			c.String(http.StatusNotFound, "frontend not found")
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", data)
	})
}

// HasAssets checks whether embedded frontend static assets (index.html) are available.
func HasAssets() bool {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		return false
	}
	f, err := sub.Open("index.html")
	if err != nil {
		return false
	}
	_ = f.Close()
	return true
}
