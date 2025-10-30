package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

var (
	fileLogger *log.Logger
	logFile    *os.File
)

// Initializes the file logger
func Init() error {
	// Get executable path
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	exeDir := filepath.Dir(exePath)

	// Create logs directory if it doesn't exist (beside the executable)
	logsDir := filepath.Join(exeDir, "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return fmt.Errorf("failed to create logs directory: %w", err)
	}

	// Create log file with date (one file per day)
	date := time.Now().Format("2006-01-02")
	logPath := filepath.Join(logsDir, fmt.Sprintf("rolodex_%s.log", date))

	var openErr error
	logFile, openErr = os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if openErr != nil {
		return fmt.Errorf("failed to open log file: %w", openErr)
	}

	// Create logger that writes to file
	fileLogger = log.New(logFile, "", log.LstdFlags|log.Lshortfile)

	fileLogger.Printf("=== Rolodex session started ===")
	return nil
}

// Closes the log file
func Close() {
	if fileLogger != nil {
		fileLogger.Printf("=== Rolodex session ended ===")
	}
	if logFile != nil {
		logFile.Close()
	}
}

// Logs a formatted message to the file
func Printf(format string, v ...any) {
	if fileLogger != nil {
		fileLogger.Printf(format, v...)
	}
}

// Logs a message to the file
func Print(v ...any) {
	if fileLogger != nil {
		fileLogger.Print(v...)
	}
}

// Logs a message with newline to the file
func Println(v ...any) {
	if fileLogger != nil {
		fileLogger.Println(v...)
	}
}

// Logs a fatal error to the file and returns the error
// Unlike log.Fatal, this doesn't call os.Exit - caller should handle the error
func Fatal(v ...any) error {
	msg := fmt.Sprint(v...)
	if fileLogger != nil {
		fileLogger.Printf("FATAL: %s", msg)
	}
	return fmt.Errorf("%s", msg)
}

// Logs a formatted fatal error to the file and returns the error
// Unlike log.Fatalf, this doesn't call os.Exit - caller should handle the error
func Fatalf(format string, v ...any) error {
	msg := fmt.Sprintf(format, v...)
	if fileLogger != nil {
		fileLogger.Printf("FATAL: %s", msg)
	}
	return fmt.Errorf("%s", msg)
}

// Returns the log file writer for use with other loggers
func GetWriter() io.Writer {
	if logFile != nil {
		return logFile
	}
	return io.Discard
}
