//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/happy-sdk/space-cli/e2e/framework"
)

// TestSpacePsNoServices tests space ps when no services are running
func TestSpacePsNoServices(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	f := framework.New(t).WithFixture("simple-app")
	defer f.Cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Make sure nothing is running
	f.RunSpaceCmd(ctx, "down")

	// Test space ps with no running services
	result := f.RunSpaceCmd(ctx, "ps")
	assert := framework.Assert(t)

	assert.CmdSucceeds(result, "space ps should succeed even with no services")
}

// TestSpacePsWithServices tests space ps when services are running
func TestSpacePsWithServices(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	f := framework.New(t).WithFixture("simple-app")
	defer f.Cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	assert := framework.Assert(t)

	// Start services
	upResult := f.RunSpaceCmd(ctx, "up")
	assert.CmdSucceeds(upResult, "space up should succeed")

	// Wait for service to be healthy
	err := f.WaitForService(ctx, "http://localhost:8080", 30*time.Second)
	assert.NoError(err, "service should be accessible")

	// Test space ps
	psResult := f.RunSpaceCmd(ctx, "ps")
	assert.CmdSucceeds(psResult, "space ps should succeed")
	assert.CmdOutputContains(psResult, "web", "should show web service")

	// Cleanup
	f.RunSpaceCmd(ctx, "down")
}

// TestSpacePsMultiService tests space ps with multiple services
func TestSpacePsMultiService(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	f := framework.New(t).WithFixture("multi-service")
	defer f.Cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	assert := framework.Assert(t)

	// Start services
	upResult := f.RunSpaceCmd(ctx, "up")
	assert.CmdSucceeds(upResult, "space up should succeed")

	// Wait for services to be healthy
	err := f.WaitForService(ctx, "http://localhost:8080", 60*time.Second)
	assert.NoError(err, "frontend service should be accessible")

	// Test space ps
	psResult := f.RunSpaceCmd(ctx, "ps")
	assert.CmdSucceeds(psResult, "space ps should succeed")

	// Should list all services
	assert.CmdOutputContains(psResult, "api", "should show api service")
	assert.CmdOutputContains(psResult, "frontend", "should show frontend service")
	assert.CmdOutputContains(psResult, "postgres", "should show postgres service")
	assert.CmdOutputContains(psResult, "redis", "should show redis service")

	// Cleanup
	f.RunSpaceCmd(ctx, "down")
}

// TestSpacePsJsonOutput tests space ps with JSON output format
func TestSpacePsJsonOutput(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	f := framework.New(t).WithFixture("simple-app")
	defer f.Cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	assert := framework.Assert(t)

	// Start services
	upResult := f.RunSpaceCmd(ctx, "up")
	assert.CmdSucceeds(upResult, "space up should succeed")

	// Wait for service
	err := f.WaitForService(ctx, "http://localhost:8080", 30*time.Second)
	assert.NoError(err, "service should be accessible")

	// Test space ps --json
	psResult := f.RunSpaceCmd(ctx, "ps", "--json")
	assert.CmdSucceeds(psResult, "space ps --json should succeed")

	// Output should be valid JSON (contains array brackets or object braces)
	assert.True(
		psResult.Contains("[") || psResult.Contains("{"),
		"JSON output should contain array or object",
	)

	// Cleanup
	f.RunSpaceCmd(ctx, "down")
}
