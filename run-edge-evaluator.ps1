# --------------------------------------------
# run-edge-evaluator.ps1
# Loads .env and starts the Edge Evaluator
# --------------------------------------------

# Load variables from .env
Get-Content ".env" |
  Where-Object { $_ -match '^\s*\w+=' } |
  ForEach-Object {
      $kv = $_ -split '=',2
      [Environment]::SetEnvironmentVariable($kv[0], $kv[1])
      Write-Host "Set $($kv[0])"
  }

# Override port for Edge Evaluator (8081 instead of 8080)
[Environment]::SetEnvironmentVariable("FF_SERVER_PORT", "8081")
Write-Host "Set FF_SERVER_PORT=8081 (Edge Evaluator)"

# Run edge evaluator (port 8081)
go run ./cmd/edge-evaluator
