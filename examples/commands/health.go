// Health check all services
//
// Usage: space run health

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

type Context struct {
	ProjectName string `json:"project_name"`
	Services    map[string]struct {
		Name string `json:"name"`
		URL  string `json:"url"`
		Port int    `json:"internal_port"`
	} `json:"services"`
}

func main() {
	var ctx Context
	if err := json.NewDecoder(os.Stdin).Decode(&ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading context: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Health check for %s\n\n", ctx.ProjectName)

	client := &http.Client{Timeout: 5 * time.Second}
	allHealthy := true

	for name, svc := range ctx.Services {
		// Try common health endpoints
		endpoints := []string{"/health", "/healthz", "/api/health", "/"}
		healthy := false

		for _, endpoint := range endpoints {
			url := svc.URL + endpoint
			resp, err := client.Get(url)
			if err == nil && resp.StatusCode < 400 {
				fmt.Printf("✅ %s: healthy (%s -> %d)\n", name, endpoint, resp.StatusCode)
				resp.Body.Close()
				healthy = true
				break
			}
			if resp != nil {
				resp.Body.Close()
			}
		}

		if !healthy {
			fmt.Printf("❌ %s: unhealthy or unreachable\n", name)
			allHealthy = false
		}
	}

	fmt.Println()
	if allHealthy {
		fmt.Println("All services healthy!")
	} else {
		fmt.Println("Some services are unhealthy.")
		os.Exit(1)
	}
}
