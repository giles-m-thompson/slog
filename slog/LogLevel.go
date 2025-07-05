package slog

import "fmt"

// LogLevel represents the severity of a log message.
type LogLevel int

const (
	ERROR LogLevel = iota // 0
	WARN                  // 1
	INFO                  // 2
	DEBUG                 // 3
	FINE                  // 4
)


// String returns the string representation of a LogLevel.
func (l LogLevel) String() string {
	switch l {
	case ERROR:
		return "ERROR"
	case WARN:
		return "WARN"
	case INFO:
		return "INFO"
	case DEBUG:
		return "DEBUG"
	case FINE:
		return "FINE"
	default:
		return fmt.Sprintf("UNKNOWN_LOG_LEVEL(%d)", l)
	}
}