package logger

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"commit_craft_reborn/internal/config"

	"charm.land/lipgloss/v2"
	"charm.land/log/v2"
	"github.com/charmbracelet/colorprofile"
)

// ringBuffer keeps the last maxLines log lines in memory and fans out new lines
// to any registered subscribers. It implements io.Writer so it can be plugged
// into a log.Logger via io.MultiWriter alongside the log file.
type ringBuffer struct {
	mu          sync.Mutex
	lines       []string
	maxLines    int
	partial     bytes.Buffer
	subscribers []chan string
}

func newRingBuffer(maxLines int) *ringBuffer {
	return &ringBuffer{lines: make([]string, 0, maxLines), maxLines: maxLines}
}

func (r *ringBuffer) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.partial.Write(p)
	data := r.partial.Bytes()
	for {
		idx := bytes.IndexByte(data, '\n')
		if idx < 0 {
			break
		}
		line := string(data[:idx])
		data = data[idx+1:]
		r.appendLine(line)
	}
	// Keep any leftover (no trailing newline) for the next write.
	remaining := append([]byte{}, data...)
	r.partial.Reset()
	r.partial.Write(remaining)

	return len(p), nil
}

func (r *ringBuffer) appendLine(line string) {
	if len(r.lines) >= r.maxLines {
		copy(r.lines, r.lines[1:])
		r.lines = r.lines[:len(r.lines)-1]
	}
	r.lines = append(r.lines, line)
	for _, ch := range r.subscribers {
		select {
		case ch <- line:
		default:
		}
	}
}

func (r *ringBuffer) snapshot() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, len(r.lines))
	copy(out, r.lines)
	return out
}

func (r *ringBuffer) subscribe() <-chan string {
	r.mu.Lock()
	defer r.mu.Unlock()
	ch := make(chan string, 64)
	r.subscribers = append(r.subscribers, ch)
	return ch
}

// Logger is a thin wrapper around a charm log v2 logger that writes both to a
// file and to an in-memory ring buffer. The ring buffer feeds the in-app logs
// popup so the user can inspect logs without leaving the TUI.
type Logger struct {
	inner *log.Logger
	ring  *ringBuffer
}

func New() *Logger {
	home, _ := os.UserHomeDir()
	logDir := filepath.Join(home, config.GlobalConfigDir)
	_ = os.MkdirAll(logDir, 0o755)
	logFilePath := filepath.Join(logDir, "commit-crafter.log")

	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o666)

	ring := newRingBuffer(500)

	var writer io.Writer
	if err != nil {
		writer = ring
	} else {
		writer = io.MultiWriter(logFile, ring)
	}

	inner := log.NewWithOptions(writer, log.Options{
		ReportCaller:    true,
		ReportTimestamp: true,
		TimeFormat:      time.Kitchen,
	})
	inner.SetLevel(log.DebugLevel)
	// charm log wraps our writer in its own colorprofile.Writer and defaults
	// to NoTTY because we're not writing to a real terminal. Force TrueColor
	// so the ring buffer (and therefore the in-TUI popup) receives coloured
	// ANSI output instead of plain text.
	inner.SetColorProfile(colorprofile.TrueColor)

	styles := log.DefaultStyles()
	styles.Caller = lipgloss.NewStyle().Foreground(lipgloss.Color("#7C7C7C"))
	styles.Key = lipgloss.NewStyle().Foreground(lipgloss.Color("#978E20"))
	inner.SetStyles(styles)

	return &Logger{inner: inner, ring: ring}
}

// Snapshot returns the log lines currently in the ring buffer.
func (l *Logger) Snapshot() []string { return l.ring.snapshot() }

// Subscribe returns a channel that receives new log lines as they are written.
func (l *Logger) Subscribe() <-chan string { return l.ring.subscribe() }

func (l *Logger) Debug(msg string, args ...any) { l.inner.Helper(); l.inner.Debug(msg, args...) }
func (l *Logger) Info(msg string, args ...any)  { l.inner.Helper(); l.inner.Info(msg, args...) }
func (l *Logger) Warn(msg string, args ...any)  { l.inner.Helper(); l.inner.Warn(msg, args...) }
func (l *Logger) Error(msg string, args ...any) { l.inner.Helper(); l.inner.Error(msg, args...) }
func (l *Logger) Fatal(msg string, args ...any) { l.inner.Helper(); l.inner.Fatal(msg, args...) }
func (l *Logger) Print(msg string, args ...any) { l.inner.Helper(); l.inner.Print(msg, args...) }
