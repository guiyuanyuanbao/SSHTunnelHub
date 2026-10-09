package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"sshtunnelhub/internal/service"
)

type HostHandler struct {
	svc *service.HostService
}

func NewHostHandler(svc *service.HostService) *HostHandler {
	return &HostHandler{svc: svc}
}

func (h *HostHandler) List(c *gin.Context) {
	hosts, err := h.svc.ListHosts()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": hosts})
}

func (h *HostHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid host id"})
		return
	}
	host, err := h.svc.GetHost(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "host not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": host})
}

func (h *HostHandler) Create(c *gin.Context) {
	var req service.CreateHostReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	host, err := h.svc.CreateHost(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": host})
}

func (h *HostHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid host id"})
		return
	}
	var req service.UpdateHostReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	host, err := h.svc.UpdateHost(uint(id), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": host})
}

func (h *HostHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid host id"})
		return
	}
	if err := h.svc.DeleteHost(uint(id)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "host deleted successfully"})
}

func (h *HostHandler) TestSavedHost(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid host id"})
		return
	}
	latencyMs, banner, err := h.svc.TestSavedHost(uint(id))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   err.Error(),
			"latency": latencyMs,
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"latency": latencyMs,
		"banner":  banner,
	})
}

func (h *HostHandler) TestRawHost(c *gin.Context) {
	var req service.TestHostReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	latencyMs, banner, err := h.svc.TestRawHost(req)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   err.Error(),
			"latency": latencyMs,
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"latency": latencyMs,
		"banner":  banner,
	})
}
