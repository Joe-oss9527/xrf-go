# CLAUDE.md

Project memory file for XRF-Go v1.0.0-RC2 - provides specific development guidance and conventions for Claude Code.

## Project Overview

XRF-Go is a production-ready Xray installation and configuration tool with 100% DESIGN.md alignment. Core design: "High efficiency, ultra-fast, extremely easy to use" with multi-configuration concurrent operation achieving ~0.12ms protocol addition speed.

## Code Architecture Requirements

### Module Structure (Enforce Strictly)
- `cmd/xrf/` - Single main CLI entry point
- `pkg/config/` - Use ConfigManager for all configuration operations
- `pkg/system/` - System detection via detector.go, service management via service.go
- `pkg/tls/` - ACME operations via acme.go, Caddy management via caddy.go
- `pkg/api/` - Xray gRPC client integration for runtime operations
- `pkg/utils/` - All utilities must use existing modules (crypto, validator, logger, etc.)

### Configuration File Conventions (Must Follow)
- Use `/etc/xray/confs/` directory only
- File naming: numeric prefix (00-09: base, 10-19: inbound, 20-29: outbound, 90-99: routing)
- Each protocol = one dedicated JSON file
- Always validate with `xray test -confdir` before writing
- Implement rollback mechanism for failed operations

## Development Commands (Use These Exact Commands)

### Building
- Production build: `go build -ldflags="-s -w" -o xrf cmd/xrf/main.go`
- Cross-compile Linux: `GOOS=linux GOARCH=amd64 go build -o xrf-linux-amd64 cmd/xrf/main.go`
- Cross-compile macOS: `GOOS=darwin GOARCH=arm64 go build -o xrf-darwin-arm64 cmd/xrf/main.go`

### Testing
- Run all tests: `go test ./...`
- TLS module tests: `go test -cover ./pkg/tls/`
- Configuration validation: `xray test -confdir /etc/xray/confs`

### Code Quality
- Format: `go fmt ./...`
- Lint: `golangci-lint run`
- Security: `go mod audit`

## Protocol Implementation Rules

### Supported Protocols (Use Exact Aliases)
- `vr` → VLESS-REALITY with xtls-rprx-vision flow
- `vw` → VLESS-WebSocket-TLS
- `vmess` → VMess-WebSocket-TLS
- `hu` → VLESS-HTTPUpgrade
- `tw` → Trojan-WebSocket-TLS
- `ss` → Shadowsocks
- `ss2022` → Shadowsocks-2022

### Configuration Templates
- All templates embedded in `pkg/config/templates.go`
- Use Go string constants only
- Include performance optimizations: `tcpKeepAliveIdle: 300`, `tcpUserTimeout: 10000`
- Apply BBR congestion control settings automatically

## CLI Command Implementation

### Core Operations (Must Implement Error Handling)
- `xrf install` - Use pkg/system/installer.go with GitHub API integration
- `xrf add [protocol]` - Must achieve <1ms performance, use ConfigManager.AddProtocol
- `xrf list` - Display protocols with color coding via utils/colors.go
- `xrf remove [tag]` - Implement with automatic backup and rollback
- `xrf info [tag]` - Show detailed config via structured output
- `xrf change [tag] [key] [value]` - Validate before change, rollback on failure

### Service Management (Use systemd Integration)
- All service commands must use pkg/system/service.go
- `xrf reload` - Send USR1 signal to Xray process
- `xrf check-port` - Use pkg/utils/network.go port validation

### TLS Automation (ACME Integration Required)
- Use pkg/tls/acme.go for Let's Encrypt operations
- Use pkg/tls/caddy.go for reverse proxy setup
- Implement 30-day auto-renewal mechanism

### Utility Commands
- `xrf generate [type]` - Use pkg/utils/crypto.go functions
- `xrf ip` - Use pkg/utils/http.go GetPublicIP function
- `xrf logs` - Integrate systemd journal reading

## Performance Requirements (Must Meet)

### Speed Targets
- Protocol addition: <1ms (current: ~0.12ms)
- Memory usage: <20MB
- Binary size: <10MB
- Configuration ops: >5000/sec

### Error Handling Strategy
- Use pkg/utils/errors.go for consistent error types
- Implement createAutoBackup() before any config change
- Call restoreFromBackup() on operation failure
- Validate with `xray test -confdir` after every change
- Provide specific fix suggestions in error messages

### Security Implementation
- systemd hardening: NoNewPrivileges=true, ProtectSystem=strict
- Port conflict detection via pkg/utils/network.go
- Certificate validation via pkg/utils/crypto.go
- BBR setup via system optimization commands

## Coding Standards (Enforce Strictly)

### Module Usage Rules
- Use existing ConfigManager methods, never direct file operations
- Use pkg/utils/logger.go for all logging with color support
- Use pkg/utils/validator.go for all input validation
- Use pkg/utils/crypto.go for UUID, password, key generation
- Use pkg/system/detector.go for OS detection before installation

### Error Handling Pattern
```go
// Always implement this pattern
if err := createAutoBackup(); err != nil {
    return fmt.Errorf("backup failed: %v", err)
}
if err := operation(); err != nil {
    restoreFromBackup()
    return fmt.Errorf("operation failed: %v", err)
}
if err := validateConfigAfterChange(); err != nil {
    restoreFromBackup()
    return fmt.Errorf("validation failed: %v", err)
}
```

### Testing Requirements
- Write unit tests for new functions
- Test with real Xray binary validation
- Benchmark performance-critical functions
- Test rollback mechanisms

## Dependencies and Constraints

### Runtime Requirements
- Xray binary: Auto-download from GitHub releases via pkg/system/installer.go
- System: Ubuntu/CentOS/Debian with systemd support
- Network: Port availability for protocols (use smart allocation)

### Build Requirements
- Go 1.19+ required
- No external dependencies beyond standard library
- Embed all templates in binary (no external files)

## Current Development Focus

### Completed (Production Ready)
- ✅ All 25+ CLI commands implemented
- ✅ DESIGN.md 100% alignment achieved
- ✅ Performance target exceeded (0.12ms vs 1ms goal)
- ✅ Full system management infrastructure
- ✅ TLS automation with ACME + Caddy integration

### Quality Improvement Priorities
- Unit test coverage expansion
- Error message improvements
- Input validation enhancement
- Documentation updates