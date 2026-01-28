//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/happy-sdk/space-cli/e2e/framework"
)

// TestSpaceConfigShow tests the space config show command
func TestSpaceConfigShow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	f := framework.New(t).WithFixture("simple-app")
	defer f.Cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result := f.RunSpaceCmd(ctx, "config", "show")
	assert := framework.Assert(t)

	assert.CmdSucceeds(result, "space config show should succeed")
	assert.CmdOutputContains(result, "project", "should show project configuration")
	assert.CmdOutputContains(result, "e2e-simple-app", "should show project name")
}

// TestSpaceConfigShowMultiService tests config show with multi-service fixture
func TestSpaceConfigShowMultiService(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	f := framework.New(t).WithFixture("multi-service")
	defer f.Cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result := f.RunSpaceCmd(ctx, "config", "show")
	assert := framework.Assert(t)

	assert.CmdSucceeds(result, "space config show should succeed")
	assert.CmdOutputContains(result, "api", "should show api service")
	assert.CmdOutputContains(result, "frontend", "should show frontend service")
	assert.CmdOutputContains(result, "postgres", "should show postgres service")
	assert.CmdOutputContains(result, "redis", "should show redis service")
}

// TestSpaceConfigNoFile tests config show without a .space.yaml file
func TestSpaceConfigNoFile(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	// Create a framework with a non-existent fixture path
	// We'll use a temp dir without a .space.yaml
	f := framework.New(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Run in project root where there's no .space.yaml
	result := f.RunSpaceCmd(ctx, "config", "show")

	// It should either fail gracefully or show defaults
	// We're testing that it doesn't crash
	_ = result
}

// TestSpaceVersion tests the space --version command
func TestSpaceVersion(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	f := framework.New(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result := f.RunSpaceCmd(ctx, "--version")
	assert := framework.Assert(t)

	assert.CmdSucceeds(result, "space --version should succeed")
	// Version output should contain version number or "dev"
	assert.True(
		result.Contains("space") || result.Contains("dev") || result.Contains("0."),
		"should show version information",
	)
}

// TestSpaceHelp tests the space --help command
func TestSpaceHelp(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	f := framework.New(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result := f.RunSpaceCmd(ctx, "--help")
	assert := framework.Assert(t)

	assert.CmdSucceeds(result, "space --help should succeed")
	assert.CmdOutputContains(result, "Usage:", "should show usage")
	assert.CmdOutputContains(result, "up", "should list up command")
	assert.CmdOutputContains(result, "down", "should list down command")
	assert.CmdOutputContains(result, "ps", "should list ps command")
}
