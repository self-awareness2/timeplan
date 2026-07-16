package app

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"chrona/server/internal/admin"
	"chrona/server/internal/auth"
	"chrona/server/internal/db"
	"chrona/server/internal/schedules"
)

func Run() error {
	cfg := LoadConfig()
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
	router.Use(func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
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
	router.GET("/", func(c *gin.Context) {
		c.File(filepath.Join(distDir, "index.html"))
	})
	router.NoRoute(func(c *gin.Context) {
		c.File(filepath.Join(distDir, "index.html"))
	})
}
