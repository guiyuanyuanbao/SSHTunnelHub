package tunnel

import (
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// TunnelMetrics tracks real-time traffic and connection counts
type TunnelMetrics struct {
	ActiveConns int64
	BytesIn     int64
	BytesOut    int64
	ConnectedAt time.Time
}

func NewTunnelMetrics() *TunnelMetrics {
	return &TunnelMetrics{}
}

func (m *TunnelMetrics) IncActiveConns() {
	atomic.AddInt64(&m.ActiveConns, 1)
}

func (m *TunnelMetrics) DecActiveConns() {
	atomic.AddInt64(&m.ActiveConns, -1)
}

func (m *TunnelMetrics) AddBytesIn(n int64) {
	atomic.AddInt64(&m.BytesIn, n)
}

func (m *TunnelMetrics) AddBytesOut(n int64) {
	atomic.AddInt64(&m.BytesOut, n)
}

func (m *TunnelMetrics) Snapshot(status string, lastError string) (int64, int64, int64, int64) {
	active := atomic.LoadInt64(&m.ActiveConns)
	if active < 0 {
		active = 0
	}
	bin := atomic.LoadInt64(&m.BytesIn)
	bout := atomic.LoadInt64(&m.BytesOut)
	var uptime int64
	if status == "running" && !m.ConnectedAt.IsZero() {
		uptime = int64(time.Since(m.ConnectedAt).Seconds())
	}
	return active, bin, bout, uptime
}

// CountingConn wraps a net.Conn to count bytes read and written
type CountingConn struct {
	net.Conn
	metrics *TunnelMetrics
	isReadIn bool // if true, Read counts as BytesIn, Write as BytesOut
}

func WrapConn(c net.Conn, metrics *TunnelMetrics, isReadIn bool) *CountingConn {
	return &CountingConn{
		Conn:     c,
		metrics:  metrics,
		isReadIn: isReadIn,
	}
}

func (c *CountingConn) Read(b []byte) (n int, err error) {
	n, err = c.Conn.Read(b)
	if n > 0 {
		if c.isReadIn {
			c.metrics.AddBytesIn(int64(n))
		} else {
			c.metrics.AddBytesOut(int64(n))
		}
	}
	return
}

func (c *CountingConn) Write(b []byte) (n int, err error) {
	n, err = c.Conn.Write(b)
	if n > 0 {
		if c.isReadIn {
			c.metrics.AddBytesOut(int64(n))
		} else {
			c.metrics.AddBytesIn(int64(n))
		}
	}
	return
}

// EnableTCPKeepAlive attempts to enable TCP keepalive on connections supporting it
func EnableTCPKeepAlive(c any, period time.Duration) {
	if period <= 0 {
		period = 30 * time.Second
	}
	if tc, ok := c.(interface{ SetKeepAlive(bool) error }); ok {
		_ = tc.SetKeepAlive(true)
	}
	if tc, ok := c.(interface{ SetKeepAlivePeriod(time.Duration) error }); ok {
		_ = tc.SetKeepAlivePeriod(period)
	}
}

// Bridge copies data bidirectionally between two connections with instant mutual-teardown
func Bridge(c1, c2 io.ReadWriteCloser) {
	EnableTCPKeepAlive(c1, 30*time.Second)
	EnableTCPKeepAlive(c2, 30*time.Second)

	var once sync.Once
	closeBoth := func() {
		_ = c1.Close()
		_ = c2.Close()
	}

	var wg sync.WaitGroup
	wg.Add(2)

	pipe := func(dst io.WriteCloser, src io.Reader) {
		defer wg.Done()
		defer once.Do(closeBoth)
		_, _ = io.Copy(dst, src)
	}

	go pipe(c1, c2)
	go pipe(c2, c1)

	wg.Wait()
}
