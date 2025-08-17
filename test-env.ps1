# Load environment variables from .env file
Get-Content .env | ForEach-Object {
    if ($_ -match '^([^#][^=]+)=(.*)$') {
        $name = $matches[1]
        $value = $matches[2]
        [Environment]::SetEnvironmentVariable($name, $value, [System.EnvironmentVariableTarget]::Process)
    }
}

# Test the environment variable
Write-Host "FF_AUTH_JWT_SECRET: $env:FF_AUTH_JWT_SECRET"

# Create a simple Go program to test config loading
@"
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
"@ | Out-File -FilePath test-config.go -Encoding utf8

go run test-config.go
