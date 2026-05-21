package api

import (
	"fmt"

	"lumor_puls/config"
	"lumor_puls/service"

	"github.com/gin-gonic/gin"
)

// SetupRouter registers HTTP routes.
func SetupRouter(deps service.Deps, runner *service.Runner) *gin.Engine {
	cfg := deps.Config
	if cfg.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}
	h := NewHandler(deps, runner)

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	r.GET("/health", h.Health)
	r.GET("/tasks", h.ListTasks)
	r.GET("/tasks/:id", h.GetTask)
	r.POST("/tasks", h.CreateTask)
	r.PUT("/tasks/:id", h.UpdateTask)
	r.DELETE("/tasks/:id", h.DeleteTask)
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
