package logger

import (
	"io"
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"
)

// New initializes and configures a new logger that writes to both console and a file.
func New() *log.Logger {
	// 1. Determine the log file path.
	home, err := os.UserHomeDir()
	if err != nil {
		// Fallback to a simple console logger if home directory is not available.
		return log.New(os.Stderr)
	}
	logDir := filepath.Join(home, ".commitcraft")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return log.New(os.Stderr)
	}
	logFilePath := filepath.Join(logDir, "commit-crafter.log")

	// 2. Open the log file.
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		logger := log.New(os.Stderr)
		logger.Error("Could not open log file, falling back to console only", "path", logFilePath, "error", err)
		return logger
	}

	// 3. Create a MultiWriter to write to both Stderr (for the console) and the log file.
	writer := io.MultiWriter(os.Stderr, logFile)

	// 4. Create the logger, telling it to use our combined writer.
	//    The charmbracelet/log library handles the handler creation internally.
	logger := log.New(writer)
	
	// Set global options for the logger.
	logger.SetReportTimestamp(true)
	logger.SetLevel(log.DebugLevel)
	
	return logger
}
