package sshutil

import (
	"fmt"
	"net"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"sshtunnelhub/internal/crypto"
	"sshtunnelhub/internal/model"
)

// BuildClientConfig creates an ssh.ClientConfig for the given Host model.
func BuildClientConfig(h *model.Host, timeout time.Duration) (*ssh.ClientConfig, error) {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	var authMethods []ssh.AuthMethod

	switch h.AuthType {
	case "password":
		plainPass, err := crypto.Decrypt(h.Password)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt password: %w", err)
		}
		authMethods = append(authMethods, ssh.Password(plainPass))

	case "private_key":
		plainKey, err := crypto.Decrypt(h.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt private key: %w", err)
		}
		keyBytes := []byte(strings.TrimSpace(plainKey))
		if len(keyBytes) == 0 {
			return nil, fmt.Errorf("private key is empty")
		}

		var signer ssh.Signer
		if h.Passphrase != "" {
			plainPassphrase, err := crypto.Decrypt(h.Passphrase)
			if err != nil {
				return nil, fmt.Errorf("failed to decrypt key passphrase: %w", err)
			}
			signer, err = ssh.ParsePrivateKeyWithPassphrase(keyBytes, []byte(plainPassphrase))
			if err != nil {
				return nil, fmt.Errorf("failed to parse private key with passphrase: %w", err)
			}
		} else {
			signer, err = ssh.ParsePrivateKey(keyBytes)
			if err != nil {
				return nil, fmt.Errorf("failed to parse private key: %w", err)
			}
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))

	default:
		return nil, fmt.Errorf("unsupported auth type: %s", h.AuthType)
	}

	config := &ssh.ClientConfig{
		User:            h.Username,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         timeout,
	}

	return config, nil
}

// Dial establishes an SSH client connection with TCP keepalive enabled.
func Dial(h *model.Host, timeout time.Duration) (*ssh.Client, error) {
	config, err := BuildClientConfig(h, timeout)
	if err != nil {
		return nil, err
	}

	addr := fmt.Sprintf("%s:%d", h.Host, h.Port)
	dialer := &net.Dialer{
		Timeout:   timeout,
		KeepAlive: 30 * time.Second,
	}

	conn, err := dialer.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", addr, err)
	}

	sshConn, chans, reqs, err := ssh.NewClientConn(conn, addr, config)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("ssh handshake failed for %s: %w", addr, err)
	}

	return ssh.NewClient(sshConn, chans, reqs), nil
}

// TestConnection tries to connect to the SSH host and measures round-trip time.
func TestConnection(h *model.Host) (time.Duration, string, error) {
	start := time.Now()
	client, err := Dial(h, 8*time.Second)
	latency := time.Since(start)
	if err != nil {
		return latency, "", err
	}
	defer client.Close()

	serverVersion := string(client.ServerVersion())
	return latency, serverVersion, nil
}
