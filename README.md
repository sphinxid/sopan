# SOPAN (SOCKS Proxy Analyzer)

[![Release](https://img.shields.io/github/v/release/sphinxid/sopan)](https://github.com/sphinxid/sopan/releases)
[![Test](https://github.com/sphinxid/sopan/actions/workflows/test.yml/badge.svg)](https://github.com/sphinxid/sopan/actions/workflows/test.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/sphinxid/sopan)](https://goreportcard.com/report/github.com/sphinxid/sopan)
[![License](https://img.shields.io/github/license/sphinxid/sopan)](LICENSE)

**SOPAN** stands for **SOCKS Proxy Analyzer** - also an Indonesian word meaning "polite/courteous"

A fast, multithreaded SOCKS proxy tester written in Go that supports both authenticated and non-authenticated proxies.

## Features

- ✅ Support for SOCKS5 proxies (with and without authentication)
- ✅ Test single proxy via CLI or batch test from file
- ✅ Multithreaded testing for high performance
- ✅ Configurable timeout
- ✅ Configurable test URL (default: https://kodelatte.com/)
- ✅ Export successful proxies to file
- ✅ Detailed latency measurements
- ✅ Verbose mode for debugging

## Installation

### Option 1: Download Pre-built Binary (Recommended)

Download the latest release for your platform from the [Releases page](https://github.com/sphinxid/sopan/releases).

```bash
# Linux/macOS/FreeBSD - make it executable
chmod +x sopan-*

# Run
./sopan-* -h
```

### Option 2: Build from Source

```bash
# Clone the repository
git clone https://github.com/sphinxid/sopan.git
cd sopan

# Download dependencies
go mod download

# Build the binary
go build -o sopan

# Or install directly
go install
```

## Usage

### Test a Single Proxy

```bash
# Without authentication
./sopan -proxy socks5://127.0.0.1:1080

# With authentication
./sopan -proxy socks5://username:password@127.0.0.1:1080

# You can also omit the protocol (defaults to socks5)
./sopan -proxy 127.0.0.1:1080
./sopan -proxy username:password@127.0.0.1:1080
```

### Test Multiple Proxies from File

```bash
# Basic usage
./sopan -file proxies.txt

# With custom thread count and timeout
./sopan -file proxies.txt -threads 20 -timeout 10

# Save successful proxies to output file
./sopan -file proxies.txt -output working-proxies.txt

# Verbose mode (show all results including failures)
./sopan -file proxies.txt -verbose
```

## Command Line Options

| Flag | Description | Default |
|------|-------------|---------|
| `-proxy` | Single proxy to test (format: `socks5://[user:pass@]host:port`) | - |
| `-file` | File containing list of proxies (one per line) | - |
| `-threads` | Number of concurrent threads | 10 |
| `-timeout` | Timeout in seconds for each proxy test | 5 |
| `-url` | URL to test proxies against | https://kodelatte.com/ |
| `-output` | Output file for successful proxies | - |
| `-verbose` | Show all results including failures | false |

## Proxy File Format

Create a text file with one proxy per line. Lines starting with `#` are treated as comments.

**Example `example-proxies-file.txt`:**

```
# SOCKS5 proxies without auth
127.0.0.1:1080
192.168.1.100:1080
socks5://10.0.0.1:1080

# SOCKS5 proxies with auth
username:password@proxy.example.com:1080
socks5://user:pass@192.168.1.200:1080

# More proxies
admin:secret123@10.10.10.10:9050
```

## Examples

### Example 1: Quick Test with Default Settings

```bash
./sopan -file proxies.txt
```

Output:
```
Testing 5 proxies with 10 threads (timeout: 5s)
--------------------------------------------------------------------------------
--------------------------------------------------------------------------------
Results: 5 tested | 3 successful | 2 failed

Successful proxies:
  socks5://127.0.0.1:1080 (latency: 234ms)
  socks5://user:pass@192.168.1.200:1080 (latency: 456ms)
  socks5://admin:secret123@10.10.10.10:9050 (latency: 189ms)
```

### Example 2: High-Speed Testing with Many Threads

```bash
./sopan -file large-proxy-list.txt -threads 50 -timeout 3
```

### Example 3: Verbose Mode with Output File

```bash
./sopan -file proxies.txt -verbose -output working.txt
```

Output:
```
Testing 5 proxies with 10 threads (timeout: 5s)
--------------------------------------------------------------------------------
✓ [SUCCESS] socks5://127.0.0.1:1080 (latency: 234ms)
✗ [FAILED]  socks5://192.168.1.100:1080 - request failed: dial tcp: connection refused
✓ [SUCCESS] socks5://user:pass@192.168.1.200:1080 (latency: 456ms)
✗ [FAILED]  socks5://10.0.0.1:1080 - request failed: i/o timeout
✓ [SUCCESS] socks5://admin:secret123@10.10.10.10:9050 (latency: 189ms)
--------------------------------------------------------------------------------
Results: 5 tested | 3 successful | 2 failed

3 successful proxies saved to working.txt
```

### Example 4: Test Single Proxy with Custom Timeout

```bash
./sopan -proxy username:password@proxy.example.com:1080 -timeout 10 -verbose
```

### Example 5: Test with Custom URL

```bash
# Test against a different website
./sopan -file proxies.txt -url https://httpbin.org/ip

# Test against your own server
./sopan -file proxies.txt -url https://example.com -threads 20
```

## Performance Tips

- **Threads**: Increase `-threads` for faster testing of large proxy lists (e.g., 50-100 threads)
- **Timeout**: Reduce `-timeout` for faster testing, but may miss slower proxies
- **Network**: Testing speed depends on your network bandwidth and the target website

## How It Works

1. Parses proxy string or loads proxies from file
2. Creates a worker pool with specified number of threads
3. Each worker:
   - Creates a SOCKS5 dialer (with or without auth)
   - Establishes connection through the proxy
   - Makes HTTPS request to the test URL (default: https://kodelatte.com/)
   - Measures latency and validates response
4. Collects and displays results

## Requirements

- Go 1.21 or higher
- `golang.org/x/net` package (automatically installed via `go mod download`)

## Building from Source

```bash
# Build for current platform
go build -o sopan

# Or use the build script
./build.sh

# Build for all platforms
./build.sh all

# Manual cross-compilation
GOOS=linux GOARCH=amd64 go build -o sopan-linux
GOOS=windows GOARCH=amd64 go build -o sopan.exe
GOOS=darwin GOARCH=amd64 go build -o sopan-mac
```

## Releases

Pre-built binaries are available for download from the [Releases page](https://github.com/sphinxid/sopan/releases).

### Creating a New Release

To create a new release with automated binary builds:

```bash
# Tag the release
git tag -a v1.0.0 -m "Release v1.0.0"

# Push the tag to GitHub
git push origin v1.0.0
```

The GitHub Actions workflow will automatically:
- Build binaries for Linux, Windows, macOS, and FreeBSD (AMD64 and ARM64)
- Generate SHA256 checksums
- Create a GitHub release with all binaries attached

## Troubleshooting

**Problem**: "connection refused" errors  
**Solution**: Verify the proxy is running and accessible from your network

**Problem**: "i/o timeout" errors  
**Solution**: Increase the timeout value with `-timeout` flag

**Problem**: All proxies failing  
**Solution**: Test with a known working proxy first, check your internet connection

**Problem**: Slow testing speed  
**Solution**: Increase thread count with `-threads` flag

## License

MIT License - feel free to use and modify as needed.

## Contributing

Contributions are welcome! Feel free to submit issues or pull requests.
