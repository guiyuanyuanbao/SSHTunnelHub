package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"sshtunnelhub/internal/service"
)

type TunnelHandler struct {
	svc *service.TunnelService
}

func NewTunnelHandler(svc *service.TunnelService) *TunnelHandler {
	return &TunnelHandler{svc: svc}
}

func (h *TunnelHandler) List(c *gin.Context) {
	tunnels, err := h.svc.ListTunnels()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": tunnels})
}

func (h *TunnelHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tunnel id"})
		return
	}
	t, err := h.svc.GetTunnel(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "tunnel not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": t})
}

func (h *TunnelHandler) Create(c *gin.Context) {
	var req service.CreateTunnelReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	t, err := h.svc.CreateTunnel(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": t})
}

func (h *TunnelHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tunnel id"})
		return
	}
	var req service.UpdateTunnelReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	t, err := h.svc.UpdateTunnel(uint(id), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": t})
}

func (h *TunnelHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tunnel id"})
		return
	}
	if err := h.svc.DeleteTunnel(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "tunnel deleted successfully"})
}

func (h *TunnelHandler) Start(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tunnel id"})
		return
	}
	if err := h.svc.StartTunnel(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "tunnel start signal sent"})
}

func (h *TunnelHandler) Stop(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tunnel id"})
		return
	}
	if err := h.svc.StopTunnel(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "tunnel stopped"})
}

func (h *TunnelHandler) Restart(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tunnel id"})
		return
	}
	if err := h.svc.RestartTunnel(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "tunnel restarted"})
}

func (h *TunnelHandler) Stats(c *gin.Context) {
	stats, err := h.svc.GetDashboardStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": stats})
}
