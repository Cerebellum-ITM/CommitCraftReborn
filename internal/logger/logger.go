package logger

import (
	"commit_craft_reborn/internal/config"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
)

// Logger holds two separate loggers: one for the console and one for the file.
type Logger struct {
	consoleLogger *log.Logger
	fileLogger    *slog.Logger
}

func New() *Logger {
	// --- File Logger Setup ---
	home, _ := os.UserHomeDir()
	logDir := filepath.Join(home, config.GlobalConfigDir)
	_ = os.MkdirAll(logDir, 0755)
	logFilePath := filepath.Join(logDir, "commit-crafter.log")

	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)

	var fileLogger *slog.Logger
	if err != nil {
		fileLogger = slog.New(slog.NewTextHandler(io.Discard, nil))
	} else {
		fileHandler := slog.NewTextHandler(logFile, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
		fileLogger = slog.New(fileHandler)
	}

	// --- Console Logger Setup ---
	consoleLogger := log.NewWithOptions(os.Stderr, log.Options{
		ReportCaller:    true,
		ReportTimestamp: true,
		TimeFormat:      time.ANSIC,
		// Prefix:          "",
	})
	consoleLogger.SetLevel(log.DebugLevel)
	syles := log.DefaultStyles()
	syles.Caller = lipgloss.NewStyle().Foreground(lipgloss.Color("#7C7C7C"))
	syles.Key = lipgloss.NewStyle().Foreground(lipgloss.Color("#978E20"))
	consoleLogger.SetStyles(syles)

	return &Logger{
		consoleLogger: consoleLogger,
		fileLogger:    fileLogger,
	}
}

func (l *Logger) Debug(msg string, args ...any) {
	l.consoleLogger.Helper()
	l.consoleLogger.Debug(msg, args...)
	l.fileLogger.Debug(msg, args...)
}

func (l *Logger) Info(msg string, args ...any) {
	l.consoleLogger.Helper()
	l.consoleLogger.Info(msg, args...)
	l.fileLogger.Info(msg, args...)
}

func (l *Logger) Warn(msg string, args ...any) {
	l.consoleLogger.Helper()
	l.consoleLogger.Warn(msg, args...)
	l.fileLogger.Warn(msg, args...)
}

func (l *Logger) Error(msg string, args ...any) {
	l.consoleLogger.Helper()
	l.consoleLogger.Error(msg, args...)
	l.fileLogger.Error(msg, args...)
}

func (l *Logger) Fatal(msg string, args ...any) {
	l.consoleLogger.Helper()
	l.consoleLogger.Fatal(msg, args...)
	l.fileLogger.Error(msg, args...)
}

func (l *Logger) Print(msg string, args ...any) {
	l.consoleLogger.Helper()
	l.consoleLogger.Print(msg)
	l.fileLogger.Info(msg, args...)
}
