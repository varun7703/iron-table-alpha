$ErrorActionPreference = 'Stop'
Set-Location $PSScriptRoot
$localGo = Join-Path $PSScriptRoot '.tools/go/bin/go.exe'
if (Test-Path $localGo) { $goCommand = $localGo } else { $goCommand = 'go' }
$env:GOPATH = Join-Path $PSScriptRoot '.tools/gopath'
$env:GOCACHE = Join-Path $PSScriptRoot '.tools/gocache'
if (-not (Test-Path node_modules)) { npm install; if ($LASTEXITCODE -ne 0) { throw 'Dependency installation failed' } }
npm run build
if ($LASTEXITCODE -ne 0) { throw 'Browser build failed' }
New-Item -ItemType Directory -Force .tools | Out-Null
& $goCommand -C server build -o ../.tools/iron-table.exe .
if ($LASTEXITCODE -ne 0) { throw 'Go build failed' }
Write-Host 'Open http://localhost:8080 — share your host address with players on the same network.'
& ./.tools/iron-table.exe
