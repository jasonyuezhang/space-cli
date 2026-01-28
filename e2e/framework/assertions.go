// Package framework provides the e2e test framework for space-cli
package framework

import (
	"strings"
	"testing"
)

// Assertions provides assertion helpers for e2e tests
type Assertions struct {
	t *testing.T
}

// Assert creates a new Assertions instance
func Assert(t *testing.T) *Assertions {
	return &Assertions{t: t}
}

// CmdSucceeds asserts that a command result indicates success
func (a *Assertions) CmdSucceeds(result *CmdResult, msgAndArgs ...interface{}) {
	a.t.Helper()
	if !result.Success() {
		msg := formatMsgAndArgs(msgAndArgs...)
		a.t.Errorf("%sCommand failed with exit code %d\nStdout: %s\nStderr: %s\nError: %v",
			msg, result.ExitCode, result.Stdout, result.Stderr, result.Err)
	}
}

// CmdFails asserts that a command result indicates failure
func (a *Assertions) CmdFails(result *CmdResult, msgAndArgs ...interface{}) {
	a.t.Helper()
	if result.Success() {
		msg := formatMsgAndArgs(msgAndArgs...)
		a.t.Errorf("%sExpected command to fail, but it succeeded\nStdout: %s\nStderr: %s",
			msg, result.Stdout, result.Stderr)
	}
}

// CmdOutputContains asserts that command output contains a substring
func (a *Assertions) CmdOutputContains(result *CmdResult, substring string, msgAndArgs ...interface{}) {
	a.t.Helper()
	if !result.Contains(substring) {
		msg := formatMsgAndArgs(msgAndArgs...)
		a.t.Errorf("%sExpected output to contain %q\nStdout: %s\nStderr: %s",
			msg, substring, result.Stdout, result.Stderr)
	}
}

// CmdOutputNotContains asserts that command output does not contain a substring
func (a *Assertions) CmdOutputNotContains(result *CmdResult, substring string, msgAndArgs ...interface{}) {
	a.t.Helper()
	if result.Contains(substring) {
		msg := formatMsgAndArgs(msgAndArgs...)
		a.t.Errorf("%sExpected output to NOT contain %q\nStdout: %s\nStderr: %s",
			msg, substring, result.Stdout, result.Stderr)
	}
}

// StringContains asserts that a string contains a substring
func (a *Assertions) StringContains(s, substring string, msgAndArgs ...interface{}) {
	a.t.Helper()
	if !strings.Contains(s, substring) {
		msg := formatMsgAndArgs(msgAndArgs...)
		a.t.Errorf("%sExpected string to contain %q\nActual: %s", msg, substring, s)
	}
}

// StringNotEmpty asserts that a string is not empty
func (a *Assertions) StringNotEmpty(s string, msgAndArgs ...interface{}) {
	a.t.Helper()
	if s == "" {
		msg := formatMsgAndArgs(msgAndArgs...)
		a.t.Errorf("%sExpected string to not be empty", msg)
	}
}

// NoError asserts that an error is nil
func (a *Assertions) NoError(err error, msgAndArgs ...interface{}) {
	a.t.Helper()
	if err != nil {
		msg := formatMsgAndArgs(msgAndArgs...)
		a.t.Errorf("%sUnexpected error: %v", msg, err)
	}
}

// Error asserts that an error is not nil
func (a *Assertions) Error(err error, msgAndArgs ...interface{}) {
	a.t.Helper()
	if err == nil {
		msg := formatMsgAndArgs(msgAndArgs...)
		a.t.Errorf("%sExpected an error but got nil", msg)
	}
}

// True asserts that a condition is true
func (a *Assertions) True(condition bool, msgAndArgs ...interface{}) {
	a.t.Helper()
	if !condition {
		msg := formatMsgAndArgs(msgAndArgs...)
		a.t.Errorf("%sExpected condition to be true", msg)
	}
}

// False asserts that a condition is false
func (a *Assertions) False(condition bool, msgAndArgs ...interface{}) {
	a.t.Helper()
	if condition {
		msg := formatMsgAndArgs(msgAndArgs...)
		a.t.Errorf("%sExpected condition to be false", msg)
	}
}

// Equal asserts that two values are equal
func (a *Assertions) Equal(expected, actual interface{}, msgAndArgs ...interface{}) {
	a.t.Helper()
	if expected != actual {
		msg := formatMsgAndArgs(msgAndArgs...)
		a.t.Errorf("%sExpected: %v\nActual: %v", msg, expected, actual)
	}
}

func formatMsgAndArgs(msgAndArgs ...interface{}) string {
	if len(msgAndArgs) == 0 {
		return ""
	}

	if len(msgAndArgs) == 1 {
		if s, ok := msgAndArgs[0].(string); ok {
			return s + ": "
		}
	}

	return ""
}
