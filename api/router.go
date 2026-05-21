package api

import (
	"fmt"

	"lumor_puls/config"
	"lumor_puls/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// SetupRouter registers HTTP routes.
func SetupRouter(db *gorm.DB, cfg config.Config) *gin.Engine {
	if cfg.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}
	deps := service.Deps{DB: db, Config: cfg}
	h := NewHandler(deps)

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	r.GET("/health", h.Health)
	r.GET("/tasks", h.ListTasks)
	r.POST("/tasks/:id/run", h.RunTask)
	r.GET("/signals", h.ListSignals)

	return r
}

// ListenAddr returns host:port for the API server.
func ListenAddr(cfg config.Config) string {
	port := cfg.Port
	if port <= 0 {
		port = 8088
	}
	return fmt.Sprintf(":%d", port)
}
