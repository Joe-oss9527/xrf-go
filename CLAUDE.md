# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

XRF-Go is a streamlined Xray installation and configuration tool designed for efficiency and ease of use. The project follows the principle of "High efficiency, ultra-fast, extremely easy to use" with multi-configuration concurrent operation as its core design feature.

## Architecture

### Multi-Configuration File Strategy
The project leverages Xray's `-confdir` feature to manage multiple configurations simultaneously:
- Configuration files are stored in `/etc/xray/confs/`
- Files are named with numeric prefixes (00-99) to control loading order
- Each protocol gets its own configuration file for modular management
- Xray automatically merges configurations based on specific rules

### Core Module Structure
```
cmd/xrf/           - Main CLI entry point with all commands
pkg/config/        - Configuration management (multi-file handling)
pkg/system/        - System operations (OS detection, service management)
pkg/tls/           - TLS/certificate management (ACME, Caddy integration)
pkg/api/           - Xray API client for runtime management
pkg/utils/         - Utilities (logging, validation, HTTP tools)
```

## Common Development Commands

### Building the Project
```bash
# Build the binary
go build -o xrf cmd/xrf/main.go

# Build with optimizations
go build -ldflags="-s -w" -o xrf cmd/xrf/main.go

# Cross-compile for different platforms
GOOS=linux GOARCH=amd64 go build -o xrf-linux-amd64 cmd/xrf/main.go
GOOS=darwin GOARCH=arm64 go build -o xrf-darwin-arm64 cmd/xrf/main.go
```

### Testing
```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific package tests
go test ./pkg/config/
go test ./pkg/system/

# Test configuration validation
xray test -confdir /etc/xray/confs
```

### Code Quality
```bash
# Format code
go fmt ./...

# Run linter
golangci-lint run

# Check for vulnerabilities
go mod audit
```

## Key Implementation Details

### Configuration File Naming Convention
Files follow a strict naming pattern for proper loading order:
- `00-09`: Base configurations (log, API, stats)
- `10-19`: Inbound protocols
- `20-29`: Outbound configurations
- `90-99`: Routing rules (with tail support)

Example: `10-inbound-vless.json`, `20-outbound-direct.json`

### Protocol Shortcuts
The tool supports protocol aliases for quick configuration:
- `vr` → VLESS-REALITY
- `vw` → VLESS-WebSocket-TLS
- `tw` → Trojan-WebSocket-TLS
- `ss` → Shadowsocks
- `ss2022` → Shadowsocks-2022
- `hu` → VLESS-HTTPUpgrade

### Service Management
The Xray service is configured to run in confdir mode:
```bash
ExecStart=/usr/local/bin/xray run -confdir /etc/xray/confs
```

### Configuration Templates
All configuration templates are embedded in the binary as Go string constants in `pkg/config/templates.go`. This ensures single-binary portability without external dependencies.

## Important Conventions

### Error Handling
- Operations should fail gracefully with rollback capability
- Provide clear error messages with fix suggestions
- Validate configurations before applying changes

### Performance Optimizations
- Automatic BBR congestion control setup
- Socket optimization parameters in all protocols
- Smart port allocation to avoid conflicts

### Multi-Configuration Rules
When working with Xray's confdir feature:
1. Inbounds are appended to the array
2. Outbounds are prepended (unless marked as "tail")
3. Same-tag configurations replace each other
4. Routing rules can be added to beginning or end

## Supported Protocols (2025)

- VLESS-REALITY (with xtls-rprx-vision flow)
- VLESS-WebSocket-TLS
- VLESS-HTTPUpgrade
- VMess-WebSocket-TLS
- Trojan-WebSocket-TLS
- Shadowsocks (including 2022 variants)

All protocols include modern optimizations like:
- `tcpKeepAliveIdle: 300`
- `tcpUserTimeout: 10000`
- Proper socket buffer configurations

## Quick Development Tips

1. **Adding a New Protocol**: Create a new template in `pkg/config/templates.go` and add to `SupportedProtocols` list
2. **Modifying Configurations**: Use the ConfigManager's file operations to maintain proper file naming
3. **Testing Changes**: Always validate with `xray test -confdir` before applying
4. **Hot Reload**: Configurations support hot reload via USR1 signal to Xray process

## Dependencies

The project aims for minimal external dependencies:
- Xray-core binary (downloaded during installation)
- Standard Go libraries
- Cobra for CLI framework (if used)
- No external configuration files - everything embedded in binary