package main

import (
	"fmt"
	"log"
	"os"
	"github.com/feature-flag-platform/pkg/config"
)

func main() {
	fmt.Println("Environment variable FF_AUTH_JWT_SECRET:", os.Getenv("FF_AUTH_JWT_SECRET"))
	
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	
	fmt.Println("Config loaded successfully!")
	fmt.Println("JWT Secret in config:", cfg.Auth.JWTSecret)
}
