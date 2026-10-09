package tunnel

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
	"sshtunnelhub/internal/logger"
	"sshtunnelhub/internal/model"
	"sshtunnelhub/internal/sshutil"
)

type cancelForwardMsg struct {
	BindAddr string
	BindPort uint32
}

func cancelTCPIPForward(client *ssh.Client, addr string, port uint32) error {
	msg := cancelForwardMsg{BindAddr: addr, BindPort: port}
	_, _, err := client.SendRequest("cancel-tcpip-forward", true, ssh.Marshal(&msg))
	return err
}

// TunnelWorker manages the lifecycle, execution, and reconnection of a single tunnel
type TunnelWorker struct {
	tunnel  model.Tunnel
	host    model.Host
	metrics *TunnelMetrics

	mu            sync.RWMutex
	status        string // "stopped" | "starting" | "running" | "reconnecting" | "error"
	lastError     string
	health        string // "healthy" | "unhealthy" | "unknown"
	healthMessage string

	ctx    context.Context
	cancel context.CancelFunc

	activeListener  net.Listener
	activeSSHClient *ssh.Client
	listenerMu      sync.Mutex
}

func NewTunnelWorker(tunnel model.Tunnel, host model.Host) *TunnelWorker {
	return &TunnelWorker{
		tunnel:        tunnel,
		host:          host,
		metrics:       NewTunnelMetrics(),
		status:        "stopped",
		health:        "unknown",
		healthMessage: "",
	}
}

func (w *TunnelWorker) Status() string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.status
}

func (w *TunnelWorker) LastError() string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.lastError
}

func (w *TunnelWorker) setStatus(status, lastErr string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.status = status
	if lastErr != "" {
		w.lastError = lastErr
	} else if status == "running" {
		w.lastError = ""
	}
	if status == "stopped" {
		w.health = "unknown"
		w.healthMessage = ""
	}
}

func (w *TunnelWorker) setHealth(health, msg string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.health = health
	w.healthMessage = msg
}

func (w *TunnelWorker) GetRuntime() model.TunnelRuntime {
	w.mu.RLock()
	status := w.status
	lastErr := w.lastError
	health := w.health
	healthMsg := w.healthMessage
	w.mu.RUnlock()

	active, bin, bout, uptime := w.metrics.Snapshot(status, lastErr)
	return model.TunnelRuntime{
		Status:        status,
		LastError:     lastErr,
		ActiveConns:   active,
		BytesIn:       bin,
		BytesOut:      bout,
		Uptime:        uptime,
		ConnectedAt:   w.metrics.ConnectedAt,
		Health:        health,
		HealthMessage: healthMsg,
	}
}

// Start launches the tunnel worker in a background goroutine
func (w *TunnelWorker) Start() error {
	w.mu.Lock()
	if w.status == "running" || w.status == "starting" || w.status == "reconnecting" {
		w.mu.Unlock()
		return fmt.Errorf("tunnel is already running or starting")
	}
	ctx, cancel := context.WithCancel(context.Background())
	w.ctx = ctx
	w.cancel = cancel
	w.status = "starting"
	w.lastError = ""
	w.health = "unknown"
	w.mu.Unlock()

	go w.runLoop()
	return nil
}

// Stop terminates the tunnel worker and cleans up listeners and SSH connections
func (w *TunnelWorker) Stop() {
	w.mu.Lock()
	if w.cancel != nil {
		w.cancel()
	}
	w.status = "stopped"
	w.health = "unknown"
	w.mu.Unlock()

	w.closeActiveConns()
}

func (w *TunnelWorker) closeActiveConns() {
	w.listenerMu.Lock()
	defer w.listenerMu.Unlock()

	if w.activeListener != nil {
		_ = w.activeListener.Close()
		w.activeListener = nil
	}
	if w.activeSSHClient != nil {
		_ = w.activeSSHClient.Close()
		w.activeSSHClient = nil
	}
}

func (w *TunnelWorker) runLoop() {
	backoff := 2 * time.Second
	maxBackoff := 60 * time.Second

	for {
		select {
		case <-w.ctx.Done():
			w.setStatus("stopped", "")
			return
		default:
		}

		w.setStatus("starting", "")
		logger.TunnelLog(w.tunnel.ID, w.tunnel.Name, "INFO", "正在启动隧道 (类型: %s, 关联主机: %s:%d)", w.tunnel.Type, w.host.Host, w.host.Port)

		// 1. Establish SSH connection
		sshClient, err := sshutil.Dial(&w.host, 12*time.Second)
		if err != nil {
			errStr := fmt.Sprintf("SSH connection failed: %v", err)
			w.setStatus("reconnecting", errStr)
			w.setHealth("unhealthy", errStr)

			// Jittered backoff (+- 20%)
			jitter := float64(backoff) * (0.8 + 0.4*rand.Float64())
			actualSleep := time.Duration(jitter)
			logger.TunnelLog(w.tunnel.ID, w.tunnel.Name, "ERROR", "SSH 拨号失败: %v，将在 %v 后重试", err, actualSleep.Round(time.Millisecond))

			select {
			case <-w.ctx.Done():
				w.setStatus("stopped", "")
				return
			case <-time.After(actualSleep):
				backoff = min(backoff*2, maxBackoff)
				continue
			}
		}

		// Save active SSH client
		w.listenerMu.Lock()
		w.activeSSHClient = sshClient
		w.listenerMu.Unlock()

		logger.TunnelLog(w.tunnel.ID, w.tunnel.Name, "SUCCESS", "SSH 连接握手成功 (%s@%s:%d)", w.host.Username, w.host.Host, w.host.Port)

		// Reset backoff upon successful SSH handshake
		backoff = 2 * time.Second

		// 2. Start Keepalive with ServerAliveCountMax = 3 tolerance
		keepaliveCtx, cancelKeepalive := context.WithCancel(w.ctx)
		go StartSSHKeepAlive(keepaliveCtx, sshClient, 15*time.Second, 3, w.tunnel.ID, w.tunnel.Name, func(keepErr error) {
			w.closeActiveConns()
		})

		// 3. Start end-to-end health probing
		probeCtx, cancelProbe := context.WithCancel(w.ctx)
		go w.startHealthProbe(probeCtx, sshClient)

		// 4. Run forward or reverse loop
		var loopErr error
		if w.tunnel.Type == "forward" {
			loopErr = w.runForward(sshClient)
		} else if w.tunnel.Type == "reverse" {
			loopErr = w.runReverse(sshClient)
		} else {
			loopErr = fmt.Errorf("unsupported tunnel type: %s", w.tunnel.Type)
		}

		// Cleanup this round
		cancelProbe()
		cancelKeepalive()
		w.closeActiveConns()

		// Check if stop was requested
		select {
		case <-w.ctx.Done():
			w.setStatus("stopped", "")
			logger.TunnelLog(w.tunnel.ID, w.tunnel.Name, "INFO", "隧道已主动停止")
			return
		default:
		}

		if loopErr != nil {
			errStr := fmt.Sprintf("Forwarding interrupted: %v", loopErr)
			w.setStatus("reconnecting", errStr)
			w.setHealth("unhealthy", errStr)
		}

		// Jittered backoff (+- 20%)
		jitter := float64(backoff) * (0.8 + 0.4*rand.Float64())
		actualSleep := time.Duration(jitter)
		logger.TunnelLog(w.tunnel.ID, w.tunnel.Name, "WARN", "隧道数据流中断: %v，将在 %v 后重连", loopErr, actualSleep.Round(time.Millisecond))

		select {
		case <-w.ctx.Done():
			w.setStatus("stopped", "")
			logger.TunnelLog(w.tunnel.ID, w.tunnel.Name, "INFO", "隧道已安全退出")
			return
		case <-time.After(actualSleep):
			backoff = min(backoff*2, maxBackoff)
		}
	}
}

// runForward handles Local Port Forwarding: Go Hub local port -> SSH -> Target
func (w *TunnelWorker) runForward(sshClient *ssh.Client) error {
	listenAddr := fmt.Sprintf("%s:%d", w.tunnel.ListenHost, w.tunnel.ListenPort)
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return fmt.Errorf("failed to bind local address %s: %w", listenAddr, err)
	}

	w.listenerMu.Lock()
	w.activeListener = listener
	w.listenerMu.Unlock()

	w.metrics.ConnectedAt = time.Now()
	w.setStatus("running", "")
	logger.TunnelLog(w.tunnel.ID, w.tunnel.Name, "SUCCESS", "正向监听已就绪: 本地 %s ➔ 远程目标 %s:%d",
		listenAddr, w.tunnel.TargetHost, w.tunnel.TargetPort)

	for {
		localConn, err := listener.Accept()
		if err != nil {
			select {
			case <-w.ctx.Done():
				return nil
			default:
				return fmt.Errorf("local accept error: %w", err)
			}
		}

		go w.handleForwardConn(localConn, sshClient)
	}
}

func (w *TunnelWorker) handleForwardConn(localConn net.Conn, sshClient *ssh.Client) {
	defer localConn.Close()

	targetAddr := fmt.Sprintf("%s:%d", w.tunnel.TargetHost, w.tunnel.TargetPort)
	remoteConn, err := sshClient.Dial("tcp", targetAddr)
	if err != nil {
		logger.TunnelLog(w.tunnel.ID, w.tunnel.Name, "ERROR", "转发通道建立失败 (无法连接目标 %s): %v", targetAddr, err)
		return
	}
	defer remoteConn.Close()

	w.metrics.IncActiveConns()
	defer w.metrics.DecActiveConns()

	// Wrap localConn to monitor ingress/egress
	trackedLocal := WrapConn(localConn, w.metrics, true)
	Bridge(trackedLocal, remoteConn)
}

// runReverse handles Remote Port Forwarding: Remote SSH port -> Go Hub -> Local/Target
func (w *TunnelWorker) runReverse(sshClient *ssh.Client) error {
	remoteListenAddr := fmt.Sprintf("%s:%d", w.tunnel.ListenHost, w.tunnel.ListenPort)

	// Preemptively cancel any stale remote forward
	_ = cancelTCPIPForward(sshClient, w.tunnel.ListenHost, uint32(w.tunnel.ListenPort))

	listener, err := sshClient.Listen("tcp", remoteListenAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on remote SSH port %s: %w", remoteListenAddr, err)
	}

	// Defer cancel on clean shutdown
	defer func() {
		_ = cancelTCPIPForward(sshClient, w.tunnel.ListenHost, uint32(w.tunnel.ListenPort))
	}()

	w.listenerMu.Lock()
	w.activeListener = listener
	w.listenerMu.Unlock()

	w.metrics.ConnectedAt = time.Now()
	w.setStatus("running", "")
	logger.TunnelLog(w.tunnel.ID, w.tunnel.Name, "SUCCESS", "反向监听已就绪: 远程 %s ➔ 本地目标 %s:%d",
		remoteListenAddr, w.tunnel.TargetHost, w.tunnel.TargetPort)

	for {
		remoteConn, err := listener.Accept()
		if err != nil {
			select {
			case <-w.ctx.Done():
				return nil
			default:
				return fmt.Errorf("remote accept error: %w", err)
			}
		}

		go w.handleReverseConn(remoteConn)
	}
}

func (w *TunnelWorker) handleReverseConn(remoteConn net.Conn) {
	defer remoteConn.Close()

	targetAddr := fmt.Sprintf("%s:%d", w.tunnel.TargetHost, w.tunnel.TargetPort)
	localTargetConn, err := net.DialTimeout("tcp", targetAddr, 10*time.Second)
	if err != nil {
		logger.TunnelLog(w.tunnel.ID, w.tunnel.Name, "ERROR", "本地目标服务连接失败 (%s): %v", targetAddr, err)
		return
	}
	defer localTargetConn.Close()

	w.metrics.IncActiveConns()
	defer w.metrics.DecActiveConns()

	// In reverse tunnel, traffic comes from remoteConn
	trackedRemote := WrapConn(remoteConn, w.metrics, true)
	Bridge(trackedRemote, localTargetConn)
}

// startHealthProbe periodically checks end-to-end connectivity to the target service
func (w *TunnelWorker) startHealthProbe(ctx context.Context, sshClient *ssh.Client) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	probe := func() {
		targetAddr := fmt.Sprintf("%s:%d", w.tunnel.TargetHost, w.tunnel.TargetPort)
		if w.tunnel.Type == "forward" {
			// Probe target through SSH channel
			ch := make(chan error, 1)
			go func() {
				c, err := sshClient.Dial("tcp", targetAddr)
				if err == nil {
					_ = c.Close()
				}
				ch <- err
			}()

			select {
			case <-ctx.Done():
				return
			case err := <-ch:
				if err != nil {
					w.setHealth("unhealthy", fmt.Sprintf("目标服务 %s 不可达: %v", targetAddr, err))
				} else {
					w.setHealth("healthy", fmt.Sprintf("目标服务 %s 握手正常", targetAddr))
				}
			case <-time.After(5 * time.Second):
				w.setHealth("unhealthy", fmt.Sprintf("目标服务 %s 握手超时 (5s)", targetAddr))
			}
		} else {
			// Probe target locally
			conn, err := net.DialTimeout("tcp", targetAddr, 5*time.Second)
			if err != nil {
				w.setHealth("unhealthy", fmt.Sprintf("本地目标 %s 不可达: %v", targetAddr, err))
			} else {
				_ = conn.Close()
				w.setHealth("healthy", fmt.Sprintf("本地目标 %s 握手正常", targetAddr))
			}
		}
	}

	// Immediate probe after connection established
	select {
	case <-time.After(1 * time.Second):
		probe()
	case <-ctx.Done():
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			probe()
		}
	}
}

func min(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}
