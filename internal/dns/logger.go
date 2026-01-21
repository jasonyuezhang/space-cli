package dns

import (
	"fmt"
	"log"
	"os"
)

// SimpleLogger is a simple logger implementation
type SimpleLogger struct {
	debug bool
}

// NewSimpleLogger creates a new simple logger
func NewSimpleLogger(debug bool) *SimpleLogger {
	return &SimpleLogger{
		debug: debug,
	}
}

// Info logs an info message
func (l *SimpleLogger) Info(msg string, fields ...interface{}) {
	l.log("INFO", msg, fields...)
}

// Warn logs a warning message
func (l *SimpleLogger) Warn(msg string, fields ...interface{}) {
	l.log("WARN", msg, fields...)
}

// Error logs an error message
func (l *SimpleLogger) Error(msg string, fields ...interface{}) {
	l.log("ERROR", msg, fields...)
}

// Debug logs a debug message
func (l *SimpleLogger) Debug(msg string, fields ...interface{}) {
	if l.debug {
		l.log("DEBUG", msg, fields...)
	}
}

// log is the internal logging function
func (l *SimpleLogger) log(level, msg string, fields ...interface{}) {
	if len(fields) > 0 {
		fieldStr := ""
		for i := 0; i < len(fields); i += 2 {
			if i > 0 {
				fieldStr += " "
			}
			if i+1 < len(fields) {
				fieldStr += fmt.Sprintf("%v=%v", fields[i], fields[i+1])
			}
		}
		log.Printf("[%s] %s (%s)", level, msg, fieldStr)
	} else {
		log.Printf("[%s] %s", level, msg)
	}
}

// StdLogger wraps the standard logger
type StdLogger struct{}

// NewStdLogger creates a standard logger
func NewStdLogger() *StdLogger {
	return &StdLogger{}
}

// Info logs an info message
func (l *StdLogger) Info(msg string, fields ...interface{}) {
	fmt.Println("ℹ️ ", msg)
}

// Warn logs a warning message
func (l *StdLogger) Warn(msg string, fields ...interface{}) {
	fmt.Fprintln(os.Stderr, "⚠️ ", msg)
}

// Error logs an error message
func (l *StdLogger) Error(msg string, fields ...interface{}) {
	fmt.Fprintln(os.Stderr, "❌", msg)
}

// Debug logs a debug message
func (l *StdLogger) Debug(msg string, fields ...interface{}) {
	// Don't print debug messages to stdout
}
