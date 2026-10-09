package tunnel

import (
	"fmt"
	"log"
	"sync"

	"sshtunnelhub/internal/model"
)

type TunnelManager struct {
	mu      sync.RWMutex
	workers map[uint]*TunnelWorker
}

func NewTunnelManager() *TunnelManager {
	return &TunnelManager{
		workers: make(map[uint]*TunnelWorker),
	}
}

// StartTunnel starts or restarts the worker for the specified tunnel
func (m *TunnelManager) StartTunnel(t model.Tunnel, h model.Host) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// If existing worker is running, stop it first
	if w, exists := m.workers[t.ID]; exists {
		w.Stop()
		delete(m.workers, t.ID)
	}

	worker := NewTunnelWorker(t, h)
	if err := worker.Start(); err != nil {
		return err
	}

	m.workers[t.ID] = worker
	return nil
}

// StopTunnel stops the worker for the specified tunnel
func (m *TunnelManager) StopTunnel(tunnelID uint) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	w, exists := m.workers[tunnelID]
	if !exists {
		return fmt.Errorf("tunnel %d is not active", tunnelID)
	}

	w.Stop()
	delete(m.workers, tunnelID)
	return nil
}

// RestartTunnel stops and starts the specified tunnel
func (m *TunnelManager) RestartTunnel(t model.Tunnel, h model.Host) error {
	return m.StartTunnel(t, h)
}

// GetRuntime returns the runtime state of a specific tunnel
func (m *TunnelManager) GetRuntime(tunnelID uint) model.TunnelRuntime {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if w, exists := m.workers[tunnelID]; exists {
		return w.GetRuntime()
	}
	return model.TunnelRuntime{
		Status: "stopped",
	}
}

// GetAllRuntimes returns a snapshot of all active tunnel runtimes
func (m *TunnelManager) GetAllRuntimes() map[uint]model.TunnelRuntime {
	m.mu.RLock()
	defer m.mu.RUnlock()

	res := make(map[uint]model.TunnelRuntime, len(m.workers))
	for id, w := range m.workers {
		res[id] = w.GetRuntime()
	}
	return res
}

// StopAll cleanly terminates all tunnel workers
func (m *TunnelManager) StopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, w := range m.workers {
		log.Printf("[Manager] Stopping tunnel worker %d", id)
		w.Stop()
	}
	m.workers = make(map[uint]*TunnelWorker)
}

// AutoStartAll starts all tunnels configured with auto_start = true
func (m *TunnelManager) AutoStartAll(tunnels []model.Tunnel, hostMap map[uint]model.Host) {
	for _, t := range tunnels {
		if t.AutoStart {
			h, ok := hostMap[t.HostID]
			if !ok {
				log.Printf("[Manager] Failed to auto-start tunnel [%s]: host %d not found", t.Name, t.HostID)
				continue
			}
			log.Printf("[Manager] Auto-starting tunnel [%s]", t.Name)
			if err := m.StartTunnel(t, h); err != nil {
				log.Printf("[Manager] Failed to auto-start tunnel [%s]: %v", t.Name, err)
			}
		}
	}
}
