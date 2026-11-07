param (
    [Parameter(Position = 0)]
    [ValidateSet("test", "test-race", "build", "run", "tidy", "clean", "help")]
    [string]$Target = "build"
)

$ErrorActionPreference = "Stop"

function Check-LastExitCode {
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Operasi gagal dengan kode keluar $LASTEXITCODE"
        exit $LASTEXITCODE
    }
}

switch ($Target) {
    "test" {
        Write-Host "Menjalankan unit test..."
        go test -short -v ./...
        Check-LastExitCode
    }
    "test-race" {
        Write-Host "Menjalankan unit test dengan race detector..."
        go test -short -v -race ./...
        Check-LastExitCode
    }
    "build" {
        Write-Host "Melakukan kompilasi binary ke bin/server.exe..."
        if (-not (Test-Path -Path "bin")) {
            New-Item -ItemType Directory -Path "bin" | Out-Null
        }
        go build -o bin/server.exe ./cmd/server
        Check-LastExitCode
        Write-Host "Kompilasi selesai: bin/server.exe"
    }
    "run" {
        Write-Host "Menjalankan server lokal..."
        go run ./cmd/server
        Check-LastExitCode
    }
    "tidy" {
        Write-Host "Merapikan dependensi modul Go..."
        go mod tidy
        Check-LastExitCode
        go mod verify
        Check-LastExitCode
    }
    "clean" {
        Write-Host "Menghapus direktori bin/..."
        if (Test-Path -Path "bin") {
            Remove-Item -Recurse -Force -Path "bin"
        }
        Write-Host "Pembersihan selesai."
    }
    "help" {
        Write-Host "Penggunaan: .\build.ps1 [target]"
        Write-Host "Target yang tersedia:"
        Write-Host "  test      - Menjalankan unit test"
        Write-Host "  test-race - Menjalankan unit test dengan race detector"
        Write-Host "  build     - Melakukan kompilasi binary ke bin/server.exe"
        Write-Host "  run       - Menjalankan server lokal"
        Write-Host "  tidy      - Merapikan dependensi go.mod dan go.sum"
        Write-Host "  clean     - Menghapus direktori bin/"
        Write-Host "  help      - Menampilkan bantuan ini"
    }
}
