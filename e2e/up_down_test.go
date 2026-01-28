//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/happy-sdk/space-cli/e2e/framework"
)

// TestSpaceUpSimple tests the basic space up command with a simple app
func TestSpaceUpSimple(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	f := framework.New(t).WithFixture("simple-app")
	defer f.Cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Test space up
	result := f.RunSpaceCmd(ctx, "up")
	assert := framework.Assert(t)

	assert.CmdSucceeds(result, "space up should succeed")
	assert.CmdOutputContains(result, "Starting services", "should show starting message")
	assert.CmdOutputContains(result, "Services started successfully", "should show success message")

	// Verify web service is accessible
	err := f.WaitForService(ctx, "http://localhost:8080", 30*time.Second)
	assert.NoError(err, "web service should be accessible")

	// Test space down
	downResult := f.RunSpaceCmd(ctx, "down")
	assert.CmdSucceeds(downResult, "space down should succeed")
}

// TestSpaceUpMultiService tests space up with multiple services
func TestSpaceUpMultiService(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	f := framework.New(t).WithFixture("multi-service")
	defer f.Cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	// Test space up
	result := f.RunSpaceCmd(ctx, "up")
	assert := framework.Assert(t)

	assert.CmdSucceeds(result, "space up should succeed")

	// Wait for all services to be healthy
	t.Log("Waiting for services to be healthy...")

	// Frontend service
	err := f.WaitForService(ctx, "http://localhost:8080", 60*time.Second)
	assert.NoError(err, "frontend service should be accessible")

	// API service
	err = f.WaitForService(ctx, "http://localhost:3000/health", 60*time.Second)
	assert.NoError(err, "api service should be accessible")

	// Cleanup
	downResult := f.RunSpaceCmd(ctx, "down")
	assert.CmdSucceeds(downResult, "space down should succeed")
}

// TestSpaceUpSelectiveServices tests starting specific services
func TestSpaceUpSelectiveServices(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	f := framework.New(t).WithFixture("multi-service")
	defer f.Cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Start only redis service
	result := f.RunSpaceCmd(ctx, "up", "redis")
	assert := framework.Assert(t)

	assert.CmdSucceeds(result, "space up redis should succeed")
	assert.CmdOutputContains(result, "Starting services: redis", "should show redis being started")

	// Cleanup
	f.RunSpaceCmd(ctx, "down")
}

// TestSpaceUpIdempotent tests that running up twice works correctly
func TestSpaceUpIdempotent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	f := framework.New(t).WithFixture("simple-app")
	defer f.Cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	assert := framework.Assert(t)

	// First up
	result1 := f.RunSpaceCmd(ctx, "up")
	assert.CmdSucceeds(result1, "first space up should succeed")

	// Wait for service
	err := f.WaitForService(ctx, "http://localhost:8080", 30*time.Second)
	assert.NoError(err, "service should be accessible after first up")

	// Second up (should be idempotent)
	result2 := f.RunSpaceCmd(ctx, "up")
	assert.CmdSucceeds(result2, "second space up should succeed")

	// Service should still be accessible
	err = f.WaitForService(ctx, "http://localhost:8080", 10*time.Second)
	assert.NoError(err, "service should still be accessible after second up")

	// Cleanup
	f.RunSpaceCmd(ctx, "down")
}

// TestSpaceDown tests the space down command
func TestSpaceDown(t *testing.T) {
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

	// Bring down services
	downResult := f.RunSpaceCmd(ctx, "down")
	assert.CmdSucceeds(downResult, "space down should succeed")
	assert.CmdOutputContains(downResult, "Services stopped", "should show stopped message")

	// Verify service is no longer accessible (give it a moment to stop)
	time.Sleep(2 * time.Second)
	err = f.WaitForService(ctx, "http://localhost:8080", 5*time.Second)
	assert.Error(err, "service should not be accessible after down")
}

// TestSpaceDownIdempotent tests that running down twice works correctly
func TestSpaceDownIdempotent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	f := framework.New(t).WithFixture("simple-app")
	defer f.Cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	assert := framework.Assert(t)

	// Start and stop services
	f.RunSpaceCmd(ctx, "up")
	f.RunSpaceCmd(ctx, "down")

	// Run down again (should be idempotent)
	result := f.RunSpaceCmd(ctx, "down")
	assert.CmdSucceeds(result, "second space down should succeed")
}
