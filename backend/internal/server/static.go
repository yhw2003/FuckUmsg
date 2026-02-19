package server

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

func RegisterStatic(router *gin.Engine, staticDir string) {
	if strings.TrimSpace(staticDir) == "" {
		return
	}
	abs := staticDir
	if !filepath.IsAbs(staticDir) {
		if cwd, err := os.Getwd(); err == nil {
			abs = filepath.Join(cwd, staticDir)
		}
	}

	indexPath := filepath.Join(abs, "index.html")
	if _, err := os.Stat(indexPath); err != nil {
		log.Printf("static index not found: %s", indexPath)
		return
	}

	router.StaticFS("/assets", http.Dir(filepath.Join(abs, "assets")))
	viteIcon := filepath.Join(abs, "vite.svg")
	if _, err := os.Stat(viteIcon); err == nil {
		router.StaticFile("/vite.svg", viteIcon)
	}
	router.GET("/", func(c *gin.Context) {
		c.File(indexPath)
	})
	router.NoRoute(func(c *gin.Context) {
		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
			c.Status(http.StatusNotFound)
			return
		}
		c.File(indexPath)
	})
}
