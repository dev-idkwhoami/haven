package tui

import (
	"sync"
)

// LogBuffer is a ring buffer that captures log output for display in the TUI.
// It implements io.Writer so it can be used as the slog output target.
type LogBuffer struct {
	mu   sync.RWMutex
	lines []string
	cap   int
	pos   int
	full  bool
}

// NewLogBuffer creates a ring buffer that stores up to capacity log lines.
func NewLogBuffer(capacity int) *LogBuffer {
	return &LogBuffer{
		lines: make([]string, capacity),
		cap:   capacity,
	}
}

// Write implements io.Writer. Each call stores one log line.
func (b *LogBuffer) Write(p []byte) (n int, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	line := string(p)
	if len(line) > 0 && line[len(line)-1] == '\n' {
		line = line[:len(line)-1]
	}
	if line == "" {
		return len(p), nil
	}

	b.lines[b.pos] = line
	b.pos++
	if b.pos >= b.cap {
		b.pos = 0
		b.full = true
	}

	return len(p), nil
}

// Lines returns all stored lines in chronological order.
func (b *LogBuffer) Lines() []string {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if !b.full {
		result := make([]string, b.pos)
		copy(result, b.lines[:b.pos])
		return result
	}

	result := make([]string, b.cap)
	copy(result, b.lines[b.pos:])
	copy(result[b.cap-b.pos:], b.lines[:b.pos])
	return result
}

// Count returns the number of stored lines.
func (b *LogBuffer) Count() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.full {
		return b.cap
	}
	return b.pos
}
