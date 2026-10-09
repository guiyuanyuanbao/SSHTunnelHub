package tunnel

import (
	"bytes"
	"io"
	"net"
	"sync"
	"testing"
	"time"
)

type mockConn struct {
	readBuf  *bytes.Buffer
	writeBuf *bytes.Buffer
	closed   bool
	mu       sync.Mutex
}

func newMockConn(input []byte) *mockConn {
	return &mockConn{
		readBuf:  bytes.NewBuffer(input),
		writeBuf: &bytes.Buffer{},
	}
}

func (m *mockConn) Read(b []byte) (n int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return 0, io.EOF
	}
	n, err = m.readBuf.Read(b)
	if n == 0 && err == nil {
		err = io.EOF
	}
	return
}

func (m *mockConn) Write(b []byte) (n int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return 0, io.ErrClosedPipe
	}
	return m.writeBuf.Write(b)
}

func (m *mockConn) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

func (m *mockConn) LocalAddr() net.Addr                { return nil }
func (m *mockConn) RemoteAddr() net.Addr               { return nil }
func (m *mockConn) SetDeadline(t time.Time) error      { return nil }
func (m *mockConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *mockConn) SetWriteDeadline(t time.Time) error { return nil }

func TestCountingConnAndMetrics(t *testing.T) {
	metrics := NewTunnelMetrics()
	input := []byte("Hello SSHTunnelHub")
	rawConn := newMockConn(input)

	tracked := WrapConn(rawConn, metrics, true)

	buf := make([]byte, len(input))
	n, err := tracked.Read(buf)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if n != len(input) {
		t.Fatalf("Expected read %d bytes, got %d", len(input), n)
	}

	writePayload := []byte("Response Data")
	nw, err := tracked.Write(writePayload)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if nw != len(writePayload) {
		t.Fatalf("Expected written %d bytes, got %d", len(writePayload), nw)
	}

	metrics.ConnectedAt = time.Now().Add(-10 * time.Second)
	active, bin, bout, uptime := metrics.Snapshot("running", "")

	if bin != int64(len(input)) {
		t.Fatalf("Expected BytesIn %d, got %d", len(input), bin)
	}
	if bout != int64(len(writePayload)) {
		t.Fatalf("Expected BytesOut %d, got %d", len(writePayload), bout)
	}
	if uptime < 9 || uptime > 12 {
		t.Fatalf("Expected uptime around 10s, got %d", uptime)
	}
	if active != 0 {
		t.Fatalf("Expected active 0, got %d", active)
	}
}
