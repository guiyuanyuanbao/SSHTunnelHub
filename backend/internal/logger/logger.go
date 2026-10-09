package logger

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type LogEntry struct {
	ID        int64     `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"` // "INFO" | "WARN" | "ERROR" | "SUCCESS"
	Tag       string    `json:"tag"`   // "SYSTEM" | "TUNNEL" | "SSH" | "AUTH"
	TunnelID  uint      `json:"tunnel_id,omitempty"`
	TunnelName string   `json:"tunnel_name,omitempty"`
	Message   string    `json:"message"`
}

type RingLogger struct {
	mu       sync.RWMutex
	capacity int
	entries  []LogEntry
	nextID   int64
	listener func(entry LogEntry)
}

var (
	globalLogger *RingLogger
	once         sync.Once
)

func GetLogger() *RingLogger {
	once.Do(func() {
		globalLogger = NewRingLogger(1500)
	})
	return globalLogger
}

func NewRingLogger(capacity int) *RingLogger {
	if capacity <= 0 {
		capacity = 1000
	}
	return &RingLogger{
		capacity: capacity,
		entries:  make([]LogEntry, 0, capacity),
	}
}

func (r *RingLogger) SetListener(fn func(entry LogEntry)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.listener = fn
}

func (r *RingLogger) Add(level, tag string, tunnelID uint, tunnelName, msg string) LogEntry {
	id := atomic.AddInt64(&r.nextID, 1)
	entry := LogEntry{
		ID:         id,
		Timestamp:  time.Now(),
		Level:      strings.ToUpper(level),
		Tag:        strings.ToUpper(tag),
		TunnelID:   tunnelID,
		TunnelName: tunnelName,
		Message:    msg,
	}

	r.mu.Lock()
	if len(r.entries) >= r.capacity {
		// Drop oldest 200 items when reaching capacity
		dropCount := 200
		if dropCount > len(r.entries) {
			dropCount = len(r.entries)
		}
		r.entries = r.entries[dropCount:]
	}
	r.entries = append(r.entries, entry)
	listener := r.listener
	r.mu.Unlock()

	// Also print to standard stdout
	log.Printf("[%s][%s] %s", entry.Level, entry.Tag, entry.Message)

	if listener != nil {
		listener(entry)
	}

	return entry
}

func (r *RingLogger) Query(level, tag string, tunnelID uint, keyword string, limit int) []LogEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if limit <= 0 || limit > 500 {
		limit = 200
	}

	level = strings.ToUpper(strings.TrimSpace(level))
	tag = strings.ToUpper(strings.TrimSpace(tag))
	keyword = strings.ToLower(strings.TrimSpace(keyword))

	results := make([]LogEntry, 0, limit)
	// Iterate backwards from newest to oldest
	for i := len(r.entries) - 1; i >= 0 && len(results) < limit; i-- {
		item := r.entries[i]

		if level != "" && level != "ALL" && item.Level != level {
			continue
		}
		if tag != "" && tag != "ALL" && item.Tag != tag {
			continue
		}
		if tunnelID > 0 && item.TunnelID != tunnelID {
			continue
		}
		if keyword != "" {
			msgLower := strings.ToLower(item.Message)
			nameLower := strings.ToLower(item.TunnelName)
			if !strings.Contains(msgLower, keyword) && !strings.Contains(nameLower, keyword) {
				continue
			}
		}

		results = append(results, item)
	}

	return results
}

func (r *RingLogger) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries = make([]LogEntry, 0, r.capacity)
}

// Convenience package helpers
func Info(tag, format string, args ...any) {
	GetLogger().Add("INFO", tag, 0, "", fmt.Sprintf(format, args...))
}

func Warn(tag, format string, args ...any) {
	GetLogger().Add("WARN", tag, 0, "", fmt.Sprintf(format, args...))
}

func Error(tag, format string, args ...any) {
	GetLogger().Add("ERROR", tag, 0, "", fmt.Sprintf(format, args...))
}

func Success(tag, format string, args ...any) {
	GetLogger().Add("SUCCESS", tag, 0, "", fmt.Sprintf(format, args...))
}

func TunnelLog(tunnelID uint, tunnelName, level, format string, args ...any) {
	GetLogger().Add(level, "TUNNEL", tunnelID, tunnelName, fmt.Sprintf(format, args...))
}
