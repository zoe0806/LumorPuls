package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"lumor_puls/service"
	"lumor_puls/tools"
	"lumor_puls/types"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Handler serves REST endpoints.
type Handler struct {
	store  *service.Store
	runner *service.Runner
}

func NewHandler(deps service.Deps, runner *service.Runner) *Handler {
	return &Handler{
		store:  service.NewStore(deps.DB),
		runner: runner,
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

func (h *Handler) GetTask(c *gin.Context) {
	id := c.Param("id")
	t, err := h.store.GetTask(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			tools.Fail(c, http.StatusNotFound, 404, "task not found")
			return
		}
		tools.Fail(c, http.StatusInternalServerError, 500, err.Error())
		return
	}
	tools.OK(c, t)
}

func (h *Handler) CreateTask(c *gin.Context) {
	var req types.CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		tools.Fail(c, http.StatusBadRequest, 400, err.Error())
		return
	}
	t, err := service.CreateTask(h.store, req)
	if err != nil {
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "Duplicate") {
			status = http.StatusConflict
		}
		tools.Fail(c, status, status, err.Error())
		return
	}
	tools.OK(c, t)
}

func (h *Handler) UpdateTask(c *gin.Context) {
	id := c.Param("id")
	var req types.UpdateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		tools.Fail(c, http.StatusBadRequest, 400, err.Error())
		return
	}
	if req.URL == nil && req.Interval == nil && req.Type == nil && req.Enabled == nil && req.SignalCategory == nil {
		tools.Fail(c, http.StatusBadRequest, 400, "no fields to update")
		return
	}
	t, err := service.UpdateTask(h.store, id, req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			tools.Fail(c, http.StatusNotFound, 404, err.Error())
			return
		}
		tools.Fail(c, http.StatusBadRequest, 400, err.Error())
		return
	}
	tools.OK(c, t)
}

func (h *Handler) DeleteTask(c *gin.Context) {
	id := c.Param("id")
	if err := service.DeleteTask(h.store, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			tools.Fail(c, http.StatusNotFound, 404, "task not found")
			return
		}
		tools.Fail(c, http.StatusBadRequest, 400, err.Error())
		return
	}
	tools.OK(c, gin.H{"id": id, "deleted": true})
}

func (h *Handler) ListSignals(c *gin.Context) {
	signalType := c.Query("type")
	category := c.Query("category")
	if category == "" {
		category = c.Query("signal_category")
	}
	taskID := c.Query("task_id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	list, err := h.store.ListSignals(signalType, category, taskID, limit)
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
	if err := h.runner.RunTask(c.Request.Context(), id); err != nil {
		if strings.Contains(err.Error(), "disabled") {
			tools.Fail(c, http.StatusBadRequest, 400, err.Error())
			return
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			tools.Fail(c, http.StatusNotFound, 404, "task not found")
			return
		}
		tools.Fail(c, http.StatusInternalServerError, 500, err.Error())
		return
	}
	tools.OK(c, gin.H{"taskId": id, "status": "done"})
}
