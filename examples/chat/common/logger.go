package common

import (
	"fmt"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

// LogLevel represents the severity of a log message
type LogLevel int

const (
	LogDebug LogLevel = iota
	LogInfo
	LogWarn
	LogError
	LogFatal
)

var logLevelNames = map[LogLevel]string{
	LogDebug: "DEBUG",
	LogInfo:  "INFO",
	LogWarn:  "WARN",
	LogError: "ERROR",
	LogFatal: "FATAL",
}

// Logger provides structured logging
type Logger struct {
	level   LogLevel
	file    *os.File
	logger  *log.Logger
	mu      sync.Mutex
	metrics *LogMetrics
}

// LogMetrics tracks logging statistics
type LogMetrics struct {
	mu      sync.RWMutex
	counts  map[LogLevel]int64
	lastLog time.Time
}

// globalLogger holds the default logger. It is an atomic pointer because it is
// written by InitLogger (main goroutine) and read by every logging call, possibly
// from other goroutines - a plain package variable would be a data race if
// InitLogger were ever called after those goroutines started.
var globalLogger atomic.Pointer[Logger]

// Global returns the default logger, or nil when InitLogger has not succeeded.
func Global() *Logger {
	return globalLogger.Load()
}

// InitLogger initializes the global logger
func InitLogger(filename string, level LogLevel) error {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, FileMode())
	if err != nil {
		return err
	}

	globalLogger.Store(&Logger{
		level:  level,
		file:   file,
		logger: log.New(file, "", 0),
		metrics: &LogMetrics{
			counts: make(map[LogLevel]int64),
		},
	})

	return nil
}

// Close closes the log file. It is safe to call on a nil *Logger, so callers do
// not need to check whether InitLogger succeeded.
func (l *Logger) Close() error {
	if l == nil || l.file == nil {
		return nil
	}
	return l.file.Close()
}

// log writes a log message
func (l *Logger) log(level LogLevel, format string, args ...any) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// Update metrics
	l.metrics.mu.Lock()
	l.metrics.counts[level]++
	l.metrics.lastLog = time.Now()
	l.metrics.mu.Unlock()

	// Format message
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	levelStr := logLevelNames[level]
	message := fmt.Sprintf(format, args...)

	// Write log
	logLine := fmt.Sprintf("[%s] [%s] %s", timestamp, levelStr, message)
	l.logger.Println(logLine)

	// Also print to console for errors and above
	if level >= LogError {
		log.Println(logLine)
	}
}

// Debug logs a debug message
func Debug(format string, args ...any) {
	if l := Global(); l != nil {
		l.log(LogDebug, format, args...)
	}
}

// Info logs an info message
func Info(format string, args ...any) {
	if l := Global(); l != nil {
		l.log(LogInfo, format, args...)
	}
}

// Warn logs a warning message
func Warn(format string, args ...any) {
	if l := Global(); l != nil {
		l.log(LogWarn, format, args...)
	}
}

// Error logs an error message
func Error(format string, args ...any) {
	if l := Global(); l != nil {
		l.log(LogError, format, args...)
	}
}

// Fatal logs a fatal message and exits.
//
// os.Exit has to be HERE and not in Logger.log: when logger initialisation failed
// and the global logger is nil, the previous version logged nothing AND did not
// end the program - main returned normally and the process exited with status 0
// despite a fatal error.
//
// TEACHING NOTE: os.Exit does NOT run deferred calls. That is why the exit sits
// after Logger.log has returned (and released its mutex) rather than inside it -
// a deferred Unlock inside log would never execute. It also means the caller's
// deferred Close on the log file does not run; the OS closes the descriptor.
func Fatal(format string, args ...any) {
	if l := Global(); l != nil {
		l.log(LogFatal, format, args...)
	} else {
		log.Printf("[FATAL] "+format, args...)
	}
	os.Exit(1)
}

// Metrics returns logging metrics
func Metrics() map[string]any {
	l := Global()
	if l == nil || l.metrics == nil {
		return nil
	}

	l.metrics.mu.RLock()
	defer l.metrics.mu.RUnlock()

	metrics := make(map[string]any)
	for level, count := range l.metrics.counts {
		metrics[logLevelNames[level]] = count
	}
	metrics["last_log"] = l.metrics.lastLog

	return metrics
}
