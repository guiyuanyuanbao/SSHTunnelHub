package tunnel

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/crypto/ssh"
	"sshtunnelhub/internal/logger"
)

// StartSSHKeepAlive periodically sends keepalive requests to the remote SSH server
// with ServerAliveCountMax tolerance (continuous 3 failures to trigger disconnect)
func StartSSHKeepAlive(
	ctx context.Context,
	client *ssh.Client,
	interval time.Duration,
	maxFails int,
	tunnelID uint,
	tunnelName string,
	onFail func(err error),
) {
	if interval <= 0 {
		interval = 15 * time.Second
	}
	if maxFails <= 0 {
		maxFails = 3
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	consecutiveFails := 0

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			resCh := make(chan error, 1)
			go func() {
				_, _, err := client.SendRequest("keepalive@openssh.com", true, nil)
				resCh <- err
			}()

			var probeErr error
			select {
			case <-ctx.Done():
				return
			case err := <-resCh:
				probeErr = err
			case <-time.After(8 * time.Second):
				probeErr = fmt.Errorf("keepalive request timed out after 8s")
			}

			if probeErr != nil {
				consecutiveFails++
				logger.TunnelLog(
					tunnelID,
					tunnelName,
					"WARN",
					"KeepAlive 心跳探测无响应 (%d/%d): %v",
					consecutiveFails,
					maxFails,
					probeErr,
				)

				if consecutiveFails >= maxFails {
					logger.TunnelLog(
						tunnelID,
						tunnelName,
						"ERROR",
						"KeepAlive 连续 %d 次无响应，判定 SSH 连接假死，启动自动重连",
						maxFails,
					)
					onFail(probeErr)
					return
				}
			} else {
				if consecutiveFails > 0 {
					logger.TunnelLog(
						tunnelID,
						tunnelName,
						"SUCCESS",
						"KeepAlive 心跳恢复正常，已清除失联计数",
					)
					consecutiveFails = 0
				}
			}
		}
	}
}
