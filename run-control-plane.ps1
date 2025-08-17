# Load environment variables from .env file and run control plane
Get-Content .env | ForEach-Object {
    if ($_ -match '^([^#][^=]+)=(.*)$') {
        $name = $matches[1]
        $value = $matches[2]
        [Environment]::SetEnvironmentVariable($name, $value, [System.EnvironmentVariableTarget]::Process)
        Write-Host "Set $name"
    }
}

# Run the control plane
go run ./cmd/control-plane
