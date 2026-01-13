// Package api provides HTTP handlers for sync schedule management.
package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/Veritas-Calculus/vc-terraform-registry/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

// SyncHandler handles sync schedule related HTTP requests.
type SyncHandler struct {
	db          *gorm.DB
	storagePath string
}

// NewSyncHandler creates a new SyncHandler.
func NewSyncHandler(db *gorm.DB, storagePath string) *SyncHandler {
	return &SyncHandler{
		db:          db,
		storagePath: storagePath,
	}
}

// CreateScheduleRequest represents the request to create a sync schedule.
type CreateScheduleRequest struct {
	Namespace string `json:"namespace" binding:"required"`
	Name      string `json:"name" binding:"required"`
	CronExpr  string `json:"cron_expr" binding:"required"`
	Enabled   bool   `json:"enabled"`
	SyncOS    string `json:"sync_os"`
	SyncArch  string `json:"sync_arch"`
}

// UpdateScheduleRequest represents the request to update a sync schedule.
type UpdateScheduleRequest struct {
	CronExpr *string `json:"cron_expr"`
	Enabled  *bool   `json:"enabled"`
	SyncOS   *string `json:"sync_os"`
	SyncArch *string `json:"sync_arch"`
}

// ListSchedules returns all sync schedules.
func (h *SyncHandler) ListSchedules(c *gin.Context) {
	var schedules []models.SyncSchedule
	if err := h.db.Find(&schedules).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list schedules"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"schedules": schedules})
}

// CreateSchedule creates a new sync schedule.
func (h *SyncHandler) CreateSchedule(c *gin.Context) {
	var req CreateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	schedule, err := parser.Parse(req.CronExpr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid cron expression: " + err.Error()})
		return
	}

	var existing models.SyncSchedule
	if err := h.db.Where("namespace = ? AND name = ?", req.Namespace, req.Name).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Schedule already exists for this provider"})
		return
	}

	syncOS := req.SyncOS
	if syncOS == "" {
		syncOS = "all"
	}
	syncArch := req.SyncArch
	if syncArch == "" {
		syncArch = "all"
	}

	nextRun := schedule.Next(time.Now())
	newSchedule := models.SyncSchedule{
		Namespace: req.Namespace,
		Name:      req.Name,
		CronExpr:  req.CronExpr,
		Enabled:   req.Enabled,
		SyncOS:    syncOS,
		SyncArch:  syncArch,
		NextRunAt: &nextRun,
	}

	if err := h.db.Create(&newSchedule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create schedule"})
		return
	}

	c.JSON(http.StatusCreated, newSchedule)
}

// UpdateSchedule updates an existing sync schedule.
func (h *SyncHandler) UpdateSchedule(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid schedule ID"})
		return
	}

	var schedule models.SyncSchedule
	if err := h.db.First(&schedule, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Schedule not found"})
		return
	}

	var req UpdateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.CronExpr != nil {
		parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
		cronSchedule, err := parser.Parse(*req.CronExpr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid cron expression: " + err.Error()})
			return
		}
		schedule.CronExpr = *req.CronExpr
		nextRun := cronSchedule.Next(time.Now())
		schedule.NextRunAt = &nextRun
	}

	if req.Enabled != nil {
		schedule.Enabled = *req.Enabled
	}
	if req.SyncOS != nil {
		schedule.SyncOS = *req.SyncOS
	}
	if req.SyncArch != nil {
		schedule.SyncArch = *req.SyncArch
	}

	if err := h.db.Save(&schedule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update schedule"})
		return
	}

	c.JSON(http.StatusOK, schedule)
}

// DeleteSchedule deletes a sync schedule.
func (h *SyncHandler) DeleteSchedule(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid schedule ID"})
		return
	}

	if err := h.db.Delete(&models.SyncSchedule{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete schedule"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Schedule deleted"})
}

// RunScheduleNow triggers an immediate sync for a schedule.
func (h *SyncHandler) RunScheduleNow(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid schedule ID"})
		return
	}

	var schedule models.SyncSchedule
	if err := h.db.First(&schedule, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Schedule not found"})
		return
	}

	now := time.Now()
	schedule.LastRunAt = &now
	schedule.LastStatus = "running"
	schedule.LastError = ""
	h.db.Save(&schedule)

	c.JSON(http.StatusOK, gin.H{
		"message":  "Sync triggered",
		"schedule": schedule,
	})
}
