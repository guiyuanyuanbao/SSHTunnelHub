package model

import (
	"time"
)

// Host represents a target SSH host credential and connection configuration
type Host struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	Name       string    `gorm:"size:100;not null" json:"name"`
	Host       string    `gorm:"size:255;not null" json:"host"`
	Port       int       `gorm:"default:22" json:"port"`
	Username   string    `gorm:"size:100;not null" json:"username"`
	AuthType   string    `gorm:"size:20;not null" json:"auth_type"` // "password" | "private_key"
	Password   string    `gorm:"type:text" json:"-"`                // Encrypted with AES-GCM
	PrivateKey string    `gorm:"type:text" json:"-"`                // Encrypted with AES-GCM
	Passphrase string    `gorm:"type:text" json:"-"`                // Encrypted with AES-GCM
	Remark     string    `gorm:"size:255" json:"remark"`
}

// HostVO is the safe view-object for frontend presentation
type HostVO struct {
	ID            uint      `json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	Name          string    `json:"name"`
	Host          string    `json:"host"`
	Port          int       `json:"port"`
	Username      string    `json:"username"`
	AuthType      string    `json:"auth_type"`
	HasPassword   bool      `json:"has_password"`
	HasPrivateKey bool      `json:"has_private_key"`
	HasPassphrase bool      `json:"has_passphrase"`
	Remark        string    `json:"remark"`
}

func (h *Host) ToVO() HostVO {
	return HostVO{
		ID:            h.ID,
		CreatedAt:     h.CreatedAt,
		UpdatedAt:     h.UpdatedAt,
		Name:          h.Name,
		Host:          h.Host,
		Port:          h.Port,
		Username:      h.Username,
		AuthType:      h.AuthType,
		HasPassword:   h.Password != "",
		HasPrivateKey: h.PrivateKey != "",
		HasPassphrase: h.Passphrase != "",
		Remark:        h.Remark,
	}
}

// Tunnel represents a persistent forward/reverse port forwarding configuration
type Tunnel struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	Name       string    `gorm:"size:100;not null" json:"name"`
	HostID     uint      `gorm:"not null;index" json:"host_id"`
	Host       Host      `gorm:"foreignKey:HostID;constraint:OnDelete:CASCADE" json:"host,omitempty"`
	Type       string    `gorm:"size:20;not null" json:"type"` // "forward" | "reverse"
	ListenHost string    `gorm:"size:100;default:'127.0.0.1'" json:"listen_host"`
	ListenPort int       `gorm:"not null" json:"listen_port"`
	TargetHost string    `gorm:"size:255;not null" json:"target_host"`
	TargetPort int       `gorm:"not null" json:"target_port"`
	AutoStart  bool      `gorm:"default:false" json:"auto_start"`
	Remark     string    `gorm:"size:255" json:"remark"`
}

// TunnelRuntime holds dynamic in-memory metrics and state
type TunnelRuntime struct {
	Status        string    `json:"status"` // "stopped" | "starting" | "running" | "reconnecting" | "error"
	LastError     string    `json:"last_error"`
	ActiveConns   int64     `json:"active_conns"`
	BytesIn       int64     `json:"bytes_in"`
	BytesOut      int64     `json:"bytes_out"`
	Uptime        int64     `json:"uptime"` // in seconds
	ConnectedAt   time.Time `json:"connected_at,omitempty"`
	Health        string    `json:"health"`         // "healthy" | "unhealthy" | "unknown"
	HealthMessage string    `json:"health_message"` // Probing diagnostic details
}

// TunnelVO combines configuration and real-time state
type TunnelVO struct {
	Tunnel
	Runtime TunnelRuntime `json:"runtime"`
}

// DashboardStats provides overview metrics for the dashboard
type DashboardStats struct {
	TotalHosts     int64 `json:"total_hosts"`
	TotalTunnels   int64 `json:"total_tunnels"`
	RunningTunnels int64 `json:"running_tunnels"`
	TotalConns     int64 `json:"total_conns"`
	TotalBytesIn   int64 `json:"total_bytes_in"`
	TotalBytesOut  int64 `json:"total_bytes_out"`
}
