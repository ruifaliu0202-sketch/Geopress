package web

import (
	"embed"
	"errors"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"

	"github.com/gin-gonic/gin"
)

//go:embed dist/*
var assets embed.FS

func Register(router *gin.Engine) {
	staticFS, err := fs.Sub(assets, "dist")
	if err != nil {
		panic(err)
	}

	fileServer := http.FileServer(http.FS(staticFS))
	router.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/") || c.Request.URL.Path == "/api" {
			c.JSON(http.StatusNotFound, gin.H{"error": "api route not found"})
			return
		}

		requestPath := strings.TrimPrefix(path.Clean(c.Request.URL.Path), "/")
		if requestPath == "." || requestPath == "" {
			serveEmbeddedFile(c, staticFS, "index.html")
			return
		}

		if info, err := fs.Stat(staticFS, requestPath); err == nil && !info.IsDir() {
			fileServer.ServeHTTP(c.Writer, c.Request)
			return
		}

		serveEmbeddedFile(c, staticFS, "index.html")
	})
}

func serveEmbeddedFile(c *gin.Context, staticFS fs.FS, name string) {
	data, err := fs.ReadFile(staticFS, name)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			c.String(http.StatusNotFound, "frontend asset not found")
			return
		}
		c.String(http.StatusInternalServerError, "read frontend asset failed")
		return
	}

	contentType := mime.TypeByExtension(path.Ext(name))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	c.Data(http.StatusOK, contentType, data)
}
