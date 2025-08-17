package main

import (
	"context"
	"fmt"
	"log"
	"time"

	featureflags "github.com/Sidd-007/feature-flag-platform/sdk/go"
)

func main() {
	// Create SDK configuration
	config := &featureflags.Config{
		APIKey:      "ff_your_api_key_here", // Replace with your actual API key
		Environment: "development",          // Environment key from your setup

		// Point to your edge evaluator endpoint
		EvaluatorEndpoint: "http://localhost:8081",
		EventsEndpoint:    "http://localhost:8083",

		// Optional: Configure caching
		CacheEnabled: true,
		CacheTTL:     5 * time.Minute,

		// Optional: Configure streaming updates
		StreamingEnabled: true,

		// Optional: Configure events
		EventsEnabled: true,

		// Optional: Configure timeouts
		EvaluationTimeout: 200 * time.Millisecond,
		HTTPTimeout:       5 * time.Second,

		// Optional: Configure logging
		LogLevel: "info",
	}

	// Create the feature flags client
	client, err := featureflags.NewClient(config)
	if err != nil {
		log.Fatalf("Failed to create feature flags client: %v", err)
	}
	defer client.Close()

	// Start the client (starts background services)
	ctx := context.Background()
	if err := client.Start(ctx); err != nil {
		log.Fatalf("Failed to start feature flags client: %v", err)
	}

	// Create user context
	userContext := &featureflags.UserContext{
		UserID:   "user-123",
		Email:    "user@example.com",
		Country:  "US",
		Platform: "web",
		Attributes: map[string]interface{}{
			"plan":         "premium",
			"signup_date":  "2023-01-15",
			"feature_beta": true,
		},
	}

	// Example 1: Evaluate a boolean flag
	fmt.Println("=== Boolean Flag Example ===")
	showNewFeature, err := client.BooleanFlag("show-new-feature", userContext, false)
	if err != nil {
		log.Printf("Error evaluating boolean flag: %v", err)
	} else {
		fmt.Printf("show-new-feature: %t\n", showNewFeature)

		if showNewFeature {
			fmt.Println("✅ Showing new feature to user")
		} else {
			fmt.Println("❌ Hiding new feature from user")
		}
	}

	// Example 2: Evaluate a string flag
	fmt.Println("\n=== String Flag Example ===")
	buttonColor, err := client.StringFlag("button-color", userContext, "blue")
	if err != nil {
		log.Printf("Error evaluating string flag: %v", err)
	} else {
		fmt.Printf("button-color: %s\n", buttonColor)
		fmt.Printf("🎨 Using %s button color\n", buttonColor)
	}

	// Example 3: Evaluate a number flag
	fmt.Println("\n=== Number Flag Example ===")
	maxItems, err := client.NumberFlag("max-items-per-page", userContext, 10)
	if err != nil {
		log.Printf("Error evaluating number flag: %v", err)
	} else {
		fmt.Printf("max-items-per-page: %.0f\n", maxItems)
		fmt.Printf("📄 Showing %.0f items per page\n", maxItems)
	}

	// Example 4: Evaluate a JSON flag
	fmt.Println("\n=== JSON Flag Example ===")
	defaultConfig := map[string]interface{}{
		"theme":    "light",
		"language": "en",
		"features": []string{"basic"},
	}

	appConfig, err := client.JSONFlag("app-config", userContext, defaultConfig)
	if err != nil {
		log.Printf("Error evaluating JSON flag: %v", err)
	} else {
		fmt.Printf("app-config: %+v\n", appConfig)
		if config, ok := appConfig.(map[string]interface{}); ok {
			fmt.Printf("⚙️  Using theme: %v, language: %v\n", config["theme"], config["language"])
		}
	}

	// Example 5: Evaluate multiple flags at once
	fmt.Println("\n=== Multiple Flags Example ===")
	flagKeys := []string{"show-new-feature", "button-color", "max-items-per-page"}
	defaults := map[string]interface{}{
		"show-new-feature":   false,
		"button-color":       "blue",
		"max-items-per-page": 10,
	}

	results, err := client.EvaluateMultiple(ctx, flagKeys, userContext, defaults)
	if err != nil {
		log.Printf("Error evaluating multiple flags: %v", err)
	} else {
		fmt.Println("Multiple flag evaluation results:")
		for flagKey, result := range results {
			fmt.Printf("  %s: %v (reason: %s, cache_hit: %t)\n",
				flagKey, result.Value, result.Reason, result.CacheHit)
		}
	}

	// Example 6: Track custom events
	fmt.Println("\n=== Custom Event Example ===")
	err = client.TrackEvent(ctx, &featureflags.Event{
		Type:      featureflags.EventTypeCustom,
		UserID:    userContext.UserID,
		EventName: "button_clicked",
		Properties: map[string]interface{}{
			"button_color": buttonColor,
			"page":         "homepage",
			"timestamp":    time.Now().Unix(),
		},
	})
	if err != nil {
		log.Printf("Error tracking custom event: %v", err)
	} else {
		fmt.Println("📊 Custom event tracked successfully")
	}

	fmt.Println("\n=== SDK Usage Complete ===")
	fmt.Println("💡 Pro tips:")
	fmt.Println("  - Use streaming for real-time updates")
	fmt.Println("  - Enable caching to reduce latency")
	fmt.Println("  - Track exposure events for analytics")
	fmt.Println("  - Set up offline mode for resilience")
}
