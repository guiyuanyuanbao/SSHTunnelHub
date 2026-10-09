package service

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
	"sshtunnelhub/internal/db"
	"sshtunnelhub/internal/model"
	"sshtunnelhub/internal/tunnel"
)

type TunnelService struct {
	manager *tunnel.TunnelManager
}

func NewTunnelService(mgr *tunnel.TunnelManager) *TunnelService {
	return &TunnelService{
		manager: mgr,
	}
}

type CreateTunnelReq struct {
	Name       string `json:"name" binding:"required"`
	HostID     uint   `json:"host_id" binding:"required"`
	Type       string `json:"type" binding:"required"` // "forward" | "reverse"
	ListenHost string `json:"listen_host"`
	ListenPort int    `json:"listen_port" binding:"required"`
	TargetHost string `json:"target_host" binding:"required"`
	TargetPort int    `json:"target_port" binding:"required"`
	AutoStart  bool   `json:"auto_start"`
	Remark     string `json:"remark"`
}

type UpdateTunnelReq struct {
	Name       string `json:"name" binding:"required"`
	HostID     uint   `json:"host_id" binding:"required"`
	Type       string `json:"type" binding:"required"`
	ListenHost string `json:"listen_host"`
	ListenPort int    `json:"listen_port" binding:"required"`
	TargetHost string `json:"target_host" binding:"required"`
	TargetPort int    `json:"target_port" binding:"required"`
	AutoStart  bool   `json:"auto_start"`
	Remark     string `json:"remark"`
}

func (s *TunnelService) ListTunnels() ([]model.TunnelVO, error) {
	var tunnels []model.Tunnel
	if err := db.DB.Preload("Host", func(db *gorm.DB) *gorm.DB {
		return db.Select("id", "name", "host", "port", "username", "auth_type")
	}).Order("id desc").Find(&tunnels).Error; err != nil {
		return nil, err
	}

	res := make([]model.TunnelVO, len(tunnels))
	for i, t := range tunnels {
		runtime := s.manager.GetRuntime(t.ID)
		res[i] = model.TunnelVO{
			Tunnel:  t,
			Runtime: runtime,
		}
	}
	return res, nil
}

func (s *TunnelService) GetTunnel(id uint) (*model.TunnelVO, error) {
	var t model.Tunnel
	if err := db.DB.Preload("Host", func(db *gorm.DB) *gorm.DB {
		return db.Select("id", "name", "host", "port", "username", "auth_type")
	}).First(&t, id).Error; err != nil {
		return nil, err
	}

	runtime := s.manager.GetRuntime(t.ID)
	return &model.TunnelVO{
		Tunnel:  t,
		Runtime: runtime,
	}, nil
}

func (s *TunnelService) CreateTunnel(req CreateTunnelReq) (*model.TunnelVO, error) {
	listenHost := strings.TrimSpace(req.ListenHost)
	if listenHost == "" {
		listenHost = "127.0.0.1"
	}

	// Validate host exists
	var host model.Host
	if err := db.DB.First(&host, req.HostID).Error; err != nil {
		return nil, fmt.Errorf("host not found: %w", err)
	}

	t := model.Tunnel{
		Name:       req.Name,
		HostID:     req.HostID,
		Type:       req.Type,
		ListenHost: listenHost,
		ListenPort: req.ListenPort,
		TargetHost: req.TargetHost,
		TargetPort: req.TargetPort,
		AutoStart:  req.AutoStart,
		Remark:     req.Remark,
	}

	if err := db.DB.Create(&t).Error; err != nil {
		return nil, err
	}

	t.Host = host
	runtime := s.manager.GetRuntime(t.ID)
	return &model.TunnelVO{
		Tunnel:  t,
		Runtime: runtime,
	}, nil
}

func (s *TunnelService) UpdateTunnel(id uint, req UpdateTunnelReq) (*model.TunnelVO, error) {
	var t model.Tunnel
	if err := db.DB.First(&t, id).Error; err != nil {
		return nil, err
	}

	listenHost := strings.TrimSpace(req.ListenHost)
	if listenHost == "" {
		listenHost = "127.0.0.1"
	}

	var host model.Host
	if err := db.DB.First(&host, req.HostID).Error; err != nil {
		return nil, fmt.Errorf("host not found: %w", err)
	}

	wasRunning := s.manager.GetRuntime(id).Status == "running" || s.manager.GetRuntime(id).Status == "reconnecting"

	t.Name = req.Name
	t.HostID = req.HostID
	t.Type = req.Type
	t.ListenHost = listenHost
	t.ListenPort = req.ListenPort
	t.TargetHost = req.TargetHost
	t.TargetPort = req.TargetPort
	t.AutoStart = req.AutoStart
	t.Remark = req.Remark

	if err := db.DB.Save(&t).Error; err != nil {
		return nil, err
	}

	t.Host = host

	// If it was running, restart with new params
	if wasRunning {
		_ = s.manager.RestartTunnel(t, host)
	}

	runtime := s.manager.GetRuntime(t.ID)
	return &model.TunnelVO{
		Tunnel:  t,
		Runtime: runtime,
	}, nil
}

func (s *TunnelService) DeleteTunnel(id uint) error {
	_ = s.manager.StopTunnel(id)
	return db.DB.Delete(&model.Tunnel{}, id).Error
}

func (s *TunnelService) StartTunnel(id uint) error {
	var t model.Tunnel
	if err := db.DB.First(&t, id).Error; err != nil {
		return err
	}

	var h model.Host
	if err := db.DB.First(&h, t.HostID).Error; err != nil {
		return fmt.Errorf("host not found: %w", err)
	}

	return s.manager.StartTunnel(t, h)
}

func (s *TunnelService) StopTunnel(id uint) error {
	return s.manager.StopTunnel(id)
}

func (s *TunnelService) RestartTunnel(id uint) error {
	var t model.Tunnel
	if err := db.DB.First(&t, id).Error; err != nil {
		return err
	}

	var h model.Host
	if err := db.DB.First(&h, t.HostID).Error; err != nil {
		return fmt.Errorf("host not found: %w", err)
	}

	return s.manager.RestartTunnel(t, h)
}

func (s *TunnelService) GetDashboardStats() (*model.DashboardStats, error) {
	var totalHosts int64
	if err := db.DB.Model(&model.Host{}).Count(&totalHosts).Error; err != nil {
		return nil, err
	}

	var totalTunnels int64
	if err := db.DB.Model(&model.Tunnel{}).Count(&totalTunnels).Error; err != nil {
		return nil, err
	}

	runtimes := s.manager.GetAllRuntimes()
	var runningTunnels int64
	var totalConns int64
	var totalBytesIn int64
	var totalBytesOut int64

	for _, r := range runtimes {
		if r.Status == "running" || r.Status == "reconnecting" {
			runningTunnels++
		}
		totalConns += r.ActiveConns
		totalBytesIn += r.BytesIn
		totalBytesOut += r.BytesOut
	}

	return &model.DashboardStats{
		TotalHosts:     totalHosts,
		TotalTunnels:   totalTunnels,
		RunningTunnels: runningTunnels,
		TotalConns:     totalConns,
		TotalBytesIn:   totalBytesIn,
		TotalBytesOut:  totalBytesOut,
	}, nil
}
