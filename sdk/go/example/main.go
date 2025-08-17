package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	featureflags "github.com/Sidd-007/feature-flag-platform/sdk/go"
)

func main() {
	// Create SDK configuration
	config := featureflags.DefaultConfig()
	config.APIKey = "your-api-key-here"
	config.Environment = "development"
	config.EvaluatorEndpoint = "http://localhost:8081"
	config.EventsEndpoint = "http://localhost:8083"

	// Optional: customize configuration
	config.CacheTTL = 5 * time.Minute
	config.StreamingEnabled = true
	config.EventsEnabled = true
	config.OfflineEnabled = true
	config.LogLevel = "info"

	// Create client
	client, err := featureflags.NewClient(config)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Start the client
	ctx := context.Background()
	if err := client.Start(ctx); err != nil {
		log.Fatalf("Failed to start client: %v", err)
	}

	// Wait for client to be ready (optional)
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := client.WaitForReady(ctx); err != nil {
		log.Printf("Client not ready: %v", err)
	} else {
		log.Println("Client is ready")
	}

	// Create user context
	user := &featureflags.UserContext{
		UserID:   "user-123",
		Email:    "user@example.com",
		Country:  "US",
		Platform: "web",
		Attributes: map[string]interface{}{
			"plan":        "premium",
			"signup_date": "2024-01-15",
			"trial_days":  30,
		},
	}

	// Example 1: Evaluate a boolean flag
	showNewFeature, err := client.EvaluateBoolFlag(ctx, "show-new-feature", user, false)
	if err != nil {
		log.Printf("Error evaluating boolean flag: %v", err)
	} else {
		fmt.Printf("Show new feature: %t\n", showNewFeature)
	}

	// Example 2: Evaluate a string flag
	theme, err := client.EvaluateStringFlag(ctx, "ui-theme", user, "light")
	if err != nil {
		log.Printf("Error evaluating string flag: %v", err)
	} else {
		fmt.Printf("UI Theme: %s\n", theme)
	}

	// Example 3: Evaluate an integer flag
	maxItems, err := client.EvaluateIntFlag(ctx, "max-items-per-page", user, 10)
	if err != nil {
		log.Printf("Error evaluating integer flag: %v", err)
	} else {
		fmt.Printf("Max items per page: %d\n", maxItems)
	}

	// Example 4: Evaluate a JSON flag
	apiConfig, err := client.EvaluateJSONFlag(ctx, "api-config", user, map[string]interface{}{
		"timeout": 5000,
		"retries": 3,
	})
	if err != nil {
		log.Printf("Error evaluating JSON flag: %v", err)
	} else {
		fmt.Printf("API Config: %+v\n", apiConfig)
	}

	// Example 5: Get detailed evaluation result
	result, err := client.EvaluateFlag(ctx, "show-new-feature", user, false)
	if err != nil {
		log.Printf("Error getting evaluation result: %v", err)
	} else {
		fmt.Printf("Detailed result:\n")
		fmt.Printf("  Flag: %s\n", result.FlagKey)
		fmt.Printf("  Value: %v\n", result.Value)
		fmt.Printf("  Variation: %s\n", result.VariationID)
		fmt.Printf("  Reason: %s\n", result.Reason)
		fmt.Printf("  Cache Hit: %t\n", result.CacheHit)
		fmt.Printf("  Default Used: %t\n", result.DefaultUsed)
		fmt.Printf("  Evaluated At: %s\n", result.EvaluatedAt.Format(time.RFC3339))
	}

	// Example 6: Track a custom event
	err = client.TrackEvent(ctx, "feature_used", user, map[string]interface{}{
		"feature_name": "new-dashboard",
		"duration_ms":  1250,
		"success":      true,
	})
	if err != nil {
		log.Printf("Error tracking event: %v", err)
	} else {
		fmt.Println("Event tracked successfully")
	}

	// Example 7: Track a metric
	err = client.TrackMetric(ctx, "page_load_time", 1.25, user, map[string]interface{}{
		"page":    "dashboard",
		"browser": "chrome",
	})
	if err != nil {
		log.Printf("Error tracking metric: %v", err)
	} else {
		fmt.Println("Metric tracked successfully")
	}

	// Example 8: Check offline status
	if client.IsOffline() {
		fmt.Println("Client is in offline mode")
	} else {
		fmt.Println("Client is online")
	}

	// Example 9: Toggle offline mode (for testing)
	fmt.Println("Testing offline mode...")
	client.SetOffline(true)
	time.Sleep(1 * time.Second)

	// Evaluate flag in offline mode
	offlineResult, err := client.EvaluateBoolFlag(ctx, "show-new-feature", user, false)
	if err != nil {
		log.Printf("Error evaluating flag in offline mode: %v", err)
	} else {
		fmt.Printf("Offline evaluation result: %t\n", offlineResult)
	}

	// Return to online mode
	client.SetOffline(false)

	// Example 10: Flush events before closing
	if err := client.Flush(ctx); err != nil {
		log.Printf("Error flushing events: %v", err)
	} else {
		fmt.Println("Events flushed successfully")
	}

	fmt.Println("Example completed successfully!")
}

// Example of using the SDK in a web application
func webApplicationExample() {
	// Initialize client once at application startup
	config := featureflags.DefaultConfig()
	config.APIKey = "your-api-key-here"
	config.Environment = "production"

	client, err := featureflags.NewClient(config)
	if err != nil {
		log.Fatalf("Failed to create feature flags client: %v", err)
	}
	defer client.Close()

	// Start the client
	ctx := context.Background()
	if err := client.Start(ctx); err != nil {
		log.Fatalf("Failed to start feature flags client: %v", err)
	}

	// Use in request handlers
	http.HandleFunc("/api/dashboard", func(w http.ResponseWriter, r *http.Request) {
		// Create user context from request
		user := &featureflags.UserContext{
			UserID:   getUserIDFromRequest(r),
			Email:    getEmailFromRequest(r),
			Country:  getCountryFromRequest(r),
			Platform: "web",
		}

		// Check feature flags
		showNewDashboard, _ := client.EvaluateBoolFlag(r.Context(), "new-dashboard", user, false)
		dashboardTheme, _ := client.EvaluateStringFlag(r.Context(), "dashboard-theme", user, "light")
		maxWidgets, _ := client.EvaluateIntFlag(r.Context(), "max-dashboard-widgets", user, 6)

		// Use flags in your application logic
		response := map[string]interface{}{
			"show_new_dashboard": showNewDashboard,
			"theme":              dashboardTheme,
			"max_widgets":        maxWidgets,
		}

		// Track usage
		client.TrackEvent(r.Context(), "dashboard_viewed", user, map[string]interface{}{
			"new_dashboard": showNewDashboard,
			"theme":         dashboardTheme,
		})

		// Return response (simplified)
		fmt.Fprintf(w, "Dashboard config: %+v", response)
	})

	// Start web server
	log.Println("Starting web server on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// Helper functions for web example
func getUserIDFromRequest(r *http.Request) string {
	// Extract user ID from JWT token, session, etc.
	return "user-123"
}

func getEmailFromRequest(r *http.Request) string {
	// Extract email from JWT token, session, etc.
	return "user@example.com"
}

func getCountryFromRequest(r *http.Request) string {
	// Extract country from IP geolocation, headers, etc.
	return "US"
}

// Example of using the SDK with context and timeouts
func advancedUsageExample() {
	config := featureflags.DefaultConfig()
	config.APIKey = "your-api-key-here"
	config.Environment = "production"
	config.EvaluationTimeout = 50 * time.Millisecond // Fast timeout

	client, err := featureflags.NewClient(config)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	client.Start(ctx)

	user := &featureflags.UserContext{
		UserID: "user-456",
	}

	// Evaluate with custom timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	result, err := client.EvaluateFlag(ctx, "experimental-feature", user, false)
	if err != nil {
		if err == context.DeadlineExceeded {
			log.Println("Flag evaluation timed out, using default value")
		} else {
			log.Printf("Flag evaluation failed: %v", err)
		}
	} else {
		fmt.Printf("Feature enabled: %v (reason: %s)\n", result.Value, result.Reason)
	}
}

// Example of handling different flag types
func flagTypesExample() {
	config := featureflags.DefaultConfig()
	config.APIKey = "your-api-key-here"
	config.Environment = "development"

	client, err := featureflags.NewClient(config)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	client.Start(ctx)

	user := &featureflags.UserContext{
		UserID: "user-789",
		Attributes: map[string]interface{}{
			"beta_user": true,
			"plan":      "enterprise",
		},
	}

	// Boolean flags
	if enabled, _ := client.EvaluateBoolFlag(ctx, "feature-enabled", user, false); enabled {
		fmt.Println("Feature is enabled")
	}

	// String flags for configuration
	apiVersion, _ := client.EvaluateStringFlag(ctx, "api-version", user, "v1")
	fmt.Printf("Using API version: %s\n", apiVersion)

	// Numeric flags for limits and thresholds
	rateLimit, _ := client.EvaluateIntFlag(ctx, "api-rate-limit", user, 1000)
	fmt.Printf("API rate limit: %d requests/hour\n", rateLimit)

	// JSON flags for complex configuration
	dbConfig, _ := client.EvaluateJSONFlag(ctx, "database-config", user, map[string]interface{}{
		"host":            "localhost",
		"port":            5432,
		"max_connections": 10,
	})

	if config, ok := dbConfig.(map[string]interface{}); ok {
		fmt.Printf("Database config: host=%s, port=%v, max_connections=%v\n",
			config["host"], config["port"], config["max_connections"])
	}
}
