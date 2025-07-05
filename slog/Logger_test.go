package slog

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"
	"testing"
)

// Helper function to create a test logger that writes to a bytes.Buffer
// and doesn't include standard log flags (like date/time) for easier string comparison.
func newTestLogger(output io.Writer, component string) *Logger {
	// Temporarily create a log.Logger directly for testing purposes.
	// In production code, NewLogger always uses log.LstdFlags.
	return &Logger{
		internalLogger: log.New(output, "", 0), // 0 flags for clean output
		component:      component,
	}
}

// TestSetGlobalMinLevel ensures the global log level can be set correctly.
func TestSetGlobalMinLevel(t *testing.T) {
	// Ensure we reset the global log level after the test
	originalLevel := GetGlobalMinLevel()
	t.Cleanup(func() {
		SetGlobalMinLevel(originalLevel)
	})

	testCases := []struct {
		level    LogLevel
		expected LogLevel
	}{
		{INFO, INFO},
		{DEBUG, DEBUG},
		{ERROR, ERROR},
		{FINE, FINE},
		{WARN, WARN},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("SetTo%s", tc.level.String()), func(t *testing.T) {
			SetGlobalMinLevel(tc.level)
			if GetGlobalMinLevel() != tc.expected {
				t.Errorf("Expected global log level to be %s, got %s", tc.expected, GetGlobalMinLevel())
			}
		})
	}
}

// TestLoggerFiltering tests that messages are only logged when their level
// is at or above the global minimum log level.
func TestLoggerFiltering(t *testing.T) {
	// Ensure global log level is reset after the test
	originalLevel := GetGlobalMinLevel()
	t.Cleanup(func() {
		SetGlobalMinLevel(originalLevel)
	})

	testCases := []struct {
		minLevel     LogLevel
		logLevel     LogLevel
		message      string
		expectLogged bool
	}{
		// --- ERROR level as minimum ---
		{ERROR, ERROR, "Error message", true},
		{ERROR, WARN, "Warning message", false}, // WARN (1) > ERROR (0) -> should NOT be logged
		{ERROR, INFO, "Info message", false},
		{ERROR, DEBUG, "Debug message", false},
		{ERROR, FINE, "Fine message", false},

		// --- WARN level as minimum ---
		{WARN, ERROR, "Error message", true},
		{WARN, WARN, "Warning message", true},
		{WARN, INFO, "Info message", false}, // INFO (2) > WARN (1) -> should NOT be logged
		{WARN, DEBUG, "Debug message", false},
		{WARN, FINE, "Fine message", false},

		// --- INFO level as minimum ---
		{INFO, ERROR, "Error message", true},
		{INFO, WARN, "Warning message", true},
		{INFO, INFO, "Info message", true},
		{INFO, DEBUG, "Debug message", false}, // DEBUG (3) > INFO (2) -> should NOT be logged
		{INFO, FINE, "Fine message", false},

		// --- DEBUG level as minimum ---
		{DEBUG, ERROR, "Error message", true},
		{DEBUG, WARN, "Warning message", true},
		{DEBUG, INFO, "Info message", true},
		{DEBUG, DEBUG, "Debug message", true},
		{DEBUG, FINE, "Fine message", false}, // FINE (4) > DEBUG (3) -> should NOT be logged

		// --- FINE level as minimum ---
		{FINE, ERROR, "Error message", true},
		{FINE, WARN, "Warning message", true},
		{FINE, INFO, "Info message", true},
		{FINE, DEBUG, "Debug message", true},
		{FINE, FINE, "Fine message", true},
	}

	for _, tc := range testCases {
		testName := fmt.Sprintf("MinLevel_%s_LogLevel_%s_ExpectLogged_%t", tc.minLevel.String(), tc.logLevel.String(), tc.expectLogged)
		t.Run(testName, func(t *testing.T) {
			var buf bytes.Buffer
			logger := newTestLogger(&buf, "TestComponent") // Use a test logger

			// Set the global minimum level for this test
			SetGlobalMinLevel(tc.minLevel)

			// Call the appropriate logging method
			switch tc.logLevel {
			case ERROR:
				logger.Error(tc.message)
			case WARN:
				logger.Warn(tc.message)
			case INFO:
				logger.Info(tc.message)
			case DEBUG:
				logger.Debug(tc.message)
			case FINE:
				logger.Fine(tc.message)
			}

			output := strings.TrimSpace(buf.String()) // Trim whitespace from output

			if tc.expectLogged {
				// We expect the message to be present and contain the level and message
				expectedOutputPart := fmt.Sprintf("[%s][TestComponent] %s", tc.logLevel.String(), tc.message)
				if !strings.Contains(output, expectedOutputPart) {
					t.Errorf("Expected log for %s but got:\n%q\nExpected part: %q", tc.logLevel.String(), output, expectedOutputPart)
				}
			} else {
				// We expect the buffer to be empty (no log output)
				if output != "" {
					t.Errorf("Expected no log for %s (min level %s), but got:\n%q", tc.logLevel.String(), tc.minLevel.String(), output)
				}
			}
		})
	}
}

// TestLoggerMessageFormatting ensures messages are formatted correctly, including params and component.
func TestLoggerMessageFormatting(t *testing.T) {
	// Ensure global log level allows all messages for this test
	originalLevel := GetGlobalMinLevel()
	t.Cleanup(func() {
		SetGlobalMinLevel(originalLevel)
	})
	SetGlobalMinLevel(FINE) // Set to FINE to ensure all levels print

	testCases := []struct {
		name        string
		component   string
		logFunc     func(l *Logger, msg string, args ...interface{})
		logLevel    LogLevel
		messageFmt  string
		args        []interface{}
		expectedMsg string // The message part expected in the output
	}{
		{
			"Error with params", "App",
			func(l *Logger, msg string, args ...interface{}) { l.Error(msg, args...) },
			ERROR, "Failed with error: %v", []interface{}{fmt.Errorf("disk full")},
			"[ERROR][App] Failed with error: disk full",
		},
		{
			"Warn with multiple params", "Worker",
			func(l *Logger, msg string, args ...interface{}) { l.Warn(msg, args...) },
			WARN, "Job %d failed for user %s", []interface{}{123, "Alice"},
			"[WARN][Worker] Job 123 failed for user Alice",
		},
		{
			"Info without params", "Service",
			func(l *Logger, msg string, args ...interface{}) { l.Info(msg, args...) },
			INFO, "Service started", nil,
			"[INFO][Service] Service started",
		},
		{
			"Debug with empty component", "",
			func(l *Logger, msg string, args ...interface{}) { l.Debug(msg, args...) },
			DEBUG, "Debug info for process: %s", []interface{}{"xyz"},
			"[DEBUG] Debug info for process: xyz", // No component tag
		},
		{
			"Fine with complex object", "DB",
			func(l *Logger, msg string, args ...interface{}) { l.Fine(msg, args...) },
			FINE, "DB state: %+v", []interface{}{struct{ Status string }{Status: "Connected"}},
			"[FINE][DB] DB state: {Status:Connected}",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := newTestLogger(&buf, tc.component)

			tc.logFunc(logger, tc.messageFmt, tc.args...)

			output := strings.TrimSpace(buf.String())

			if !strings.Contains(output, tc.expectedMsg) {
				t.Errorf("Expected output to contain:\n%q\nGot:\n%q", tc.expectedMsg, output)
			}
		})
	}
}

// TestLogLevelStringer ensures the String() method for LogLevel returns correct strings.
func TestLogLevelStringer(t *testing.T) {
	testCases := []struct {
		level    LogLevel
		expected string
	}{
		{ERROR, "ERROR"},
		{WARN, "WARN"},
		{INFO, "INFO"},
		{DEBUG, "DEBUG"},
		{FINE, "FINE"},
		{LogLevel(99), "UNKNOWN_LOG_LEVEL(99)"}, // Test an unknown level
	}

	for _, tc := range testCases {
		t.Run(tc.expected, func(t *testing.T) {
			if tc.level.String() != tc.expected {
				t.Errorf("Expected String() for %d to be %q, got %q", tc.level, tc.expected, tc.level.String())
			}
		})
	}
}

// TestLoggerThreadSafety (basic check)
func TestLoggerThreadSafety(t *testing.T) {
	originalLevel := GetGlobalMinLevel()
	t.Cleanup(func() {
		SetGlobalMinLevel(originalLevel)
	})

	var buf bytes.Buffer
	logger := newTestLogger(&buf, "ThreadTest")
	SetGlobalMinLevel(FINE) // Ensure all logs are written

	var wg sync.WaitGroup
	numGoroutines := 100
	messagesPerGoroutine := 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for j := 0; j < messagesPerGoroutine; j++ {
				logger.Info("Goroutine %d: Message %d", g, j)
			}
		}(i)
	}

	wg.Wait()

	// Just a basic check: ensure the total number of expected messages are logged.
	// This doesn't catch all concurrency issues, but helps verify no deadlocks/panics
	// and that messages aren't mysteriously lost.
	expectedTotalMessages := numGoroutines * messagesPerGoroutine
	actualLines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(actualLines) != expectedTotalMessages {
		// Note: The `log` package (which we wrap) uses mutexes internally, so it's inherently thread-safe for writes.
		// This test primarily checks that our wrapper doesn't introduce *new* concurrency issues.
		t.Errorf("Expected %d log messages, got %d", expectedTotalMessages, len(actualLines))
	}

	// Also check if setting global level concurrently is safe
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			SetGlobalMinLevel(INFO)
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			SetGlobalMinLevel(DEBUG)
		}
	}()
	wg.Wait()
	// No specific assertion needed here other than it completes without panicking
	// indicating the mutex usage is preventing deadlocks during writes.
}

/**
Explanation of the Tests:
newTestLogger Helper:

This is crucial. Instead of using os.Stdout, it directs the log.Logger's output to a bytes.Buffer. This allows the test function to read what was "printed."

log.New(output, "", 0): We pass 0 as the flags to the underlying log.Logger. This removes the default date/time stamp, making it much easier to assert exact string matches in the output.

t.Cleanup(func() { ... }):

This is a best practice in Go tests. It schedules a function to be run after the test (or subtest) completes, regardless of whether it passed or failed.

We use it here to reset globalLogLevel to its original value. This prevents one test from unintentionally affecting the global state for subsequent tests.

TestSetGlobalMinLevel:

Simply tests if SetGlobalMinLevel correctly updates the globalLogLevel variable.

TestLoggerFiltering:

This is the core test for your log level logic.

It uses a testCases slice to define various scenarios: different minLevel settings and different logLevel calls.

For each scenario, it sets the globalLogLevel, calls the corresponding logging method, captures the output, and then asserts whether a message was expected or not.

strings.Contains is used for flexible string matching, and strings.TrimSpace cleans up the output.

TestLoggerMessageFormatting:

Ensures that your messages, including fmt.Printf style parameters and the component name, are correctly assembled in the final log string.

It temporarily sets the globalLogLevel to FINE to ensure all test messages are logged, regardless of their level.

TestLogLevelStringer:

Verifies that the String() method on your LogLevel type returns the correct string representation (e.g., INFO for INFO). It also checks the UNKNOWN_LOG_LEVEL case.

TestLoggerThreadSafety:

This is a basic concurrency test. It starts multiple goroutines that concurrently log messages and concurrently try to change the global log level.

It doesn't make strict assertions about the order of messages (which can vary in concurrent scenarios) but primarily ensures that the code runs without panics or deadlocks, and that the expected number of messages are eventually logged (implying the mutexes for globalLogLevel and the internal log.Logger are working correctly).

These tests provide good coverage for the core functionality of your slog package.
*/
