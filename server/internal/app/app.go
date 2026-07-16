package app

import (
	"net/http"
	"os"
	"path/filepath"

	"chrona/server/internal/admin"
	"chrona/server/internal/auth"
	"chrona/server/internal/db"
	"chrona/server/internal/schedules"
	"github.com/gin-gonic/gin"
)

func Run() error {
	cfg := LoadConfig()
	if err := cfg.Validate(); err != nil {
		return err
	}
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		return err
	}

	store, err := db.Open(cfg.DBPath)
	if err != nil {
		return err
	}
	defer store.Close()

	authService := auth.NewService(store, cfg.Secret)
	scheduleService := schedules.NewService(store)
	adminService := admin.NewService(store, admin.Config{DBPath: cfg.DBPath, Token: cfg.AdminToken})

	router := gin.Default()
	router.GET("/healthz", func(c *gin.Context) {
		if err := store.DB.Ping(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"ok": false, "code": "database_unavailable"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true, "service": "chrona"})
	})
	router.Use(func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		origin := c.GetHeader("Origin")
		if origin == "http://localhost" || origin == "https://localhost" || origin == "capacitor://localhost" {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
			c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			c.Header("Vary", "Origin")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
	})

	auth.RegisterRoutes(router.Group("/auth"), authService)
	schedules.RegisterRoutes(router.Group("/api"), authService, scheduleService)
	admin.RegisterRoutes(router, adminService)
	registerStaticRoutes(router, cfg.DistDir)

	return router.Run(":" + cfg.Port)
}

func registerStaticRoutes(router *gin.Engine, distDir string) {
	if _, err := os.Stat(filepath.Join(distDir, "index.html")); err != nil {
		router.NoRoute(func(c *gin.Context) {
			c.String(http.StatusNotFound, "Run: cd client/web && npm run build")
		})
		return
	}

	router.Static("/assets", filepath.Join(distDir, "assets"))
	router.StaticFile("/chrona-mark.svg", filepath.Join(distDir, "chrona-mark.svg"))
	router.GET("/", func(c *gin.Context) {
		c.File(filepath.Join(distDir, "index.html"))
	})
	router.NoRoute(func(c *gin.Context) {
		c.File(filepath.Join(distDir, "index.html"))
	})
}
