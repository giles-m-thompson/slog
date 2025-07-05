package slog

import (
	"fmt"
	"log"
	"os"
	"sync"
)



// --- Global Log Level Configuration ---

// This mutex ensures thread-safe access to the global LOG_LEVEL
var globalLogLevelMutex sync.RWMutex
var globalLogLevel LogLevel = INFO // Default to INFO, can be changed via Logger methods

// SetGlobalMinLevel sets the minimum log level for ALL Logger instances.
// This is useful if you want a single, application-wide log verbosity setting.
// It's thread-safe.
func SetGlobalMinLevel(level LogLevel) {
	globalLogLevelMutex.Lock()
	defer globalLogLevelMutex.Unlock()
	globalLogLevel = level
}

// GetGlobalMinLevel returns the current global minimum log level.
// It's thread-safe.
func GetGlobalMinLevel() LogLevel {
	globalLogLevelMutex.RLock()
	defer globalLogLevelMutex.RUnlock()
	return globalLogLevel
}

// Logger provides a structured logging utility with configurable levels.
type Logger struct {
	internalLogger *log.Logger
	component      string // New field to store the explicit component/struct name
}

// NewLogger creates and returns a new Logger instance.
//
// component: An optional string to identify the source of the log (e.g., struct name, module name).
//            If empty, no component prefix will be added.
// output: An optional os.File to direct logs to. If nil, os.Stdout is used.
func NewLogger(component string, output *os.File) *Logger {
	if output == nil {
		output = os.Stdout
	}
	return &Logger{
		internalLogger: log.New(output, "", log.LstdFlags),
		component:      component,
	}
}

// logf is the internal function that handles the actual logging logic.
// It checks against the global minimum log level and includes the component name.
func (l *Logger) logf(level LogLevel, msg string, params ...interface{}) {
	// Check if the message's level is higher than the currently configured global minimum level.
	if level > GetGlobalMinLevel() {
		return // Do not log if the level is too low
	}

	// Build the prefix: [LEVEL][COMPONENT]
	prefix := fmt.Sprintf("[%s]", level.String())
	if l.component != "" {
		prefix = fmt.Sprintf("%s[%s]", prefix, l.component)
	}

	// Print the final message.
	l.internalLogger.Printf("%s %s", prefix, fmt.Sprintf(msg, params...))
}

//LOG LEVEL METHODS.

// Error logs an error message.
func (l *Logger) Error(msg string, params ...interface{}) {
	l.logf(ERROR, msg, params...)
}

// Warn logs a warning message.
func (l *Logger) Warn(msg string, params ...interface{}) {
	l.logf(WARN, msg, params...)
}

// Info logs an informational message.
func (l *Logger) Info(msg string, params ...interface{}) {
	l.logf(INFO, msg, params...)
}

// Debug logs a debug message.
func (l *Logger) Debug(msg string, params ...interface{}) {
	l.logf(DEBUG, msg, params...)
}

// Fine logs a fine-grained debug message.
func (l *Logger) Fine(msg string, params ...interface{}) {
	l.logf(FINE, msg, params...)
}
