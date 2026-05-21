package api

import (
	"net/http"
	"strconv"

	"lumor_puls/service"
	"lumor_puls/tools"

	"github.com/gin-gonic/gin"
)

// Handler serves REST endpoints.
type Handler struct {
	deps     service.Deps
	store    *service.Store
	pipeline *service.Pipeline
}

func NewHandler(deps service.Deps) *Handler {
	return &Handler{
		deps:     deps,
		store:    service.NewStore(deps.DB),
		pipeline: service.NewPipeline(deps),
	}
}

func (h *Handler) Health(c *gin.Context) {
	tools.OK(c, gin.H{"status": "ok", "service": "lumor_puls"})
}

func (h *Handler) ListTasks(c *gin.Context) {
	list, err := h.store.ListTasks()
	if err != nil {
		tools.Fail(c, http.StatusInternalServerError, 500, err.Error())
		return
	}
	tools.OK(c, list)
}

func (h *Handler) ListSignals(c *gin.Context) {
	signalType := c.Query("type")
	taskID := c.Query("task_id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	list, err := h.store.ListSignals(signalType, taskID, limit)
	if err != nil {
		tools.Fail(c, http.StatusInternalServerError, 500, err.Error())
		return
	}
	tools.OK(c, list)
}

func (h *Handler) RunTask(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		tools.Fail(c, http.StatusBadRequest, 400, "missing task id")
		return
	}
	if err := h.pipeline.RunTask(c.Request.Context(), id); err != nil {
		tools.Fail(c, http.StatusInternalServerError, 500, err.Error())
		return
	}
	tools.OK(c, gin.H{"taskId": id, "status": "done"})
}
