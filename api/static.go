package api

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

// registerWeb serves the dashboard at GET / and /dashboard.
func registerWeb(r *gin.Engine) {
	dir := webDir()
	fmt.Println("dir", dir)
	r.GET("/", func(c *gin.Context) {
		c.File(filepath.Join(dir, "index.html"))
	})
	r.GET("/dashboard", func(c *gin.Context) {
		c.File(filepath.Join(dir, "index.html"))
	})
}

func webDir() string {
	if wd, err := os.Getwd(); err == nil {
		return filepath.Join(wd, "web")
	}
	return "web"
}
