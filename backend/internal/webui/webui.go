package webui

import (
	"embed"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

//go:embed all:dist
var embeddedFS embed.FS

// Register attaches web static file handlers to Gin router
func Register(router *gin.Engine) {
	subFS, err := fs.Sub(embeddedFS, "dist")
	hasEmbedded := err == nil
	if hasEmbedded {
		if _, err := subFS.Open("index.html"); err != nil {
			hasEmbedded = false
		}
	}

	if hasEmbedded {
		fileServer := http.FileServer(http.FS(subFS))

		router.NoRoute(func(c *gin.Context) {
			path := strings.TrimPrefix(c.Request.URL.Path, "/")
			if path == "" || path == "/" {
				c.Header("Content-Type", "text/html; charset=utf-8")
				f, err := subFS.Open("index.html")
				if err == nil {
					defer f.Close()
					if rs, ok := f.(io.ReadSeeker); ok {
						http.ServeContent(c.Writer, c.Request, "index.html", time.Now(), rs)
						return
					}
				}
			}

			// Try opening exact file in subFS
			if f, err := subFS.Open(path); err == nil {
				_ = f.Close()
				fileServer.ServeHTTP(c.Writer, c.Request)
				return
			}

			// Single Page Application (SPA) fallback to index.html
			f, err := subFS.Open("index.html")
			if err != nil {
				c.Status(http.StatusNotFound)
				return
			}
			defer f.Close()
			c.Header("Content-Type", "text/html; charset=utf-8")
			if rs, ok := f.(io.ReadSeeker); ok {
				stat, _ := f.Stat()
				modTime := time.Now()
				if stat != nil {
					modTime = stat.ModTime()
				}
				http.ServeContent(c.Writer, c.Request, "index.html", modTime, rs)
				return
			}
			c.Status(http.StatusInternalServerError)
		})
		return
	}

	// Fallback to local disk directory if not embedded
	distDir := filepath.Join(".", "frontend", "dist")
	if stat, err := os.Stat(distDir); err == nil && stat.IsDir() {
		router.Static("/assets", filepath.Join(distDir, "assets"))
		router.NoRoute(func(c *gin.Context) {
			filePath := filepath.Join(distDir, c.Request.URL.Path)
			if stat, err := os.Stat(filePath); err == nil && !stat.IsDir() {
				c.File(filePath)
				return
			}
			c.File(filepath.Join(distDir, "index.html"))
		})
	}
}
