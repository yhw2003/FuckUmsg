package server

import (
	"io/fs"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func RegisterStatic(router *gin.Engine, staticDir string, useEmbedded bool, logger *zap.Logger) {
	if useEmbedded {
		if registerEmbeddedStatic(router, embeddedStatic, logger) {
			logger.Info("static mode", zap.String("mode", "embedded"))
			return
		}
		logger.Warn("register embedded static failed, fallback to disk static")
	}
	registerDiskStatic(router, staticDir, logger)
}

func registerDiskStatic(router *gin.Engine, staticDir string, logger *zap.Logger) {
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
		logger.Warn("static index not found", zap.String("index_path", indexPath))
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
	logger.Info("static mode", zap.String("mode", "disk"), zap.String("static_dir", abs))
}

func registerEmbeddedStatic(router *gin.Engine, embedded fs.FS, logger *zap.Logger) bool {
	const (
		embeddedRoot  = "web_dist"
		indexPath     = embeddedRoot + "/index.html"
		assetsSubPath = embeddedRoot + "/assets"
		viteIconPath  = embeddedRoot + "/vite.svg"
	)

	if _, err := fs.Stat(embedded, indexPath); err != nil {
		logger.Warn("embedded static index not found", zap.Error(err))
		return false
	}

	assetsFS, err := fs.Sub(embedded, assetsSubPath)
	if err == nil {
		router.StaticFS("/assets", http.FS(assetsFS))
	} else {
		logger.Warn("embedded assets not found", zap.Error(err))
	}

	if _, err := fs.Stat(embedded, viteIconPath); err == nil {
		router.GET("/vite.svg", func(c *gin.Context) {
			serveEmbeddedAsset(c, embedded, viteIconPath)
		})
	}

	router.GET("/", func(c *gin.Context) {
		serveEmbeddedAsset(c, embedded, indexPath)
	})
	router.NoRoute(func(c *gin.Context) {
		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
			c.Status(http.StatusNotFound)
			return
		}
		serveEmbeddedAsset(c, embedded, indexPath)
	})

	return true
}

func serveEmbeddedAsset(c *gin.Context, embedded fs.FS, path string) {
	content, err := fs.ReadFile(embedded, path)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	ext := strings.ToLower(filepath.Ext(path))
	if ext != "" {
		if contentType := mime.TypeByExtension(ext); contentType != "" {
			c.Header("Content-Type", contentType)
		}
	}
	c.Data(http.StatusOK, c.ContentType(), content)
}
