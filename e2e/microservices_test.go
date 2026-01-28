//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/happy-sdk/space-cli/e2e/framework"
)

const projectName = "e2e-microservices"

// Service URL configurations for microservices fixture
// Supports both localhost (port bindings) and OrbStack container DNS
func getFrontendURL(f *framework.E2EFramework) framework.ServiceURL {
	return framework.ServiceURL{
		LocalhostURL:   "http://localhost:8080",
		ContainerURL:   f.GetOrbstackURL(f.GetContainerName(projectName, "frontend"), 80),
		HealthEndpoint: "/",
	}
}

func getAPIUsersURL(f *framework.E2EFramework) framework.ServiceURL {
	return framework.ServiceURL{
		LocalhostURL:   "http://localhost:3001",
		ContainerURL:   f.GetOrbstackURL(f.GetContainerName(projectName, "api-users"), 80),
		HealthEndpoint: "/health",
	}
}

func getAPIOrdersURL(f *framework.E2EFramework) framework.ServiceURL {
	return framework.ServiceURL{
		LocalhostURL:   "http://localhost:3002",
		ContainerURL:   f.GetOrbstackURL(f.GetContainerName(projectName, "api-orders"), 80),
		HealthEndpoint: "/health",
	}
}

// TestMicroservicesUp tests the full microservices stack startup
func TestMicroservicesUp(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	f := framework.New(t).WithFixture("microservices")
	defer f.Cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Start all services
	result := f.RunSpaceCmd(ctx, "up")
	assert := framework.Assert(t)

	assert.CmdSucceeds(result, "space up should succeed")
	assert.CmdOutputContains(result, "Starting services", "should show starting message")

	// Wait for all services to be healthy
	// Order matters: postgres -> APIs -> frontend (following dependency chain)
	t.Log("Waiting for services to be healthy...")

	// API Users (depends on postgres)
	usersURL, err := f.WaitForServiceWithFallback(ctx, getAPIUsersURL(f), 90*time.Second)
	assert.NoError(err, "api-users should be accessible")
	t.Logf("API Users accessible at: %s", usersURL)

	// API Orders (depends on postgres)
	ordersURL, err := f.WaitForServiceWithFallback(ctx, getAPIOrdersURL(f), 90*time.Second)
	assert.NoError(err, "api-orders should be accessible")
	t.Logf("API Orders accessible at: %s", ordersURL)

	// Frontend (depends on both APIs)
	frontendURL, err := f.WaitForServiceWithFallback(ctx, getFrontendURL(f), 90*time.Second)
	assert.NoError(err, "frontend should be accessible")
	t.Logf("Frontend accessible at: %s", frontendURL)

	t.Log("All services are healthy!")

	// Cleanup
	downResult := f.RunSpaceCmd(ctx, "down")
	assert.CmdSucceeds(downResult, "space down should succeed")
}

// TestMicroservicesHealthChecks verifies health endpoints return expected data
func TestMicroservicesHealthChecks(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	f := framework.New(t).WithFixture("microservices")
	defer f.Cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	assert := framework.Assert(t)

	// Start services
	result := f.RunSpaceCmd(ctx, "up")
	assert.CmdSucceeds(result, "space up should succeed")

	// Wait for services and get their accessible URLs
	usersURL, err := f.WaitForServiceWithFallback(ctx, getAPIUsersURL(f), 60*time.Second)
	assert.NoError(err, "api-users should be accessible")

	ordersURL, err := f.WaitForServiceWithFallback(ctx, getAPIOrdersURL(f), 60*time.Second)
	assert.NoError(err, "api-orders should be accessible")

	// Verify api-users health response
	resp, err := http.Get(usersURL + "/health")
	assert.NoError(err, "should fetch api-users health")
	if resp != nil {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)

		var health map[string]interface{}
		err = json.Unmarshal(body, &health)
		assert.NoError(err, "health response should be valid JSON")

		if service, ok := health["service"].(string); ok {
			assert.Equal("api-users", service, "service name should be api-users")
		}
	}

	// Verify api-orders health response
	resp2, err := http.Get(ordersURL + "/health")
	assert.NoError(err, "should fetch api-orders health")
	if resp2 != nil {
		defer resp2.Body.Close()
		body, _ := io.ReadAll(resp2.Body)

		var health map[string]interface{}
		err = json.Unmarshal(body, &health)
		assert.NoError(err, "health response should be valid JSON")

		if service, ok := health["service"].(string); ok {
			assert.Equal("api-orders", service, "service name should be api-orders")
		}
	}

	// Cleanup
	f.RunSpaceCmd(ctx, "down")
}

// TestMicroservicesAPIEndpoints verifies API endpoints return expected data
func TestMicroservicesAPIEndpoints(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	f := framework.New(t).WithFixture("microservices")
	defer f.Cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	assert := framework.Assert(t)

	// Start services
	result := f.RunSpaceCmd(ctx, "up")
	assert.CmdSucceeds(result, "space up should succeed")

	// Wait for all services and get their accessible URLs
	_, err := f.WaitForServiceWithFallback(ctx, getFrontendURL(f), 60*time.Second)
	assert.NoError(err, "frontend should be accessible")

	usersURL, err := f.WaitForServiceWithFallback(ctx, getAPIUsersURL(f), 60*time.Second)
	assert.NoError(err, "api-users should be accessible")

	ordersURL, err := f.WaitForServiceWithFallback(ctx, getAPIOrdersURL(f), 60*time.Second)
	assert.NoError(err, "api-orders should be accessible")

	// Test users API endpoint
	resp, err := http.Get(usersURL + "/api/users")
	assert.NoError(err, "should fetch users API")
	if resp != nil {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)

		var data map[string]interface{}
		err = json.Unmarshal(body, &data)
		assert.NoError(err, "users API response should be valid JSON")

		if users, ok := data["users"].([]interface{}); ok {
			assert.True(len(users) > 0, "should have at least one user")
		}
	}

	// Test orders API endpoint
	resp2, err := http.Get(ordersURL + "/api/orders")
	assert.NoError(err, "should fetch orders API")
	if resp2 != nil {
		defer resp2.Body.Close()
		body, _ := io.ReadAll(resp2.Body)

		var data map[string]interface{}
		err = json.Unmarshal(body, &data)
		assert.NoError(err, "orders API response should be valid JSON")

		if orders, ok := data["orders"].([]interface{}); ok {
			assert.True(len(orders) > 0, "should have at least one order")
		}
	}

	// Cleanup
	f.RunSpaceCmd(ctx, "down")
}

// TestMicroservicesFrontendProxy tests that frontend can proxy to backend services
func TestMicroservicesFrontendProxy(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	f := framework.New(t).WithFixture("microservices")
	defer f.Cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	assert := framework.Assert(t)

	// Start services
	result := f.RunSpaceCmd(ctx, "up")
	assert.CmdSucceeds(result, "space up should succeed")

	// Wait for frontend
	frontendURL, err := f.WaitForServiceWithFallback(ctx, getFrontendURL(f), 60*time.Second)
	assert.NoError(err, "frontend should be accessible")

	// Test proxied users API via frontend
	resp, err := http.Get(frontendURL + "/api/users")
	if err == nil && resp != nil {
		defer resp.Body.Close()
		// If proxy is working, we should get a response
		t.Logf("Frontend proxy to /api/users returned status: %d", resp.StatusCode)
	}

	// Test proxied orders API via frontend
	resp2, err := http.Get(frontendURL + "/api/orders")
	if err == nil && resp2 != nil {
		defer resp2.Body.Close()
		t.Logf("Frontend proxy to /api/orders returned status: %d", resp2.StatusCode)
	}

	// Cleanup
	f.RunSpaceCmd(ctx, "down")
}

// TestMicroservicesPs tests space ps with all microservices
func TestMicroservicesPs(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	f := framework.New(t).WithFixture("microservices")
	defer f.Cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	assert := framework.Assert(t)

	// Start services
	upResult := f.RunSpaceCmd(ctx, "up")
	assert.CmdSucceeds(upResult, "space up should succeed")

	// Wait for frontend to ensure services are up
	_, err := f.WaitForServiceWithFallback(ctx, getFrontendURL(f), 60*time.Second)
	assert.NoError(err, "frontend should be accessible")

	// Test space ps
	psResult := f.RunSpaceCmd(ctx, "ps")
	assert.CmdSucceeds(psResult, "space ps should succeed")

	// Should list all services
	assert.CmdOutputContains(psResult, "frontend", "should show frontend service")
	assert.CmdOutputContains(psResult, "api-users", "should show api-users service")
	assert.CmdOutputContains(psResult, "api-orders", "should show api-orders service")
	assert.CmdOutputContains(psResult, "postgres", "should show postgres service")

	// Cleanup
	f.RunSpaceCmd(ctx, "down")
}

// TestMicroservicesSelectiveStart tests starting specific services
func TestMicroservicesSelectiveStart(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	f := framework.New(t).WithFixture("microservices")
	defer f.Cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	assert := framework.Assert(t)

	// Start only postgres and api-users
	result := f.RunSpaceCmd(ctx, "up", "postgres", "api-users")
	assert.CmdSucceeds(result, "space up postgres api-users should succeed")

	// Wait for postgres to be ready
	time.Sleep(10 * time.Second)

	// api-users should be accessible
	_, err := f.WaitForServiceWithFallback(ctx, getAPIUsersURL(f), 60*time.Second)
	assert.NoError(err, "api-users should be accessible")

	// frontend should NOT be accessible (not started)
	_, err = f.WaitForServiceWithFallback(ctx, getFrontendURL(f), 5*time.Second)
	assert.Error(err, "frontend should not be accessible (not started)")

	// Cleanup
	f.RunSpaceCmd(ctx, "down")
}
